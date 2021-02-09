/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package certificates

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Certificate, error) {
	var s Certificate
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

func (r CreateResult) Extract() ([]*Certificate, error) {
	var s CreateResponse
	err := r.ExtractInto(&s)
	if s.Error != "" {
		err = fmt.Errorf("certificate install error: %s", s.Error)
	}
	return s.Certificate, err
}

// GetResult represents the result of a get operation.
type GetResult struct {
	commonResult
}

// UpdateResult represents the result of an update operation.
type UpdateResult struct {
	commonResult
}

// DeleteResult represents the result of a delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// CertificateResponse defines a special wrapper to deal with the non-standard
// response format of certificate install API.
type CreateResponse struct {
	Certificate []*Certificate `json:"certificates,omitempty"`
	Body        string         `json:"body"`
	Success     string         `json:"success"`
	Error       string         `json:"error"`
}

// Certificate defines the data associated to a single Certificate instance.
type Certificate struct {
	// ID defines the system assigned unique UUID value.
	ID string `json:"uuid"`

	// Mode defines the operational purpose of the
	Type string `json:"certtype"`

	// Signature defines the x.509 certificate hash
	Signature string `json:"signature"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// CertificatePage is the page returned by a pager when traversing over a
// collection of Certificates.
type CertificatePage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a CertificatePage struct is empty.
func (r CertificatePage) IsEmpty() (bool, error) {
	is, err := ExtractCertificates(r)
	return len(is) == 0, err
}

// ExtractCertificates accepts a Page struct, specifically a CertificatePage
// struct, and extracts the elements into a slice of Certificate structs. In
// other words, a generic collection is mapped into a relevant slice.
func ExtractCertificates(r pagination.Page) ([]Certificate, error) {
	var s struct {
		Certificate []Certificate `json:"certificates"`
	}

	err := (r.(CertificatePage)).ExtractInto(&s)

	return s.Certificate, err
}
