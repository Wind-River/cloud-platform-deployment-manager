/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2021 Wind River Systems, Inc. */

package system

import (
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

type SystemOpts struct {
	Name         *string `json:"name,omitempty" mapstructure:"name"`
	Description  *string `json:"description,omitempty" mapstructure:"description"`
	Location     *string `json:"location,omitempty" mapstructure:"location"`
	Latitude     *string `json:"latitude,omitempty" mapstructure:"latitude"`
	Longitude    *string `json:"longitude,omitempty" mapstructure:"longitude"`
	Contact      *string `json:"contact,omitempty" mapstructure:"contact"`
	HTTPSEnabled *string `json:"https_enabled,omitempty" mapstructure:"https_enabled"`
	SDNEnabled   *string `json:"sdn_enabled,omitempty" mapstructure:"sdn_enabled"`
	VSwitchType  *string `json:"vswitch_type,omitempty" mapstructure:"vswitch_type"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToSystemListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the system attributes you want to see returned. SortKey allows you to sort
// by a particular system attribute. SortDir sets the direction, and is either
// `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToSystemListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToSystemListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// systems. It accepts a ListOpts struct, which allows you to filter and sort
// the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToSystemListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return SystemPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific system based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(getURL(c, id), &r.Body, nil)
	return r
}

// Update accepts a PatchOpts struct and updates an existing system using the
// values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts SystemOpts) (r UpdateResult) {
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

// GetDefaultSystem returns a single system record.  There should only be a
// single record available therefore an error is returned if there are more or
// less than 1 record.
func GetDefaultSystem(c *gophercloud.ServiceClient) (*System, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractSystems(pages)
	if err != nil {
		return nil, err
	}

	if len(objs) != 1 {
		return nil, errors.New(fmt.Sprintf("Unexpected number of system records: %d", len(objs)))
	}

	return &objs[0], err
}
