/*
Copyright 2022 The Kubernetes Authors.
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

package fakes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:storageversion
//+kubebuilder:subresource:status

// CloudStackMachineOwner is the Schema for CloudStack Machine owner (machineset, etcdadmcluster, kubeadmcontrolplane)
type CloudStackMachineOwner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudStackMachineOwnerSpec   `json:"spec,omitempty"`
	Status CloudStackMachineOwnerStatus `json:"status,omitempty"`
}

// CloudStackMachineOwnerSpec defines the desired state of CloudStackMachineOwner
type CloudStackMachineOwnerSpec struct {
	Replicas *int32 `json:"replicas,omitempty"`
}

// CloudStackMachineOwnerStatus defines the name of the template
type CloudStackMachineOwnerStatus struct {
	Replicas *int32 `json:"replicas"`
	//+optional
	Ready *bool `json:"ready,omitempty"`
	//+optional
	ReadyReplicas *int32 `json:"readyReplicas,omitempty"`
}

//+kubebuilder:object:root=true
// CloudStackMachineOwnerrList contains a list of CloudStackMachineOwner
type CloudStackMachineOwnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudStackMachineOwner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudStackMachineOwner{}, &CloudStackMachineOwnerList{})
}
