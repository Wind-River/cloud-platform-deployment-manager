/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2025 Wind River Systems, Inc. */

package v1

import (
	"reflect"
	"sort"
)

func areSlicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Create copies to avoid modifying original slices
	aCopy := make([]string, len(a))
	copy(aCopy, a)
	bCopy := make([]string, len(b))
	copy(bCopy, b)

	sort.Strings(aCopy)
	sort.Strings(bCopy)

	return reflect.DeepEqual(aCopy, bCopy)
}

// deepequal-gen does not seem to iterate through []string of map[string][]string for DeepEqual,
// forced not to generate DeepEqual, and created DeepEqual manually here
// deeply comparing the receiver with other, it must be non-nil. It compares unordered list.
func (in *PtpInstanceSpec) DeepEqual(other *PtpInstanceSpec) bool {
	if other == nil {
		return false
	}

	if in.Service != other.Service {
		return false
	}
	if ((in.InstanceParameters != nil) && (other.InstanceParameters != nil)) || ((in.InstanceParameters == nil) != (other.InstanceParameters == nil)) {
		in, other := &in.InstanceParameters, &other.InstanceParameters
		if other == nil {
			return false
		}

		if len(*in) != len(*other) {
			return false
		} else {
			for i, inElement := range *in {
				if areSlicesEqualUnordered(inElement, (*other)[i]) == false {
					return false
				}
			}
		}
	}

	return true
}
