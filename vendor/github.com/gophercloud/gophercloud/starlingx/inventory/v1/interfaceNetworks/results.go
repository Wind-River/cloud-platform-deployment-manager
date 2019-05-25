/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package interfaceNetworks

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*InterfaceNetwork, error) {
	var s InterfaceNetwork
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

// InterfaceNetwork defines the data associated to a single interface network
// association.
type InterfaceNetwork struct {
	// UUID is a system generated unique UUID for the network association
	UUID string `json:"uuid"`

	// ID is a sequential integer assigned by the system.
	ID int `json:"id"`

	// NetworkUUID defines the system assigned unique UUID of the associated
	// network.
	NetworkUUID string `json:"network_uuid"`

	// NetworkID defines the system assigned sequential integer id value of the
	// associated network.
	NetworkID int `json:"network_id"`

	// NetworkType defines the type value assigned to the network.  Note: this
	// currently appears to be set to the same value as the network name.
	NetworkType string `json:"network_type"`

	// NetworkName defines the name assigned to the network.
	NetworkName string `json:"network_name"`

	// InterfaceName defines the name assigned to the associated interface.
	InterfaceName string `json:"ifname"`

	// InterfaceUUID defines the system assigned unique UUID value of the
	// associated network.
	InterfaceUUID string `json:"interface_uuid"`
}

// InterfaceNetworkPage is the page returned by a pager when traversing over a
// collection of interface networks.
type InterfaceNetworkPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a InterfaceNetworkPage struct is empty.
func (r InterfaceNetworkPage) IsEmpty() (bool, error) {
	is, err := ExtractInterfaceNetworks(r)
	return len(is) == 0, err
}

// ExtractInterfaceNetworks accepts a Page struct, specifically a
// InterfaceNetworkPage struct, and extracts the elements into a slice of
// InterfaceNetwork structs. In other words, a generic collection is mapped into
// a relevant slice.
func ExtractInterfaceNetworks(r pagination.Page) ([]InterfaceNetwork, error) {
	var s struct {
		InterfaceNetwork []InterfaceNetwork `json:"interface_networks"`
	}

	err := (r.(InterfaceNetworkPage)).ExtractInto(&s)

	return s.InterfaceNetwork, err
}
