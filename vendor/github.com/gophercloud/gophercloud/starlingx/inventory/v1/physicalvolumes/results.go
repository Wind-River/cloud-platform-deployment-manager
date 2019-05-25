/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package physicalvolumes

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an VolumeGroup.
func (r commonResult) Extract() (*PhysicalVolume, error) {
	var s PhysicalVolume
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

type LVMInfo struct {
	Name            string `json:"lvm_pv_name"`
	VolumeGroupName string `json:"lvm_vg_name"`
	UUID            string `json:"lvm_pv_uuid"`
	Size            int    `json:"lvm_pv_size"`
	TotalPE         int    `json:"lvm_pe_total"`
	AllocatedPE     int    `json:"lvm_pe_alloced"`
}

// Capabilities defines the set of system or resource capabilities associated to
// this resource.
type Capabilities struct {
}

// PhysicalVolume represents the host volume group.
type PhysicalVolume struct {
	// ID is the system assigned unique UUID value for the physical volume.
	ID string `json:"uuid"`

	// Type is the physical volume type.
	Type string `json:"pv_type"`

	// State is the current provisioning state of the physical volume
	State string `json:"pv_state"`

	// HostID is the unique UUID of the host.
	HostID string `json:"ihost_uuid"`

	// DevicePath is the full device path of the disk or partition.
	DevicePath string `json:"disk_or_part_device_path"`

	// DeviceNode is the /dev symlink to the full device path.
	DeviceNode string `json:"disk_or_part_device_node"`

	// DeviceUUID is the UUID value of the disk or partition.
	DeviceUUID string `json:"disk_or_part_uuid"`

	// Capabilities is a map of applicable device and system capabilities.
	Capabilities Capabilities `json:"capabilities"`

	// VolumeGroupID is the UUID value of the associated volume group.
	VolumeGroupID string `json:"ilvg_uuid"`

	// LVMInfo are the LVM related attributes for this volume group.
	LVMInfo `json:",inline"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// PhysicalVolumePage is the page returned by a pager when traversing over a
// collection of volume groups.
type PhysicalVolumePage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a PhysicalVolumePage struct is empty.
func (r PhysicalVolumePage) IsEmpty() (bool, error) {
	is, err := ExtractPhysicalVolumes(r)
	return len(is) == 0, err
}

// ExtractPhysicalVolumes accepts a Page struct, specifically a
// PhysicalVolumePage struct, and extracts the elements into a slice of
// PhysicalVolume structs. In other words, a generic collection is mapped into
// a relevant slice.
func ExtractPhysicalVolumes(r pagination.Page) ([]PhysicalVolume, error) {
	var s struct {
		PhysicalVolume []PhysicalVolume `json:"ipvs"`
	}

	err := (r.(PhysicalVolumePage)).ExtractInto(&s)

	return s.PhysicalVolume, err
}
