/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package ports

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Port, error) {
	var s Port
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

// Port represents a host physical port
type Port struct {
	// ID is the system assign unique UUID value for the port
	ID string `json:"uuid"`

	// SystemName is the system assign unique name value for the port (i.e., Linux
	// interface name).
	Name string `json:"name"`

	// PCIAddress is the system assign PCI bus address for the port.
	PCIAddress string `json:"pciaddr"`

	// InterfaceID is the unique UUID value of the interface to which this port
	// is related.
	InterfaceID string `json:"interface_uuid"`
}

// PortPage is the page returned by a pager when traversing over a
// collection of ports.
type PortPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a PortPage struct is empty.
func (r PortPage) IsEmpty() (bool, error) {
	is, err := ExtractPorts(r)
	return len(is) == 0, err
}

// ExtractPorts accepts a Page struct, specifically a PortPage struct,
// and extracts the elements into a slice of Port structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractPorts(r pagination.Page) ([]Port, error) {
	var s struct {
		Port []Port `json:"ethernet_ports"`
	}

	err := (r.(PortPage)).ExtractInto(&s)

	return s.Port, err
}
