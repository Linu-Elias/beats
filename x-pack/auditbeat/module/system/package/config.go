// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package pkg

import (
	"time"
)

// config defines the package metricset's configuration options.
type config struct {
	StatePeriod        time.Duration `config:"state.period"`
	PackageStatePeriod time.Duration `config:"package.state.period"`
	// PackageSuidDrop runs RPM queries with suid to drop out of root
	// see the comment in package.go for more context
	PackageSuidDrop *int64 `config:"package.rpm_drop_to_uid"`
}

func (c *config) effectiveStatePeriod() time.Duration {
	if c.PackageStatePeriod != 0 {
		return c.PackageStatePeriod
	}
	return c.StatePeriod
}

func defaultConfig() config {
	return config{
		StatePeriod: 12 * time.Hour,
	}
}
