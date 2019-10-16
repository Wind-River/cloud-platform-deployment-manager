/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package system

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/controllerFilesystems"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1beta1"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/manager"
	"time"
)

// DefaultAvailableControllerNodeMonitorInterval represents the default interval
// between polling attempts to check whether all controller nodes have reached
// the unlocked/enabled/available state.  The DRDB file systems have to be
// fully synchronized before the degraded state is cleared therefore there is
// no need to poll this frequently.
const DefaultAvailableControllerNodeMonitorInterval = 60 * time.Second

// availableControllerNodeMonitor waits all controller nodes to reach the
// unlocked/available state.  Once the required state has been reached a
// reconcilable event is generated to kick the reconciler.
type availableControllerNodeMonitor struct {
	manager.CommonMonitorBody
	required int
}

// NewAvailableControllerNodeMonitor defines a convenience function to
// instantiate a new available controller monitor with all required attributes.
func NewAvailableControllerNodeMonitor(instance *v1beta1.System, required int) *manager.Monitor {
	logger := log.WithName("available-controllers-monitor")
	return &manager.Monitor{
		MonitorBody: &availableControllerNodeMonitor{
			required: required,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultAvailableControllerNodeMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *availableControllerNodeMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	objects, err := hosts.ListHosts(client)
	if err != nil {
		m.SetState("failed to query host list: %s", err.Error())
		return false, err
	}

	if controllerNodesAvailable(objects, m.required) {
		m.SetState("required number of controllers are available")
		return true, nil
	}

	m.SetState("waiting for %d controller(s) to be available", m.required)

	return false, nil
}

// fileSystemResizeMonitor defines a monitor that can check the state of the
// controller file systems and determine when it is safe to proceed with file
// system resizing.
type fileSystemResizeMonitor struct {
	manager.CommonMonitorBody
}

// DefaultPartitionMonitorInterval represents the default interval between
// polling attempts to check file system states.
const DefaultFileSystemResizeMonitorInterval = 15 * time.Second

// NewPartitionStateMonitor defines a convenience function to instantiate
// a new partition monitor with all required attributes.
func NewFileSystemResizeMonitor(instance *v1beta1.System) *manager.Monitor {
	logger := log.WithName("fs-resize-monitor")
	return &manager.Monitor{
		MonitorBody: &fileSystemResizeMonitor{},
		Logger:      logger,
		Object:      instance,
		Interval:    DefaultFileSystemResizeMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *fileSystemResizeMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	objects, err := controllerFilesystems.ListFileSystems(client)
	if err != nil {
		m.SetState("failed to get disk partitions: %s", err.Error())
		return false, err
	}

	for _, fs := range objects {
		if fs.State == controllerFilesystems.ResizeInProgress {
			m.SetState("waiting for filesystem %q to finish resizing", fs.Name)
			return false, nil
		}
	}

	m.SetState("all filesystems are ready for resizing")

	return true, nil
}
