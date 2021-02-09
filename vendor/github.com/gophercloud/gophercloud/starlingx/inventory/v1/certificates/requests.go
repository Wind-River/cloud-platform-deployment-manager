/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package certificates

import (
	"bytes"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	"mime/multipart"
)

type CertificateOpts struct {
	Type       string  `json:"mode,omitempty" mapstructure:"mode"`
	Passphrase *string `json:"-" mapstructure:"passphrase"`
	File       []byte  `json:"-" mapstructure:"file"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToCertificateListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. SortKey allows you to sort by a particular Certificate attribute.
// SortDir sets the direction, and is either `asc' or `desc'. Marker and Limit
// are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToCertificateListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToCertificateListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// Certificates. It accepts a ListOpts struct, which allows you to filter and
// sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToCertificateListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return CertificatePage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific Certificate based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(getURL(c, id), &r.Body, nil)
	return r
}

// Create accepts a CreateOpts struct and creates a new Certificate using the
// values provided. The operation parameters and certificate file contents are
// encoded as a MIME multipart message.
func Create(c *gophercloud.ServiceClient, opts CertificateOpts) (r CreateResult) {
	var b bytes.Buffer

	// Setup a new multipart write that is backed by a byte buffer.  As new
	// parts are created they are written to the byte buffer.
	w := multipart.NewWriter(&b)

	err := w.WriteField("mode", opts.Type)
	if err != nil {
		r.Err = err
		return r
	}

	err = w.WriteField("force", "true")
	if err != nil {
		r.Err = err
		return r
	}

	if opts.Passphrase != nil {
		err = w.WriteField("passphrase", *opts.Passphrase)
		if err != nil {
			r.Err = err
			return r
		}
	}

	// Create a new mime part to hold the certificate contents.
	p, err := w.CreateFormFile("file", "certificate.pem")
	if err != nil {
		r.Err = err
		return r
	}

	_, err = p.Write(opts.File)
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

// ListCertificates is a convenience function to list and extract the entire
// list of system certificates
func ListCertificates(c *gophercloud.ServiceClient) ([]Certificate, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractCertificates(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
