/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package snmpCommunity

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*SNMPCommunity, error) {
	var s SNMPCommunity
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

// SNMPCommunity defines the data associated to a single SNMPCommunity instance.
type SNMPCommunity struct {
	// ID is the generated unique UUID for the SNMP community resource.
	ID string `json:"uuid"`

	// Community represents the SNMP community to which this client has access.
	Community string `json:"community"`

	// Access is the SNMP GET/SET access control setting for this client.
	Access string `json:"access"`

	// View is the SNMP MIB view to which this community has access.
	View string `json:"view"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// SNMPCommunityPage is the page returned by a pager when traversing over a
// collection of routes.
type SNMPCommunityPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a SNMPCommunityPage struct is empty.
func (r SNMPCommunityPage) IsEmpty() (bool, error) {
	is, err := ExtractSNMPCommunities(r)
	return len(is) == 0, err
}

// ExtractSNMPCommunities accepts a Page struct, specifically a
// SNMPCommunityPage struct, and extracts the elements into a slice of
// SNMPCommunity structs. In other words, a generic collection is mapped into a
// relevant slice.
func ExtractSNMPCommunities(r pagination.Page) ([]SNMPCommunity, error) {
	var s struct {
		SNMPCommunity []SNMPCommunity `json:"icommunity"`
	}

	err := (r.(SNMPCommunityPage)).ExtractInto(&s)

	return s.SNMPCommunity, err
}
