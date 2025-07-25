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

package registrar

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	helper "github.com/elastic/elastic-agent-libs/file"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

type registryVersion string

const (
	noRegistry    registryVersion = ""
	legacyVersion registryVersion = "<legacy>"
	version0      registryVersion = "0"
	version1      registryVersion = "1"
)

const currentVersion = version1

type Migrator struct {
	dataPath    string
	migrateFile string
	permissions os.FileMode
	logger      *logp.Logger
}

func NewMigrator(cfg config.Registry, logger *logp.Logger) *Migrator {
	path := paths.Resolve(paths.Data, cfg.Path)
	migrateFile := cfg.MigrateFile
	if migrateFile != "" {
		migrateFile = paths.Resolve(paths.Data, migrateFile)
	}

	return &Migrator{
		dataPath:    path,
		migrateFile: migrateFile,
		permissions: cfg.Permissions,
		logger:      logger,
	}
}

// Run checks the on disk registry version and updates
// old on disk and data layouts to the current supported storage format.
func (m *Migrator) Run() error {
	migrateFile := m.migrateFile
	if migrateFile == "" {
		if isFile(m.dataPath, m.logger) {
			migrateFile = m.dataPath
		}
	}

	fbRegHome := filepath.Join(m.dataPath, "filebeat")
	version, err := readVersion(fbRegHome, migrateFile, m.logger)
	if err != nil {
		return err
	}

	m.logger.Named("registrar").Debugf("Registry type '%v' found", version)

	for {
		switch version {
		case legacyVersion:
			// first migrate to verion0 style registry and go from there
			if err := m.updateToVersion0(fbRegHome, migrateFile); err != nil {
				return err
			}
			fallthrough
		case version0:
			return m.updateToVersion1(fbRegHome)

		case currentVersion:
			return nil

		case noRegistry:
			// check if we've been in the middle of a migration from the legacy
			// format to the current version. If so continue with
			// the migration and try again.
			backupFile := migrateFile + ".bak"
			if isFile(backupFile, m.logger) {
				migrateFile = backupFile
				version = legacyVersion
				break
			}

			// postpone registry creation, until we open and configure it.
			return nil
		default:
			return fmt.Errorf("registry file version %v not supported", version)
		}
	}
}

