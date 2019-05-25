/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package datanetworks

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

const (
	DefaultMTU = 1500
)

const (
	TypeFlat  = "flat"
	TypeVLAN  = "vlan"
	TypeVxLAN = "vxlan"
)

const (
	EndpointModeStatic  = "static"
	EndpointModeDynamic = "dynamic"
)

type DataNetworkOpts struct {
	Name           *string `json:"name,omitempty" mapstructure:"name"`
	Description    *string `json:"description,omitempty" mapstructure:"description"`
	Type           *string `json:"network_type,omitempty" mapstructure:"network_type"`
	MTU            *int    `json:"mtu,omitempty" mapstructure:"mtu"`
	TTL            *int    `json:"ttl,omitempty" mapstructure:"ttl"`
	Mode           *string `json:"mode,omitempty" mapstructure:"mode"`
	MulticastGroup *string `json:"multicast_group,omitempty" mapstructure:"multicast_group"`
	PortNumber     *int    `json:"port_num,omitempty" mapstructure:"port_num"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToDataNetworkListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the dataNetwork attributes you want to see returned. SortKey allows you to
// sort by a particular dataNetwork attribute. SortDir sets the direction, and
// is either `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToDataNetworkListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToDataNetworkListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// dataNetworks. It accepts a ListOpts struct, which allows you to filter and
// sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToDataNetworkListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return DataNetworkPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific dataNetwork based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(getURL(c, id), &r.Body, nil)
	return r
}

// Create accepts a CreateOpts struct and creates a new AddressPool using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts DataNetworkOpts) (r CreateResult) {
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

// Update accepts a PatchOpts struct and updates an existing dataNetwork using
// the values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts DataNetworkOpts) (r UpdateResult) {
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

// ListDataNetworks is a convenience function to list and extract the entire
// list of data networks
func ListDataNetworks(c *gophercloud.ServiceClient) ([]DataNetwork, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractDataNetworks(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
