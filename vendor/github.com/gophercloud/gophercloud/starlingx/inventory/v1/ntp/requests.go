/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package ntp

import (
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

type NTPOpts struct {
	NTPServers *string `json:"ntpservers,omitempty" mapstructure:"ntpservers"`
	Enabled    *string `json:"enabled,omitempty" mapstructure:"enabled"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToNTPListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. SortKey allows you to sort by a particular NTP attribute. SortDir
// sets the direction, and is either `asc' or `desc'. Marker and Limit are used
// for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToNTPListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToNTPListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// NTPs. It accepts a ListOpts struct, which allows you to filter and sort
// the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToNTPListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return NTPPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific NTP based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) GetResult {
	var res GetResult
	_, res.Err = c.Get(getURL(c, id), &res.Body, nil)
	return res
}

// Update accepts a PatchOpts struct and updates an existing NTP using the
// values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts NTPOpts) (r UpdateResult) {
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

// GetDefaultNTP returns a single NTP record.  There should only be a single
// record available therefore an error is returned if there are more or less
// than 1 record.
func GetDefaultNTP(c *gophercloud.ServiceClient) (*NTP, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractNTPs(pages)
	if err != nil {
		return nil, err
	}

	if len(objs) != 1 {
		return nil, errors.New(fmt.Sprintf("Unexpected number of NTP records: %d", len(objs)))
	}

	return &objs[0], err
}

// GetSystemNTP returns a single NTP record for the specific system id.
func GetSystemNTP(c *gophercloud.ServiceClient, systemID string) (*NTP, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractNTPs(pages)
	if err != nil {
		return nil, err
	}

	for _, record := range objs {
		if record.SystemID == systemID {
			return &record, nil
		}
	}

	err = fmt.Errorf("no matching NTP records found for id %s", systemID)

	return nil, err
}
