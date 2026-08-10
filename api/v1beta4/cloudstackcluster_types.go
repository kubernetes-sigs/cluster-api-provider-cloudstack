/*
Copyright 2024 The Kubernetes Authors.

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

package v1beta4

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterFinalizer = "cloudstackcluster.infrastructure.cluster.x-k8s.io"
)

var K8sClient client.Client

// FailureDomain describes a failure domain for machine placement, serialized as a JSON array
// element compatible with the CAPI v1beta2 infrastructure contract.
type FailureDomain struct {
	// name is the name of the failure domain.
	// +required
	Name string `json:"name,omitempty"`

	// controlPlane determines if this failure domain is suitable for use by control plane machines.
	// +optional
	ControlPlane *bool `json:"controlPlane,omitempty"`

	// attributes is a free form map of attributes an infrastructure provider might use or require.
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
}

// ClusterInitializationStatus tracks the initialization state of the CloudStackCluster,
// implementing the CAPI v1beta2 infrastructure contract.
type ClusterInitializationStatus struct {
	// provisioned is true when the infrastructure provider reports that the cluster's
	// infrastructure is fully provisioned.
	// NOTE: this field is part of the CAPI v1beta2 contract.
	// +optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

// CloudStackClusterSpec defines the desired state of CloudStackCluster.
type CloudStackClusterSpec struct {
	FailureDomains []CloudStackFailureDomainSpec `json:"failureDomains"`

	// controlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// Populated by the controller after LB provisioning; empty at create time.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty,omitzero"`

	// SyncWithACS determines if an externalManaged CKS cluster should be created on ACS.
	// +optional
	SyncWithACS *bool `json:"syncWithACS,omitempty"`
}

// CloudStackClusterStatus defines the observed state of CloudStackCluster.
type CloudStackClusterStatus struct {
	// failureDomains is a slice of failure domain objects synced to the CAPI Cluster.
	// Serialized as a JSON array to comply with the CAPI v1beta2 infrastructure contract.
	// +optional
	// +listType=map
	// +listMapKey=name
	FailureDomains []FailureDomain `json:"failureDomains,omitempty"`

	// Id of CAPC managed kubernetes cluster created in CloudStack.
	// +optional
	CloudStackClusterID string `json:"cloudStackClusterId"`

	// Ready reflects the readiness of the CS cluster.
	// Kept for backward compatibility; prefer Initialization.Provisioned.
	Ready bool `json:"ready"`

	// initialization contains the initialization state of the cluster infrastructure.
	// NOTE: this field implements the CAPI v1beta2 infrastructure contract.
	// +optional
	Initialization *ClusterInitializationStatus `json:"initialization,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:metadata:labels="cluster.x-k8s.io/v1beta2=v1beta4"

// CloudStackCluster is the Schema for the cloudstackclusters API.
type CloudStackCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CloudStackClusterSpec `json:"spec,omitempty"`

	// The actual cluster state reported by CloudStack.
	Status CloudStackClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CloudStackClusterList contains a list of CloudStackCluster.
type CloudStackClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudStackCluster `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &CloudStackCluster{}, &CloudStackClusterList{})
}
