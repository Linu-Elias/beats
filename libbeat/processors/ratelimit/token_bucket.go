// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package ratelimit

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jonboulle/clockwork"

	"github.com/elastic/go-concert/unison"

	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	register("token_bucket", newTokenBucket)
}

type bucket struct {
	mu sync.Mutex

	tokens        float64
	lastReplenish time.Time
}

func (b *bucket) withdraw() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

func (b *bucket) replenish(rate rate, clock clockwork.Clock) float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	secsSinceLastReplenish := clock.Now().Sub(b.lastReplenish).Seconds()
	tokensToReplenish := secsSinceLastReplenish * rate.valuePerSecond()

	b.tokens += tokensToReplenish
	b.lastReplenish = clock.Now()
	return b.tokens
}

type tokenBucket struct {
	mu unison.Mutex

	limit   rate
	depth   float64
	buckets sync.Map

	// GC thresholds and metrics
	gc struct {
		thresholds tokenBucketGCConfig
		metrics    struct {
			numCalls atomic.Uint64
		}
	}

	clock  clockwork.Clock
	logger *logp.Logger
}

type tokenBucketGCConfig struct {
	// NumCalls is the number of calls made to IsAllowed. When more than
	// the specified number of calls are made, GC is performed.
	NumCalls uint `config:"num_calls"`
}

type tokenBucketConfig struct {
	BurstMultiplier float64 `config:"burst_multiplier"`

	// GC governs when completely filled token buckets must be deleted
	// to free up memory. GC is performed when _any_ of the GC conditions
	// below are met. After each GC, counters corresponding to _each_ of
	// the GC conditions below are reset.
	GC tokenBucketGCConfig `config:"gc"`
}

func newTokenBucket(config algoConfig, logger *logp.Logger) (algorithm, error) {
	cfg := tokenBucketConfig{
		BurstMultiplier: 1.0,
		GC: tokenBucketGCConfig{
			NumCalls: 10000,
		},
	}

	if err := config.config.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("could not unpack token_bucket algorithm configuration: %w", err)
	}

	return &tokenBucket{
		limit:   config.limit,
		depth:   config.limit.value * cfg.BurstMultiplier,
		buckets: sync.Map{},
		gc: struct {
			thresholds tokenBucketGCConfig
			metrics    struct {
				numCalls atomic.Uint64
			}
		}{
			thresholds: tokenBucketGCConfig{
				NumCalls: cfg.GC.NumCalls,
			},
		},
		clock:  clockwork.NewRealClock(),
		logger: logger.Named("token_bucket"),
		mu:     unison.MakeMutex(),
	}, nil
}

func (t *tokenBucket) IsAllowed(key uint64) bool {
	t.runGC()

	b := t.getBucket(key)
	allowed := b.withdraw()

	t.gc.metrics.numCalls.Add(1)
	return allowed
}

// setClock allows test code to inject a fake clock
func (t *tokenBucket) setClock(c clockwork.Clock) {
	t.clock = c
}

func (t *tokenBucket) getBucket(key uint64) *bucket {
	v, exists := t.buckets.Load(key)
	if exists {
		//nolint:errcheck // ignore
		b := v.(*bucket)
		b.replenish(t.limit, t.clock)
		return b
	}

	v, _ = t.buckets.LoadOrStore(key, &bucket{
		tokens:        t.depth,
		lastReplenish: t.clock.Now(),
	})
	//nolint:errcheck // ignore
	return v.(*bucket)
}

func (t *tokenBucket) runGC() {
	// Don't run GC if thresholds haven't been crossed.
	if t.gc.metrics.numCalls.Load() < uint64(t.gc.thresholds.NumCalls) {
		return
	}

	if !t.mu.TryLock() {
		return
	}

	go func() {
		defer t.mu.Unlock()
		gcStartTime := time.Now()

		// Add tokens to all buckets according to the rate limit
		// and flag full buckets for deletion.
		toDelete := make([]uint64, 0)
		numBucketsBefore := 0
		t.buckets.Range(func(k, v interface{}) bool {
			//nolint:errcheck // ignore
			key := k.(uint64)
			//nolint:errcheck // ignore
			b := v.(*bucket)

			tokens := b.replenish(t.limit, t.clock)
			if tokens >= t.depth {
				toDelete = append(toDelete, key)
			}

			numBucketsBefore++
			return true
		})

		// Cleanup full buckets to free up memory
		for _, key := range toDelete {
			t.buckets.Delete(key)
		}

		// Reset GC metrics
		t.gc.metrics.numCalls.Store(0)

		gcDuration := time.Since(gcStartTime)
		numBucketsDeleted := len(toDelete)
		numBucketsAfter := numBucketsBefore - numBucketsDeleted
		t.logger.Debugf("gc duration: %v, buckets: (before: %v, deleted: %v, after: %v)",
			gcDuration, numBucketsBefore, numBucketsDeleted, numBucketsAfter)
	}()
}
