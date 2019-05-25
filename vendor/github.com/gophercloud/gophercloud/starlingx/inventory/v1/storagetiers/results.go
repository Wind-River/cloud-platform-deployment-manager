/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package storagetiers

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*StorageTier, error) {
	var s StorageTier
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

// StorageTier defines the data associated to a single StorageTier
// instance.
type StorageTier struct {
	// ID is the system assigned unique UUID value.
	ID string `json:"uuid"`

	// SystemName is the human readable name of the storage tier.
	Name string `json:"name"`

	// Type is the storage backend type associated to this tier.
	Type string `json:"type"`

	// Status is the current operational status of the storage tier.
	Status string `json:"status"`

	// Stors is the list of OSD resources associated to this storage tier.
	Stors []int `json:"stors"`

	// Capabilities is a map of applicable device and system capabilities.
	Capabilities Capabilities `json:"capabilities"`

	// ClusterID is the unique UUID value of the cluster to which this tier is
	// associated.
	ClusterID string `json:"cluster_uuid"`

	// BackendID is the unique UUID value of the backend to which this tier is
	// associated.
	BackendID string `json:"backend_uuid"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// StorageTierPage is the page returned by a pager when traversing over a
// collection of StorageTiers.
type StorageTierPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a StorageTierPage struct is empty.
func (r StorageTierPage) IsEmpty() (bool, error) {
	is, err := ExtractStorageTiers(r)
	return len(is) == 0, err
}

// ExtractStorageTiers accepts a Page struct, specifically a
// StorageTierPage struct, and extracts the elements into a slice of
// StorageTier structs. In other words, a generic collection is mapped into
// a relevant slice.
func ExtractStorageTiers(r pagination.Page) ([]StorageTier, error) {
	var s struct {
		StorageTier []StorageTier `json:"storage_tiers"`
	}

	err := (r.(StorageTierPage)).ExtractInto(&s)

	return s.StorageTier, err
}
