/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package hosts

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

const (
	// BMTypeDisabled is a special placeholder value to request that the BM
	// configuration be removed from a host.
	BMTypeDisabled = "none"
)

// Defines system defined host names.  These hosts are named by the system and
// they cannot be renamed by the operator.
const (
	Controller0 = "controller-0"
)

// Defines system defined personality specific capabilities for hosts.
const (
	ActiveController = "Controller-Active"
)

// Administrative States
const (
	AdminLocked   = "locked"
	AdminUnlocked = "unlocked"
)

// Operational States
const (
	OperEnabled  = "enabled"
	OperDisabled = "disabled"
)

// Availability States
const (
	AvailAvailable = "available"
	AvailOnline    = "online"
	AvailOffline   = "offline"
	AvailPowerOff  = "power-off"
)

// Mtce task values
const (
	TaskLocking     = "Locking"
	TaskPoweringOn  = "Powering-on"
	TaskPoweringOff = "Powering-off"
)

// Inventory collection state
const (
	InventoryCollected = "inventoried"
)

const (
	PersonalityController = "controller"
	PersonalityWorker     = "worker"
	PersonalityStorage    = "storage"
)

const (
	SubFunctionController = "controller"
	SubFunctionWorker     = "worker"
	SubFunctionStorage    = "storage"
)

const (
	StorFunctionMonitor = "monitor"
)

const OSDMinimumMonitorCount = 2

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*Host, error) {
	var s Host
	err := r.ExtractInto(&s)
	return &s, err
}

type commonResult struct {
	gophercloud.Result
}

// CreateResult represents the result of a create operation.
type CreateResult struct {
	commonResult
}

// GetResult represents the result of a get operation.
type GetResult struct {
	commonResult
}

// UpdateResult represents the result of an update operation.
type UpdateResult struct {
	commonResult
}

// DeleteResult represents the result of a delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// Location defines the format used to represent a host location in the system
// API.
type Location struct {
	Name *string `json:"locn,omitempty"`
}

type Capabilities struct {
	StorFunction *string `json:"stor_function,omitempty"`
	Personality  *string `json:"Personality,omitempty"`
}

// Host represents, well, a host.
type Host struct {
	// ID is the system assigned unique UUID value
	ID string `json:"uuid"`

	// Human-readable name for the host. Must be unique.
	Hostname string `json:"hostname"`

	// Personality is the role assigned to the host.
	Personality string `json:"personality"`

	// Subfunctions is the sub-roles assigned to the host.
	SubFunctions string `json:"subfunctions"`

	// Capabilities is the set of enabled capabilities on the host.
	Capabilities Capabilities `json:"capabilities"`

	// Location is the physical location of the host in the data centre.
	Location Location `json:"location"`

	// InstallOutput is the configured installation output type.
	InstallOutput string `json:"install_output"`

	// Console is the configured output device and speed.
	Console string `json:"console"`

	// BootMAC is the host boot interface MAC address.
	BootMAC string `json:"mgmt_mac"`

	// BootIP is the host boot interface IP address.
	BootIP string `json:"mgmt_ip"`

	// RootDevice is the configured root file system device.
	RootDevice string `json:"rootfs_device"`

	// BootDevice is the configured boot file system device.
	BootDevice string `json:"boot_device"`

	// BMType is the configured board management controller type.
	BMType *string `json:"bm_type,omitempty"`

	// BMAddress is the configured board management IP address.
	BMAddress *string `json:"bm_ip,omitempty"`

	// BMUsername is the configured board management username.
	BMUsername *string `json:"bm_username,omitempty"`

	// SerialNumber is the DMI serial number attribute.
	SerialNumber *string `json:"serial_number,omitempty"`

	// AssetTag is the DMI asset tag attribute.
	AssetTag *string `json:"asset_tag,omitempty"`

	// ConfigurationStatus is the current configuration status.
	ConfigurationStatus string `json:"config_status"`

	// ConfigurationApplied is the uuid value of the currently applied
	// configuration.
	ConfigurationApplied string `json:"config_applied"`

	// ConfigurationTarget is the uuid value of the next applied configuration.
	ConfigurationTarget string `json:"config_target"`

	// Task reports whether there is a mtce task in progress.
	Task *string `json:"task,omitempty"`

	// AdministrativeState is the current administrative state of the host.
	AdministrativeState string `json:"administrative"`

	// OperationalStatus is the current operational state of the host.
	OperationalStatus string `json:"operational"`

	// AvailabilityStatus is the current availability state of the host.
	AvailabilityStatus string `json:"availability"`

	// InventoryState is the current state of the inventory collection process
	// of the host.
	InventoryState *string `json:"inv_state,omitempty"`

	// ClockSynchronization is the chosen clock source for the host.
	ClockSynchronization *string `json:"clock_synchronization,omitempty"`

	//MaxCPUFrequency is the max limit of the CPU frequency set on the host.
	MaxCPUFrequency string `json:"max_cpu_frequency,omitempty"`
}

// HostPage is the page returned by a pager when traversing over a
// collection of hosts.
type HostPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a HostPage struct is empty.
func (r HostPage) IsEmpty() (bool, error) {
	is, err := ExtractHosts(r)
	return len(is) == 0, err
}

// ExtractHosts accepts a Page struct, specifically a HostPage struct,
// and extracts the elements into a slice of Host structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractHosts(r pagination.Page) ([]Host, error) {
	var s struct {
		Host []Host `json:"ihosts"`
	}

	err := (r.(HostPage)).ExtractInto(&s)

	return s.Host, err
}
