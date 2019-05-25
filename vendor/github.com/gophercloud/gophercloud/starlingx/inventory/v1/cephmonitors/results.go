/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package cephmonitors

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Extract interprets any commonResult as an Image.
func (r commonResult) Extract() (*CephMonitor, error) {
	var s CephMonitor
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

// UpdateResult represents the result of an update operation.
type UpdateResult struct {
	commonResult
}

// DeleteResult represents the result of a delete operation.
type DeleteResult struct {
	gophercloud.ErrResult
}

// CephMonitor defines the data associated to a single CephMonitor instance.
type CephMonitor struct {
	// ID defines the system assigned unique UUID value
	ID string `json:"uuid"`

	// HostUUID defines the unique UUID value associated to the host.
	HostUUID string `json:"ihost_uuid"`

	// Hostname defines the name associated to the host.
	Hostname string `json:"hostname"`

	// State defines the current state of the monitor.
	State string `json:"state"`

	// Task defines the name of the current in progress task (if any).
	Task *string `json:"task,omitempty"`

	// Size defines the space allocated to the monitor - in gigabytes.
	Size int `json:"ceph_mon_gib"`

	// DevicePath defines the name of the device on which the monitor runs.
	DevicePath *string `json:"device_path,omitempty"`

	// CreatedAt defines the timestamp at which the resource was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt defines the timestamp at which the resource was last updated.
	UpdatedAt *string `json:"updated_at,omitempty"`
}

// CephMonitorPage is the page returned by a pager when traversing over a
// collection of CephMonitors.
type CephMonitorPage struct {
	pagination.SinglePageBase
}

// IsEmpty checks whether a CephMonitorPage struct is empty.
func (r CephMonitorPage) IsEmpty() (bool, error) {
	is, err := ExtractCephMonitors(r)
	return len(is) == 0, err
}

// ExtractCephMonitors accepts a Page struct, specifically a
// CephMonitorPage struct, and extracts the elements into a slice of
// CephMonitor structs. In other words, a generic collection is mapped into
// a relevant slice.
func ExtractCephMonitors(r pagination.Page) ([]CephMonitor, error) {
	var s struct {
		CephMonitor []CephMonitor `json:"ceph_mon"`
	}

	err := (r.(CephMonitorPage)).ExtractInto(&s)

	return s.CephMonitor, err
}
