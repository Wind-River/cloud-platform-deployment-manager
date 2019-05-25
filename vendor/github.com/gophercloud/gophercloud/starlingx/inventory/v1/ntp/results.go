/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package ntp

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*NTP, error) {
	var s NTP
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

// NTP defines the data associated to a single NTP instance.
type NTP struct {
	// ID defines the system NTP record
	ID string `json:"uuid"`

	// NTPServers defines the comma separated list of NTP servers configured.
	NTPServers string `json:"ntpservers"`

	// Enables defines whether NTP is enabled on the system or not
	Enabled bool `json:"enabled"`

	// SystemID is the unique UUID value of the system to which this resource
	// is associated.
	SystemID string `json:"isystem_uuid"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// NTPPage is the page returned by a pager when traversing over a
// collection of NTPs.
type NTPPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a NTPPage struct is empty.
func (r NTPPage) IsEmpty() (bool, error) {
	is, err := ExtractNTPs(r)
	return len(is) == 0, err
}

// ExtractNTPs accepts a Page struct, specifically a NTPPage struct,
// and extracts the elements into a slice of NTP structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractNTPs(r pagination.Page) ([]NTP, error) {
	var s struct {
		NTP []NTP `json:"intps"`
	}

	err := (r.(NTPPage)).ExtractInto(&s)

	return s.NTP, err
}
