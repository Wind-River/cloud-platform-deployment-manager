/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package ptp

import (
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

type PTPOpts struct {
	Enabled   *bool   `json:"enabled,omitempty" mapstructure:"enabled"`
	Mode      *string `json:"mode,omitempty" mapstructure:"mode"`
	Transport *string `json:"transport,omitempty" mapstructure:"transport"`
	Mechanism *string `json:"mechanism,omitempty" mapstructure:"mechanism"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToPTPListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. SortKey allows you to sort by a particular PTP attribute. SortDir
// sets the direction, and is either `asc' or `desc'. Marker and Limit are used
// for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToPTPListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToPTPListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// PTPs. It accepts a ListOpts struct, which allows you to filter and sort
// the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToPTPListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return PTPPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific PTP based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(getURL(c, id), &r.Body, nil)
	return r
}

// Update accepts a PatchOpts struct and updates an existing PTP using the
// values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts PTPOpts) (r UpdateResult) {
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

// GetDefaultPTP returns a single PTP record.  There should only be a single
// record available therefore an error is returned if there are more or less
// than 1 record.
func GetDefaultPTP(c *gophercloud.ServiceClient) (*PTP, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractPTPs(pages)
	if err != nil {
		return nil, err
	}

	if len(objs) != 1 {
		return nil, errors.New(fmt.Sprintf("Unexpected number of PTP records: %d", len(objs)))
	}

	return &objs[0], err
}

// GetSystemPTP returns a single PTP record for the specific system id.
func GetSystemPTP(c *gophercloud.ServiceClient, systemID string) (*PTP, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractPTPs(pages)
	if err != nil {
		return nil, err
	}

	for _, record := range objs {
		if record.SystemID == systemID {
			return &record, nil
		}
	}

	err = fmt.Errorf("no matching PTP records found for id %s", systemID)

	return nil, err
}
