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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	ClusterFinalizer           = "cloudstackcluster.infrastructure.cluster.x-k8s.io"
	defaultIdentityRefKind     = "Secret"
	CloudStackClusterLabelName = "cloudstackcluster.infrastructure.cluster.x-k8s.io/name"
	NetworkTypeIsolated        = "Isolated"
	NetworkTypeShared          = "Shared"
)

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

type Network struct {
	// Cloudstack Network ID the cluster is built in.
	// +optional
	ID string `json:"id,omitempty"`

	// Cloudstack Network Type the cluster is built in.
	// + optional
	Type string `json:"type,omitempty"`

	// Cloudstack Network Name the cluster is built in.
	// +optional
	Name string `json:"name"`
}

// MetaName returns a lower cased name to be used in a k8s object meta.
// It prefers the zone's name, but will use the ID if that's the only present identifier.
func (z *CloudStackZoneSpec) MetaName() string {
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

	// +optional
	// +k8s:conversion-gen=false
	IdentityRef *CloudStackIdentityReference `json:"identityRef,omitempty"`
}

// The status of the abstract CS k8s (not an actual Cloudstack Cluster) cluster.
type CloudStackClusterStatus struct {

	// CAPI recognizes failure domains as a method to spread machines.
	// CAPC sets failure domains to indicate functioning Zones.
	// +optional
	FailureDomains clusterv1.FailureDomains `json:"failureDomains,omitempty"`

	// Reflects the readiness of the CS cluster.
	Ready bool `json:"ready"`

	// Cloudstack Domain ID the cluster is built in.
	// +optional
	DomainID string `json:"domainID,omitempty"`

	// The CS public IP ID to use for the k8s endpoint.
	// +optional
	PublicIPID string `json:"publicIPID,omitempty"`

	// The ID of the network the PublicIP is in.
	// +optional
	PublicIPNetworkID string `json:"publicIPNetworkID,omitempty"`

	// The ID of the lb rule used to assign VMs to the lb.
	// +optional
	LBRuleID string `json:"loadBalancerRuleID,omitempty"`
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