// updateToVersion0 converts a single old registry file to a version 0 style registry.
// The version 0 registry is a directory, that contains two files: meta.json, and data.json.
// The meta.json reports the current storage version, and the data.json
// contains a JSON array with all entries known to the registrar.
//
// The data representation has changed multiple times before version 0 was introduced. The update function tries
// to fix the state, allow us to migrarte from older Beats registry files.
// NOTE: The oldest known filebeat registry file format this was tested with is from Filebeat 6.3.
// Support for older Filebeat versions is best effort.
func (m *Migrator) updateToVersion0(regHome, migrateFile string) error {
	m.logger.Info("Migrate registry file to registry directory")

	if m.dataPath == migrateFile {
		backupFile := migrateFile + ".bak"
		if isFile(migrateFile, m.logger) {
			m.logger.Infof("Move registry file to backup file: %v", backupFile)
			if err := helper.SafeFileRotate(backupFile, migrateFile); err != nil {
				return err
			}
			migrateFile = backupFile
		} else if isFile(backupFile, m.logger) {
			m.logger.Info("Old registry backup file found, continue migration")
			migrateFile = backupFile
		}
	}

	if err := initVersion0Registry(regHome, m.permissions, m.logger); err != nil {
		return err
	}

	dataFile := filepath.Join(regHome, "data.json")
	if !isFile(dataFile, m.logger) && isFile(migrateFile, m.logger) {
		m.logger.Info("Migrate old registry file to new data file")
		err := helper.SafeFileRotate(dataFile, migrateFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func initVersion0Registry(regHome string, perm os.FileMode, logger *logp.Logger) error {
	if !isDir(regHome, logger) {
		logger.Infof("No registry home found. Create: %v", regHome)
		if err := os.MkdirAll(regHome, 0o750); err != nil {
			return fmt.Errorf("failed to create registry dir '%v': %w", regHome, err)
		}
	}

	metaFile := filepath.Join(regHome, "meta.json")
	if !isFile(metaFile, logger) {
		logger.Info("Initialize registry meta file")
		err := safeWriteFile(metaFile, []byte(`{"version": "0"}`), perm)
		if err != nil {
			return fmt.Errorf("failed writing registry meta.json: %w", err)
		}
	}

	return nil
}

// updateToVersion1 updates the filebeat registry from version 0 to version 1
// only. Version 1 is based on the implementation of version 1 in
// libbeat/statestore/backend/memlog.
func (m *Migrator) updateToVersion1(regHome string) error {
	m.logger.Info("Migrate registry version 0 to version 1")

	origDataFile := filepath.Join(regHome, "data.json")
	if !isFile(origDataFile, m.logger) {
		return fmt.Errorf("missing original data file at: %v", origDataFile)
	}

	// read states from file and ensure file is closed immediately.
	states, err := func() ([]file.State, error) {
		origIn, err := os.Open(origDataFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open original data file: %w", err)
		}
		defer origIn.Close()

		var states []file.State
		decoder := json.NewDecoder(origIn)
		if err := decoder.Decode(&states); err != nil {
			return nil, fmt.Errorf("error decoding original data file '%v': %w", origDataFile, err)
		}
		return states, nil
	}()
	if err != nil {
		return err
	}

	states = resetStates(fixStates(states, m.logger))

	registryBackend, err := memlog.New(m.logger.Named("migration"), memlog.Settings{
		Root:               m.dataPath,
		FileMode:           m.permissions,
		Checkpoint:         func(sz uint64) bool { return false },
		IgnoreVersionCheck: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create new registry backend: %w", err)
	}
	defer registryBackend.Close()

	store, err := registryBackend.Access("filebeat")
	if err != nil {
		return fmt.Errorf("failed to open filebeat registry store: %w", err)
	}
	defer store.Close()

	if err := writeStates(store, states); err != nil {
		return fmt.Errorf("failed to migrate registry states: %w", err)
	}

	if checkpointer, ok := store.(interface{ Checkpoint() error }); ok {
		err := checkpointer.Checkpoint()
		if err != nil {
			return fmt.Errorf("failed to fsync filebeat storage state: %w", err)
		}
	}

	if err := os.Remove(origDataFile); err != nil {
		return fmt.Errorf("migration complete but failed to remove original data file: %v: %w", origDataFile, err)
	}

	if err := os.WriteFile(filepath.Join(regHome, "meta.json"), []byte(`{"version": "1"}`), m.permissions); err != nil {
		return fmt.Errorf("failed to update the meta.json file: %w", err)
	}

	return nil
}

func readVersion(regHome, migrateFile string, log *logp.Logger) (registryVersion, error) {
	if isFile(migrateFile, log) {
		return legacyVersion, nil
	}

	if !isDir(regHome, log) {
		return noRegistry, nil
	}

	metaFile := filepath.Join(regHome, "meta.json")
	if !isFile(metaFile, log) {
		return noRegistry, nil
	}

	tmp, err := os.ReadFile(metaFile)
	if err != nil {
		return noRegistry, fmt.Errorf("failed to open meta file: %w", err)
	}

	meta := struct{ Version string }{}
	if err := json.Unmarshal(tmp, &meta); err != nil {
		return noRegistry, fmt.Errorf("failed reading meta file: %w", err)
	}

	return registryVersion(meta.Version), nil
}

func isDir(path string, log *logp.Logger) bool {
	fi, err := os.Stat(path)
	exists := err == nil && fi.IsDir()
	log.Named("test").Debugf("isDir(%v) -> %v", path, exists)
	return exists
}

func isFile(path string, log *logp.Logger) bool {
	fi, err := os.Stat(path)
	exists := err == nil && fi.Mode().IsRegular()
	log.Named("test").Debugf("isFile(%v) -> %v", path, exists)
	return exists
}

func safeWriteFile(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	for len(data) > 0 {
		var n int
		n, err = f.Write(data)
		if err != nil {
			break
		}

		data = data[n:]
	}

	if err == nil {
		err = f.Sync()
	}

	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

// fixStates cleans up the registry states when updating from an older version
// of filebeat potentially writing invalid entries.
func fixStates(states []file.State, logger *logp.Logger) []file.State {
	if len(states) == 0 {
		return states
	}

	// we use a map of states here, so to identify and merge duplicate entries.
	idx := map[string]*file.State{}
	for i := range states {
		state := &states[i]
		fixState(state, logger)

		old, exists := idx[state.Id]
		if !exists {
			idx[state.Id] = state
		} else {
			mergeStates(old, state) // overwrite the entry in 'old'
		}
	}

	if len(idx) == len(states) {
		return states
	}

	i := 0
	newStates := make([]file.State, len(idx))
	for _, state := range idx {
		newStates[i] = *state
		i++
	}
	return newStates
}

// fixState updates a read state to fullfil required invariantes:
// - "Meta" must be nil if len(Meta) == 0
// - "Id" must be initialized
func fixState(st *file.State, logger *logp.Logger) {
	if len(st.Meta) == 0 {
		st.Meta = nil
	}

	if len(st.IdentifierName) == 0 {
		identifier, _ := file.NewStateIdentifier(nil, logger)
		st.Id, st.IdentifierName = identifier.GenerateID(*st)
	}
}

// resetStates sets all states to finished and disable TTL on restart
// For all states covered by an input, TTL will be overwritten with the input value
func resetStates(states []file.State) []file.State {
	for key, state := range states {
		state.Finished = true
		// Set ttl to -2 to easily spot which states are not managed by a input
		state.TTL = -2
		states[key] = state
	}
	return states
}

// mergeStates merges 2 states by trying to determine the 'newer' state.
// The st state is overwritten with the updated fields.
func mergeStates(st, other *file.State) {
	st.Finished = st.Finished || other.Finished
	if st.Offset < other.Offset { // always select the higher offset
		st.Offset = other.Offset
	}

	// update file meta-data. As these are updated concurrently by the
	// inputs, select the newer state based on the update timestamp.
	var meta, metaOld, metaNew map[string]string
	if st.Timestamp.Before(other.Timestamp) {
		st.Source = other.Source
		st.Timestamp = other.Timestamp
		st.TTL = other.TTL
		st.FileStateOS = other.FileStateOS

		metaOld, metaNew = st.Meta, other.Meta
	} else {
		metaOld, metaNew = other.Meta, st.Meta
	}

	if len(metaOld) == 0 || len(metaNew) == 0 {
		meta = metaNew
	} else {
		meta = map[string]string{}
		for k, v := range metaOld {
			meta[k] = v
		}
		for k, v := range metaNew {
			meta[k] = v
		}
	}

	if len(meta) == 0 {
		meta = nil
	}
	st.Meta = meta
}
