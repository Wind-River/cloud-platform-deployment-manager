/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package storagebackends

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*StorageBackend, error) {
	var s StorageBackend
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
	Replication string `json:"replication,omitempty"`
	MinReplication string `json:"min_replication,omitempty"`
}

// StorageBackend defines the data associated to a single StorageBackend
// instance.
type StorageBackend struct {
	ID           string       `json:"uuid"`
	Backend      string       `json:"backend"`
	Name         string       `json:"name"`
	State        string       `json:"state"`
	Task         string       `json:"task"`
	Services     string       `json:"services"`
	Capabilities Capabilities `json:"capabilities"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// StorageBackendPage is the page returned by a pager when traversing over a
// collection of StorageBackends.
type StorageBackendPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a StorageBackendPage struct is empty.
func (r StorageBackendPage) IsEmpty() (bool, error) {
	is, err := ExtractStorageBackends(r)
	return len(is) == 0, err
}

// ExtractStorageBackends accepts a Page struct, specifically a
// StorageBackendPage struct, and extracts the elements into a slice of
// StorageBackend structs. In other words, a generic collection is mapped into
// a relevant slice.
func ExtractStorageBackends(r pagination.Page) ([]StorageBackend, error) {
	var s struct {
		StorageBackend []StorageBackend `json:"storage_backends"`
	}

	err := (r.(StorageBackendPage)).ExtractInto(&s)

	return s.StorageBackend, err
}
