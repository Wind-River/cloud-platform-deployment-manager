/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package ptpinstances

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*PTPInstance, error) {
	var s PTPInstance
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

// ApplyResult represents the result of an apply operation.
type ApplyResult struct {
	commonResult
}

// DeleteResult represents the result of an delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// PTPInstance defines the data associated to a single ptp instance.
type PTPInstance struct {
	// UUID is generated unique UUID for the ptp instance.
	UUID string `json:"uuid"`

	// ID is the resource ID of the ptp instance
	ID int `json:"id"`

	// Name is the human readable and assignable name of the ptp instance.
	Name string `json:"name"`

	// Service is the service protocol type of the ptp instance.
	Service string `json:"service"`

	// Hostnames is the list of the host names assigned to the ptp instance.
	HostNames []string `json:"hostnames,omitempty"`

	// Parameters is the list of the parameters assigned to the ptp instance.
	Parameters []string `json:"parameters,omitempty"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// PTPInstancePage is the page returned by a pager when traversing over a
// collection of PTPInstances.
type PTPInstancePage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a PTPInstancePage struct is empty.
func (r PTPInstancePage) IsEmpty() (bool, error) {
	is, err := ExtractPTPInstances(r)
	return len(is) == 0, err
}

// ExtractPTPInstances accepts a Page struct, specifically a PTPInstancePage
// struct, and extracts the elements into a slice of PTPInstance structs. In
// other words, a generic collection is mapped into a relevant slice.
func ExtractPTPInstances(r pagination.Page) ([]PTPInstance, error) {
	var s struct {
		PTPInstance []PTPInstance `json:"ptp_instances"`
	}

	err := (r.(PTPInstancePage)).ExtractInto(&s)

	return s.PTPInstance, err
}
