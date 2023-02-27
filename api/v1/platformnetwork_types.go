/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AllocationRange defines the start and end address for an allocation range
type AllocationRange struct {
	// Start defines the beginning of the address range (inclusively)
	Start string `json:"start"`

	// End defines the end of the address range (inclusively)
	End string `json:"end"`
}

// AllocationInfo defines the allocation scheme details for a network
type AllocationInfo struct {
	// Ranges defines the pools from which host addresses are allocated.   If
	// omitted addresses the entire network
	// address space is considered available.
	Ranges []AllocationRange `json:"ranges,omitempty"`

	// Type defines whether network addresses are allocated dynamically or
	// statically.
	// +kubebuilder:validation:Enum=static;dynamic
	Type string `json:"type"`

	// Order defines whether host address are allocation randomly or sequential
	// from the available pool or addresses.
	// +kubebuilder:validation:Enum=sequential;random
	// +optional
	Order *string `json:"order,omitempty"`
}

// PlatformNetworkSpec defines the desired state of PlatformNetwork
type PlatformNetworkSpec struct {
	// Type defines the intended usage of the network
	// +kubebuilder:validation:Enum=mgmt;pxeboot;infra;oam;multicast;system-controller;cluster-host;cluster-pod;cluster-service;storage;admin;other
	Type string `json:"type"`

	// Subnet defines the IPv4 or IPv6 network address for the network
	Subnet string `json:"subnet"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	Prefix int `json:"prefix"`

	// Gateway defines the nexthop gateway IP address if applicable
	// +optional
	Gateway *string `json:"gateway,omitempty"`

	// Allocation defines the allocation scheme details for the network
	Allocation AllocationInfo `json:"allocation"`
}

// PlatformNetworkStatus defines the observed state of PlatformNetwork
type PlatformNetworkStatus struct {
	// ID defines the system assigned unique identifier.  This will only exist
	// once this resource has been provisioned into the system.
	// +optional
	ID *string `json:"id,omitempty"`

	// PoolUUID defines the system assigned unique identifier that is represents
	// the networks underlying address pool resource.  This will only exist
	// once this resource has been provisioned into the system.
	// +optional
	PoolUUID *string `json:"poolUUID,omitempty"`

	// Reconciled defines whether the network has been successfully reconciled
	// at least once.  If further changes are made they will be ignored by the
	// reconciler.
	Reconciled bool `json:"reconciled"`

	// Defines whether the resource has been provisioned on the target system.
	InSync bool `json:"inSync"`
}

// +kubebuilder:object:root=true
// PlatformNetwork defines the attributes that represent the network level
// attributes of a StarlingX system.  This is a composition of the following
// StarlingX API endpoints.
//
//   https://docs.starlingx.io/api-ref/stx-config/api-ref-sysinv-v1-config.html#networks
//   https://docs.starlingx.io/api-ref/stx-config/api-ref-sysinv-v1-config.html#address-pools
//
// +deepequal-gen=false
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="type",type="string",JSONPath=".spec.type",description="The platform network type."
// +kubebuilder:printcolumn:name="subnet",type="string",JSONPath=".spec.subnet",description="The platform network address subnet."
// +kubebuilder:printcolumn:name="prefix",type="string",JSONPath=".spec.prefix",description="The platform network address prefix."
// +kubebuilder:printcolumn:name="insync",type="boolean",JSONPath=".status.inSync",description="The current synchronization state."
// +kubebuilder:printcolumn:name="reconciled",type="boolean",JSONPath=".status.reconciled",description="The current reconciliation state."
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
