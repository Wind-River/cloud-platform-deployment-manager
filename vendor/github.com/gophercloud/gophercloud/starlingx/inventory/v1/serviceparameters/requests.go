/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2020 Wind River Systems, Inc. */

package serviceparameters

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

/* POST /v1/service_parameter/apply  supports an optional "service" request parameter  */
type ServiceApplyOpts struct {
	Service *string `json:"service,omitempty" mapstructure:"service"`
}

/* POST /v1/service_parameter expects: service, section and one or more  name:value parameters */
type ServiceParameterOpts struct {
	Service     *string            `json:"service,omitempty" mapstructure:"service"`
	Section     *string            `json:"section,omitempty" mapstructure:"section"`
	Parameters  *map[string]string `json:"parameters" mapstructure:"parameters"`
	Personality *string            `json:"personality,omitempty" mapstructure:"personality"`
	Resource    *string            `json:"resource,omitempty" mapstructure:"resource"`
}

/* PATCH /v1/service_parameter/{parameter_id} expects: name:value */
type ServiceParameterPatchOpts struct {
	ParamName   *string `json:"name,omitempty" mapstructure:"name"`
	ParamValue  *string `json:"value,omitempty" mapstructure:"value"`
	Personality *string `json:"personality,omitempty" mapstructure:"personality"`
	Resource    *string `json:"resource,omitempty" mapstructure:"resource"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToServiceParameterListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through the API.
// Filtering is achieved by passing in struct field values that map to the serviceparameter
// attributes you want to see returned.
// SortKey allows you to sort by a particular serviceParameter attribute.
// SortDir sets the direction, and is either `asc' or `desc'.
// Marker and Limit are used for pagination.
type ListOpts struct {
	Service string `q:"service"`
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToServiceParameterListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToServiceParameterListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// serviceParameters. It accepts a ListOpts struct, which allows you to filter and
// sort the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToServiceParameterListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return ServiceParameterPage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific serviceParameter based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) GetResult {
	var res GetResult
	_, res.Err = c.Get(getURL(c, id), &res.Body, nil)
	return res
}

// Create accepts a CreateOpts struct and creates a new ServiceParameter using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts ServiceParameterOpts) (r CreateResult) {
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

// Apply applies the service parameters through a POST. Optionally a service name can be passed'
func Apply(c *gophercloud.ServiceClient, opts ServiceApplyOpts) (r ApplyResult) {
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

// Update accepts a PatchOpts struct and updates an existing serviceParameter using
// the values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts ServiceParameterPatchOpts) (r UpdateResult) {
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

// Delete accepts a unique ID and deletes the related resource.
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, id), nil)
	return r
}

// ListServiceParameters is a convenience function to list and extract the entire
// list of service parameters
func ListServiceParameters(c *gophercloud.ServiceClient) ([]ServiceParameter, error) {
	pages, err := List(c, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractServiceParameters(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
