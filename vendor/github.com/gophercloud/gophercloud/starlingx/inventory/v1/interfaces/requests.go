/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package interfaces

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
	inventoryv1 "github.com/gophercloud/gophercloud/starlingx/inventory/v1"
)

// Defines common system defined interface names
const (
	LoopbackInterfaceName = "lo"
)

// Defines the valid interface addressing modes
const (
	AddressModeStatic   = "static"
	AddressModePool     = "pool"
	AddressModeDisabled = "disabled"
)

// Defines the valid interface classes
const (
	IFClassPlatform       = "platform"
	IFClassData           = "data"
	IFClassNone           = "none"
	IFClassPCISRIOV       = "pci-sriov"
	IFClassPCIPassthrough = "pci-passthrough"
)

// Defines the interface type values used by the system API
const (
	IFTypeAE       = "ae"
	IFTypeVLAN     = "vlan"
	IFTypeEthernet = "ethernet"
	IFTypeVirtual  = "virtual"
	IFTypeVF       = "vf"
)

// Defines the system defined value for interface MTU settings
const (
	DefaultMTU = 1500
)

// Defines the valid PTP settings
const (
	PTPRoleMaster = "master"
	PTPRoleSlave  = "slave"
	PTPRoleNone   = "none"
)

// InterfaceOpts provides configured interface options
type InterfaceOpts struct {
	HostUUID         *string   `json:"ihost_uuid,omitempty" mapstructure:"ihost_uuid"`
	Type             *string   `json:"iftype,omitempty" mapstructure:"iftype"`
	Name             *string   `json:"ifname,omitempty" mapstructure:"ifname"`
	Class            *string   `json:"ifclass,omitempty" mapstructure:"ifclass"`
	MTU              *int      `json:"imtu,omitempty" mapstructure:"imtu"`
	VID              *int      `json:"vlan_id,omitempty" mapstructure:"vlan_id"`
	IPv4Mode         *string   `json:"ipv4_mode,omitempty" mapstructure:"ipv4_mode"`
	IPv4Pool         *string   `json:"ipv4_pool,omitempty" mapstructure:"ipv4_pool"`
	IPv6Mode         *string   `json:"ipv6_mode,omitempty" mapstructure:"ipv6_mode"`
	IPv6Pool         *string   `json:"ipv6_pool,omitempty" mapstructure:"ipv6_pool"`
	Networks         *[]string `json:"networks,omitempty" mapstructure:"networks"`
	NetworksToAdd    *[]string `json:"networks_to_add,omitempty" mapstructure:"networks_to_add"`
	NetworksToDelete *[]string `json:"interface_networks_to_remove,omitempty" mapstructure:"interface_networks_to_remove"`
	DataNetworks     *[]string `json:"datanetworks,omitempty" mapstructure:"datanetworks"`
	AEMode           *string   `json:"aemode,omitempty" mapstructure:"aemode"`
	AETransmitHash   *string   `json:"txhashpolicy,omitempty" mapstructure:"txhashpolicy"`
	VFCount          *int      `json:"sriov_numvfs,omitempty" mapstructure:"sriov_numvfs"`
	VFDriver         *string   `json:"sriov_vf_driver,omitempty" mapstructure:"sriov_vf_driver"`
	Uses             *[]string `json:"uses,omitempty" mapstructure:"uses"`
	UsesModify       *[]string `json:"usesmodify,omitempty" mapstructure:"usesmodify"`
	PTPRole          *string   `json:"ptp_role,omitempty" mapstructure:"ptp_role"`
	MaxTxRate        *int      `json:"max_tx_rate,omitempty" mapstructure:"max_tx_rate"`
}

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToInterfaceListQuery() (string, error)
}

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the interface attributes you want to see returned. SortKey allows you to sort
// by a particular interface attribute. SortDir sets the direction, and is
// either `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	Name    string `q:"ifname"`
	ID      string `q:"id"`
	Marker  string `q:"marker"`
	Limit   int    `q:"limit"`
	SortKey string `q:"sort_key"`
	SortDir string `q:"sort_dir"`
}

// ToInterfaceListQuery formats a ListOpts into a query string.
func (opts ListOpts) ToInterfaceListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}

// List returns a Pager which allows you to iterate over a collection of
// interfaces. It accepts a ListOpts struct, which allows you to filter and sort
// the returned collection for greater efficiency.
func List(c *gophercloud.ServiceClient, hostid string, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c, hostid)
	if opts != nil {
		query, err := opts.ToInterfaceListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return InterfacePage{pagination.SinglePageBase(r)}
	})
}

// Get retrieves a specific interface based on its unique ID.
func Get(c *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = c.Get(getURL(c, id), &r.Body, nil)
	return r
}

// Create accepts a CreateOpts struct and creates a new interface using the
// values provided.
func Create(c *gophercloud.ServiceClient, opts InterfaceOpts) (r CreateResult) {
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

// Update accepts a PatchOpts struct and updates an existing interface using the
// values provided. For more information, see the Create function.
func Update(c *gophercloud.ServiceClient, id string, opts InterfaceOpts) (r UpdateResult) {
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

// Delete accepts a unique ID and deletes the interface associated with it.
func Delete(c *gophercloud.ServiceClient, id string) (r DeleteResult) {
	_, r.Err = c.Delete(deleteURL(c, id), nil)
	return r
}

// ListInterfaces is a convenience function to list and extract the entire
// list of interfaces on a specific host.
func ListInterfaces(c *gophercloud.ServiceClient, hostid string) ([]Interface, error) {
	pages, err := List(c, hostid, nil).AllPages()
	if err != nil {
		return nil, err
	}

	empty, err := pages.IsEmpty()
	if empty || err != nil {
		return nil, err
	}

	objs, err := ExtractInterfaces(pages)
	if err != nil {
		return nil, err
	}

	return objs, err
}
