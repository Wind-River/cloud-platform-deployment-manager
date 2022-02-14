/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package ptpinterfaces

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

type PTPInterfaceOpts struct {
	Name		    *string `json:"name,omitempty" mapstructure:"name"`
	PTPInstanceUUID *string `json:"ptp_instance_uuid, omitempty" mapstructure:"ptp_instance_uuid,omitempty"`
}

// PATCH /v1/ptp_interfaces/{ptpinterface_uuid}
type PTPParamToPTPIntOpts struct {
	Parameter *string `json:"ptp_parameters/-,omitempty" mapstructure:"ptp_parameters/-"`
}

// Patch /v1/iinterfaces/{interface_uuid}
// [{"path": "/ptp_interfaces/-", "value": {ptpinterface_id}, "op": "op"}]
type PTPIntToIntOpt struct {
	PTPinterfaceID *int `json:"ptp_interfaces/-,omitempty" mapstructure:"ptp_interfaces/-"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToPTPInterfaceListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the PTP interface attributes you want to see returned. SortKey allows you to
// sort by a particular PTP interface attribute. SortDir sets the direction, and
// is either `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToPTPInterfaceListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToPTPInterfaceListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// PTP interfaces. It accepts a ListOpts struct, which allows you to filter and
// sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToPTPInterfaceListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return PTPInterfacePage{pagination.SinglePageBase(r)}
	})
}

// HostList returns a pager which allows you to iterate over a collection of
// PTP interface assigned to a host.
func HostList(c *gophercloud.ServiceClient, hostID string, opts ListOptsBuilder) pagination.Pager {
	url := hostListURL(c, hostID)
	if opts != nil {
		query, err := opts.ToPTPInterfaceListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return PTPInterfacePage{pagination.SinglePageBase(r)}
	})
}

// InterfaceList returns a pager which allows you to iterate over a collection of
// PTP interface assigned to an interface.
func InterfaceList(c *gophercloud.ServiceClient, interfaceID string, opts ListOptsBuilder) pagination.Pager {
	url := interfaceListURL(c, interfaceID)
	if opts != nil {
		query, err := opts.ToPTPInterfaceListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return PTPInterfacePage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific PTPInterface based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) GetResult {
	var res GetResult
	_, res.Err = c.Get(getURL(c, id), &res.Body, nil)
	return res
}

// Create accepts a CreateOpts struct and creates a new PTPInterface using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts PTPInterfaceOpts) (r CreateResult) {
	reqBody, err := inventoryv1.ConvertToCreateMap(opts)
	if err != nil {
		r.Err = err
		return r
	}

	// Send request to API
	_, r.Err = c.Post(createURL(c), reqBody, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201, 202},
	})
	return r
}

// Delete accepts a unique ID and deletes the related resource.
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, id), nil)
	return r
}

// ListHostPTPInterfaces is a convenience function to list and extract the entire
// list of ptp interfaces assigned to a host.
func ListHostPTPInterfaces(c *gophercloud.ServiceClient, hostID string) ([]PTPInterface, error) {
	pages, err := HostList(c, hostID, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractPTPInterfaces(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}

// ListInterfacePTPInterfaces is a convenience function to list and extract the entire
// list of ptp interfaces assigned to an interface.
func ListInterfacePTPInterfaces(c *gophercloud.ServiceClient, interfaceID string) ([]PTPInterface, error) {
	pages, err := InterfaceList(c, interfaceID, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractPTPInterfaces(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}

// ListPTPInterfaces is a convenience function to list and extract the entire
// list of ptp interfaces
func ListPTPInterfaces(c *gophercloud.ServiceClient) ([]PTPInterface, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractPTPInterfaces(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}

// AddPTPParamToPTPInt accepts a PatchOpts struct and updates an existing PTPInterface to
// associate with a PTP parameter
func AddPTPParamToPTPInt(c *gophercloud.ServiceClient, id string, opts PTPParamToPTPIntOpts) (r UpdateResult) {
	reqBody, err := inventoryv1.ConvertToPatchMap(opts, inventoryv1.AddOp)
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


// RemovePTPParamFromPTPInt accepts a PatchOpts struct and updates an existing PTPInterface
// to remove a certain PTP parameter
func RemovePTPParamFromPTPInt(c *gophercloud.ServiceClient, id string, opts PTPParamToPTPIntOpts) (r UpdateResult) {
	reqBody, err := inventoryv1.ConvertToPatchMap(opts, inventoryv1.RemoveOp)
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

// AddPTPIntToInt accepts a PTPIntToIntOpt to associate the interface
// with a PTP interface
func AddPTPIntToInt(c *gophercloud.ServiceClient, interfaceID string, opts PTPIntToIntOpt) (r UpdateResult) {
	reqBody, err := inventoryv1.ConvertToPatchMap(opts, inventoryv1.AddOp)
	if err != nil {
		r.Err = err
		return r
	}

	// Send request to API
	_, r.Err = c.Patch(interfaceUpdateURL(c, interfaceID), reqBody, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201},
	})

	return r
}

// RemovePTPIntFromInt accepts a PTPIntToIntOpt to associate the interface
// with a PTP interface
func RemovePTPIntFromInt(c *gophercloud.ServiceClient, interfaceID string, opts PTPIntToIntOpt) (r UpdateResult) {
	reqBody, err := inventoryv1.ConvertToPatchMap(opts, inventoryv1.RemoveOp)
	if err != nil {
		r.Err = err
		return r
	}

	// Send request to API
	_, r.Err = c.Patch(interfaceUpdateURL(c, interfaceID), reqBody, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201},
	})

	return r
}
