/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package partitions

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an DiskPartition.
func (r commonResult) Extract() (*DiskPartition, error) {
	var s DiskPartition
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

// Capabilities defines the set of system or resource capabilities associated to
// this resource.
type Capabilities struct {
}

// DiskPartition represents the host physical partition inventory.
type DiskPartition struct {
	// ID is the unique system assigned numerical UUID of the partition.
	ID string `json:"uuid"`

	// HostID is the UUID of the associated host resource.
	HostID string `json:"ihost_uuid"`

	// DiskID is the unique system assigned UUID of the associated disk.
	DiskID string `json:"idisk_uuid"`

	// DevicePath is the full device path of the partition.
	DevicePath string `json:"device_path"`

	// DeviceNode is the /dev symlink to the full device path.
	DeviceNode string `json:"device_node"`

	// TypeName is the assigned partition type for this partition.
	TypeName string `json:"type_name"`

	// TypeGUID is the partition GUID value associated to the TypeName value.
	TypeGUID string `json:"type_guid"`

	// Size is the size of the disk partition; in megabytes.
	Size int `json:"size_mib"`

	// Start is the start position on disk for this partition; in megabytes.
	Start int `json:"start_mib"`

	// End is the end position on disk for this partition; in megabytes.
	End int `json:"end_mib"`

	// PhysicalVolumeID is the UUID value of the associated physical volume.
	PhysicalVolumeID *string `json:"ipv_uuid"`

	// Capabilities is a map of applicable device and system capabilities.
	Capabilities Capabilities `json:"capabilities"`

	// Status is the current readiness status of the partition.
	Status int

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// DiskPartitionPage is the page returned by a pager when traversing over a
// collection of partitions.
type DiskPartitionPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a DiskPartitionPage struct is empty.
func (r DiskPartitionPage) IsEmpty() (bool, error) {
	is, err := ExtractDiskPartitions(r)
	return len(is) == 0, err
}

// ExtractDiskPartitions accepts a Page struct, specifically a DiskPartitionPage
// struct, and extracts the elements into a slice of DiskPartition structs. In
// other words, a generic collection is mapped into a relevant slice.
func ExtractDiskPartitions(r pagination.Page) ([]DiskPartition, error) {
	var s struct {
		DiskPartition []DiskPartition `json:"partitions"`
	}

	err := (r.(DiskPartitionPage)).ExtractInto(&s)

	return s.DiskPartition, err
}
