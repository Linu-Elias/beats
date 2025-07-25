// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"

	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

// limiter, is used to limit the number of goroutines from blowing up the stack
type limiter struct {
	wg sync.WaitGroup
	// limit specifies the maximum number
	// of concurrent jobs to perform.
	limit chan struct{}
}

// acquire gets an available worker thread.
func (l *limiter) acquire() {
	l.wg.Add(1)
	l.limit <- struct{}{}
}

func (l *limiter) wait() {
	l.wg.Wait()
}

// release puts pack a worker thread.
func (l *limiter) release() {
	<-l.limit
	l.wg.Done()
}

type scheduler struct {
	publisher  cursor.Publisher
	client     *azcontainer.Client
	credential *serviceCredentials
	src        *Source
	cfg        *config
	state      *state
	log        *logp.Logger
	limiter    *limiter
	serviceURL string
	status     status.StatusReporter
	metrics    *inputMetrics
}

// newScheduler, returns a new scheduler instance
func newScheduler(publisher cursor.Publisher, client *azcontainer.Client,
	credential *serviceCredentials, src *Source, cfg *config,
	state *state, serviceURL string, stat status.StatusReporter, metrics *inputMetrics, log *logp.Logger,
) *scheduler {
	if metrics == nil {
		// metrics are optional, initialize a stub if not provided
		metrics = newInputMetrics("", nil)
	}
	return &scheduler{
		publisher:  publisher,
		client:     client,
		credential: credential,
		src:        src,
		cfg:        cfg,
		state:      state,
		log:        log,
		limiter:    &limiter{limit: make(chan struct{}, src.MaxWorkers)},
		serviceURL: serviceURL,
		status:     stat,
		metrics:    metrics,
	}
}

// schedule, is responsible for fetching & scheduling jobs using the workerpool model
func (s *scheduler) schedule(ctx context.Context) error {
	if !s.src.Poll {
		return s.scheduleOnce(ctx)
	}

	for {
		err := s.scheduleOnce(ctx)
		if err != nil {
			return err
		}

		err = timed.Wait(ctx, s.src.PollInterval)
		if err != nil {
			s.metrics.errorsTotal.Inc()
			return err
		}
	}
}

func (s *scheduler) scheduleOnce(ctx context.Context) error {
	defer s.limiter.wait()
	pager := s.fetchBlobPager(int32(s.src.BatchSize))
	fileSelectorLen := len(s.src.FileSelectors)
	var numBlobs, numJobs int

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			s.metrics.errorsTotal.Inc()
			s.status.UpdateStatus(status.Failed, "failed to fetch next page during pagination: "+err.Error())
			return err
		}

		numBlobs += len(resp.Segment.BlobItems)
		s.log.Debugf("scheduler: %d blobs fetched for current batch", len(resp.Segment.BlobItems))
		s.metrics.absBlobsListedTotal.Add(uint64(len(resp.Segment.BlobItems)))

		var jobs []*job
		for _, v := range resp.Segment.BlobItems {
			// if file selectors are present, then only select the files that match the regex
			if fileSelectorLen != 0 && !s.isFileSelected(*v.Name) {
				continue
			}
			// date filter is applied on last modified time of the blob
			if s.src.TimeStampEpoch != nil && v.Properties.LastModified.Unix() < *s.src.TimeStampEpoch {
				continue
			}
			blobURL := s.serviceURL + s.src.ContainerName + "/" + *v.Name
			blobCreds := &blobCredentials{
				serviceCreds:  s.credential,
				blobName:      *v.Name,
				containerName: s.src.ContainerName,
			}

			blobClient, err := fetchBlobClient(blobURL, blobCreds, *s.cfg, s.log)
			if err != nil {
				s.metrics.errorsTotal.Inc()
				s.log.Errorf("Job creation failed for container %s with error %v", s.src.ContainerName, err)
				s.status.UpdateStatus(status.Failed, "failed to fetch blob client while scheduling jobs: "+err.Error())
				return err
			}

			job := newJob(blobClient, v, blobURL, s.state, s.src, s.publisher, s.status, s.metrics, s.log)
			jobs = append(jobs, job)
		}

		// If previous checkpoint was saved then look up starting point for new jobs
		if !s.state.checkpoint().LatestEntryTime.IsZero() {
			jobs = s.moveToLastSeenJob(jobs)
		}

		s.log.Debugf("scheduler: %d jobs scheduled for current batch", len(jobs))
		s.metrics.absJobsScheduledAfterValidation.Update(int64(len(jobs)))
		numJobs += len(jobs)

		// distributes jobs among workers with the help of a limiter
		for i, job := range jobs {
			id := fetchJobID(i, s.src.ContainerName, job.name())
			job := job
			// sets the content type and encoding for the job blob properties based on the reader configuration.
			// If the override flags are set, it will use the provided content type and encoding. If not,
			// it will only set them if they are not already defined.
			readerCfg := s.src.ReaderConfig
			if readerCfg.ContentType != "" {
				if readerCfg.OverrideContentType || isStringUnset(job.blob.Properties.ContentType) {
					job.blob.Properties.ContentType = &readerCfg.ContentType
				}
			}
			if readerCfg.Encoding != "" {
				if readerCfg.OverrideEncoding || isStringUnset(job.blob.Properties.ContentEncoding) {
					job.blob.Properties.ContentEncoding = &readerCfg.Encoding
				}
			}
			// acquire a worker thread from the limiter, and schedule the job
			// to be executed in a goroutine.
			s.limiter.acquire()
			go func() {
				defer s.limiter.release()
				job.do(ctx, id)
			}()
		}

		s.log.Debugf("scheduler: total objects read till now: %d\nscheduler: total jobs scheduled till now: %d", numBlobs, numJobs)
		if len(jobs) != 0 {
			s.log.Debugf("scheduler: first job in current batch: %s\nscheduler: last job in current batch: %s", jobs[0].name(), jobs[len(jobs)-1].name())
		}
	}

	return nil
}

