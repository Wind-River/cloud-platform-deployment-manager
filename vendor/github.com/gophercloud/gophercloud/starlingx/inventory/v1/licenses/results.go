/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package licenses

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
)

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

func (r CreateResult) ExtractErr() error {
	var s CreateResponse
	err := r.ExtractInto(&s)
	if s.Error != "" {
		err = fmt.Errorf("license install error: %s", s.Error)
	}
	return err
}

func (r GetResult) Extract() (*License,  error) {
	var result License
	var s GetResponse
	err := r.ExtractInto(&s)
	if s.Error != "" {
		err = fmt.Errorf("license get error: %s", s.Error)
		return nil, err
	}

	result.Content = s.Content
	return &result, nil
}

// GetResponse defines a special wrapper to deal with the non-standard response
// format of the license show API.
type GetResponse struct {
	Content     string       `json:"content"`
	Error       string       `json:"error"`
}

// LicenseResponse defines a special wrapper to deal with the non-standard
// response format of license install API.
type CreateResponse struct {
	Success     string       `json:"success"`
	Error       string       `json:"error"`
}

// License defines the internal representation of a license file which is just
// a plaintext string.
type License struct {
	Content string
}