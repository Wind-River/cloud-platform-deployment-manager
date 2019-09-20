/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package disks

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Disk, error) {
	var s Disk
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

// Capabilities defines the set of system or resource capabilities associated to
// this resource.
type Capabilities struct {
}

// Disk represents the host physical disk inventory.
type Disk struct {
	// ID is the unique system UUID for the disk.
	ID string `json:"uuid"`

	// DevicePath is the full device path.
	DevicePath string `json:"device_path"`

	// DeviceNode is the /dev symlink to the full device path.
	DeviceNode string `json:"device_node"`

	// DeviceType is the type of physical device (i.e., hdd, ssd).
	DeviceType string `json:"device_type"`

	// TODO(alegacy): DeviceWWN
	DeviceWWN *string `json:"device_wwn,omitempty"`

	// DeviceID is the manufacturer disk identifier.
	DeviceID string `json:"device_id"`

	// TODO(alegacy): DeviceNumber
	DeviceNumber int `json:"device_num"`

	// Size is the total size of the disk in megabytes.
	Size int `json:"size_mib"`

	// AvailableSpace is the total available space in megabytes.
	AvailableSpace int `json:"available_mib"`

	// PhysicalVolumeID is the UUID value of the associated physical volume.
	PhysicalVolumeID *string `json:"ipv_uuid,omitempty"`

	// SerialID is the manufacturer serial number of the disk
	SerialID *string `json:"serial_id,omitempty"`

	// StorID is the UUID value of the associated storage function.
	StorID *string `json:"istor_uuid,omitempty"`

	// Capabilities is a map of applicable device and system capabilities.
	Capabilities Capabilities `json:"capabilities"`

	// RPM is the rotational speed of the disk.
	RPM string `json:"rpm"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// DiskPage is the page returned by a pager when traversing over a
// collection of disks.
type DiskPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a DiskPage struct is empty.
func (r DiskPage) IsEmpty() (bool, error) {
	is, err := ExtractDisks(r)
	return len(is) == 0, err
}

// ExtractDisks accepts a Page struct, specifically a DiskPage struct,
// and extracts the elements into a slice of Disk structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractDisks(r pagination.Page) ([]Disk, error) {
	var s struct {
		Disk []Disk `json:"idisks"`
	}

	err := (r.(DiskPage)).ExtractInto(&s)

	return s.Disk, err
}

// ListDisks is a convenience function to list and extract the entire
// list of Disks on a given host.
func ListDisks(c *gophercloud.ServiceClient, hostid string) ([]Disk, error) {
	pages, err := List(c, hostid, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractDisks(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
