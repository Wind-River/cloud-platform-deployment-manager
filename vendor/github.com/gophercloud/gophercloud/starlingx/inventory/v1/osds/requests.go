/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package osds

import (
	"github.com/alecthomas/units"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

const (
	FunctionOSD     = "osd"
	FunctionJournal = "journal"
)

type OSDOpts struct {
	HostID          *string `json:"ihost_uuid,omitempty" mapstructure:"ihost_uuid"`
	Function        *string `json:"function,omitempty" mapstructure:"function"`
	DiskID          *string `json:"idisk_uuid,omitempty" mapstructure:"idisk_uuid"`
	TierUUID        *string `json:"tier_uuid,omitempty" mapstructure:"tier_uuid"`
	JournalLocation *string `json:"journal_location,omitempty" mapstructure:"journal_location"`
	JournalSize     *int    `json:"journal_size_mib,omitempty" mapstructure:"journal_size_mib"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToOSDListQuery() (string, error)
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
func (opts ListOpts) ToOSDListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// OSDs. It accepts a ListOpts struct, which allows you to filter
// and sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, hostID string, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c, hostID)
	if opts != nil {
		query, err := opts.ToOSDListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return OSDPage{pagination.SinglePageBase(r)}
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
func (opts *OSDOpts) fixUnits() {
	if opts.JournalSize != nil {
		*opts.JournalSize = *opts.JournalSize * int(units.Kibibyte)
	}
}

// Create accepts a CreateOpts struct and creates a new volume group using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts OSDOpts) (r CreateResult) {
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

// Update accepts a PatchOpts struct and updates an existing volume group using
// the values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts OSDOpts) (r UpdateResult) {
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

// Delete accepts a unique ID and deletes the volume group associated with it.
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, id), nil)
	return r
}

// ListOSDs is a convenience function to list and extract the entire list of
// OSD resources on a given host.
func ListOSDs(c *gophercloud.ServiceClient, hostid string) ([]OSD, error) {
	pages, err := List(c, hostid, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractOSDs(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
