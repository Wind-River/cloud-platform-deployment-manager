/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package controllerFilesystems

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

const (
	CGCS               = "glance"
	Cinder             = "cinder"
	Database           = "database"
	DockerDistribution = "docker-distribution"
	Extension          = "extension"
	ETCD               = "etcd"
	PatchVault         = "patch-vault"
)

const (
	Available        = "available"
	ResizeInProgress = "drbd_fs_resizing_in_progress"
)

type FileSystemOpts struct {
	Name   string  `json:"name,omitempty" mapstructure:"name"`
	Size   int     `json:"size,omitempty" mapstructure:"size"`
	Action *string `json:"action,omitempty" mapstructure:"action"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToFileSystemListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the filesystem attributes you want to see returned. SortKey allows you to
// sort by a particular filesystem attribute. SortDir sets the direction, and is
// either `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToFileSystemListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToFileSystemListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// filesystems. It accepts a ListOpts struct, which allows you to filter and
// sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToFileSystemListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return FileSystemPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific filesystem based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(getURL(c, id), &r.Body, nil)
	return r
}

// Update accepts an array of FileSystemOpts and creates a new FileSystem using
// the values provided.
func Update(c *gophercloud.ServiceClient, systemId string, opts []FileSystemOpts) (r UpdateResult) {
	requests := make([]interface{}, 0)

	for _, opt := range opts {
		reqBody, err := inventoryv1.ConvertToPatchMap(opt, inventoryv1.ReplaceOp)
		if err != nil {
			r.Err = err
			return r
		}

		requests = append(requests, reqBody)
	}

	_, r.Err = c.Put(updateURL(c, systemId), requests, nil, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201, 202, 204},
	})

	return r
}

// ListFileSystems is a convenience function to list and extract the entire
// list of filesystems
func ListFileSystems(c *gophercloud.ServiceClient) ([]FileSystem, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractFileSystems(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
