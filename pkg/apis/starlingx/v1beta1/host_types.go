/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MatchBMInfo defines the board management attributes that can be used to
// match a system host resource to a
// host CR definition.
type MatchBMInfo struct {
	// Address defines the board management IP address.
	Address *string `json:"address,omitempty"`
}

// MatchDMIInfo defines the Desktop Management Interface attributes that can
// be used to match a system host
// resource to a host CR definition.
type MatchDMIInfo struct {
	// SerialNumber defines the board serial number as stored in the DMI block.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	SerialNumber *string `json:"serialNumber,omitempty"`

	// AssetTag defines the board asset tag as stored in the DMI block.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	AssetTag *string `json:"assetTag,omitempty"`
}

// MatchInfo defines the attributes that can be used to dynamically match a
// system host resource to a host CR definition.  To be considered a match
// all of the fields defined with the match criteria must match the actual
// attributes of the system host resource.
type MatchInfo struct {
	// BootMAC defines the MAC address that a host used to perform the initial
	// software installation.
	// +kubebuilder:validation:Pattern=^([0-9a-fA-Z]{2}[:-]){5}([0-9a-fA-Z]{2})$
	// +optional
	BootMAC *string `json:"bootMAC,omitempty"`

	// BoardManagement defines the board management attributes that can be used
	// to match a system host resource to a system CR definition.
	// NOTE:  Not yet supported.
	// +optional
	BoardManagement *MatchBMInfo `json:"boardManagement,omitempty"`

	// DMI defines the Desktop Management Interface attributes that can be used
	// to match a system host resource to a system CR definition.
	// NOTE:  Not yet supported.
	// +optional
	DMI *MatchDMIInfo `json:"dmi,omitempty"`
}

// HostSpec defines the desired state of a Host resource.
type HostSpec struct {
	// Profile defines the name of the HostProfile to use as a configuration
	// template for the host.  A host may point to a single HostProfile resource
	// or may point to a chain or hierarchy of HostProfile resources.  At
	// configuration time the Deployment Manager will flatten the Host
	// attributes defined in the hierarchy of HostProfiles and produce a final
	// composite profile that represents the intended configuration state of
	// an individual Host resources.  This composite profile also include any
	// individual host specific attributes defined in the "overrides" attribute
	// defined below.
	Profile string `json:"profile"`

	// Match defines the attributes used to match a system host resource to a
	// host CR definition.
	// +optional
	Match *MatchInfo `json:"match,omitempty"`

	// Overrides defines a set of HostProfile attributes that must be overridden
	// from the base HostProfile before configuring the host.  The schema for
	// this field is intentionally a copy of the full HostProfileSpec schema
	// so that any HostProfile attribute can be overridden on a per-host basis.
	//
	// For example, it may be necessary to define IP addresses that are unique
	// to each host, or to override storage device paths if the installed
	// devices does not align completely with the HostProfile pointed to by the
	// "profile" attribute.
	// +optional
	Overrides *HostProfileSpec `json:"overrides,omitempty"`
}

// HostStatus defines the observed state of a Host resource.
type HostStatus struct {
	// ID defines the system assigned unique identifier.  This will only exist
	// once this resource has been provisioned into the system.
	ID *string `json:"id,omitempty"`

	// AdministrativeState is the last known administrative state of the host.
	AdministrativeState *string `json:"administrativeState,omitempty"`

	// OperationalStatus is the last known operational status of the host.
	OperationalStatus *string `json:"operationalStatus,omitempty"`

	// AvailabilityStatus is the last known availability status of the host.
	AvailabilityStatus *string `json:"availabilityStatus,omitempty"`

	// InSync defines whether the desired state matches the operational state.
	InSync bool `json:"inSync"`

	// Reconciled defines whether the host has been successfully reconciled
	// at least once.  If further changes are made they will be ignored by the
	// reconciler.
	Reconciled bool `json:"reconciled"`

	// Defaults defines the configuration attributed collected before applying
	// any user configuration values.
	Defaults *string `json:"defaults,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Host defines the attributes that represent the host level attributes
// of a StarlingX system.
//
// +k8s:openapi-gen=true
// +deepequal-gen=false
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="administrative",type="string",JSONPath=".status.administrativeState",description="The administrative state of the host."
// +kubebuilder:printcolumn:name="operational",type="string",JSONPath=".status.operationalStatus",description="The operational status of the host."
// +kubebuilder:printcolumn:name="availability",type="string",JSONPath=".status.availabilityStatus",description="The availability status of the host."
// +kubebuilder:printcolumn:name="profile",type="string",JSONPath=".spec.profile",description="The configuration profile of the host."
// +kubebuilder:printcolumn:name="insync",type="boolean",JSONPath=".status.inSync",description="The current synchronization state."
type Host struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HostSpec   `json:"spec,omitempty"`
	Status HostStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HostList contains a list of Host
// +deepequal-gen=false
type HostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Host `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Host{}, &HostList{})
}
