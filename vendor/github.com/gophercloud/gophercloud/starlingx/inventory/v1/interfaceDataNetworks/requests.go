/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package interfaceDataNetworks

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

type InterfaceDataNetworkOpts struct {
	InterfaceUUID   string `json:"interface_uuid,omitempty" mapstructure:"interface_uuid"`
	DataNetworkUUID string `json:"datanetwork_uuid,omitempty" mapstructure:"datanetwork_uuid"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToInterfaceDataNetworkListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the InterfaceDataNetwork attributes you want to see returned. SortKey allows
// you to sort by a particular InterfaceDataNetwork attribute. SortDir sets the
// direction, and is either `asc' or `desc'. Marker and Limit are used for
// pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToInterfaceDataNetworkListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToInterfaceDataNetworkListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// InterfaceDataNetworks. It accepts a ListOpts struct, which allows you to
// filter and sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, hostid string, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c, hostid)
	if opts != nil {
		query, err := opts.ToInterfaceDataNetworkListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return InterfaceDataNetworkPage{pagination.SinglePageBase(r)}
	})
}

// Create accepts a CreateOpts struct and creates a new AddressPool using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts InterfaceDataNetworkOpts) (r CreateResult) {
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
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, id), nil)
	return r
}

// ListInterfaceDataNetworks is a convenience function to list and extract the
// entire list of platform InterfaceDataNetworks
func ListInterfaceDataNetworks(c *gophercloud.ServiceClient, hostid string) ([]InterfaceDataNetwork, error) {
	pages, err := List(c, hostid, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractInterfaceDataNetworks(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
