/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PtpInstanceSpec defines the desired state of a PtpInstance resource
type PtpInstanceSpec struct {

	// Serivce defines the service type of the ptp instance
	// +kubebuilder:validation:Enum=ptp4l,phc2sys,ts2phc,clock
	Service string `json:"service"`

	// Parameters contains a list of parameters assigned to the ptp instance
	// +optional
	InstanceParameters []string `json:"parameters,omitempty"`
}

// PtpInstanceStatus defines the observed state of a PtpInstance resource
type PtpInstanceStatus struct {
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
// +k8s:openapi-gen=true
// +deepequal-gen=false
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="insync",type="boolean",JSONPath=".status.inSync",description="The current synchronization state."
// +kubebuilder:printcolumn:name="reconciled",type="boolean",JSONPath=".status.reconciled",description="The current reconciliation state."
type PtpInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PtpInstanceSpec   `json:"spec,omitempty"`
	Status PtpInstanceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

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
