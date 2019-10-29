/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package licenses

import (
	"bytes"
	"github.com/gophercloud/gophercloud"
	"mime/multipart"
)

type LicenseOpts struct {
	Contents []byte `json:"-" mapstructure:"contents"`
}

// Get retrieves a specific Certificate based on its unique ID.
func Get(c *gophercloud.ServiceClient) (r GetResult) {
	_, r.Err = c.Get(getURL(c), &r.Body, nil)
	return r
}

// Create accepts a CreateOpts struct and creates a new License using the
// values provided. The operation parameters and license file contents are
// encoded as a MIME multipart message.
func Create(c *gophercloud.ServiceClient, opts LicenseOpts) (r CreateResult) {
	var b bytes.Buffer

	// Setup a new multipart write that is backed by a byte buffer.  As new
	// parts are created they are written to the byte buffer.
	w := multipart.NewWriter(&b)

	// Create a new mime part to hold the license contents.
	p, err := w.CreateFormFile("file", "license.lic")
	if err != nil {
		r.Err = err
		return r
	}

	_, err = p.Write(opts.Contents)
	if err != nil {
		r.Err = err
		return r
	}

	err = w.Close()
	if err != nil {
		r.Err = err
		return r
	}

	// Issue the post request using the byte buffer as the message body.
	_, r.Err = c.Post(createURL(c), &b, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201, 202},
		MoreHeaders: map[string]string{
			"Content-Type": w.FormDataContentType(),
		},
	})

	return r
}
