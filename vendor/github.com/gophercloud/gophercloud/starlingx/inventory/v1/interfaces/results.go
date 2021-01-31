/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package interfaces

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Interface, error) {
	var s Interface
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

// UpdateResult represelnts the result of an update operation.
type UpdateResult struct {
	commonResult
}

// DeleteResult represents the result of a delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// Interface represents a host interface.
type Interface struct {
	// ID is the system assigned unique UUID value for the interface
	ID string `json:"uuid"`

	// SystemName is the human-readable name for the interface. Must be unique.
	Name string `json:"ifname"`

	// Type is the interface type of the interface (e.g., vlan, ethernet, ae)
	Type string `json:"iftype"`

	// Class is the assigned interface class (e.g., platform, data, sriov)
	Class string `json:"ifclass"`

	// NetworkType is the type assigned to the interface. (e.g. mgmt, oam)
	NetworkType string `json:"network_type"`

	// MTU is the maximum transmit unit of the interface.
	MTU int `json:"imtu"`

	// VID is the VLAN ID value assign to the interface if the interface is a
	// VLAN interface.
	VID *int `json:"vlan_id,omitempty"`

	// IPv4Mode is the addressing mode assigned to the interface (e.g., static,
	// dhcp).
	IPv4Mode *string `json:"ipv4_mode,omitempty"`

	// IPv4Pool is the UUID value of the address pool to be associated with
	// the interface if the mode is set to "pool".
	IPv4Pool *string `json:"ipv4_pool,omitempty"`

	// IPv6Mode is the addressing mode assigned to the interface (e.g., static,
	// dhcp).
	IPv6Mode *string `json:"ipv6_mode,omitempty"`

	// IPv6Pool is the UUID value of the address pool to be associated with
	// the interface if the mode is set to "pool".
	IPv6Pool *string `json:"ipv6_pool,omitempty"`

	// Networks is the list of networks assigned to this interface.
	Networks []string `json:"networks"`

	// DataNetwork is the list of data networks assigned to this interface.
	DataNetworks []string `json:"datanetworks"`

	// AEMode is the link protection mode assigned to the interface if the
	// interface is a Bond interface.
	AEMode *string `json:"aemode,omitempty"`

	// AETransmitHash is the link selection policy assigned to the interface if
	// the interface is a Bond interface.
	AETransmitHash *string `json:"txhashpolicy,omitempty"`

	// VFCount is the number of SRIOV VF interfaces configured.
	VFCount *int `json:"sriov_numvfs,omitempty"`

	// VFDriver is the NIC driver to be bound on the host for each VF device
	VFDriver *string `json:"sriov_vf_driver,omitempty"`

	// Uses is the list of interfaces upon which this interface depends. This is
	// only applicable to VLAN and Bond interfaces.
	Uses []string `json:"uses"`

	// Users is the list of interfaces that depend on this interface.  This is
	// only applicable to Ethernet and Bond interfaces.
	Users []string `json:"used_by"`

	// PTPRole is the configuration of the interface as ptp master, slave, or none.
	PTPRole *string `json:"ptp_role,omitempty"`

	// VFCount is the number of SRIOV VF interfaces configured.
	MaxTxRate *int `json:"max_tx_rate,omitempty"`

}

// InterfacePage is the page returned by a pager when traversing over a
// collection of host interfaces.
type InterfacePage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a InterfacePage struct is empty.
func (r InterfacePage) IsEmpty() (bool, error) {
	is, err := ExtractInterfaces(r)
	return len(is) == 0, err
}

// ExtractInterfaces accepts a Page struct, specifically a InterfacePage struct,
// and extracts the elements into a slice of Interface structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractInterfaces(r pagination.Page) ([]Interface, error) {
	var s struct {
		Interface []Interface `json:"iinterfaces"`
	}

	err := (r.(InterfacePage)).ExtractInto(&s)

	return s.Interface, err
}
