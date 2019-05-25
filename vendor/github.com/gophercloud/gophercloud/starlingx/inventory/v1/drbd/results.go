/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package drbd

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*DRBD, error) {
	var s DRBD
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

// UpdateResult represents the result of a create operation.
type UpdateResult struct {
	commonResult
}

// DRBD defines the data associated to a DRBD configuration instance.
type DRBD struct {
	// ID is the generated unique UUID for the DRBD configuration instance.
	ID string `json:"uuid"`

	// LinkUtilisation defines the maximum link utilisation percentage during
	// resync activities.
	LinkUtilization int `json:"link_util"`

	// ParallelDevices defines the number of devices to sync in parallel.
	ParallelDevices int `json:"num_parallel"`

	// RoundTripDelay defines the acceptable round-trip delay in milliseconds.
	RoundTripDelay float64 `json:"rtt_ms"`

	// SystemID is the system defined unique UUID of the system that these
	// settings related to.
	SystemID string `json:"isystem_uuid"`
}

// DRBDPage is the page returned by a pager when traversing over a
// collection of routes.
type DRBDPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a DRBDPage struct is empty.
func (r DRBDPage) IsEmpty() (bool, error) {
	is, err := ExtractDRBDs(r)
	return len(is) == 0, err
}

// ExtractDRBDs accepts a Page struct, specifically a DRBDPage struct, and
// extracts the elements into a slice of DRBD structs. In other words, a generic
// collection is mapped into a relevant slice.
func ExtractDRBDs(r pagination.Page) ([]DRBD, error) {
	var s struct {
		DRBD []DRBD `json:"drbdconfigs"`
	}

	err := (r.(DRBDPage)).ExtractInto(&s)

	return s.DRBD, err
}
