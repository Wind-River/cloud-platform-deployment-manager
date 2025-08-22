/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */

package manager

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/nfv/v1/systemconfigupdate"
	"github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Dummymanager for unit test
type Dummymanager struct {
	vimClientAvailable    bool
	gcShow                string
	strategyCreated       bool
	strategyCreateError   bool
	strategySent          bool
	strategyDeleted       bool
	strategyActionSend    bool
	strategyActionError   bool
	config_version        int
	monitor_version       int
	Resource              map[string]*ResourceInfo
	strategyCreateRequest systemconfigupdate.SystemConfigUpdateOpts
	retryCount            int

	DefaultUpdateError error         // Simulate error for default update method
	Client             client.Client // Added client field for the Kubernetes client
}

func (m *Dummymanager) ResetPlatformClient(namespace string) error {
	return nil
}
func (m *Dummymanager) GetPlatformClient(namespace string) *gophercloud.ServiceClient {
	c := &gophercloud.ServiceClient{}
	return c
}
func (m *Dummymanager) GetKubernetesClient() client.Client {
	return nil
}
func (m *Dummymanager) BuildPlatformClient(namespace string, endpointName string, endpointType string) (*gophercloud.ServiceClient, error) {
	c := &gophercloud.ServiceClient{}
	return c, nil
}
func (m *Dummymanager) NotifySystemDependencies(namespace string) error {
	return nil
}
func (m *Dummymanager) NotifyResource(object client.Object) error {
	return nil
}
func (m *Dummymanager) SetSystemReady(namespace string, value bool) {

}
func (m *Dummymanager) GetSystemReady(namespace string) bool {
	return true
}
func (m *Dummymanager) SetSystemType(namespace string, value SystemType) {

}
func (m *Dummymanager) GetSystemType(namespace string) SystemType {
	return ""
}
func (m *Dummymanager) StartMonitor(monitor *Monitor, message string) error {
	return nil
}
func (m *Dummymanager) CancelMonitor(object client.Object) {

}
func (m *Dummymanager) GetActiveHost(namespace string, client *gophercloud.ServiceClient) (*starlingxv1.Host, error) {
	return nil, nil
}
func (m *Dummymanager) GetSystemInfo(namespace string, client *gophercloud.ServiceClient) (*SystemInfo, error) {
	return nil, nil
}
func (m *Dummymanager) SetResourceInfo(resourcetype string, personality string, resourcename string, reconciled bool, required string) {

}
func (m *Dummymanager) GetStrategyRequiredList() map[string]*ResourceInfo {
	return m.Resource
}
func (m *Dummymanager) ListStrategyRequired() string {
	return ""
}
func (m *Dummymanager) UpdateConfigVersion() {

}
func (m *Dummymanager) GetConfigVersion() int {
	return m.config_version
}
func (m *Dummymanager) GetMonitorVersion() int {
	return m.monitor_version
}
func (m *Dummymanager) SetMonitorVersion(i int) {

}
func (m *Dummymanager) StrategySent() {
	m.strategySent = true
}
func (m *Dummymanager) GetStrategySent() bool {
	return m.strategySent
}
func (m *Dummymanager) ClearStrategy() {

}
func (m *Dummymanager) GetNamespace() string {
	return ""
}
func (m *Dummymanager) GetVimClient() *gophercloud.ServiceClient {
	if m.vimClientAvailable {
		c := &gophercloud.ServiceClient{}
		return c
	} else {
		return nil
	}
}
func (m *Dummymanager) SetStrategyAppliedSent(namespace string, applied bool) error {
	return nil
}
func (m *Dummymanager) StartStrategyMonitor() {

}
func (m *Dummymanager) SetStrategyRetryCount(c int) error {
	return nil
}
func (m *Dummymanager) GetStrategyRetryCount() (int, error) {
	return m.retryCount, nil
}
func (m *Dummymanager) IsPlatformNetworkReconciling() bool {
	return false
}
func (m *Dummymanager) SetPlatformNetworkReconciling(status bool) {

}
func (m *Dummymanager) IsNotifyingActiveHost() bool {
	return false
}
func (m *Dummymanager) SetNotifyingActiveHost(status bool) {

}
func (m *Dummymanager) SetStrategyExpectedByOtherReconcilers(status bool) {

}
func (m *Dummymanager) GetStrategyExpectedByOtherReconcilers() bool {
	return false
}
func (m *Dummymanager) GetHostByPersonality(namespace string, client *gophercloud.ServiceClient, personality string) (*starlingxv1.Host, *hosts.Host, error) {
	return nil, nil, nil
}
func (m *Dummymanager) GcShow(c *gophercloud.ServiceClient) (*systemconfigupdate.SystemConfigUpdate, error) {
	if len(m.gcShow) != 0 {
		s := &systemconfigupdate.SystemConfigUpdate{
			State: m.gcShow,
			ID:    "abc-def",
		}
		return s, nil
	}

	// VIM API sends json response {"strategy": null} when there are no
	// strategies on the system and not error responses such as 404.
	// Hence we are returning nil, nil without error.

	return nil, nil
}
func (m *Dummymanager) GcActionStrategy(c *gophercloud.ServiceClient, opts systemconfigupdate.StrategyActionOpts) (*systemconfigupdate.SystemConfigUpdate, error) {
	m.strategyActionSend = true
	if m.strategyActionError {
		err := errors.New("test: action sent error")
		return nil, err
	} else {
		s := &systemconfigupdate.SystemConfigUpdate{}
		return s, nil
	}
}
func (m *Dummymanager) GcCreate(c *gophercloud.ServiceClient, opts systemconfigupdate.SystemConfigUpdateOpts) (*systemconfigupdate.SystemConfigUpdate, error) {
	if m.strategyCreateError {
		return nil, errors.New("Test strategy create error")
	} else {
		m.strategyCreated = true
		m.strategyCreateRequest = opts
		s := &systemconfigupdate.SystemConfigUpdate{}
		return s, nil
	}
}
func (m *Dummymanager) GcDelete(c *gophercloud.ServiceClient) (r systemconfigupdate.DeleteResult) {
	m.strategyDeleted = true
	re := systemconfigupdate.DeleteResult{}
	return re
}
func (m *Dummymanager) SetDefaultGetPlatformClient() {

}
func (m *Dummymanager) SetGetPlatformClient(f func(namespace string) *gophercloud.ServiceClient) {

}
