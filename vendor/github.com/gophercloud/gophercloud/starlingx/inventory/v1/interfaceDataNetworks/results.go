/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package interfaceDataNetworks

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*InterfaceDataNetwork, error) {
	var s InterfaceDataNetwork
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

// InterfaceDataNetwork defines the data associated to a single interface network
// association.
type InterfaceDataNetwork struct {
	// UUID is a system generated unique UUID for the network association
	UUID string `json:"uuid"`

	// ID is a sequential integer assigned by the system.
	ID int `json:"id"`

	// DataNetworkUUID defines the system assigned unique UUID of the associated
	// data network.
	DataNetworkUUID string `json:"network_uuid"`

	// DataNetworkID defines the system assigned sequential integer id value of
	// the associated network.
	DataNetworkID int `json:"network_id"`

	// DataNetworkType defines the type value assigned to the data network.
	DataNetworkType string `json:"network_type"`

	// NetworkName defines the name assigned to the network.
	DataNetworkName string `json:"datanetwork_name"`

	// InterfaceName defines the name assigned to the associated interface.
	InterfaceName string `json:"ifname"`

	// InterfaceUUID defines the system assigned unique UUID value of the
	// associated data network.
	InterfaceUUID string `json:"interface_uuid"`
}

// InterfaceDataNetworkPage is the page returned by a pager when traversing over a
// collection of interface networks.
type InterfaceDataNetworkPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a InterfaceDataNetworkPage struct is empty.
func (r InterfaceDataNetworkPage) IsEmpty() (bool, error) {
	is, err := ExtractInterfaceDataNetworks(r)
	return len(is) == 0, err
}

// ExtractInterfaceDataNetworks accepts a Page struct, specifically a
// InterfaceDataNetworkPage struct, and extracts the elements into a slice of
// InterfaceDataNetwork structs. In other words, a generic collection is mapped into
// a relevant slice.
func ExtractInterfaceDataNetworks(r pagination.Page) ([]InterfaceDataNetwork, error) {
	var s struct {
		InterfaceDataNetwork []InterfaceDataNetwork `json:"interface_datanetworks"`
	}

	err := (r.(InterfaceDataNetworkPage)).ExtractInto(&s)

	return s.InterfaceDataNetwork, err
}
