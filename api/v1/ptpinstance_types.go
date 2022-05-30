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

// PtpInstanceSpec defines the desired state of PtpInstance
type PtpInstanceSpec struct {
	// Serivce defines the service type of the ptp instance
	// +kubebuilder:validation:Enum=ptp4l;phc2sys;ts2phc
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
	Reconciled bool `json:"reconciled"`

	// Defines whether the resource has been provisioned on the target system.
	InSync bool `json:"inSync"`
}

// +kubebuilder:object:root=true
// +deepequal-gen=false
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="insync",type="boolean",JSONPath=".status.inSync",description="The current synchronization state."
// +kubebuilder:printcolumn:name="reconciled",type="boolean",JSONPath=".status.reconciled",description="The current reconciliation state."
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
