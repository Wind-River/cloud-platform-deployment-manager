/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package labels

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Label, error) {
	var s Label
	err := r.ExtractInto(&s)
	return &s, err
}

type commonResult struct {
	gophercloud.Result
}

// CreateResult represents the result of a create operation.
type CreateResult struct {
	commonResult
}

// DeleteResult represents the result of a delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// Label defines the data associated to a single Label instance.
type Label struct {
	// ID defines the system assigned unique UUID value
	ID string `json:"uuid"`

	// HostUUID defines the unique UUID value associated to the host.
	HostUUID string `json:"host_uuid"`

	// Key defines the name of the label instance.
	Key string `json:"label_key"`

	// Value defines the value of the label instance.
	Value string `json:"label_value"`
}

// LabelPage is the page returned by a pager when traversing over a
// collection of Labels.
type LabelPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a LabelPage struct is empty.
func (r LabelPage) IsEmpty() (bool, error) {
	is, err := ExtractLabels(r)
	return len(is) == 0, err
}

// ExtractLabels accepts a Page struct, specifically a
// LabelPage struct, and extracts the elements into a slice of
// Label structs. In other words, a generic collection is mapped into
// a relevant slice.
func ExtractLabels(r pagination.Page) ([]Label, error) {
	var s struct {
		Label []Label `json:"labels"`
	}

	err := (r.(LabelPage)).ExtractInto(&s)

	return s.Label, err
}
