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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// The presence of a finalizer prevents CAPI from deleting the corresponding CAPI data.
	ClusterFinalizer       = "cloudstackcluster.infrastructure.cluster.x-k8s.io"
	defaultIdentityRefKind = "Secret"
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

// GetOne just returns a Zone from the map of zone statuses
// Needed as there's no short way to do this.
func (zones ZoneStatusMap) GetOne() *Zone {
	for _, zone := range zones {
		return &zone
	}
	return nil
}

// GetByName fetches a zone by name if present in the map of zone statuses.
// Needed as there's no short way to do this.
func (zones ZoneStatusMap) GetByName(name string) *Zone {
	for _, zone := range zones {
		if zone.Name == name {
			return &zone
		}
	}
	return nil
}

type Zone struct {
	// The Zone name.
	// + optional
	Name string `json:"name,omitempty"`

	// The CS zone ID the cluster is built in.
	// + optional
	ID string `json:"id,omitempty"`

	// The network within the Zone to use.
	Network Network `json:"network"`
}

// CloudStackClusterSpec defines the desired state of CloudStackCluster.
type CloudStackClusterSpec struct {
	Zones []Zone `json:"zones"`

	// The kubernetes control plane endpoint.
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`

	// CloudStack account.
	// +optional
	Account string `json:"account,omitempty"`

	// CloudStack domain.
	// +optional
	Domain CloudStackResourceIdentifier `json:"domain,omitempty"`

	// +optional
	// +k8s:conversion-gen=false
	IdentityRef *CloudStackIdentityReference `json:"identityRef,omitempty"`
}

// The status of the abstract CS k8s (not an actual Cloudstack Cluster) cluster.
type CloudStackClusterStatus struct {

	// The status of the cluster's ACS Zones.
	// +optional
	Zones ZoneStatusMap `json:"zones,omitempty"`

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
