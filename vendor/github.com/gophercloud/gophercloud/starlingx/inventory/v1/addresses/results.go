/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package addresses

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Address, error) {
	var s Address
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

// DeleteResult represents the result of an delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// Address defines the data associated to a single address instance.
type Address struct {
	// ID is the generated unique UUID for the address
	ID string `json:"uuid"`

	// Address is the IPv4 or IPv6 address value.
	Address string `json:"address"`

	// Prefix is the numeric prefix length of the address.
	Prefix int `json:"prefix"`

	// InterfaceName is the name of the interface to which the address is
	// associated.
	InterfaceName string `json:"ifname"`

	// InterfaceUUID is the UUID of the interface to which the address is
	// associated.
	InterfaceUUID string `json:"interface_uuid"`

	// EnableDAD is the state of Duplicate Address Detection (DAD) on this
	// address.
	EnableDAD bool `json:"enable_dad"`

	// PoolUUID is the unique uuid value of the pool to which the address is
	// associated.
	PoolUUID *string `json:"pool_uuid,omitempty"`
}

// AddressPage is the page returned by a pager when traversing over a
// collection of addresss.
type AddressPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a AddressPage struct is empty.
func (r AddressPage) IsEmpty() (bool, error) {
	is, err := ExtractAddresses(r)
	return len(is) == 0, err
}

// ExtractAddresses accepts a Page struct, specifically a AddressPage struct,
// and extracts the elements into a slice of Address structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractAddresses(r pagination.Page) ([]Address, error) {
	var s struct {
		Address []Address `json:"addresses"`
	}

	err := (r.(AddressPage)).ExtractInto(&s)

	return s.Address, err
}
