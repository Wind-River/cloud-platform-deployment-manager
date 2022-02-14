/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package ptpparameters

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*PTPParameter, error) {
	var s PTPParameter
	err := r.ExtractInto(&s)
	return &s, err
}

type commonResult struct {
	gophercloud.Result
}

// GetResult represents the result of a get operation.
type GetResult struct {
	commonResult
}

// UpdateResult represents the result of an update operation.
type UpdateResult struct {
	commonResult
}

// CreateResult represents the result of a create operation.
type CreateResult struct {
	commonResult
}

// DeleteResult represents the result of an delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// PTPParameter defines the data associated to a single PTP parameter.
type PTPParameter struct {
	// UUID is generated unique UUID for the ptp parameter.
	UUID string `json:"uuid"`

	// Name is the key of the PTP parameter.
	Name string `json:"name"`

	// Value is the value of the PTP parameter.
	Value string `json:"value"`

	// Owners is a list of UUID of PTP instance or PTP interface that the PTP
	// parameter is assigned to.
	Owners []string `json:"owners,omitempty"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// PTPParameterPage is the page returned by a pager when traversing over a
// collection of PTPInterfaces.
type PTPParameterPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a PTPParameterPage struct is empty.
func (r PTPParameterPage) IsEmpty() (bool, error) {
	is, err := ExtractPTPParameters(r)
	return len(is) == 0, err
}

// ExtractPTPParameters accepts a Page struct, specifically a PTPParameterPage
// struct, and extracts the elements into a slice of PTPParameter structs. In
// other words, a generic collection is mapped into a relevant slice.
func ExtractPTPParameters(r pagination.Page) ([]PTPParameter, error) {
	var s struct {
		PTPParameter []PTPParameter `json:"ptp_parameters"`
	}

	err := (r.(PTPParameterPage)).ExtractInto(&s)

	return s.PTPParameter, err
}

