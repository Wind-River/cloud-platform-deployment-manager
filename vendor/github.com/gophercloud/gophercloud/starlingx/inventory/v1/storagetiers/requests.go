/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package storagetiers

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

const StorageTierName = "storage"

type StorageTierOpts struct {
	ClusterID string `json:"cluster_uuid" mapstructure:"cluster_uuid"`
	Name      string `json:"name" mapstructure:"name"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToStorageTierListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. SortKey allows you to sort by a particular StorageTier attribute.
// SortDir sets the direction, and is either `asc' or `desc'. Marker and Limit
// are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToStorageTierListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToStorageTierListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// StorageTiers. It accepts a ListOpts struct, which allows you to filter
// and sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, clusterID string, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c, clusterID)
	if opts != nil {
		query, err := opts.ToStorageTierListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return StorageTierPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific StorageTier based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(getURL(c, id), &r.Body, nil)
	return r
}

// Create accepts a CreateOpts struct and creates a new StorageTier using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts StorageTierOpts) (r CreateResult) {
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

// Update accepts a PatchOpts struct and updates an existing StorageTier
// using the values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts StorageTierOpts) (r UpdateResult) {
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

// Delete accepts a unique ID and deletes the storage backend associated with
// it.
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, id), nil)
	return r
}

// ListTiers returns the current set of configured storage backends
func ListTiers(c *gophercloud.ServiceClient, clusterID string) ([]StorageTier, error) {
	pages, err := List(c, clusterID, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractStorageTiers(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
