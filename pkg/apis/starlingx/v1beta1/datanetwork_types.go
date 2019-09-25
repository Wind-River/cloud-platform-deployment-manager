/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VxLANInfo defines VxLAN specific attributes of a data network
type VxLANInfo struct {
	// MulticastGroup defines the multicast IP address to be used for the data
	// network.
	// +optional
	MulticastGroup *string `json:"multicastGroup,omitempty"`

	// UDPPortNumber defines the UDP protocol number to be used for the data
	// network.  The IANA or Legacy port
	// number values can be used.
	// +kubebuilder:validation:Enum=4789,8472
	UDPPortNumber *int `json:"udpPortNumber,omitempty"`

	// TTL defines the time-to-live value to assign to the data network.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=255
	// +optional
	TTL *int `json:"ttl,omitempty"`

	// EndpointMode defines the endpoint port learning mode for the data network
	// network.  The dynamic mode allows the virtual network to use multicast
	// addressing when transmitting a packet to an unknown endpoint to
	// dynamically discover that node's VTEP IP address.   The static mode
	// requires that all VTEP IP addresses be programmed into the virtual switch
	// in advance and any packets destined to an unknown endpoint are dropped.
	// +kubebuilder:validation:Enum=static,dynamic
	// +optional
	EndpointMode *string `json:"endpointMode,omitempty"`
}

// DataNetworkSpec defines the desired state of a DataNetwork resource
type DataNetworkSpec struct {
	// Type defines the encapsulation method used for the data network.
	// +kubebuilder:validation:Enum=flat,vlan,vxlan
	Type string `json:"type"`

	// Description defines a user define description which explains the purpose
	// of the data network.
	// +optional
	Description *string `json:"description,omitempty"`

	// MTU defines the maximum transmit unit for any virtual network derived
	// from this data network.
	// +kubebuilder:validation:Minimum=576
	// +kubebuilder:validation:Maximum=9216
	// +optional
	MTU *int `json:"mtu,omitempty"`

	// VxLan defines VxLAN specific attributes for the data network.
	// +optional
	VxLAN *VxLANInfo `json:"vxlan,omitempty"`
}

// DataNetworkStatus defines the observed state of a DataNetwork resource
type DataNetworkStatus struct {
	// ID defines the system assigned unique identifier.  This will only exist
	// once this resource has been provisioned into the system.
	// +optional
	ID *string `json:"id,omitempty"`

	// Reconciled defines whether the host has been successfully reconciled
	// at least once.  If further changes are made they will be ignored by the
	// reconciler.
	Reconciled bool `json:"reconciled"`

	// Defines whether the resource has been provisioned on the target system.
	InSync bool `json:"inSync"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DataNetworks defines the attributes that represent the data network level
// attributes of a StarlingX system.  This is a composition of the following
// StarlingX API endpoints.
//
//   https://docs.starlingx.io/api-ref/stx-config/api-ref-sysinv-v1-config.html#data-networks
//
// +k8s:openapi-gen=true
// +deepequal-gen=false
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="type",type="string",JSONPath=".spec.type",description="The data network encapsulation type."
// +kubebuilder:printcolumn:name="insync",type="boolean",JSONPath=".status.inSync",description="The current synchronization state."
type DataNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataNetworkSpec   `json:"spec,omitempty"`
	Status DataNetworkStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DataNetworkNameList contains a list of DataNetwork
// +deepequal-gen=false
type DataNetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataNetwork `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataNetwork{}, &DataNetworkList{})
}
