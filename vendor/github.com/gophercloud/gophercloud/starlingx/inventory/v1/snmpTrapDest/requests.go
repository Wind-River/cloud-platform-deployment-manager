/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package snmpTrapDest

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

type SNMPTrapDestOpts struct {
	Community *string `json:"community,omitempty" mapstructure:"community"`
	IPAddress *string `json:"ip_address,omitempty" mapstructure:"ip_address"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToSNMPTrapDestListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the SNMPTrapDest attributes you want to see returned. SortKey allows you to
// sort by a particular SNMPTrapDest attribute. SortDir sets the direction, and
// is either `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToSNMPTrapDestListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToSNMPTrapDestListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// SNMPCommunities. It accepts a ListOpts struct, which allows you to filter and
// sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToSNMPTrapDestListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return SNMPTrapDestPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific SNMPTrapDest based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(getURL(c, id), &r.Body, nil)
	return r
}

// Create accepts a CreateOpts struct and creates a new SNMPTrapDest using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts SNMPTrapDestOpts) (r CreateResult) {
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

// ListSNMPTrapDests is a convenience function to list and extract the entire
// list of SNMPTrapDests
func ListSNMPTrapDests(c *gophercloud.ServiceClient) ([]SNMPTrapDest, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractSNMPTrapDests(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
