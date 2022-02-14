/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package ptpinterfaces

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*PTPInterface, error) {
	var s PTPInterface
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

// PTPInterface defines the data associated to a single ptp interface.
type PTPInterface struct {
	// UUID is generated unique UUID for the ptp interface.
	UUID string `json:"uuid"`

	// ID is the resource ID of the ptp interface
	ID int `json:"id"`

	// Name is the human readable and assignable name of the ptp interface.
	Name string `json:"name"`

	// PTPInstanceUUID is the UUID of the ptp instance associated with this
	// ptp interface.
	PTPInstanceUUID string `json:"ptp_instance_uuid"`

	// PTPinstanceName is the human readable and assignable name of the ptp
	// instance that associated with this ptp interface.
	PTPInstanceName string `json:"ptp_instance_name,omitempty"`

	// Hostnames is the list of the host names assigned to the ptp interface.
	HostNames []string `json:"hostnames,omitempty"`

	// InterfaceNames is the list of the interface names assigned to the ptp
	// interface.
	InterfaceNames []string `json:"interface_names,omitempty"`

	// Parameters is the list of the parameters assigned to the ptp instance.
	Parameters []string `json:"parameters,omitempty"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// PTPInterfacePage is the page returned by a pager when traversing over a
// collection of PTPInterfaces.
type PTPInterfacePage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a PTPInterfacePage struct is empty.
func (r PTPInterfacePage) IsEmpty() (bool, error) {
	is, err := ExtractPTPInterfaces(r)
	return len(is) == 0, err
}

// ExtractPTPInterfaces accepts a Page struct, specifically a PTPInterfacePage
// struct, and extracts the elements into a slice of PTPInterface structs. In
// other words, a generic collection is mapped into a relevant slice.
func ExtractPTPInterfaces(r pagination.Page) ([]PTPInterface, error) {
	var s struct {
		PTPInterface []PTPInterface `json:"ptp_interfaces"`
	}

	err := (r.(PTPInterfacePage)).ExtractInto(&s)

	return s.PTPInterface, err
}
