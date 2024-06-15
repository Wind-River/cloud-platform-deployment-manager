/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */

package manager

import (
	"context"
	"fmt"
	"strconv"

	perrors "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConfigMap to record and drive the configuration after factory install
// Sample configMap:
/*
apiVersion: v1
kind: ConfigMap
metadata:
	name: factory-install
	namespace: deployment
data:
	factory-installed: "false"  # Initial state (change to "true" if factory-installed)
	factory-config-finalized: "false" # Initial state (change to "true" if configuration is complete)
	system-abcd-default-updated: "false" # Initial state (change to "true" if resource default updated)
*/
const FactoryInstallConfigMapName = "factory-install"
const FactoryInstalled = "factory-installed"
const FactoryConfigFinalized = "factory-config-finalized"

// GetFactoryInstall is to get if the system is in a process of configuration after
// factory install. It reads the factory-install configmap, returns:
// true: factory-installed = true and (factory-config-finalized in (false or nil))
// false:
//  1. factory-installed in (false or nil)
//  2. factory-config-finalized = true
func (m *PlatformManager) GetFactoryInstall(ns string) (bool, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	configMap := &v1.ConfigMap{}
	configMapName := types.NamespacedName{Namespace: ns, Name: FactoryInstallConfigMapName}

	// Lookup the factory install ConfigMap from this namespace
	err := m.GetClient().Get(context.TODO(), configMapName, configMap)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			// Any error other than NotFound is returned
			return false, err
		}
		// ConfigMap not found, return false with no error
		log.V(2).Info("Factory install configmap not found", "namespace", ns)
		return false, nil
	}

	// Read the factory-installed status
	factoryInstalledValue, exists := configMap.Data[FactoryInstalled]
	if !exists {
		return false, nil
	}
	log.V(2).Info("Data factory-installed in configmap", "factoryInstalledValue", factoryInstalledValue)

	factoryInstalled, err := strconv.ParseBool(factoryInstalledValue)
	if err != nil {
		return false, err
	}
	if !factoryInstalled {
		return false, err
	}

	// Read the factory-config-finalized status
	factoryConfigFinalizedValue, exists := configMap.Data[FactoryConfigFinalized]
	if !exists {
		// installed but not finalized
		return true, nil
	}

	factoryConfigFinalized, err2 := strconv.ParseBool(factoryConfigFinalizedValue)
	if err2 != nil {
		// installed but finalized value has error, expected to be updated later
		return true, err
	}

	return !factoryConfigFinalized, nil
}

// SetFactoryConfigFinalized is to set the FactoryConfigFinalized with the value.
func (m *PlatformManager) SetFactoryConfigFinalized(ns string, value bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	configMap := &v1.ConfigMap{}
	configMapName := types.NamespacedName{Namespace: ns, Name: FactoryInstallConfigMapName}

	// Lookup the factory install ConfigMap from this namespace
	err := m.GetClient().Get(context.TODO(), configMapName, configMap)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			// Any error other than NotFound is returned
			return err
		}
		// ConfigMap not found, return false with no error
		log.V(2).Info("Factory install configmap not found", "namespace", ns)
		return nil
	}

	// Update the factory-installed data field
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}
	configMap.Data[FactoryConfigFinalized] = strconv.FormatBool(value)

	// Update the ConfigMap
	err = m.GetClient().Update(context.TODO(), configMap)
	if err != nil {
		return err // Return error if update fails
	}

	log.Info("Factory config has been finalized", "namespace", ns, "value", value)
	return nil
}

// SetResourceDefaultUpdated is to set the <resource name>-default-updated to
// block another update in the next around of reconciliation.
func (m *PlatformManager) SetResourceDefaultUpdated(ns string, name string, value bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	configMap := &v1.ConfigMap{}
	configMapName := types.NamespacedName{Namespace: ns, Name: FactoryInstallConfigMapName}
	// Lookup the factory install configMap from this namespace
	err := m.GetClient().Get(context.TODO(), configMapName, configMap)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			// Any error other than NotFound is returned
			return err
		}
		// ConfigMap not found, return false with no error
		log.V(2).Info("Factory install configmap not found", "namespace", ns)
		return nil
	}

	// Add new data entries
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}
	key := name + "-default-updated"
	configMap.Data[key] = strconv.FormatBool(value)
	// Update the ConfigMap
	err = m.GetClient().Update(context.TODO(), configMap)
	if err != nil {
		err = perrors.Wrapf(err, "Error updating ConfigMap: %s", FactoryInstallConfigMapName)
		return err
	}
	log.V(2).Info("Resource defaults updated", "key", key)

	return nil
}

// GetResourceDefaultUpdated is to get the <resource name>-default-updated to
// block another update in the next around of reconciliation.
func (m *PlatformManager) GetResourceDefaultUpdated(ns string, name string) (bool, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	configMap := &v1.ConfigMap{}
	configMapName := types.NamespacedName{Namespace: ns, Name: FactoryInstallConfigMapName}
	// Lookup the factory install configMap from this namespace
	err := m.GetClient().Get(context.TODO(), configMapName, configMap)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			// Any error other than NotFound is returned
			return false, err
		}
		// ConfigMap not found, return false with no error
		log.V(2).Info("Factory install configmap not found", "namespace", ns)
		return false, nil
	}

	// Retrieve the data entry
	key := name + "-default-updated"
	valueStr, ok := configMap.Data[key]
	if !ok {
		log.V(2).Info("Default updated info not found", "key", key)
		return false, nil
	}

	// Convert the string value to a boolean
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return false, fmt.Errorf("error parsing boolean value from key %s in ConfigMap %s: %v", key, configMapName, err)
	}

	return value, nil
}
