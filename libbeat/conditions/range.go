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

package conditions

import (
	"fmt"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
)

type rangeValue struct {
	gte *float64
	gt  *float64
	lte *float64
	lt  *float64
}

// Range is a Condition type for checking against ranges.
type Range struct {
	rangemap map[string]rangeValue
	logger   *logp.Logger
}

// NewRangeCondition builds a new Range from a map of ranges.
func NewRangeCondition(config map[string]interface{}, log *logp.Logger) (c Range, err error) {
	c = Range{logger: log, rangemap: make(map[string]rangeValue)}

	updateRangeValue := func(key string, op string, value float64) error {
		field := strings.TrimSuffix(key, "."+op)
		_, exists := c.rangemap[field]
		if !exists {
			c.rangemap[field] = rangeValue{}
		}
		rv := c.rangemap[field]
		switch op {
		case "gte":
			rv.gte = &value
		case "gt":
			rv.gt = &value
		case "lt":
			rv.lt = &value
		case "lte":
			rv.lte = &value
		default:
			return fmt.Errorf("unexpected range operator %s", op)
		}
		c.rangemap[field] = rv
		return nil
	}

	for key, value := range config {

		floatValue, err := ExtractFloat(value)
		if err != nil {
			return c, err
		}

		list := strings.Split(key, ".")
		err = updateRangeValue(key, list[len(list)-1], floatValue)
		if err != nil {
			return c, err
		}

	}

	return c, nil
}

// Check determines whether the given event matches this condition.
func (c Range) Check(event ValuesMap) bool {
	checkValue := func(value float64, rangeValue rangeValue) bool {
		if rangeValue.gte != nil {
			if value < *rangeValue.gte {
				return false
			}
		}
		if rangeValue.gt != nil {
			if value <= *rangeValue.gt {
				return false
			}
		}
		if rangeValue.lte != nil {
			if value > *rangeValue.lte {
				return false
			}
		}
		if rangeValue.lt != nil {
			if value >= *rangeValue.lt {
				return false
			}
		}
		return true
	}

	for field, rangeValue := range c.rangemap {

		value, err := event.GetValue(field)
		if err != nil {
			return false
		}

		floatValue, err := ExtractFloat(value)
		if err != nil {
			c.logger.Named(logName).Warnf(err.Error())
			return false
		}

		if !checkValue(floatValue, rangeValue) {
			return false
		}

	}
	return true
}

func (c Range) String() string {
	return fmt.Sprintf("range: %v", map[string]rangeValue(c.rangemap))
}
