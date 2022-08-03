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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	ClusterFinalizer           = "cloudstackcluster.infrastructure.cluster.x-k8s.io"
	CloudStackClusterLabelName = "cloudstackcluster.infrastructure.cluster.x-k8s.io/name"
)

var K8sClient client.Client

// CloudStackIdentityReference is a reference to an infrastructure
// provider identity to be used to provision cluster resources.
type CloudStackIdentityReference struct {
	// Kind of the identity. Must be supported by the infrastructure provider
	// and may be either cluster or namespace-scoped.
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`

	// Name of the infrastructure identity to be used.
	Name string `json:"name"`
}

type ZoneStatusMap map[string]Zone

type Zone struct {
	// Name.
	//+optional
	Name string `json:"name,omitempty"`

	// ID.
	//+optional
	ID string `json:"id,omitempty"`

	// The network within the Zone to use.
	Network Network `json:"network"`
}

// MetaName returns a lower cased name to be used in a k8s object meta.
// It prefers the zone's name, but will use the ID if that's the only present identifier.
func (z *Zone) MetaName() string {
	s := z.Name
	if s == "" {
		s = z.ID
	}
	return strings.ToLower(s)
}

// CloudStackClusterSpec defines the desired state of CloudStackCluster.
type CloudStackClusterSpec struct {
	FailureDomains []CloudStackFailureDomainSpec `json:"failureDomains"`

	// The kubernetes control plane endpoint.
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`

	// CloudStack account.
	// +optional
	Account string `json:"account,omitempty"`

	// CloudStack domain.
	// +optional
	Domain string `json:"domain,omitempty"`
}

// The status of the CloudStackCluster object.
type CloudStackClusterStatus struct {

	// CAPI recognizes failure domains as a method to spread machines.
	// CAPC sets failure domains to indicate functioning CloudStackFailureDomains.
	// +optional
	FailureDomains clusterv1.FailureDomains `json:"failureDomains,omitempty"`

	// Reflects the readiness of the CS cluster.
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// CloudStackCluster is the Schema for the cloudstackclusters API
type CloudStackCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CloudStackClusterSpec `json:"spec,omitempty"`

	// The actual cluster state reported by CloudStack.
	Status CloudStackClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CloudStackClusterList contains a list of CloudStackCluster
type CloudStackClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudStackCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudStackCluster{}, &CloudStackClusterList{})
}
