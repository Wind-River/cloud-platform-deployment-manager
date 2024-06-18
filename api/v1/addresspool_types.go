/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddressPoolStatus defines the observed state of AddressPool
type AddressPoolStatus struct {
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

// AllocationRange defines the start and end address for an allocation range
type AllocationRange struct {
	// Start defines the beginning of the address range (inclusively)
	Start string `json:"start"`

	// End defines the end of the address range (inclusively)
	End string `json:"end"`
}

// AllocationInfo defines the allocation scheme details for a network
type AllocationInfo struct {
	// Ranges defines the pools from which host addresses are allocated.
	// If omitted addresses the entire network address space is
	// considered available.
	Ranges []AllocationRange `json:"ranges,omitempty"`

	// Order defines whether host address are allocation randomly or sequential
	// from the available pool or addresses.
	// +kubebuilder:validation:Enum=sequential;random
	// +optional
	Order *string `json:"order,omitempty"`
}

// AddressPoolSpec defines the desired state of AddressPool
type AddressPoolSpec struct {
	// Subnet defines the subdivision IPv4 or IPv6 network address for the network
	Subnet string `json:"subnet"`

	// FloatingAddress defines the floating IPv4 or IPv6 network address for the network
	FloatingAddress *string `json:"floatingAddress,omitempty"`

	// Controller0Address is the controller-0 IPv4 or IPv6 network address value.
	Controller0Address *string `json:"controller0Address,omitempty"`

	// Controller1Address is the controller-1 IPv4 or IPv6 network address value.
	Controller1Address *string `json:"controller1Address,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	Prefix int `json:"prefix"`

	// Gateway defines the nexthop gateway IP address if applicable
	// +optional
	Gateway *string `json:"gateway,omitempty"`

	// Allocation defines the allocation scheme details for the network
	Allocation AllocationInfo `json:"allocation"`
}

// +kubebuilder:object:root=true
// AddressPool defines the attributes that represent the addresspool level
// attributes of a StarlingX system.  This represents following StarlingX endpoint:
//
//	https://docs.starlingx.io/api-ref/stx-config/api-ref-sysinv-v1-config.html#address-pools
//
// +deepequal-gen=false
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="subnet",type="string",JSONPath=".spec.subnet",description="The address pool subnet."
// +kubebuilder:printcolumn:name="prefix",type="string",JSONPath=".spec.prefix",description="The address pool prefix."
// +kubebuilder:printcolumn:name="insync",type="boolean",JSONPath=".status.inSync",description="The current synchronization state."
// +kubebuilder:printcolumn:name="reconciled",type="boolean",JSONPath=".status.reconciled",description="The current reconciliation state."
type AddressPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AddressPoolSpec   `json:"spec,omitempty"`
	Status AddressPoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// AddressPoolList contains a list of AddressPool
// +deepequal-gen=false
type AddressPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AddressPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AddressPool{}, &AddressPoolList{})
}
