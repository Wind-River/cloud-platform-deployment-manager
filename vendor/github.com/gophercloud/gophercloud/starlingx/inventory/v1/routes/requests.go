/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package routes

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

const (
	DefaultMetric = 1
)

type RouteOpts struct {
	InterfaceUUID *string `json:"interface_uuid,omitempty" mapstructure:"interface_uuid"`
	Network       *string `json:"network,omitempty" mapstructure:"network"`
	Prefix        *int    `json:"prefix,omitempty" mapstructure:"prefix"`
	Gateway       *string `json:"gateway,omitempty" mapstructure:"gateway"`
	Metric        *int    `json:"metric,omitempty" mapstructure:"metric"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToRouteListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the Route attributes you want to see returned. SortKey allows you to sort
// by a particular Route attribute. SortDir sets the direction, and is either
// `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToRouteListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToRouteListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// Routes. It accepts a ListOpts struct, which allows you to filter and sort
// the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, hostid string, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c, hostid)
	if opts != nil {
		query, err := opts.ToRouteListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return RoutePage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific Route based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(getURL(c, id), &r.Body, nil)
	return r
}

// Create accepts a CreateOpts struct and creates a new Route using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts RouteOpts) (r CreateResult) {
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

// Delete accepts a unique ID and deletes the related resource.
func Delete(c *gophercloud.ServiceClient, hostID string) DeleteResult {
	var res DeleteResult
	_, res.Err = c.Delete(deleteURL(c, hostID), nil)
	return res
}

// ListRoutes is a convenience function to list and extract the entire
// list of Routes
func ListRoutes(c *gophercloud.ServiceClient, hostid string) ([]Route, error) {
	pages, err := List(c, hostid, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractRoutes(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
