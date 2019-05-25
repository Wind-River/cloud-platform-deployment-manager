/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package routes

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Route, error) {
	var s Route
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

// Route defines the data associated to a single Route instance.
type Route struct {
	// ID is the generated unique UUID for the route
	ID string `json:"uuid"`

	// Network is the IPv4 or IPv6 network address value.
	Network string `json:"network"`

	// Prefix is the numeric prefix length of the network value.
	Prefix int `json:"prefix"`

	// Gateway is the IPv4 or IPv6 next hop gateway address.
	Gateway string `json:"gateway"`

	// Metric is the numeric prefix length of the route.
	Metric int `json:"metric"`

	// InterfaceName is the name of the interface associated to which the
	// route is associated.
	InterfaceName string `json:"ifname"`

	// InterfaceUUID is the UUID of the interface associated to which the
	// route is associated.
	InterfaceUUID string `json:"interface_uuid"`
}

// RoutePage is the page returned by a pager when traversing over a
// collection of routes.
type RoutePage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a RoutePage struct is empty.
func (r RoutePage) IsEmpty() (bool, error) {
	is, err := ExtractRoutes(r)
	return len(is) == 0, err
}

// ExtractRoutes accepts a Page struct, specifically a RoutePage struct,
// and extracts the elements into a slice of Route structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractRoutes(r pagination.Page) ([]Route, error) {
	var s struct {
		Route []Route `json:"routes"`
	}

	err := (r.(RoutePage)).ExtractInto(&s)

	return s.Route, err
}
