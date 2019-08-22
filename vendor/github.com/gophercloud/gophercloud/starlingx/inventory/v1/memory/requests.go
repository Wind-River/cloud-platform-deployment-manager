/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package memory

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

// Defines the accepted function values for memory configurations.
const (
	MemoryFunctionPlatform = "platform"
	MemoryFunctionVSwitch  = "vswitch"
	MemoryFunctionVM       = "vm"
)

type MemoryOpts struct {
	Function            string `json:"function,omitempty" mapstructure:"-"`
	Platform            *int   `json:"platform_reserved_mib,omitempty" mapstructure:"platform_reserved_mib"`
	VMHugepages2M       *int   `json:"vm_hugepages_nr_2M_pending,omitempty" mapstructure:"vm_hugepages_nr_2M_pending"`
	VMHugepages1G       *int   `json:"vm_hugepages_nr_1G_pending,omitempty" mapstructure:"vm_hugepages_nr_1G_pending"`
	VSwitchHugepages    *int   `json:"vswitch_hugepages_reqd,omitempty" mapstructure:"vswitch_hugepages_reqd"`
	VSwitchHugepageSize *int   `json:"vswitch_hugepages_size_mib,omitempty" mapstructure:"vswitch_hugepages_size_mib"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToMemoryListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the Memory attributes you want to see returned. SortKey allows you to sort
// by a particular Memory attribute. SortDir sets the direction, and is either
// `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToMemoryListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToMemoryListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// Memorys. It accepts a ListOpts struct, which allows you to filter and sort
// the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, hostid string, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c, hostid)
	if opts != nil {
		query, err := opts.ToMemoryListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return MemoryPage{pagination.SinglePageBase(r)}
	})
}

// Update accepts an array of MemoryOpts and updates the specified host with the
// desired Memory configuration.
func Update(c *gophercloud.ServiceClient, memid string, opts MemoryOpts) (r UpdateResult) {
	reqBody, err := inventoryv1.ConvertToPatchMap(opts, inventoryv1.ReplaceOp)
	if err != nil {
		r.Err = err
		return r
	}

	// Send request to API
	_, r.Err = c.Patch(updateURL(c, memid), reqBody, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201},
	})

	return r
}

// ListMemory is a convenience function to list and extract the entire list of
// Memory configurations for a given host.
func ListMemory(c *gophercloud.ServiceClient, hostid string) ([]Memory, error) {
	pages, err := List(c, hostid, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractMemorys(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
