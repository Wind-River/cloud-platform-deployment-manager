/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package addresspools

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*AddressPool, error) {
	var s AddressPool
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

// UpdateResult represents the result of a create operation.
type UpdateResult struct {
	commonResult
}

// DeleteResult represents the result of an update operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// AddressPool defines the data associated to a single addressPool instance.
type AddressPool struct {
	// Is addressPool generated unique UUID for the address pool
	ID string `json:"uuid"`

	// SystemName is the human readable and assignable name of the address pool.
	Name string `json:"name"`

	// Network is the IPv4 or IPv6 network address value.
	Network string `json:"network"`

	// Prefix is the numeric prefix length of the network.
	Prefix int `json:"prefix"`

	// Gateway is the next hop gateway address for the network (if applicable).
	Gateway *string `json:"gateway_address,omitempty"`

	// Order defines whether addresses are allocated randomly or sequentially
	// from the list of available addresses.
	Order string `json:"order"`

	// Ranges is a list of start/end pairs defining the available address space.
	// Each pair is a two element array where the first element is start and the
	// last element is end.
	Ranges [][]string `json:"ranges"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// AddressPoolPage is the page returned by a pager when traversing over a
// collection of addressPools.
type AddressPoolPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a AddressPoolPage struct is empty.
func (r AddressPoolPage) IsEmpty() (bool, error) {
	is, err := ExtractAddressPools(r)
	return len(is) == 0, err
}

// ExtractAddressPools accepts a Page struct, specifically a AddressPoolPage
// struct, and extracts the elements into a slice of AddressPool structs. In
// other words, a generic collection is mapped into a relevant slice.
func ExtractAddressPools(r pagination.Page) ([]AddressPool, error) {
	var s struct {
		AddressPool []AddressPool `json:"addrpools"`
	}

	err := (r.(AddressPoolPage)).ExtractInto(&s)

	return s.AddressPool, err
}
