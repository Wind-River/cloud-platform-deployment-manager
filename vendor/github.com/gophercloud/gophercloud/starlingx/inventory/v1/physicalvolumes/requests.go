/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package physicalvolumes

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

const (
	PVTypeDisk      = "disk"
	PVTypePartition = "partition"
)

type PhysicalVolumeOpts struct {
	HostID        string `json:"ihost_uuid" mapstructure:"ihost_uuid"`
	VolumeGroupID string `json:"ilvg_uuid" mapstructure:"ilvg_uuid"`
	DeviceID      string `json:"disk_or_part_uuid" mapstructure:"disk_or_part_uuid"`
	Type          string `json:"pv_type" mapstructure:"pv_type"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToPhysicalVolumeListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the interface attributes you want to see returned. SortKey allows you to
// sort by a particular interface attribute. SortDir sets the direction, and is
// either `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	ID      string `q:"id"`
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToInterfaceListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToPhysicalVolumeListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// PhysicalVolumes. It accepts a ListOpts struct, which allows you to filter
// and sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, hostID string, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c, hostID)
	if opts != nil {
		query, err := opts.ToPhysicalVolumeListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return PhysicalVolumePage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific interface based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) GetResult {
	var res GetResult
	_, res.Err = c.Get(getURL(c, id), &res.Body, nil)
	return res
}

// Create accepts a CreateOpts struct and creates a new volume group using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts PhysicalVolumeOpts) (r CreateResult) {
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

// Update accepts a PatchOpts struct and updates an existing volume group using
// the values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts PhysicalVolumeOpts) (r UpdateResult) {
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

// Delete accepts a unique ID and deletes the volume group associated with it.
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, id), nil)
	return r
}

// ListPhysicalVolumes is a convenience function to list and extract the entire
// list of Physical Volumes on a given host.
func ListPhysicalVolumes(c *gophercloud.ServiceClient, hostid string) ([]PhysicalVolume, error) {
	pages, err := List(c, hostid, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractPhysicalVolumes(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
