/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package storagebackends

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

type StorageBackendOpts struct {
	Confirmed    bool                    `json:"confirmed,omitempty" mapstructure:"confirmed"`
	Backend      *string                 `json:"backend,omitempty" mapstructure:"backend"`
	Name         *string                 `json:"name,omitempty" mapstructure:"name"`
	State        *string                 `json:"state,omitempty" mapstructure:"state"`
	Task         *string                 `json:"task,omitempty" mapstructure:"task"`
	Services     *string                 `json:"services,omitempty" mapstructure:"services"`
	Network      *string                 `json:"network,omitempty" mapstructure:"network"`
	Capabilities *map[string]interface{} `json:"capabilities,omitempty" mapstructure:"capabilities"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToStorageBackendListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. SortKey allows you to sort by a particular StorageBackend attribute.
// SortDir sets the direction, and is either `asc' or `desc'. Marker and Limit
// are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToStorageBackendListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToStorageBackendListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// StorageBackends. It accepts a ListOpts struct, which allows you to filter
// and sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToStorageBackendListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return StorageBackendPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific StorageBackend based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) GetResult {
	var res GetResult
	_, res.Err = c.Get(getURL(c, id), &res.Body, nil)
	return res
}

// Create accepts a CreateOpts struct and creates a new StorageBackend using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts StorageBackendOpts) (r CreateResult) {
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

// Update accepts a PatchOpts struct and updates an existing StorageBackend
// using the values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts StorageBackendOpts) (r UpdateResult) {
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

// ListBackends returns the current set of configured storage backends
func ListBackends(c *gophercloud.ServiceClient) ([]StorageBackend, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractStorageBackends(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
