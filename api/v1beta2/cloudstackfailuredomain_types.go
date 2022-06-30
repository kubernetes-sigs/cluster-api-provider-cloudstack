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

package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const FailureDomainFinalizer = "cloudstackfailuredomain.infrastructure.cluster.x-k8s.io"

// CloudStackFailureDomainSpec defines the desired state of CloudStackFailureDomain
type CloudStackFailureDomainSpec struct {
	// The ACS Zone for this failure domain.
	Zone Zone `json:"zone"`

	// CloudStack account.
	// +optional
	Account string `json:"account,omitempty"`

	// CloudStack domain.
	// +optional
	Domain string `json:"domain,omitempty"`

	// +k8s:conversion-gen=false
	CredentialsRef *corev1.ObjectReference `json:"credentialsRef"`
}

type CloudStackFailureDomainStatusMap map[string]CloudStackFailureDomainStatus

// CloudStackFailureDomainStatus defines the observed state of CloudStackFailureDomain
type CloudStackFailureDomainStatus struct {
	// Cloudstack Domain ID the failure domain is built in.
	DomainID string `json:"domainID,omitempty"`

	// Reflects the readiness of the CS failure domain.
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+ks8:conversion-gen=false

// CloudStackFailureDomain is the Schema for the cloudstackfailuredomains API
type CloudStackFailureDomain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudStackFailureDomainSpec   `json:"spec,omitempty"`
	Status CloudStackFailureDomainStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CloudStackFailureDomainList contains a list of CloudStackFailureDomain
type CloudStackFailureDomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudStackFailureDomain `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudStackFailureDomain{}, &CloudStackFailureDomainList{})
}
