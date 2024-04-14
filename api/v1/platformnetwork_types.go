/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PlatformNetworkSpec defines the desired state of PlatformNetwork
type PlatformNetworkSpec struct {
	// Type defines the intended usage of the network
	// +kubebuilder:validation:Enum=mgmt;pxeboot;infra;oam;multicast;system-controller;cluster-host;cluster-pod;cluster-service;storage;admin;other
	Type string `json:"type"`

	// Dynamic defines whether network addresses are allocated dynamically (true) or
	// statically (false).
	Dynamic bool `json:"dynamic"`

	// AssociatedAddressPools defines the set of address pool names to associate
	// the network with.
	AssociatedAddressPools []string `json:"associatedAddressPools"`
}

// PlatformNetworkStatus defines the observed state of PlatformNetwork
type PlatformNetworkStatus struct {
	// ID defines the system assigned unique identifier.  This will only exist
	// once this resource has been provisioned into the system.
	// +optional
	ID *string `json:"id,omitempty"`

	// Reconciled defines whether the network has been successfully reconciled
	// at least once.  If further changes are made they will be ignored by the
	// reconciler.
	// +optional
	Reconciled bool `json:"reconciled"`

	// Defines whether the resource has been provisioned on the target system.
	// +optional
	InSync bool `json:"inSync"`

	// DeploymentScope defines whether the resource has been deployed
	// on the initial setup or during an update.
	// +kubebuilder:validation:Enum=bootstrap;principal;Bootstrap;Principal;BOOTSTRAP;PRINCIPAL
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

	// Delta between final profile vs current configuration
	// +optional
	Delta string `json:"delta"`
}

// +kubebuilder:object:root=true
// PlatformNetwork defines the attributes that represent the network level
// attributes of a StarlingX system.  This is a composition of the following
// StarlingX API endpoints.
//
//	https://docs.starlingx.io/api-ref/stx-config/api-ref-sysinv-v1-config.html#networks
//	https://docs.starlingx.io/api-ref/stx-config/api-ref-sysinv-v1-config.html#address-pools
//
// +deepequal-gen=false
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="type",type="string",JSONPath=".spec.type",description="The platform network type."
// +kubebuilder:printcolumn:name="dynamic",type="boolean",JSONPath=".spec.dynamic",description="The platform network address IP Allocation Type."
// +kubebuilder:printcolumn:name="insync",type="boolean",JSONPath=".status.inSync",description="The current synchronization state."
// +kubebuilder:printcolumn:name="reconciled",type="boolean",JSONPath=".status.reconciled",description="The current reconciliation state."
// +kubebuilder:printcolumn:name="scope",type="string",JSONPath=".status.deploymentScope",description="The current deploymentScope state."
type PlatformNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlatformNetworkSpec   `json:"spec,omitempty"`
	Status PlatformNetworkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// PlatformNetworkNameList contains a list of PlatformNetwork
// +deepequal-gen=false
type PlatformNetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlatformNetwork `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlatformNetwork{}, &PlatformNetworkList{})
}
