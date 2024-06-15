/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */

package manager

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/nfv/v1/systemconfigupdate"
	"github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	defaultUpdated     bool          // Simulate default update status
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
func (m *Dummymanager) StrageySent() {
	m.strategySent = true
}
func (m *Dummymanager) GetStrageySent() bool {
	return m.strategySent
}
func (m *Dummymanager) ClearStragey() {

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
func (m *Dummymanager) GcShow(c *gophercloud.ServiceClient) (*systemconfigupdate.SystemConfigUpdate, error) {
	if len(m.gcShow) == 0 {
		err := errors.New("test: no info available")
		return nil, err
	} else {
		s := &systemconfigupdate.SystemConfigUpdate{
			State: m.gcShow,
		}
		return s, nil
	}
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

func (m *Dummymanager) GetFactoryInstall(namespace string) (bool, error) {
	// Assuming factory install is always false for this dummy implementation
	return false, nil
}

func (m *Dummymanager) SetFactoryConfigFinalized(namespace string, value bool) error {
	configMap := &v1.ConfigMap{}
	configMapName := client.ObjectKey{Namespace: namespace, Name: FactoryInstallConfigMapName}
	err := m.Client.Get(context.TODO(), configMapName, configMap)
	if err != nil {
		return err
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}
	configMap.Data[FactoryConfigFinalized] = strconv.FormatBool(value)

	return m.Client.Update(context.TODO(), configMap)
}

func (m *Dummymanager) SetResourceDefaultUpdated(namespace string, name string, value bool) error {
	if m.Client == nil {
		return errors.New("kubernetes client is not initialized")
	}

	configMap := &v1.ConfigMap{}
	configMapName := client.ObjectKey{Namespace: namespace, Name: FactoryInstallConfigMapName}
	err := m.Client.Get(context.TODO(), configMapName, configMap)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		// Create a new ConfigMap if not found
		configMap = &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      FactoryInstallConfigMapName,
				Namespace: namespace,
			},
			Data: make(map[string]string),
		}
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}
	key := name + "-default-updated"
	configMap.Data[key] = strconv.FormatBool(value)

	if err := m.Client.Update(context.TODO(), configMap); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		// Create the ConfigMap if it does not exist
		return m.Client.Create(context.TODO(), configMap)
	}
	m.defaultUpdated = value
	return nil
}

func (m *Dummymanager) GetResourceDefaultUpdated(namespace, name string) (bool, error) {
	if m.Client == nil {
		return false, errors.New("kubernetes client is not initialized")
	}

	configMap := &v1.ConfigMap{}
	configMapName := client.ObjectKey{Namespace: namespace, Name: FactoryInstallConfigMapName}
	err := m.Client.Get(context.TODO(), configMapName, configMap)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return false, err
		}
		return false, nil
	}

	key := name + "-default-updated"
	valueStr, ok := configMap.Data[key]
	if !ok {
		return false, nil
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return false, fmt.Errorf("error parsing boolean value from key %s in ConfigMap %s: %v", key, configMapName, err)
	}

	return value, nil
}
