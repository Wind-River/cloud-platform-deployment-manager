/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package cpus

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Defines the accepted function values for memory configurations.
const (
	CPUFunctionPlatform     = "platform"
	CPUFunctionVSwitch      = "vswitch"
	CPUFunctionShared       = "shared"
	CPUFunctionApplications = "applications"
)

type CPUOpts struct {
	Function string           `json:"function" mapstructure:"function"`
	Sockets  []map[string]int `json:"sockets" mapstructure:"sockets"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToCPUListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the cPU attributes you want to see returned. SortKey allows you to sort
// by a particular cPU attribute. SortDir sets the direction, and is either
// `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToCPUListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToCPUListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// cPUs. It accepts a ListOpts struct, which allows you to filter and sort
// the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, hostid string, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c, hostid)
	if opts != nil {
		query, err := opts.ToCPUListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return CPUPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific cPU based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(getURL(c, id), &r.Body, nil)
	return r
}

// Update accepts an array of CPUOpts and updates the specified host with the
// desired CPU configuration.
func Update(c *gophercloud.ServiceClient, hostid string, opts []CPUOpts) (r UpdateResult) {
	// Send request to API
	_, r.Err = c.Put(updateURL(c, hostid), opts, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201},
	})

	return r
}

// ListCPUs is a convenience function to list and extract the entire list of CPU
// instances on a given host
func ListCPUs(c *gophercloud.ServiceClient, hostid string) ([]CPU, error) {
	pages, err := List(c, hostid, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractCPUs(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
