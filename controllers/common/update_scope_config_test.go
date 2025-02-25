/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2025 Wind River Systems, Inc. */

package common

import (
	"testing"

	"github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type MockStarlingxInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            MockStatus `json:"status,omitempty"`
}

type MockStatus struct {
	StrategyRequired     string `json:"strategyRequired,omitempty"`
	DeploymentScope      string `json:"deploymentScope,omitempty"`
	Reconciled           bool   `json:"reconciled,omitempty"`
	ConfigurationUpdated bool   `json:"configurationUpdated,omitempty"`
	ObservedGeneration   int64  `json:"observedGeneration,omitempty"`
}

func (m *MockStarlingxInstance) GetObjectMeta() metav1.Object {
	return &m.ObjectMeta
}

func (m *MockStarlingxInstance) GetStrategyRequired() string {
	return m.Status.StrategyRequired
}

func (m *MockStarlingxInstance) SetStrategyRequired(strategy string) {
	m.Status.StrategyRequired = strategy
}

func (m *MockStarlingxInstance) GetDeploymentScope() string {
	return m.Status.DeploymentScope
}

func (m *MockStarlingxInstance) SetDeploymentScope(scope string) {
	m.Status.DeploymentScope = scope
}

func (m *MockStarlingxInstance) DeepCopyObject() runtime.Object {
	copied := *m
	return &copied
}

func TestExtractScope(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		want    string
		wantErr bool
	}{
		{
			name:    "Valid JSON with deployment scope",
			config:  `{"status":{"deploymentScope":"principal"}}`,
			want:    "principal",
			wantErr: false,
		},
		{
			name:    "Invalid JSON format",
			config:  `{"status":{"deploymentScope":"principal"`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "Missing deployment scope field",
			config:  `{"status":{}}`,
			want:    "",
			wantErr: false,
		},
		{
			name:    "Missing Status",
			config:  `{}`,
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractScope(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractScope() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDeploymentScope(t *testing.T) {
	tests := []struct {
		name          string
		annotations   map[string]string
		expectedScope string
		expectError   bool
	}{
		{
			name: "Valid scope in annotations",
			annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": `{"status":{"deploymentScope":"principal"}}`,
			},
			expectedScope: "principal",
			expectError:   false,
		},
		{
			name:          "No annotations",
			annotations:   map[string]string{},
			expectedScope: "bootstrap", // Assuming bootstrap is the default
			expectError:   false,
		},
		{
			name: "Invalid JSON in annotations",
			annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": `{"status":{"deploymentScope":}`,
			},
			expectedScope: "bootstrap",
			expectError:   true,
		},
		{
			name: "Missing deploymentScope field",
			annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": `{"status":{}}`,
			},
			expectedScope: "bootstrap",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &MockStarlingxInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-instance",
					Namespace:   "default",
					Annotations: tt.annotations,
				},
			}

			scope, err := GetDeploymentScope(instance)

			if scope != tt.expectedScope {
				t.Errorf("Test %s failed: expected scope %v, got %v", tt.name, tt.expectedScope, scope)
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("Test %s expected an error but did not get one", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Test %s did not expect an error but got one: %v", tt.name, err)
				}
			}
		})
	}
}

func TestUpdateDeploymentScope(t *testing.T) {
	instance := &MockStarlingxInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": `{"status":{"deploymentScope":"principal"}}`,
			},
		},
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(schema.GroupVersion{Group: "test-group", Version: "v1"}, instance)

	client := fake.NewClientBuilder().WithScheme(s).WithObjects(instance).WithStatusSubresource(instance).Build()

	updated, err := UpdateDeploymentScope(client, instance)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !updated {
		t.Fatalf("expected true, got false")
	}

	if instance.GetDeploymentScope() != manager.ScopePrincipal {
		t.Fatalf("expected principal, got %s", instance.GetDeploymentScope())
	}
}
