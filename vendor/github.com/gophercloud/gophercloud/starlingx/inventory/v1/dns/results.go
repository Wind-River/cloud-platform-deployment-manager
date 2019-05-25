/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package dns

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*DNS, error) {
	var s DNS
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

// DNS defines the data associated to a single DNS instance.
type DNS struct {
	// ID defines the system DNS record
	ID string `json:"uuid"`

	// Nameservers defines the comma separated list of DNS servers configured.
	Nameservers string `json:"nameservers"`

	// SystemID is the unique UUID value of the system to which this resource
	// is associated.
	SystemID string `json:"isystem_uuid"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// DNSPage is the page returned by a pager when traversing over a
// collection of DNSs.
type DNSPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a DNSPage struct is empty.
func (r DNSPage) IsEmpty() (bool, error) {
	is, err := ExtractDNSs(r)
	return len(is) == 0, err
}

// ExtractDNSs accepts a Page struct, specifically a DNSPage struct,
// and extracts the elements into a slice of DNS structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractDNSs(r pagination.Page) ([]DNS, error) {
	var s struct {
		DNS []DNS `json:"idnss"`
	}

	err := (r.(DNSPage)).ExtractInto(&s)

	return s.DNS, err
}
