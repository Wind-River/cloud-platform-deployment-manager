/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package networks

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Network, error) {
	var s Network
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
type CreateResult struct {
	commonResult
}

// UpdateResult represents the result of an update operation.
type UpdateResult struct {
	commonResult
}

// DeleteResult represents the result of an delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// Network defines the data associated to a single network instance.
type Network struct {
	// UUID is a system generated unique UUID for the network
	UUID string `json:"uuid"`

	// ID is a sequential integer assigned by the system.
	ID int `json:"id"`

	// SystemName is the human readable and assignable name of the network.
	Name string `json:"name"`

	// Type is the usage type of the network
	Type string `json:"type"`

	// Dynamic defines whether addresses are allocated dynamically or statically
	Dynamic bool `json:"dynamic"`

	// Pool UUID is a reference to the underlying address pool resource.
	PoolUUID string `json:"pool_uuid"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// NetworkPage is the page returned by a pager when traversing over a
// collection of networks.
type NetworkPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a NetworkPage struct is empty.
func (r NetworkPage) IsEmpty() (bool, error) {
	is, err := ExtractNetworks(r)
	return len(is) == 0, err
}

// ExtractNetworks accepts a Page struct, specifically a NetworkPage struct,
// and extracts the elements into a slice of Network structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractNetworks(r pagination.Page) ([]Network, error) {
	var s struct {
		Network []Network `json:"networks"`
	}

	err := (r.(NetworkPage)).ExtractInto(&s)

	return s.Network, err
}
