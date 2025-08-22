package common

import (
	"strconv"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const FactoryInstallConfigMapName = "factory-install"
const FactoryInstalled = "factory-installed"
const FactoryConfigFinalized = "factory-config-finalized"

// GetFactoryConfigMapPredicate return struct of predicates function to be used in the
// controller-runtime watch mechanism.
func GetFactoryConfigMapPredicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return true },
		UpdateFunc:  func(e event.UpdateEvent) bool { return false },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
	}
}

// FactoryReconfigAllowed check if factory re-configuration is allowed based on the factory
// install config-map data.
func FactoryReconfigAllowed(namespace string, obj client.Object) (*v1.ConfigMap, bool) {
	if obj.GetName() != FactoryInstallConfigMapName {
		return nil, false
	}
	if obj.GetNamespace() != namespace {
		return nil, false
	}
	configMap, ok := obj.(*v1.ConfigMap)
	if !ok {
		return nil, false
	}
	factoryInstalledValue, exists := configMap.Data[FactoryInstalled]
	if !exists {
		return nil, false
	}
	if factoryInstalled, _ := strconv.ParseBool(factoryInstalledValue); !factoryInstalled {
		return nil, false
	}

	if factoryConfigFinalizedValue, exists := configMap.Data[FactoryConfigFinalized]; exists {
		factoryConfigFinalized, err := strconv.ParseBool(factoryConfigFinalizedValue)
		if err != nil {
			return nil, false
		}
		if factoryConfigFinalized {
			return nil, false
		}
	}

	return configMap, true
}
