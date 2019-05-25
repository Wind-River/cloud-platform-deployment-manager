/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package cephmonitors

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

type CephMonitorOpts struct {
	HostUUID *string `json:"ihost_uuid,omitempty" mapstructure:"ihost_uuid"`
	Size     *int    `json:"ceph_mon_gib" mapstructure:"ceph_mon_gib"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToCephMonitorListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. SortKey allows you to sort by a particular CephMonitor attribute.
// SortDir sets the direction, and is either `asc' or `desc'. Marker and Limit
// are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToCephMonitorListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToCephMonitorListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// CephMonitors. It accepts a ListOpts struct, which allows you to filter
// and sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToCephMonitorListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return CephMonitorPage{pagination.SinglePageBase(r)}
	})
}

// Create accepts a CreateOpts struct and creates a new CephMonitor using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts CephMonitorOpts) (r CreateResult) {
	reqBody, err := inventoryv1.ConvertToCreateMap(opts)
	if err != nil {
		r.Err = err
		return r
	}

	_, r.Err = c.Post(createURL(c), reqBody, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201, 202},
	})
	return r
}

// Update accepts a PatchOpts struct and updates an existing CephMonitor using
// the values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts CephMonitorOpts) (r UpdateResult) {
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

// Delete accepts a unique ID and deletes the volume group associated with it.
func Delete(c *gophercloud.ServiceClient, hostid string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, hostid), nil)
	return r
}

// ListCephMonitors returns the current set of configured CephMonitors on a
// host.
func ListCephMonitors(c *gophercloud.ServiceClient) ([]CephMonitor, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractCephMonitors(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
