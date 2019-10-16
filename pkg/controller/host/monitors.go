/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/clusters"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/partitions"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/storagetiers"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1beta1"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/manager"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

// partitionStateMonitor defines a monitor that can check the state of the
// disk partitions on a host and signal the host reconciler once all of the
// partitions have become "available".
type partitionStateMonitor struct {
	manager.CommonMonitorBody
	id string
}

// DefaultPartitionMonitorInterval represents the default interval between
// polling attempts to check disk partition states on a host.  Partitions do
// not usually take much time to be ready so this interval is kept short.
const DefaultPartitionMonitorInterval = 15 * time.Second

// NewPartitionStateMonitor defines a convenience function to instantiate
// a new partition monitor with all required attributes.
func NewPartitionStateMonitor(instance *v1beta1.Host, id string) *manager.Monitor {
	logger := log.WithName("partition-monitor")
	return &manager.Monitor{
		MonitorBody: &partitionStateMonitor{
			id: id,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultPartitionMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *partitionStateMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	objects, err := partitions.ListPartitions(client, m.id)
	if err != nil {
		m.SetState("failed to get disk partitions: %s", err.Error())
		return false, err
	}

	for _, p := range objects {
		switch p.Status {
		case partitions.StatusDeleting, partitions.StatusModifying, partitions.StatusCreating:
			m.SetState("waiting for partition %q to be available", p.ID)
			return false, nil
		}
	}

	m.SetState("all partitions are available")

	return true, nil
}

// DefaultPartitionMonitorInterval represents the default interval between
// polling attempts to check whether a cluster exists or not.  A user may
// need to intervene to create a cluster so set this to a value long enough
// to avoid unnecessary polling of the API.
const DefaultClusterPresenceMonitorInterval = 2 * time.Minute

// clusterPresenceMonitor waits for a given cluster to be provisioned in the
// system being reconciled.  Once it finds the specified cluster a reconcilable
// event is generated to kick the reconciler.
type clusterPresenceMonitor struct {
	manager.CommonMonitorBody
	clusterName string
}

// NewClusterPresenceMonitor defines a convenience function to instantiate
// a new cluster presence monitor with all required attributes.
func NewClusterPresenceMonitor(instance *v1beta1.Host, cluster string) *manager.Monitor {
	logger := log.WithName("cluster-presence-monitor")
	return &manager.Monitor{
		MonitorBody: &clusterPresenceMonitor{
			clusterName: cluster,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultClusterPresenceMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *clusterPresenceMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	objects, err := clusters.ListClusters(client)
	if err != nil {
		m.SetState("failed to get cluster list: %s", err.Error())
		return false, err
	}

	for _, c := range objects {
		if c.Name == m.clusterName {
			m.SetState("cluster %q found", m.clusterName)
			return true, nil
		}
	}

	m.SetState("waiting for cluster %q", m.clusterName)

	return false, nil
}

// DefaultClusterDeploymentModelMonitorInterval represents the default interval
// between polling attempts to check whether the deployment model has been set
// on a cluster.  The deployment model requires multiple hosts to be enabled
// therefore there is no point in polling frequently as other hosts need to
// boot, install, and be configured.
const DefaultClusterDeploymentModelMonitorInterval = 30 * time.Second

// clusterDeploymentModelMonitor waits for a given cluster to have a valid
// deployment model set.  Once a deployment model is set on the cluster a
// reconcilable event is generated to kick the reconciler.
type clusterDeploymentModelMonitor struct {
	manager.CommonMonitorBody
	clusterId string
}

// NewClusterDeploymentModelMonitor defines a convenience function to
// instantiate a new cluster deployment model monitor.
func NewClusterDeploymentModelMonitor(instance *v1beta1.Host, clusterId string) *manager.Monitor {
	logger := log.WithName("deployment-model-monitor")
	return &manager.Monitor{
		MonitorBody: &clusterDeploymentModelMonitor{
			clusterId: clusterId,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultClusterDeploymentModelMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *clusterDeploymentModelMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	cluster, err := clusters.Get(client, m.clusterId).Extract()
	if err != nil {
		m.SetState("failed to get cluster list: %s", err.Error())
		return false, err
	}

	if cluster.DeploymentModel != clusters.DeploymentModelUndefined {
		m.SetState("deployment model is set on cluster %q", m.clusterId)
		return true, nil
	}

	m.SetState("waiting for deployment model to be set on cluster %q", m.clusterId)

	return false, nil
}

// DefaultMonitorCountMonitorInterval represents the default interval
// between polling attempts to check whether sufficient storage monitors are
// enabled prior to enabling other hosts or creating OSDs.
const DefaultMonitorCountMonitorInterval = 30 * time.Second

// storageMonitorCountMonitor waits for a given number of storage monitors to be
// enabled.  Once the required number of monitors is enabled a reconcilable
// event is generated to kick the reconciler.
type storageMonitorCountMonitor struct {
	manager.CommonMonitorBody
	required int
}

// NewMonitorCountMonitor defines a convenience function to
// instantiate a new storage monitor count monitor with all required attributes.
func NewStorageMonitorCountMonitor(instance *v1beta1.Host, required int) *manager.Monitor {
	logger := log.WithName("storage-mon-monitor")
	return &manager.Monitor{
		MonitorBody: &storageMonitorCountMonitor{
			required: required,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultMonitorCountMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *storageMonitorCountMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	objects, err := hosts.ListHosts(client)
	if err != nil {
		m.SetState("failed to query host list: %q", err.Error())
		return false, err
	}

	if monitorsEnabled(objects, m.required) {
		m.SetState("required number of monitors now enabled: %d", m.required)
		return true, nil
	}

	m.SetState("waiting for %d monitor(s) to be enabled", m.required)

	return false, nil
}

// DefaultStorageTierMonitorInterval represents the default interval between
// polling attempts to check whether the specified storage tier has been
// created.
const DefaultStorageTierMonitorInterval = 30 * time.Second

// storageMonitorCountMonitor waits for a specified storage tier to be created.
// Once the required storage tier has been found a reconcilable event is
// generated to kick the reconciler.
type storageTierMonitor struct {
	manager.CommonMonitorBody
	clusterID string
	tierName  string
}

// NewStorageTierMonitor defines a convenience function to instantiate a new
// storage monitor count monitor with all required attributes.
func NewStorageTierMonitor(instance *v1beta1.Host, clusterID, tierName string) *manager.Monitor {
	logger := log.WithName("storage-tier-monitor")
	return &manager.Monitor{
		MonitorBody: &storageTierMonitor{
			clusterID: clusterID,
			tierName:  tierName,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultStorageTierMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *storageTierMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	tiers, err := storagetiers.ListTiers(client, m.clusterID)
	if err != nil {
		m.SetState("failed to get storage tier list: %s", err.Error())
		return false, err
	}

	for _, t := range tiers {
		if t.Name == storagetiers.StorageTierName {
			m.SetState("storage tier for cluster %q has been found", m.clusterID)
			return true, nil
		}
	}

	m.SetState("waiting for storage tier for cluster %q", m.clusterID)

	return false, nil
}

// DefaultStateMonitorInterval represents the default interval between polling
// attempts to check whether a host has reached the desired state.
const DefaultStateMonitorInterval = 30 * time.Second

// stateMonitor waits for a host to reach a desired state.  Once the host has
// reached the desired state a reconcilable event is generated to kick the
// reconciler.
type stateMonitor struct {
	manager.CommonMonitorBody
	lastLoggedState     string
	hostID              string
	administrativeState *string
	availabilityStatus  *string
	operationalStatus   *string
}

func (m *stateMonitor) stableOnly() bool {
	return m.administrativeState == nil && m.availabilityStatus == nil && m.operationalStatus == nil
}

func (m *stateMonitor) desiredState() string {
	if m.stableOnly() {
		return "idle"
	}

	admin := "*"
	if m.administrativeState != nil {
		admin = *m.administrativeState
	}

	oper := "*"
	if m.operationalStatus != nil {
		oper = *m.operationalStatus
	}

	avail := "*"
	if m.availabilityStatus != nil {
		avail = *m.availabilityStatus
	}

	return fmt.Sprintf("%s/%s/%s/-", admin, oper, avail)
}

// NewMonitorCountMonitor defines a convenience function to instantiate
// a new host state monitor with all required attributes.
func NewStateMonitor(instance *v1beta1.Host, id string, admin, oper, avail *string) *manager.Monitor {
	logger := log.WithName("state-monitor")
	return &manager.Monitor{
		MonitorBody: &stateMonitor{
			hostID:              id,
			administrativeState: admin,
			availabilityStatus:  avail,
			operationalStatus:   oper,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultStateMonitorInterval,
	}
}

// NewUnlockedEnabledHostMonitor is a convenience wrapper around
// NewStateMonitor to wait for a host to reach the unlocked/enabled state.
func NewUnlockedEnabledHostMonitor(instance *v1beta1.Host, id string) *manager.Monitor {
	admin := hosts.AdminUnlocked
	oper := hosts.OperEnabled
	return NewStateMonitor(instance, id, &admin, &oper, nil)
}

// NewUnlockedAvailableHostMonitor is a convenience wrapper around
// NewStateMonitor to wait for a host to reach the unlocked/enabled/available
// state.
func NewUnlockedAvailableHostMonitor(instance *v1beta1.Host, id string) *manager.Monitor {
	admin := hosts.AdminUnlocked
	oper := hosts.OperEnabled
	avail := hosts.AvailAvailable
	return NewStateMonitor(instance, id, &admin, &oper, &avail)
}

// NewLockedDisabledHostMonitor is a convenience wrapper around
// NewStateMonitor to wait for a host to reach the locked/disabled state.
func NewLockedDisabledHostMonitor(instance *v1beta1.Host, id string) *manager.Monitor {
	admin := hosts.AdminLocked
	oper := hosts.OperDisabled
	return NewStateMonitor(instance, id, &admin, &oper, nil)
}

// NewIdleHostMonitor is a convenience wrapper around NewStateMonitor to wait
// for a host to reach the idle state regardless of what its administrative
// state, operational status, or availability status values are.
func NewIdleHostMonitor(instance *v1beta1.Host, id string) *manager.Monitor {
	return NewStateMonitor(instance, id, nil, nil, nil)
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *stateMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	host, err := hosts.Get(client, m.hostID).Extract()
	if err != nil {
		m.SetState("failed to get host %q: %s", m.hostID, err.Error())
		return false, err
	}

	task := "-"
	if host.Task != nil && *host.Task != "" {
		task = *host.Task
	}

	state := fmt.Sprintf("%s/%s/%s/%s", host.AdministrativeState, host.OperationalStatus, host.AvailabilityStatus, task)

	if !host.Idle() {
		m.SetState("waiting for host to reach stable state: %s", state)
		goto done
	}

	if m.administrativeState != nil && (host.AdministrativeState != *m.administrativeState) {
		goto done
	}

	if m.operationalStatus != nil && (host.OperationalStatus != *m.operationalStatus) {
		goto done
	}

	if m.availabilityStatus != nil && (host.AvailabilityStatus != *m.availabilityStatus) {
		goto done
	}

	stop = true

done:
	if state != m.lastLoggedState {
		m.lastLoggedState = state
	}

	if stop {
		m.SetState("desired state has been reached: %s", m.desiredState())
	} else if m.stableOnly() {
		m.SetState("waiting for stable state; current: %s", state)
	} else {
		m.SetState("waiting for state: %s; current: %s", m.desiredState(), state)
	}

	return stop, nil
}

// DefaultInventoryCollectedMonitorInterval represents the default interval
// between polling attempts to check whether a host has reached the desired
// state before beginning to collect default values.
const DefaultInventoryCollectedMonitorInterval = 30 * time.Second

// inventoryCollectedMonitor waits for a host to reach an idle state and for
// system inventory to have been collected on that host.  This is determined
// based on whether there are any disk reports against the host since disks
// are the last piece of information collected by the agent.  Once the required
// state has been reached a reconcilable event is generated to kick the
// reconciler.
type inventoryCollectedMonitor struct {
	manager.CommonMonitorBody
	id string
}

// NewInventoryCollectedMonitor defines a convenience function to
// instantiate a new inventory collected monitor with all required attributes.
func NewInventoryCollectedMonitor(instance *v1beta1.Host, id string) *manager.Monitor {
	logger := log.WithName("inventory-monitor")
	return &manager.Monitor{
		MonitorBody: &inventoryCollectedMonitor{
			id: id,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultInventoryCollectedMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *inventoryCollectedMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	host, err := hosts.Get(client, m.id).Extract()
	if err != nil {
		m.SetState("failed to get host %q: %s", m.id, err.Error())
		return false, err
	}

	if !host.Idle() || host.AvailabilityStatus == hosts.AvailOffline {
		m.SetState("waiting for stable state before collecting defaults")
		return false, nil
	}

	if host.IsInventoryCollected() {
		m.SetState("inventory has completed host %q", m.id)
		return true, nil
	}

	m.SetState("waiting for inventory to complete before collecting defaults")

	return false, nil
}

// DefaultEnabledControllerNodeMonitorInterval represents the default interval
// between polling attempts to check whether all controller nodes have reached
// the unlocked/enabled state.
const DefaultEnabledControllerNodeMonitorInterval = 30 * time.Second

// enabledControllerNodeMonitor waits all controller nodes to reach the
// unlocked/enabled state.  Once the required state has been reached a
// reconcilable event is generated to kick the reconciler.
type enabledControllerNodeMonitor struct {
	manager.CommonMonitorBody
	required int
}

// NewEnabledControllerNodeMonitor defines a convenience function to
// instantiate a new enabled controller monitor with all required attributes.
func NewEnabledControllerNodeMonitor(instance *v1beta1.Host, required int) *manager.Monitor {
	logger := log.WithName("controller-node-monitor")
	return &manager.Monitor{
		MonitorBody: &enabledControllerNodeMonitor{
			required: required,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultEnabledControllerNodeMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *enabledControllerNodeMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	objects, err := hosts.ListHosts(client)
	if err != nil {
		m.SetState("failed to query host list: %s", err.Error())
		return false, err
	}

	if allControllerNodesEnabled(objects, m.required) {
		m.SetState("required number of controllers are enabled: %d", m.required)
		return true, nil
	}

	m.SetState("waiting for %d controller(s) to be enabled", m.required)

	return false, nil
}

// DefaultProvisioningAllowedMonitorInterval represents the default interval
// between polling attempts to check whether host provisioning is allowed based
// on the state of the primary controller.
const DefaultProvisioningAllowedMonitorInterval = 30 * time.Second

// provisioningAllowedMonitor waits for the first controller to be enabled.
// Once the required state has been reached a reconcilable event is generated to
// kick the reconciler.
type provisioningAllowedMonitor struct {
	manager.CommonMonitorBody
}

// NewProvisioningAllowedMonitor defines a convenience function to
// instantiate a new provisioning allowed monitor with all required attributes.
func NewProvisioningAllowedMonitor(instance *v1beta1.Host) *manager.Monitor {
	logger := log.WithName("provisioning-monitor")
	return &manager.Monitor{
		MonitorBody: &provisioningAllowedMonitor{},
		Logger:      logger,
		Object:      instance,
		Interval:    DefaultProvisioningAllowedMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *provisioningAllowedMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	objects, err := hosts.ListHosts(client)
	if err != nil {
		m.SetState("failed to query host list: %s", err.Error())
		return false, err
	}

	if provisioningAllowed(objects) {
		m.SetState("host provisioning is now allowed")
		return true, nil
	}

	m.SetState("waiting for host provisioning to be allowed")

	return false, nil
}

// DefaultDynamicHostMonitorInterval represents the default interval
// between polling attempts to check whether a host has appeared in
// system inventory.
const DefaultDynamicHostMonitorInterval = 30 * time.Second

// dynamicHostMonitor waits for a host to appear in system inventory. Once
// the required resource exists a reconcilable event is generated to kick the
// reconciler.
type dynamicHostMonitor struct {
	manager.CommonMonitorBody
	hostname string
	match    *v1beta1.MatchInfo
	bootMAC  *string
}

// NewDynamicHostMonitor defines a convenience function to instantiate a
// new kubernetes resource monitor with all required attributes.
func NewDynamicHostMonitor(instance *v1beta1.Host, hostname string, match *v1beta1.MatchInfo, bootMAC *string) *manager.Monitor {
	logger := log.WithName("dynamic-host-monitor")
	return &manager.Monitor{
		MonitorBody: &dynamicHostMonitor{
			hostname: hostname,
			match:    match,
			bootMAC:  bootMAC,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultDynamicHostMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *dynamicHostMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	objects, err := hosts.ListHosts(client)
	if err != nil {
		m.SetState("failed to query host list: %s", err.Error())
		return false, err
	}

	host := findExistingHost(objects, m.hostname, m.match, m.bootMAC)
	if host != nil {
		m.SetState("host inventory record has been found for %q", m.hostname)
		return true, nil
	}

	m.SetState("waiting for %q to appear in system inventory", m.hostname)

	return false, nil
}

// DefaultKubernetesResourceMonitorInterval represents the default interval
// between polling attempts to check whether a kubernetes resource is present.
const DefaultKubernetesResourceMonitorInterval = 30 * time.Second

// kubernetesResourceMonitor waits for a kubernetes resource to be present.
// Once the required resource exists a reconcilable event is generated to kick
// the reconciler.
type kubernetesResourceMonitor struct {
	manager.CommonMonitorBody
	manager manager.TitaniumManager
	object  runtime.Object
	name    types.NamespacedName
}

// NewKubernetesResourceMonitor defines a convenience function to instantiate a
// new kubernetes resource monitor with all required attributes.
func NewKubernetesResourceMonitor(instance *v1beta1.Host, target runtime.Object, name types.NamespacedName) *manager.Monitor {
	logger := log.WithName("kubernetes-monitor")
	return &manager.Monitor{
		MonitorBody: &kubernetesResourceMonitor{
			CommonMonitorBody: manager.CommonMonitorBody{},
			object:            target,
			name:              name,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultKubernetesResourceMonitorInterval,
	}
}

// NewKubernetesSecretMonitor is a convenience wrapper around
// NewKubernetesResourceMonitor that instantiates a monitor explicitly for
// Kubernetes Secret resources.
func NewKubernetesSecretMonitor(instance *v1beta1.Host, name types.NamespacedName) *manager.Monitor {
	secret := v1.Secret{}
	return NewKubernetesResourceMonitor(instance, &secret, name)
}

// SetManager implements the MonitorManager interface which allows the parent
// monitor to provide access to the manager reference.
func (m *kubernetesResourceMonitor) SetManager(manager manager.TitaniumManager) {
	m.manager = manager
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *kubernetesResourceMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	result := unstructured.Unstructured{}
	err = m.manager.GetKubernetesClient().Get(context.Background(), m.name, &result)
	if err == nil {
		m.SetState("kubernetes %q resource %q is now available", m.object.GetObjectKind(), m.name)
		return true, nil
	} else if errors.IsNotFound(err) {
		m.SetState("waiting for kubernetes %q resource %q", m.object.GetObjectKind(), m.name)
	} else {
		m.SetState("failed to query kubernetes %q resources", m.object.GetObjectKind())
	}

	// The error is intentionally not returned because we do not attempt
	// to recover from those errors with any specific actions other than to
	// retry later.

	return false, nil
}

// DefaultStateChangeMonitorInterval represents the default interval between
// polling attempts to check whether a host has change state.
const DefaultStateChangeMonitorInterval = 2 * time.Minute

// stateChangeMonitor waits for a host to reach a desired state.  Once the host has
// reached the desired state a reconcilable event is generated to kick the
// reconciler.
type stateChangeMonitor struct {
	manager.CommonMonitorBody
	lastLoggedState string
	hostID          string
}

// NewMonitorCountMonitor defines a convenience function to instantiate
// a new host state monitor with all required attributes.
func NewStateChangeMonitor(instance *v1beta1.Host, id string) *manager.Monitor {
	logger := log.WithName("state-change-monitor")
	return &manager.Monitor{
		MonitorBody: &stateChangeMonitor{
			hostID: id,
		},
		Logger:   logger,
		Object:   instance,
		Interval: DefaultStateChangeMonitorInterval,
	}
}

// Run implements the MonitorBody interface Run method which is responsible
// for monitor one or more resources and returning true when all conditions
// are satisfied.
func (m *stateChangeMonitor) Run(client *gophercloud.ServiceClient) (stop bool, err error) {
	host, err := hosts.Get(client, m.hostID).Extract()
	if err != nil {
		m.SetState("failed to get host %q: %s", m.hostID, err.Error())
		return false, err
	}

	state := fmt.Sprintf("%s/%s/%s", host.AdministrativeState, host.OperationalStatus, host.AvailabilityStatus)

	if m.lastLoggedState == "" || state == m.lastLoggedState {
		m.SetState("monitoring for state changes")
		m.lastLoggedState = state
		return false, nil

	} else {
		m.SetState("state changed from %s to %s", m.lastLoggedState, state)
		m.lastLoggedState = state

		// Force the reconciler to run to pick up this change and reflect it
		// in the database.
		return true, nil
	}
}
