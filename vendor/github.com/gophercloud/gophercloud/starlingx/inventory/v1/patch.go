/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package v1

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// PatchOp represents a valid update operation.
type PatchOp string

const (
	AddOp     PatchOp = "add"
	ReplaceOp PatchOp = "replace"
	RemoveOp  PatchOp = "remove"
)

// PatchMap represents a patch property request.
type PatchMap struct {
	op    PatchOp
	name  string
	value string
}

type Patch interface {
	ToPatchMap() map[string]interface{}
}

func ConvertToPatchMap(obj interface{}, op PatchOp) ([]interface{}, error) {
	result := make(map[string]interface{})
	err := mapstructure.Decode(obj, &result)
	if err != nil {
		return nil, err
	}

	m := make([]interface{}, 0)
	for k := range result {
		reflectValue := reflect.ValueOf(result[k])
		if reflectValue.Kind() == reflect.Ptr && reflectValue.IsNil() {
			// TODO(alegacy): replace with omitempty when it is supported by
			//  mapstructure
			continue
		}

		value := result[k]
		if reflectValue.Kind() == reflect.Ptr && reflectValue.Elem().Kind() == reflect.Slice {
			// The patch operation at the system API does not support arrays
			// therefore if we find one in the set of attributes that need to
			// change then automatically convert it to a comma separate list.
			value = strings.Join(*value.(*[]string), ",")
			if value == "" {
				// The system API expects empty list as "none"
				value = "none"
			}
		}

		p := map[string]interface{}{
			"op":    op,
			"path":  fmt.Sprintf("/%s", k),
			"value": value,
		}

		m = append(m, p)
	}

	return m, nil
}

func ConvertToCreateMap(obj interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	err := mapstructure.Decode(obj, &result)
	if err != nil {
		return nil, err
	}

	m := make(map[string]interface{})
	for k := range result {
		value := reflect.ValueOf(result[k])
		if value.Kind() == reflect.Ptr && value.IsNil() {
			// TODO(alegacy): replace with omitempty when it is supported by
			//  mapstructure
			continue
		}

		m[k] = result[k]
	}

	return m, nil
}
