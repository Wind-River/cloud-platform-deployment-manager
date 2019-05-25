/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package partitions

import (
	"github.com/alecthomas/units"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

const (
	// Disk Partitions: From sysinv constants
	// User creatable disk partitions, system managed,  GUID partitions types
	PartitionUserManagedGUIDPrefix = "ba5eba11-0000-1111-2222-"
	UserPartitionPhysicalVolume    = PartitionUserManagedGUIDPrefix + "000000000001"
)

// Defines the disk partition type values.
const (
	PartitionTypeLVM = "lvm_phys_vol"
)

// Defines the mapping between disk partition type values and the partition
// GUID to be used for that partition type.
var PartitionTypeMap = map[string]string{
	PartitionTypeLVM: UserPartitionPhysicalVolume,
}

// Defines the disk partition status values.
const (
	StatusCreating  = 2
	StatusDeleting  = 4
	StatusModifying = 5
)

type DiskPartitionOpts struct {
	HostID   string  `json:"ihost_uuid" mapstructure:"ihost_uuid"`
	DiskID   string  `json:"idisk_uuid" mapstructure:"idisk_uuid"`
	Size     int     `json:"size_mib" mapstructure:"size_mib"`
	TypeName *string `json:"type_name" mapstructure:"type_name"`
	TypeGUID *string `json:"type_guid" mapstructure:"type_guid"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToDiskPartitionListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the interface attributes you want to see returned. SortKey allows you to
// sort by a particular interface attribute. SortDir sets the direction, and is
// either `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Path    string `q:"device_path"`
	ID      string `q:"id"`
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToInterfaceListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToDiskPartitionListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// partitions. It accepts a ListOpts struct, which allows you to filter and
// sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, hostID string, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c, hostID)
	if opts != nil {
		query, err := opts.ToDiskPartitionListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return DiskPartitionPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific interface based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) GetResult {
	var res GetResult
	_, res.Err = c.Get(getURL(c, id), &res.Body, nil)
	return res
}

// fixUnits is a utility method which converts the incoming size attributes
// from GiB units to the system API MiB equivalent.
// TODO(alegacy): remove once system API is converted to GiB units.
//  See: https://bugs.launchpad.net/bugs/1823737
func (opts *DiskPartitionOpts) fixUnits() {
	opts.Size = opts.Size * int(units.Kibibyte) // GiB -> MiB
}

// Create accepts a CreateOpts struct and creates a new partition using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts DiskPartitionOpts) (r CreateResult) {
	opts.fixUnits()

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

// Update accepts a PatchOpts struct and updates an existing partition using the
// values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts DiskPartitionOpts) (r UpdateResult) {
	opts.fixUnits()

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

// Delete accepts a unique ID and deletes the partition associated with it.
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, id), nil)
	return r
}

// ListPartitions is a convenience function to list and extract the entire
// list of Partitions on a given host.
func ListPartitions(c *gophercloud.ServiceClient, hostid string) ([]DiskPartition, error) {
	pages, err := List(c, hostid, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractDiskPartitions(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
