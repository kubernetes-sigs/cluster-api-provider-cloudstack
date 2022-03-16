/*
Copyright 2022.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloudStackZoneSpec defines the desired state of CloudStackZone
type CloudStackZoneSpec struct {
	// The name of the CloudStack zone.
	Name string `json:"name,omitempty"`

	// The ID of the CloudStack zone.
	ID string `json:"id,omitempty"`

	// The network within the Zone to use.
	Network Network `json:"network"`
}

// CloudStackZoneStatus defines the observed state of CloudStackZone
type CloudStackZoneStatus struct {
	// Reflects the readiness of the CloudStack zone.
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CloudStackZone is the Schema for the cloudstackzones API
type CloudStackZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudStackZoneSpec   `json:"spec,omitempty"`
	Status CloudStackZoneStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CloudStackZoneList contains a list of CloudStackZone
type CloudStackZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudStackZone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudStackZone{}, &CloudStackZoneList{})
}
