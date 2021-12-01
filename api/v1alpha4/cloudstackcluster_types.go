/*
Copyright 2021.

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

package v1alpha4

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
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

// CloudStackClusterSpec defines the desired state of CloudStackCluster.
type CloudStackClusterSpec struct {
	// CloudStack Zone name.
	Zone string `json:"zone"`

	// CloudStack guest network name.
	Network string `json:"network,omitempty"`

	// The kubernetes control plane endpoint.
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`

	// +optional
	// +k8s:conversion-gen=false
	IdentityRef *CloudStackIdentityReference `json:"identityRef,omitempty"`
}

// The status of the abstract CS k8s (not an actual Cloudstack Cluster) cluster.
type CloudStackClusterStatus struct {
	// Reflects the readiness of the CS cluster.
	Ready bool `json:"ready"`

	// The CS zone ID the cluster is built in.
	ZoneID string `json:"zoneID"`

	// Cloudstack Network ID the cluster is built in.
	NetworkID string `json:"networkID,omitempty"`

	// Cloudstack Network Type the cluster is built in.
	NetworkType string `json:"networkType,omitempty"`

	// The CS public IP ID to use for the k8s endpoint.
	PublicIPID string `json:"publicIPID,omitempty"`

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
