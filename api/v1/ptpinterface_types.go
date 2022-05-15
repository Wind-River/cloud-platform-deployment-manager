/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
/* Copyright(c) 2022 Wind River Systems, Inc. */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PtpInterfaceSpec defines the desired state of PtpInterface
type PtpInterfaceSpec struct {
	// ptpinstance defines the ptp instance assigned to the ptp interface
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_]+$
	PtpInstance string `json:"ptpinstance"`

	// InterfaceParameters contains a list of parameters assigned to the ptp interface
	// +optional
	InterfaceParameters []string `json:"parameters,omitempty"`
}

// PtpInterfaceStatus defines the observed state of PtpInterface
type PtpInterfaceStatus struct {
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

// +kubebuilder:object:root=true
// +deepequal-gen=false
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="insync",type="boolean",JSONPath=".status.inSync",description="The current synchronization state."
// +kubebuilder:printcolumn:name="reconciled",type="boolean",JSONPath=".status.reconciled",description="The current reconciliation state."
type PtpInterface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PtpInterfaceSpec   `json:"spec,omitempty"`
	Status PtpInterfaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// PtpInterfaceList contains a list of PtpInterface
// +deepequal-gen=false
type PtpInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PtpInterface `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PtpInterface{}, &PtpInterfaceList{})
}
