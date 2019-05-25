/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package volumegroups

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an VolumeGroup.
func (r commonResult) Extract() (*VolumeGroup, error) {
	var s VolumeGroup
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
	// SystemName is the name of the logical volume group.
	Name string `json:"lvm_vg_name"`

	// GroupUUID is the system assigned UUID value associated to the LVM volume
	// group.
	GroupUUID string `json:"lvm_vg_uuid"`

	// Access is the volume group permissions.
	Access string `json:"lvm_vg_access"`

	// Size is the total size of the volume group.
	Size int `json:"lvm_vg_size"`

	// AvailableSize is the currently available space on the volume group.
	AvailableSize int `json:"lvm_vg_avail_size"`

	// TotalPE is the total number of physical extends defined for this group.
	TotalPE int `json:"lvm_vg_total_pe"`

	// FreePE is the total number of free physical extends remaining.
	FreePE int `json:"lvm_vg_free_pe"`

	// CurrentLogicalVolumes is the current number of logical volumes associated
	// to this group.
	CurrentLogicalVolumes int `json:"lvm_cur_lv"`

	// MaximumLogicalVolumes is the total number of logical volumes permitted in
	// this group.
	MaximumLogicalVolumes int `json:"lvm_max_lv"`

	// CurrentPhysicalVolumes is the current number of physical volumes
	// associated to this group.
	CurrentPhysicalVolumes int `json:"lvm_cur_pv"`

	// MaximumPhysicalVolumes is the total number of physical volumes permitted
	// in this group.
	MaximumPhysicalVolumes int `json:"lvm_max_pv"`
}

// Capabilities defines the set of system or resource capabilities associated to
// this resource.
type Capabilities struct {
	LVMType                  *string `json:"lvm_type"`
	InstanceBacking          *string `json:"instance_backing"`
	ConcurrentDiskOperations *int    `json:"concurrent_disk_operations"`
}

// VolumeGroup represents the host volume group.
type VolumeGroup struct {
	// ID is the system assigned UUID for the volume group
	ID string `json:"uuid"`

	// HostID is the unique UUID value of the associated host resource.
	HostID string `json:"ihost_uuid"`

	// State is the current volume group state.
	State string `json:"vg_state"`

	// Capabilities is a map of applicable device and system capabilities.
	Capabilities Capabilities `json:"capabilities"`

	// LVMInfo are the LVM related attributes for this volume group.
	LVMInfo `json:",inline"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// VolumeGroupPage is the page returned by a pager when traversing over a
// collection of volume groups.
type VolumeGroupPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a VolumeGroupPage struct is empty.
func (r VolumeGroupPage) IsEmpty() (bool, error) {
	is, err := ExtractVolumeGroups(r)
	return len(is) == 0, err
}

// ExtractVolumeGroups accepts a Page struct, specifically a VolumeGroupPage
// struct, and extracts the elements into a slice of VolumeGroup structs. In
// other words, a generic collection is mapped into a relevant slice.
func ExtractVolumeGroups(r pagination.Page) ([]VolumeGroup, error) {
	var s struct {
		VolumeGroup []VolumeGroup `json:"ilvgs"`
	}

	err := (r.(VolumeGroupPage)).ExtractInto(&s)

	return s.VolumeGroup, err
}
