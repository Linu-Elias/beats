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

// Code generated by beats/dev-tools/cmd/asset/asset.go - DO NOT EDIT.

package postgresql

import (
	"github.com/elastic/beats/v7/libbeat/asset"
)

func init() {
	if err := asset.SetFields("metricbeat", "postgresql", asset.ModuleFieldsPri, AssetPostgresql); err != nil {
		panic(err)
	}
}

// AssetPostgresql returns asset data.
// This is the base64 encoded zlib format compressed contents of module/postgresql.
func AssetPostgresql() string {
	return "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"
}
