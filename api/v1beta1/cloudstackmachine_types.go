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
	"fmt"
	"strings"

	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// The presence of a finalizer prevents CAPI from deleting the corresponding CAPI data.
	MachineFinalizer = "cloudstackmachine.infrastructure.cluster.x-k8s.io"
)

// CloudStackMachineSpec defines the desired state of CloudStackMachine
type CloudStackMachineSpec struct {
	// Instance ID. Should only be useful to modify an existing instance.
	InstanceID *string `json:"instanceID,omitempty"`

	// CloudStack compute offering.
	Offering CloudStackResourceIdentifier `json:"offering"`

	// CloudStack template to use.
	Template CloudStackResourceIdentifier `json:"template"`

	// CloudStack disk offering to use.
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

	// The CS specific unique identifier. Of the form: fmt.Sprintf("cloudstack:///%s", CS Machine ID)
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	// Optionally settable Zone ID to land the machine in.
	ZoneID string `json:"zoneID,omitempty"`

	// Optionally settable Zone Name to land the machine in.
	ZoneName string `json:"zoneName,omitempty"`

	// IdentityRef is a reference to a identity to be used when reconciling this cluster
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
	MountPath                    string `json:"mountPath"`
}

// TODO: Review the use of this field/type.
type InstanceState string

// Type pulled mostly from the CloudStack API.
type CloudStackMachineStatus struct {
	// Zone ID is used so that the zone can be computed once per reconcile and then propagate.
	// +optional
	ZoneID string `json:"zoneID,omitempty"`

	// Addresses contains a CloudStack VM instance's IP addresses.
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// InstanceState is the state of the CloudStack instance for this machine.
	// +optional
	InstanceState InstanceState `json:"instanceState,omitempty"`

	// Ready indicates the readiness of the provider resource.
	Ready bool `json:"ready"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=cloudstackmachines,scope=Namespaced,categories=cluster-api,shortName=csm
// +kubebuilder:storageversion
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

// The computed affinity group name relevant to this machine.
func (csm CloudStackMachine) AffinityGroupName(
	capiMachine *capiv1.Machine,
) (string, error) {

	managerOwnerRef := csCtrlrUtils.GetManagementOwnerRef(capiMachine)
	if managerOwnerRef == nil {
		return "", errors.Errorf("could not find owner UID for %s/%s", csm.Namespace, csm.Name)
	}
	return fmt.Sprintf("%sAffinity-%s-%s", strings.Title(csm.Spec.Affinity), managerOwnerRef.Name, managerOwnerRef.UID), nil
}

//+kubebuilder:object:root=true

// CloudStackMachineList contains a list of CloudStackMachine
type CloudStackMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudStackMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudStackMachine{}, &CloudStackMachineList{})
}
