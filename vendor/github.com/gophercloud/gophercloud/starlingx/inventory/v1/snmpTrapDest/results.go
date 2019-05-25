/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package snmpTrapDest

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*SNMPTrapDest, error) {
	var s SNMPTrapDest
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

// SNMPTrapDest defines the data associated to a single SNMPTrapDest instance.
type SNMPTrapDest struct {
	// ID is the generated unique UUID for the SNMP community resource.
	ID string `json:"uuid"`

	// Community represents the SNMP community to which this client has access.
	Community string `json:"community"`

	// Access is the SNMP GET/SET access control setting for this client.
	IPAddress string `json:"ip_address"`

	// View is the SNMP MIB view to which this community has access.
	Type string `json:"type"`

	// CreatedAt defines the timestamp at which the resource was created.
	Port int `json:"port"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	Transport string `json:"transport"`
}

// SNMPTrapDestPage is the page returned by a pager when traversing over a
// collection of routes.
type SNMPTrapDestPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a SNMPTrapDestPage struct is empty.
func (r SNMPTrapDestPage) IsEmpty() (bool, error) {
	is, err := ExtractSNMPTrapDests(r)
	return len(is) == 0, err
}

// ExtractSNMPCommunities accepts a Page struct, specifically a
// SNMPTrapDestPage struct, and extracts the elements into a slice of
// SNMPTrapDest structs. In other words, a generic collection is mapped into a
// relevant slice.
func ExtractSNMPTrapDests(r pagination.Page) ([]SNMPTrapDest, error) {
	var s struct {
		SNMPTrapDest []SNMPTrapDest `json:"itrapdest"`
	}

	err := (r.(SNMPTrapDestPage)).ExtractInto(&s)

	return s.SNMPTrapDest, err
}
