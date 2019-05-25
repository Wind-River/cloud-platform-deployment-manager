/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package osds

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an VolumeGroup.
func (r commonResult) Extract() (*OSD, error) {
	var s OSD
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

type JournalInfo struct {
	Path     *string `json:"journal_path"`
	Size     *int    `json:"journal_size_mib"`
	Node     *string `json:"journal_node"`
	Location *string `json:"journal_location"`
}

// Capabilities defines the set of system or resource capabilities associated to
// this resource.
type Capabilities struct {
}

// OSD represents the host volume group.
type OSD struct {
	// ID is the system assigned unique UUID value for the OSD.
	ID string `json:"uuid"`

	// Function is the OSD assigned function type (i.e., Journal, OSD)
	Function string `json:"function"`

	// HostID is the unique UUID of the host.
	HostID string `json:"ihost_uuid"`

	// DiskID is the unique UUID of the disk.
	DiskID string `json:"idisk_uuid"`

	// State is the current provisioning state of the OSD.
	State string `json:"state"`

	// TierName is the name of the tier to which this OSD is associated.
	TierName string `json:"tier_name"`

	// TierUUID is the unique UUID value of the tier to which this OSD is
	// associated.
	TierUUID string `json:"tier_uuid"`

	// Capabilities is a map of applicable device and system capabilities.
	Capabilities Capabilities `json:"capabilities"`

	// JournalInfo is the set of journal related attributes
	JournalInfo `json:",inline"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// OSDPage is the page returned by a pager when traversing over a
// collection of volume groups.
type OSDPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a OSDPage struct is empty.
func (r OSDPage) IsEmpty() (bool, error) {
	is, err := ExtractOSDs(r)
	return len(is) == 0, err
}

// ExtractOSDs accepts a Page struct, specifically a
// OSDPage struct, and extracts the elements into a slice of
// OSD structs. In other words, a generic collection is mapped into
// a relevant slice.
func ExtractOSDs(r pagination.Page) ([]OSD, error) {
	var s struct {
		OSD []OSD `json:"istors"`
	}

	err := (r.(OSDPage)).ExtractInto(&s)

	return s.OSD, err
}
