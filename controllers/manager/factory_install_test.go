package manager

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetFactoryInstall(t *testing.T) {
	// Create a scheme and add ConfigMap to it
	scheme := runtime.NewScheme()
	err := v1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("failed to add v1 scheme: %v", err)
	}

	// Create a fake client with a ConfigMap
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FactoryInstallConfigMapName,
			Namespace: "deployment",
		},
		Data: map[string]string{
			"factory-installed": "true",
		},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap).Build()

	// Create the Dummymanager with the fake client
	manager := &Dummymanager{
		Resource: make(map[string]*ResourceInfo),
		Client:   k8sClient, // Assign the fake client to the manager
	}

	// Test GetFactoryInstall
	result, err := manager.GetFactoryInstall("deployment")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result {
		t.Fatalf("expected false, got true")
	}
}

func TestSetFactoryConfigFinalized(t *testing.T) {
	// Create a scheme and add ConfigMap to it
	scheme := runtime.NewScheme()
	err := v1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("failed to add v1 scheme: %v", err)
	}

	// Create a fake client with a ConfigMap
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FactoryInstallConfigMapName,
			Namespace: "deployment",
		},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap).Build()

	// Create the Dummymanager with the fake client
	manager := &Dummymanager{
		Resource: make(map[string]*ResourceInfo),
		Client:   k8sClient, // Assign the fake client to the manager
	}

	// Test SetFactoryConfigFinalized
	err = manager.SetFactoryConfigFinalized("deployment", true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the ConfigMap data
	updatedConfigMap := &v1.ConfigMap{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{
		Name:      FactoryInstallConfigMapName,
		Namespace: "deployment",
	}, updatedConfigMap)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updatedConfigMap.Data["factory-config-finalized"] != "true" {
		t.Fatalf("expected 'true', got %v", updatedConfigMap.Data["factory-config-finalized"])
	}
}

func TestSetResourceDefaultUpdated(t *testing.T) {
	// Create a scheme and add ConfigMap to it
	scheme := runtime.NewScheme()
	err := v1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("failed to add v1 scheme: %v", err)
	}

	// Create a fake client with a ConfigMap
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FactoryInstallConfigMapName,
			Namespace: "deployment",
		},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap).Build()

	// Create the Dummymanager with the fake client
	manager := &Dummymanager{
		Resource: make(map[string]*ResourceInfo),
		Client:   k8sClient, // Assign the fake client to the manager
	}

	// Test SetResourceDefaultUpdated
	err = manager.SetResourceDefaultUpdated("deployment", "system-abcd", true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the ConfigMap data
	updatedConfigMap := &v1.ConfigMap{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{
		Name:      FactoryInstallConfigMapName,
		Namespace: "deployment",
	}, updatedConfigMap)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updatedConfigMap.Data["system-abcd-default-updated"] != "true" {
		t.Fatalf("expected 'true', got %v", updatedConfigMap.Data["system-abcd-default-updated"])
	}
}

func TestGetResourceDefaultUpdated(t *testing.T) {
	// Create a scheme and add ConfigMap to it
	scheme := runtime.NewScheme()
	err := v1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("failed to add v1 scheme: %v", err)
	}

	// Create a fake client with a ConfigMap
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FactoryInstallConfigMapName,
			Namespace: "deployment",
		},
		Data: map[string]string{
			"system-abcd-default-updated": "true",
		},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap).Build()

	// Create the Dummymanager with the fake client
	manager := &Dummymanager{
		Resource:       make(map[string]*ResourceInfo),
		Client:         k8sClient, // Assign the fake client to the manager
		defaultUpdated: true,      // Set the defaultUpdated flag to true for testing
	}

	// Test GetResourceDefaultUpdated
	result, err := manager.GetResourceDefaultUpdated("deployment", "system-abcd")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result {
		t.Fatalf("expected true, got false")
	}
}
