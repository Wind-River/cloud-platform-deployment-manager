/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package dns

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

type DNSOpts struct {
	Nameservers *string `json:"nameservers,omitempty" mapstructure:"nameservers"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToDNSListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. SortKey allows you to sort by a particular DNS attribute. SortDir
// sets the direction, and is either `asc' or `desc'. Marker and Limit are used
// for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToDNSListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToDNSListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// DNSs. It accepts a ListOpts struct, which allows you to filter and sort
// the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToDNSListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return DNSPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific DNS based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(getURL(c, id), &r.Body, nil)
	return r
}

// Update accepts a PatchOpts struct and updates an existing DNS using the
// values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts DNSOpts) (r UpdateResult) {
	reqBody, err := inventoryv1.ConvertToPatchMap(opts, inventoryv1.ReplaceOp)
	if err != nil {
		r.Err = err
		return r
	}

	// Send request to API
	_, r.Err = c.Patch(updateURL(c, id), reqBody, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201},
	})

	return r
}

// GetDefaultDNS returns a single DNS record.
// There should only be a single record available therefore an error is
// returned if there are more or less than 1 record.
func GetDefaultDNS(c *gophercloud.ServiceClient) (*DNS, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractDNSs(pages)
	if err != nil {
		return nil, err
	}

	if len(objs) != 1 {
		return nil, fmt.Errorf("unexpected number of DNS records: %d", len(objs))
	}

	return &objs[0], err
}

// GetSystemDNS returns a single DNS record for the specific system id.
func GetSystemDNS(c *gophercloud.ServiceClient, systemID string) (*DNS, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractDNSs(pages)
	if err != nil {
		return nil, err
	}

	for _, record := range objs {
		if record.SystemID == systemID {
			return &record, nil
		}
	}

	err = fmt.Errorf("no matching DNS records found for id %s", systemID)

	return nil, err
}
