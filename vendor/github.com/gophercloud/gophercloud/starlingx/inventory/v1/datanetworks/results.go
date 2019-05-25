/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package datanetworks

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*DataNetwork, error) {
	var s DataNetwork
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

// UpdateResult represents the result of an update operation.
type UpdateResult struct {
	commonResult
}

// CreateResult represents the result of a create operation.
type CreateResult struct {
	commonResult
}

// DeleteResult represents the result of an delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// DataNetwork defines the data associated to a single dataNetwork instance.
type DataNetwork struct {
	// Is network generated unique UUID for the data network
	ID string `json:"uuid"`

	// SystemName is the human readable and assignable name of the data network.
	Name string `json:"name"`

	// Description is user assignable description of the data network purpose.
	Description string `json:"description"`

	// Type is the data network encapsulation type.
	Type string `json:"network_type"`

	// MTU is the maximum transmit unit configured for this data network.
	MTU int `json:"mtu"`

	// TTL is the VXLAN time-to-live value.  Only applicable to dynamic learning
	// VxLAN data networks.
	TTL *int `json:"ttl,omitempty"`

	// Mode is the VxLAN endpoint mode.  Only applicable to VxLAN data networks.
	Mode *string `json:"mode,omitempty"`

	// MulticastGroup is the VxLAN multicast group address.  Only applicable to
	// VxLAN data networks.
	MulticastGroup *string `json:"multicast_group,omitempty"`

	// UDPPortNumber is the UDP port number to be used for UDP encapsulation of
	// the VxLAN header.
	UDPPortNumber *int `json:"port_num,omitempty"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// DataNetworkPage is the page returned by a pager when traversing over a
// collection of dataNetworks.
type DataNetworkPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a DataNetworkPage struct is empty.
func (r DataNetworkPage) IsEmpty() (bool, error) {
	is, err := ExtractDataNetworks(r)
	return len(is) == 0, err
}

// ExtractDataNetworks accepts a Page struct, specifically a DataNetworkPage
// struct, and extracts the elements into a slice of DataNetwork structs. In
// other words, a generic collection is mapped into a relevant slice.
func ExtractDataNetworks(r pagination.Page) ([]DataNetwork, error) {
	var s struct {
		DataNetwork []DataNetwork `json:"datanetworks"`
	}

	err := (r.(DataNetworkPage)).ExtractInto(&s)

	return s.DataNetwork, err
}
