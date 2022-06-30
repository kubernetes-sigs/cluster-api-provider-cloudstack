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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"strings"
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

type FailureDomain struct {

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

// CloudStackClusterSpec defines the desired state of CloudStackCluster.
type CloudStackClusterSpec struct {
	FailureDomains []FailureDomain `json:"failureDomains"`

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

	// The status of the cluster's ACS Failure Domains.
	// +optional
	CloudStackFailureDomainStatusMap CloudStackFailureDomainStatusMap `json:"zones,omitempty"`

	// CAPI recognizes failure domains as a method to spread machines.
	// CAPC sets failure domains to indicate functioning Zones.
	// +optional
	FailureDomains clusterv1.FailureDomains `json:"failureDomains,omitempty"`

	// Reflects the readiness of the CS cluster.
	Ready bool `json:"ready"`

	// Cloudstack Domain ID the cluster is built in.
	DomainID string `json:"domainID,omitempty"`

	// The CS public IP ID to use for the k8s endpoint.
	PublicIPID string `json:"publicIPID,omitempty"`

	// The ID of the network the PublicIP is in.
	PublicIPNetworkID string `json:"publicIPNetworkID,omitempty"`

	// The ID of the lb rule used to assign VMs to the lb.
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
