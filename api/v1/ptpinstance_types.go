/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PtpInstanceSpec defines the desired state of PtpInstance
type PtpInstanceSpec struct {
	// Serivce defines the service type of the ptp instance
	// +kubebuilder:validation:Enum=ptp4l;phc2sys;ts2phc;clock
	Service string `json:"service"`

	// Parameters contains a list of parameters assigned to the ptp instance
	// +optional
	InstanceParameters []string `json:"parameters,omitempty"`
}

// PtpInstanceStatus defines the observed state of PtpInstance
type PtpInstanceStatus struct {
	// ID defines the system assigned unique identifier.  This will only exist
	// once this resource has been provisioned into the system.
	// +optional
	ID *string `json:"id,omitempty"`

	// Reconciled defines whether the host has been successfully reconciled
	// at least once.  If further changes are made they will be ignored by the
	// reconciler.
	// +optional
	Reconciled bool `json:"reconciled"`

	// Defines whether the resource has been provisioned on the target system.
	// +optional
	InSync bool `json:"inSync"`

	// DeploymentScope defines whether the resource has been deployed
	// on the initial setup or during an update.
	// +kubebuilder:validation:Enum=bootstrap;principal
	// +optional
	// +kubebuilder:default:=bootstrap
	DeploymentScope string `json:"deploymentScope"`

	// Reflect value of configuration generation.
	// The value will be set when configuration generation is updated.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration"`

	// Value for configuration is updated or not
	// +optional
	ConfigurationUpdated bool `json:"configurationUpdated"`

	// Value for configuration is updated or not
	// +kubebuilder:validation:Enum=not_required;lock_required;unlock_required
	// +optional
	// +kubebuilder:default:=not_required
	StrategyRequired string `json:"strategyRequired"`

	// Delta between final profile vs current configuration
	// +optional
	Delta string `json:"delta"`
}

// +kubebuilder:object:root=true
// +deepequal-gen=false
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="insync",type="boolean",JSONPath=".status.inSync",description="The current synchronization state."
// +kubebuilder:printcolumn:name="scope",type="string",JSONPath=".status.deploymentScope",description="The current deploymentScope state."
// +kubebuilder:printcolumn:name="reconciled",type="boolean",JSONPath=".status.reconciled",description="The current reconciliation state."
// +TODO(ecandotti): enhance docs/playbooks/wind-river-cloud-platform-deployment-manager.yaml#L431 since it's looking for the last column to get 'reconciled' value.
// PtpInstance is the Schema for the ptpinstances API
type PtpInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PtpInstanceSpec   `json:"spec,omitempty"`
	Status PtpInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// PtpInstanceList contains a list of PtpInstance
// +deepequal-gen=false
type PtpInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PtpInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PtpInstance{}, &PtpInstanceList{})
}
