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

const (
	// The presence of a finalizer prevents CAPI from deleting the corresponding CAPI data.
	IsolatedNetworkFinalizer = "cloudstackisolatednetwork.infrastructure.cluster.x-k8s.io"
)

// CloudStackIsolatedNetworkSpec defines the desired state of CloudStackIsolatedNetwork
type CloudStackIsolatedNetworkSpec struct {
	// Name is the name of the isolated network.
	// +optional
	Name string `json:"name,omitempty"`

	// ID is the CloudStack id of the isolated network.
	// +optional
	ID string `json:"id,omitempty"`

	// IdentityRef is a reference to a identity to be used when reconciling this cluster
	// +optional
	// +k8s:conversion-gen=false
	IdentityRef *CloudStackIdentityReference `json:"identityRef,omitempty"`
}

// CloudStackIsolatedNetworkStatus defines the observed state of CloudStackIsolatedNetwork
type CloudStackIsolatedNetworkStatus struct {
	// Name is the CloudStack type of the isolated network.
	// +optional
	Type string `json:"type,omitempty"`

	// Ready indicates the readiness of this provider resource.
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CloudStackIsolatedNetwork is the Schema for the cloudstackisolatednetworks API
type CloudStackIsolatedNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudStackIsolatedNetworkSpec   `json:"spec,omitempty"`
	Status CloudStackIsolatedNetworkStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CloudStackIsolatedNetworkList contains a list of CloudStackIsolatedNetwork
type CloudStackIsolatedNetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudStackIsolatedNetwork `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudStackIsolatedNetwork{}, &CloudStackIsolatedNetworkList{})
}