// fetchJobID returns a job id which is a combination of worker id, container name and blob name
func fetchJobID(workerId int, containerName string, blobName string) string {
	jobID := fmt.Sprintf("%s-%s-worker-%d", containerName, blobName, workerId)

	return jobID
}

// fetchBlobPager fetches the current blob page object given a batch size & a page marker.
// The page marker has been disabled since it was found that it operates on the basis of
// lexicographical order, and not on the basis of the latest file uploaded, meaning if a blob with a name
// of lesser lexicographic value is uploaded after a blob with a name of higher value, the latest
// marker stored in the checkpoint will not retrieve that new blob, this distorts the polling logic
// hence disabling it for now, until more feedback is given. Disabling this how ever makes the sheduler loop
// through all the blobs on every poll action to arrive at the latest checkpoint.
// [NOTE] : There are no api's / sdk functions that list blobs via timestamp/latest entry, it's always lexicographical order
func (s *scheduler) fetchBlobPager(batchSize int32) *azruntime.Pager[azblob.ListBlobsFlatResponse] {
	listBlobsFlatOptions := azcontainer.ListBlobsFlatOptions{
		Include: azcontainer.ListBlobsInclude{
			Metadata: true,
			Tags:     true,
		},
		MaxResults: &batchSize,
	}
	if s.src.PathPrefix != "" {
		listBlobsFlatOptions.Prefix = &s.src.PathPrefix
	}

	return s.client.NewListBlobsFlatPager(&listBlobsFlatOptions)
}

// moveToLastSeenJob, moves to the latest job position past the last seen job
// Jobs are stored in lexicographical order always, hence the latest position can be found either on the basis of job name or timestamp
func (s *scheduler) moveToLastSeenJob(jobs []*job) []*job {
	cp := s.state.checkpoint()
	jobs = slices.DeleteFunc(jobs, func(j *job) bool {
		return !(j.timestamp().After(cp.LatestEntryTime) || j.name() > cp.BlobName)
	})

	// In a scenario where there are some jobs which have a greater timestamp
	// but lesser lexicographic order and some jobs have greater lexicographic order
	// than the current checkpoint blob name, we then sort around the pivot checkpoint
	// timestamp.
	sort.SliceStable(jobs, func(i, _ int) bool {
		return jobs[i].timestamp().After(cp.LatestEntryTime)
	})
	return jobs
}

func (s *scheduler) isFileSelected(name string) bool {
	for _, sel := range s.src.FileSelectors {
		if sel.Regex == nil || sel.Regex.MatchString(name) {
			return true
		}
	}
	return false
}

func isStringUnset(s *string) bool {
	return s == nil || *s == ""
}
