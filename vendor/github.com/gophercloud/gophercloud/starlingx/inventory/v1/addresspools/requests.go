/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package addresspools

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

type AddressPoolOpts struct {
	Name    *string     `json:"name,omitempty" mapstructure:"name"`
	Network *string     `json:"network,omitempty" mapstructure:"network"`
	Prefix  *int        `json:"prefix,omitempty" mapstructure:"prefix"`
	Gateway *string     `json:"gateway_address,omitempty" mapstructure:"gateway_address,omitempty"`
	Order   *string     `json:"order,omitempty" mapstructure:"order"`
	Ranges  *[][]string `json:"ranges,omitempty" mapstructure:"ranges"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToAddressPoolListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the addressPool attributes you want to see returned. SortKey allows you to
// sort by a particular addressPool attribute. SortDir sets the direction, and
// is either `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToAddressPoolListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToAddressPoolListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// addressPools. It accepts a ListOpts struct, which allows you to filter and
// sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToAddressPoolListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return AddressPoolPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific addressPool based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) GetResult {
	var res GetResult
	_, res.Err = c.Get(getURL(c, id), &res.Body, nil)
	return res
}

// Create accepts a CreateOpts struct and creates a new AddressPool using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts AddressPoolOpts) (r CreateResult) {
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

// Update accepts a PatchOpts struct and updates an existing addressPool using
// the values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts AddressPoolOpts) (r UpdateResult) {
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

// ListAddressPools is a convenience function to list and extract the entire
// list of address pools
func ListAddressPools(c *gophercloud.ServiceClient) ([]AddressPool, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractAddressPools(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
