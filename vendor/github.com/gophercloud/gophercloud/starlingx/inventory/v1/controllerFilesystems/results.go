/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package controllerFilesystems

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*FileSystem, error) {
	var s FileSystem
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

// CreateResult represents the result of an update operation.
type UpdateResult struct {
	gophercloud.ErrResult
}

// FileSystem defines the data associated to a single filesystem instance.
type FileSystem struct {
	// ID is the generated unique UUID for the fileSystem
	ID string `json:"uuid"`

	// Name defines the system defined name of the file system resource.
	Name string `json:"name"`

	// State defines the current resizing state of the file system.
	State string `json:"state"`

	// SystemUUID is the system defined unique UUID value of the system
	// to which this resource is associated.
	SystemUUID string `json:"isystem_uuid"`

	// Replicated defines a bool which represents whether the file system
	// is replicated across both controllers or not.
	Replicated bool `json:"replicated"`

	// Size defines the size of the file system in gigabytes.
	Size int `json:"size"`

	// LogicalVolumes defines the system defined logical volume that is
	// backing this resource on the controllers.
	LogicalVolume string `json:"logical_volume"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// FileSystemPage is the page returned by a pager when traversing over a
// collection of fileSystems.
type FileSystemPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a FileSystemPage struct is empty.
func (r FileSystemPage) IsEmpty() (bool, error) {
	is, err := ExtractFileSystems(r)
	return len(is) == 0, err
}

// ExtractFileSystems accepts a Page struct, specifically a FileSystemPage
// struct, and extracts the elements into a slice of FileSystem structs. In
// other words, a generic collection is mapped into a relevant slice.
func ExtractFileSystems(r pagination.Page) ([]FileSystem, error) {
	var s struct {
		FileSystem []FileSystem `json:"controller_fs"`
	}

	err := (r.(FileSystemPage)).ExtractInto(&s)

	return s.FileSystem, err
}
