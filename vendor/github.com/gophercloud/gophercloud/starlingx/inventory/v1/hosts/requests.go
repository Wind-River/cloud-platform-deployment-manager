/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package hosts

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

type LocationOpts struct {
	Name string `json:"locn" mapstructure:"locn"`
}

type HostOpts struct {
	Hostname             *string       `json:"hostname,omitempty" mapstructure:"hostname"`
	Personality          *string       `json:"personality,omitempty" mapstructure:"personality"`
	SubFunctions         *string       `json:"subfunctions,omitempty" mapstructure:"subfunctions"`
	Location             *LocationOpts `json:"location,omitempty" mapstructure:"location"`
	InstallOutput        *string       `json:"install_output,omitempty" mapstructure:"install_output"`
	Console              *string       `json:"console,omitempty" mapstructure:"console"`
	BootIP               *string       `json:"mgmt_ip,omitempty" mapstructure:"mgmt_ip"`
	BootMAC              *string       `json:"mgmt_mac,omitempty" mapstructure:"mgmt_mac"`
	RootDevice           *string       `json:"rootfs_device,omitempty" mapstructure:"rootfs_device"`
	BootDevice           *string       `json:"boot_device,omitempty" mapstructure:"boot_device"`
	BMAddress            *string       `json:"bm_ip,omitempty" mapstructure:"bm_ip"`
	BMType               *string       `json:"bm_type,omitempty" mapstructure:"bm_type"`
	BMUsername           *string       `json:"bm_username,omitempty" mapstructure:"bm_username"`
	BMPassword           *string       `json:"-" mapstructure:"bm_password"`
	ClockSynchronization *string       `json:"clock_synchronization,omitempty" mapstructure:"clock_synchronization"`
	Action               *string       `json:"action,omitempty" mapstructure:"action"`
	MaxCPUFrequency      *string       `json:"max_cpu_frequency,omitempty" mapstructure:"max_cpu_frequency"`
}

const (
	ActionLock      = "lock"
	ActionUnlock    = "unlock"
	ActionPowerOn   = "power-on"
	ActionPowerOff  = "power-off"
	ActionReinstall = "reinstall"
)

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToHostListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the host attributes you want to see returned. SortKey allows you to sort
// by a particular host attribute. SortDir sets the direction, and is either
// `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	ID       string `q:"id"`
	Hostname string `q:"name"`
	Marker   string `q:"marker"`
	Limit    int    `q:"limit"`
	SortKey  string `q:"sort_key"`
	SortDir  string `q:"sort_dir"`
}

// ToHostListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToHostListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// hosts. It accepts a ListOpts struct, which allows you to filter and sort
// the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToHostListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return HostPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific host based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) GetResult {
	var res GetResult
	_, res.Err = c.Get(getURL(c, id), &res.Body, nil)
	return res
}

// Create accepts a CreateOpts struct and creates a new host using the values
// provided.
func Create(c *gophercloud.ServiceClient, opts HostOpts) (r CreateResult) {
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

// Update accepts a PatchOpts struct and updates an existing host using the
// values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts HostOpts) (r UpdateResult) {
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

// Delete accepts a unique ID and deletes the host associated with it.
func Delete(c *gophercloud.ServiceClient, hostID string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, hostID), nil)
	return r
}

// ListHosts is a convenience function to list and extract the entire
// list of hosts
func ListHosts(c *gophercloud.ServiceClient) ([]Host, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractHosts(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
