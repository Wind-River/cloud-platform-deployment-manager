/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package ptp

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*PTP, error) {
	var s PTP
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

// GetResult represents the result of a get operation.
type GetResult struct {
	commonResult
}

// UpdateResult represents the result of an update operation.
type UpdateResult struct {
	commonResult
}

// DeleteResult represents the result of a delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// PTP defines the data associated to a single PTP instance.
type PTP struct {
	// ID defines the system NTP record
	ID string `json:"uuid"`

	// Enables defines whether PTP is enabled on the system or not
	Enabled bool `json:"enabled"`

	// Mode defines the PTP operating mode.
	Mode string `json:"mode"`

	// Transport defines the PTP transport mode.
	Transport string `json:"transport"`

	// Mechanism defines the PTP mechanism
	Mechanism string `json:"mechanism"`

	// SystemID is the unique UUID value of the system to which this resource
	// is associated.
	SystemID string `json:"isystem_uuid"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// PTPPage is the page returned by a pager when traversing over a
// collection of PTPs.
type PTPPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a PTPPage struct is empty.
func (r PTPPage) IsEmpty() (bool, error) {
	is, err := ExtractPTPs(r)
	return len(is) == 0, err
}

// ExtractPTPs accepts a Page struct, specifically a PTPPage struct,
// and extracts the elements into a slice of PTP structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractPTPs(r pagination.Page) ([]PTP, error) {
	var s struct {
		PTP []PTP `json:"ptps"`
	}

	err := (r.(PTPPage)).ExtractInto(&s)

	return s.PTP, err
}
