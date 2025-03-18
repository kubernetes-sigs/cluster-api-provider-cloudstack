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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// The presence of a finalizer prevents CAPI from deleting the corresponding CAPI data.
	MachineFinalizer = "cloudstackmachine.infrastructure.cluster.x-k8s.io"
	ProAffinity      = "pro"
	AntiAffinity     = "anti"
	NoAffinity       = "no"
)

// CloudStackMachineSpec defines the desired state of CloudStackMachine
type CloudStackMachineSpec struct {
	// Name.
	//+optional
	Name string `json:"name,omitempty"`

	// ID.
	//+optional
	ID string `json:"id,omitempty"`

	// Instance ID. Should only be useful to modify an existing instance.
	InstanceID *string `json:"instanceID,omitempty"`

	// CloudStack compute offering.
	Offering CloudStackResourceIdentifier `json:"offering"`

	// CloudStack template to use.
	Template CloudStackResourceIdentifier `json:"template"`

	// CloudStack disk offering to use.
	// +optional
	DiskOffering CloudStackResourceDiskOffering `json:"diskOffering,omitempty"`

	// CloudStack ssh key to use.
	// +optional
	SSHKey string `json:"sshKey"`

	// Optional details map for deployVirtualMachine
	Details map[string]string `json:"details,omitempty"`

	// Optional affinitygroupids for deployVirtualMachine
	// +optional
	AffinityGroupIDs []string `json:"affinitygroupids,omitempty"`

	// Mutually exclusive parameter with AffinityGroupIDs.
	// Defaults to `no`. Can be `pro` or `anti`. Will create an affinity group per machine set.
	// +optional
	Affinity string `json:"affinity,omitempty"`

	// Mutually exclusive parameter with AffinityGroupIDs.
	// Is a reference to a CloudStack affiniity group CRD.
	// +optional
	AffinityGroupRef *corev1.ObjectReference `json:"cloudstackaffinityref,omitempty"`

	// The CS specific unique identifier. Of the form: fmt.Sprintf("cloudstack:///%s", CS Machine ID)
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	// Optionally settable Zone ID to land the machine in.
	// +k8s:conversion-gen=false
	ZoneID string `json:"zoneID,omitempty"`

	// +k8s:conversion-gen=false
	// Optionally settable Zone Name to land the machine in.
	ZoneName string `json:"zoneName,omitempty"`

	// IdentityRef is a reference to an identity to be used when reconciling this cluster
	// +optional
	// +k8s:conversion-gen=false
	IdentityRef *CloudStackIdentityReference `json:"identityRef,omitempty"`
}

type CloudStackResourceIdentifier struct {
	// Cloudstack resource ID.
	// +optional
	ID string `json:"id,omitempty"`

	// Cloudstack resource Name
	// +optional
	Name string `json:"name,omitempty"`
}

type CloudStackResourceDiskOffering struct {
	CloudStackResourceIdentifier `json:",inline"`
	// Desired disk size. Used if disk offering is customizable as indicated by the ACS field 'Custom Disk Size'.
	// +optional
	CustomSize int64 `json:"customSizeInGB"`
	// mount point the data disk uses to mount. The actual partition, mkfs and mount are done by cloud-init generated by kubeadmConfig.
	MountPath string `json:"mountPath"`
	// device name of data disk, for example /dev/vdb
	Device string `json:"device"`
	// filesystem used by data disk, for example, ext4, xfs
	Filesystem string `json:"filesystem"`
	// label of data disk, used by mkfs as label parameter
	Label string `json:"label"`
}

// TODO: Review the use of this field/type.
type InstanceState string

// Type pulled mostly from the CloudStack API.
type CloudStackMachineStatus struct {
	// Zone ID is used so that the zone can be computed once per reconcile and then propagate.
	// +optional
	// +k8s:conversion-gen=false
	ZoneID string `json:"zoneID,omitempty"`

	// Addresses contains a CloudStack VM instance's IP addresses.
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// InstanceState is the state of the CloudStack instance for this machine.
	// +optional
	InstanceState InstanceState `json:"instanceState,omitempty"`

	// InstanceStateLastUpdated is the time the instance state was last updated.
	// +optional
	InstanceStateLastUpdated metav1.Time `json:"instanceStateLastUpdated,omitempty"`

	// Ready indicates the readiness of the provider resource.
	Ready bool `json:"ready"`
}

// TimeSinceLastStateChange returns the amount of time that's elapsed since the state was last updated.  If the state
// hasn't ever been updated, it returns a negative value.
func (s *CloudStackMachineStatus) TimeSinceLastStateChange() time.Duration {
	if s.InstanceStateLastUpdated.IsZero() {
		return time.Duration(-1)
	}
	return time.Since(s.InstanceStateLastUpdated.Time)
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=cloudstackmachines,scope=Namespaced,categories=cluster-api,shortName=csm
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this CloudStackMachine belongs"
// +kubebuilder:printcolumn:name="InstanceState",type="string",JSONPath=".status.instanceState",description="CloudStack instance state"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Machine ready status"
// +kubebuilder:printcolumn:name="ProviderID",type="string",JSONPath=".spec.providerID",description="CloudStack instance ID"
// +kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns with this CloudStackMachine"

// CloudStackMachine is the Schema for the cloudstackmachines API
type CloudStackMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudStackMachineSpec   `json:"spec,omitempty"`
	Status CloudStackMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CloudStackMachineList contains a list of CloudStackMachine
type CloudStackMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudStackMachine `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &CloudStackMachine{}, &CloudStackMachineList{})
}
