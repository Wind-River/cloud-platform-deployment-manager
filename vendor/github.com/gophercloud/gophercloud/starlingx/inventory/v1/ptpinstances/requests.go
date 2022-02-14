/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package ptpinstances

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

const (
	ServicePtp4l   = "ptp4l"
	ServicePhc2sys = "phc2sys"
	ServiceTs2phc  = "ts2phc"
)


type PTPInstanceOpts struct {
	Name	  *string `json:"name,omitempty" mapstructure:"name"`
	Service   *string `json:"service,omitempty" mapstructure:"service"`
}

// PATCH /v1/ptp_instances/{ptpinstance_uuid}
type PTPParamToPTPInstOpts struct {
	Parameter *string `json:"ptp_parameters/-,omitempty" mapstructure:"ptp_parameters/-"`
}

// PATCH /v1/ihosts/{host_uuid}
// [{"path": "/ptp_instances/-", "value": {ptp_instance_id}, "op": "op"}]
type PTPInstToHostOpts struct {
	PTPInstanceID *int `json:"ptp_instances/-,omitempty" mapstructure:"ptp_instances/-"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToPTPInstanceListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the dataNetwork attributes you want to see returned. SortKey allows you to
// sort by a particular dataNetwork attribute. SortDir sets the direction, and
// is either `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToPTPInstanceListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToPTPInstanceListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// ptp instances. It accepts a ListOpts struct, which allows you to filter and
// sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToPTPInstanceListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return PTPInstancePage{pagination.SinglePageBase(r)}
	})
}

// List returns a Pager which allows you to iterate over a collection of
// ptp instances that assigned to a host.
func HostList(c *gophercloud.ServiceClient, hostID string, opts ListOptsBuilder) pagination.Pager {
	url := hostListURL(c, hostID)
	if opts != nil {
		query, err := opts.ToPTPInstanceListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return PTPInstancePage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific PTPInstance based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) GetResult {
	var res GetResult
	_, res.Err = c.Get(getURL(c, id), &res.Body, nil)
	return res
}

// Create accepts a CreateOpts struct and creates a new PTPInstance using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts PTPInstanceOpts) (r CreateResult) {
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

// AddPTPParamToPTPInst accepts a PatchOpts struct and updates an existing 
// PTPInstance to associate with a PTP parameter
func AddPTPParamToPTPInst(c *gophercloud.ServiceClient, id string, opts PTPParamToPTPInstOpts) (r UpdateResult) {
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

// RemovePTPParamFromPTPInst accepts a PatchOpts struct and updates an existing 
// PTPInstance to remove a certain PTP parameter
func RemovePTPParamFromPTPInst(c *gophercloud.ServiceClient, id string, opts PTPParamToPTPInstOpts) (r UpdateResult) {
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

// Delete accepts a unique ID and deletes the related resource.
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, id), nil)
	return r
}

// Apply applies the PTP instance configuration
func Apply(c *gophercloud.ServiceClient, opts PTPInstanceOpts) (r ApplyResult) {
	reqBody, err := inventoryv1.ConvertToCreateMap(opts)
	if err != nil {
		r.Err = err
		return r
	}

	_, r.Err = c.Post(applyURL(c), reqBody, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{204},
	})
	return r
}

// ListPTPInstances is a convenience function to list and extract the entire
// list of ptp instances
func ListPTPInstances(c *gophercloud.ServiceClient) ([]PTPInstance, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractPTPInstances(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}

// ListHostPTPInstances is a convenience function to list and extract the entire
// list of ptp instances of a host
func ListHostPTPInstances(c *gophercloud.ServiceClient, hostID string) ([]PTPInstance, error) {
	pages, err := HostList(c, hostID, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractPTPInstances(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}

// AddPTPInstanceToHost accepts a PatchOpts struct and updates an existing 
// PTPInstance to associate with a host.
func AddPTPInstanceToHost(c *gophercloud.ServiceClient, hostID string, opts PTPInstToHostOpts) (r UpdateResult) {
	reqBody, err := inventoryv1.ConvertToPatchMap(opts, inventoryv1.AddOp)
	if err != nil {
		r.Err = err
		return r
	}

	// Send request to API
	_, r.Err = c.Patch(hostUpdateURL(c, hostID), reqBody, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201},
	})

	return r
}

// RemovePTPInstanceFromHost accepts a PatchOpts struct and remove the PTP instance
// from a host.
func RemovePTPInstanceFromHost(c *gophercloud.ServiceClient, hostID string, opts PTPInstToHostOpts) (r UpdateResult) {
	reqBody, err := inventoryv1.ConvertToPatchMap(opts, inventoryv1.RemoveOp)
	if err != nil {
		r.Err = err
		return r
	}

	// Send request to API
	_, r.Err = c.Patch(hostUpdateURL(c, hostID), reqBody, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201},
	})

	return r
}
