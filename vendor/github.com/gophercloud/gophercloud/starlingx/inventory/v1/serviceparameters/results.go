/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2020 Wind River Systems, Inc. */

package serviceparameters

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*ServiceParameter, error) {
	var s ServiceParameter
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

// ApplyResult represents the result of an apply operation.
type ApplyResult struct {
	commonResult
}

// UpdateResult represents the result of a create operation.
type UpdateResult struct {
	commonResult
}

// DeleteResult represents the result of a delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// ServiceParameter defines the data associated to a single service parameter instance.
type ServiceParameter struct {
	// Is serviceParameter generated unique UUID for the service parameter
	ID string `json:"uuid"`

	// Service is the  human readable and assignable name of the service parameter service.
	Service string `json:"service"`

	// Section is the human readable and assignable name of the service parameter section.
	Section string `json:"section"`

	// ParamName is the name of the service parameter.
	ParamName string `json:"name"`

	// ParamValue is the value of the service parameter.
	ParamValue string `json:"value"`

	// Personality for the service parameter (controller, storage, etc..)
	Personality *string `json:"personality,omitempty"`

	// Resource for the service parameter (this is a hiera entry, required if creating a service parameter)
	Resource *string `json:"resource,omitempty"`
}

// ServiceParameterPage is the page returned by a pager when traversing over a
// collection of service parameters.
type ServiceParameterPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a ServiceParameterPage struct is empty.
func (r ServiceParameterPage) IsEmpty() (bool, error) {
	is, err := ExtractServiceParameters(r)
	return len(is) == 0, err
}

// ExtractServiceParameters accepts a Page struct, specifically a ServiceParameterPage
// struct, and extracts the elements into a slice of ServiceParameter structs. In
// other words, a generic collection is mapped into a relevant slice.
func ExtractServiceParameters(r pagination.Page) ([]ServiceParameter, error) {
	var s struct {
		ServiceParameter []ServiceParameter `json:"parameters"`
	}

	err := (r.(ServiceParameterPage)).ExtractInto(&s)

	return s.ServiceParameter, err
}
