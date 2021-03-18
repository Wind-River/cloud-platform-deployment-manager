/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2021 Wind River Systems, Inc. */

package system

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*System, error) {
	var s System
	err := r.ExtractInto(&s)
	return &s, err
}

type commonResult struct {
	gophercloud.Result
}

// GetResult represents the result of a get operation.
type GetResult struct {
	commonResult
}

// UpdateResult represents the result of an update operation.
type UpdateResult struct {
	commonResult
}

// Capabilities defines the system capabilities for a single system
// instance.
type Capabilities struct {
	// HTTPSEnabled is whether HTTPS is configured for the system.
	HTTPSEnabled bool `json:"https_enabled"`

	// SDNEnabled is whether SDN is configured for the system.
	SDNEnabled bool `json:"sdn_enabled"`

	// VSwitchType defines which vswitch implementation is configured.
	VSwitchType string `json:"vswitch_type"`

	// SharedServices defines the list of region mode shared services.
	SharedServices *string `json:"shared_services,omitempty"`

	// KubernetesEnabled defines whether Kubernetes is enabled or not.
	KubernetesEnabled bool `json:"kubernetes_enabled"`

	// RegionConfig defines whether region mode is enabled.
	RegionConfig bool `json:"region_config"`

	// BMRegion defines the current board management region.
	BMRegion string `json:"bm_region"`
}

// System defines the data associated to a single system instance.
type System struct {
	// Is system generated unique UUID for the system
	ID string `json:"uuid"`

	// SystemName is the human readable and assignable name of the system.
	Name string `json:"name"`

	// Description is the user assigned descriptive text of the system.
	Description string `json:"description"`

	// Location is a description of the physical location of the system.
	Location string `json:"location"`

	// Latitude is the latitude geolocation coordinate of the system's physical
	Latitude string `json:"latitude"`

	// Longitude is the longitude geolocation coordinate of the system's physical
	Longitude string `json:"longitude"`

	// Contact is the name and address information of the system contact person.
	Contact string `json:"contact"`

	// SystemMode is whether the system is running in simplex or duplex.
	SystemMode string `json:"system_mode"`

	// SystemType is whether the system is configured as a single node or multi
	// node system.
	SystemType string `json:"system_type"`

	// SoftwareVersion is the currently running system software version.
	SoftwareVersion string `json:"software_version"`

	// RegionName defines the region name of this system instance.
	RegionName string `json:"region_name"`

	// Capabilities defines the set of system capabilities and their current
	// states.
	Capabilities Capabilities `json:"capabilities"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// SystemPage is the page returned by a pager when traversing over a
// collection of systems.
type SystemPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a SystemPage struct is empty.
func (r SystemPage) IsEmpty() (bool, error) {
	is, err := ExtractSystems(r)
	return len(is) == 0, err
}

// ExtractSystems accepts a Page struct, specifically a SystemPage struct,
// and extracts the elements into a slice of System structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractSystems(r pagination.Page) ([]System, error) {
	var s struct {
		System []System `json:"isystems"`
	}

	err := (r.(SystemPage)).ExtractInto(&s)

	return s.System, err
}
