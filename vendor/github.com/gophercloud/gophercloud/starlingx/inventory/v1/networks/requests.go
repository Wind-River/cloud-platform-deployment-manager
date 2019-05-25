/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package networks

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

const (
	// Defines commonly used network type values
	NetworkTypeOther = "other"
)

const (
	AllocationOrderDynamic = "dynamic"
	AllocationOrderStatic  = "static"
)

type NetworkOpts struct {
	Name     *string `json:"name,omitempty" mapstructure:"name"`
	Type     *string `json:"type,omitempty" mapstructure:"type"`
	Dynamic  *bool   `json:"dynamic,omitempty" mapstructure:"dynamic"`
	PoolUUID *string `json:"pool_uuid,omitempty" mapstructure:"pool_uuid"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToNetworkListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the network attributes you want to see returned. SortKey allows you to sort
// by a particular network attribute. SortDir sets the direction, and is either
// `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToNetworkListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToNetworkListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// networks. It accepts a ListOpts struct, which allows you to filter and sort
// the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToNetworkListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return NetworkPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific network based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) GetResult {
	var res GetResult
	_, res.Err = c.Get(getURL(c, id), &res.Body, nil)
	return res
}

// Create accepts a CreateOpts struct and creates a new AddressPool using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts NetworkOpts) (r CreateResult) {
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

// Update accepts a PatchOpts struct and updates an existing network using the
// values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts NetworkOpts) (r UpdateResult) {
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

// Delete accepts a unique ID and deletes the related resource.
func Delete(c *gophercloud.ServiceClient, hostID string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, hostID), nil)
	return r
}

// ListNetworks is a convenience function to list and extract the entire list
// of platform networks
func ListNetworks(c *gophercloud.ServiceClient) ([]Network, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractNetworks(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
