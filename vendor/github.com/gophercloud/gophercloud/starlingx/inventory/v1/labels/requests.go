/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package labels

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToLabelListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. SortKey allows you to sort by a particular Label attribute.
// SortDir sets the direction, and is either `asc' or `desc'. Marker and Limit
// are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToLabelListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToLabelListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// Labels. It accepts a ListOpts struct, which allows you to filter
// and sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, hostid string, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c, hostid)
	if opts != nil {
		query, err := opts.ToLabelListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return LabelPage{pagination.SinglePageBase(r)}
	})
}

// Create accepts a CreateOpts struct and creates a new Label using the
// values provided.
func Create(c *gophercloud.ServiceClient, hostid string, labels map[string]string) (r CreateResult) {
	_, r.Err = c.Post(createURL(c, hostid), labels, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201, 202},
	})
	return r
}

// Delete accepts a unique ID and deletes the volume group associated with it.
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, id), nil)
	return r
}

// ListBackends returns the current set of configured labels on a host.
func ListLabels(c *gophercloud.ServiceClient, hostid string) ([]Label, error) {
	pages, err := List(c, hostid, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractLabels(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
