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

package v1beta3

import (
	"sort"

	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //lint:ignore SA1019 spoke pinned to CAPI core/v1beta1 APIEndpoint (see c82a9f9) to preserve pre-v1.13 spoke CRD schema
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// CloudStackCluster helpers

func convertCloudStackClusterToV4(src *CloudStackCluster, dst *v1beta4.CloudStackCluster) error {
	dst.ObjectMeta = src.ObjectMeta
	// Spec — APIEndpoint moved from CAPI v1beta1 (spoke) to v1beta2 (hub); field-copy across the two Go types.
	dst.Spec.ControlPlaneEndpoint = clusterv1beta2.APIEndpoint{
		Host: src.Spec.ControlPlaneEndpoint.Host,
		Port: src.Spec.ControlPlaneEndpoint.Port,
	}
	dst.Spec.SyncWithACS = src.Spec.SyncWithACS
	if src.Spec.FailureDomains != nil {
		dst.Spec.FailureDomains = make([]v1beta4.CloudStackFailureDomainSpec, len(src.Spec.FailureDomains))
		for i, fd := range src.Spec.FailureDomains {
			dst.Spec.FailureDomains[i] = convertFailureDomainSpecToV4(fd)
		}
	}
	// Status: map → slice for FailureDomains
	dst.Status.Ready = src.Status.Ready
	dst.Status.CloudStackClusterID = src.Status.CloudStackClusterID
	if src.Status.FailureDomains != nil {
		dst.Status.FailureDomains = make([]v1beta4.FailureDomain, 0, len(src.Status.FailureDomains))
		// Iterate in sorted name order so the resulting slice is deterministic across
		// conversions (map iteration order in Go is randomized).
		names := make([]string, 0, len(src.Status.FailureDomains))
		for name := range src.Status.FailureDomains {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fdSpec := src.Status.FailureDomains[name]
			dst.Status.FailureDomains = append(dst.Status.FailureDomains, v1beta4.FailureDomain{
				Name:         name,
				ControlPlane: ptr.To(fdSpec.ControlPlane),
				Attributes:   fdSpec.Attributes,
			})
		}
	}
	// Initialization is v1beta4-only; will be nil after up-conversion unless restored from annotation
	return nil
}

func convertCloudStackClusterFromV4(src *v1beta4.CloudStackCluster, dst *CloudStackCluster) error {
	dst.ObjectMeta = src.ObjectMeta
	// Spec — APIEndpoint moved from CAPI v1beta2 (hub) to v1beta1 (spoke); field-copy across the two Go types.
	dst.Spec.ControlPlaneEndpoint = clusterv1beta1.APIEndpoint{
		Host: src.Spec.ControlPlaneEndpoint.Host,
		Port: src.Spec.ControlPlaneEndpoint.Port,
	}
	dst.Spec.SyncWithACS = src.Spec.SyncWithACS
	if src.Spec.FailureDomains != nil {
		dst.Spec.FailureDomains = make([]CloudStackFailureDomainSpec, len(src.Spec.FailureDomains))
		for i, fd := range src.Spec.FailureDomains {
			dst.Spec.FailureDomains[i] = convertFailureDomainSpecFromV4(fd)
		}
	}
	// Status: slice → map for FailureDomains
	dst.Status.Ready = src.Status.Ready
	dst.Status.CloudStackClusterID = src.Status.CloudStackClusterID
	if src.Status.FailureDomains != nil {
		dst.Status.FailureDomains = make(FailureDomains, len(src.Status.FailureDomains))
		for _, fd := range src.Status.FailureDomains {
			cp := false
			if fd.ControlPlane != nil {
				cp = *fd.ControlPlane
			}
			dst.Status.FailureDomains[fd.Name] = FailureDomainSpec{
				ControlPlane: cp,
				Attributes:   fd.Attributes,
			}
		}
	}
	// Initialization dropped (v1beta4-only)
	return nil
}

func convertFailureDomainSpecToV4(src CloudStackFailureDomainSpec) v1beta4.CloudStackFailureDomainSpec {
	return v1beta4.CloudStackFailureDomainSpec{
		Name:        src.Name,
		Account:     src.Account,
		Domain:      src.Domain,
		Project:     src.Project,
		ACSEndpoint: src.ACSEndpoint,
		Zone: v1beta4.CloudStackZoneSpec{
			Name: src.Zone.Name,
			ID:   src.Zone.ID,
			Network: v1beta4.Network{
				ID:          src.Zone.Network.ID,
				Type:        src.Zone.Network.Type,
				Name:        src.Zone.Network.Name,
				Gateway:     src.Zone.Network.Gateway,
				Netmask:     src.Zone.Network.Netmask,
				Offering:    src.Zone.Network.Offering,
				RoutingMode: src.Zone.Network.RoutingMode,
				VPC:         convertVPCToV4(src.Zone.Network.VPC),
			},
		},
	}
}

func convertFailureDomainSpecFromV4(src v1beta4.CloudStackFailureDomainSpec) CloudStackFailureDomainSpec {
	return CloudStackFailureDomainSpec{
		Name:        src.Name,
		Account:     src.Account,
		Domain:      src.Domain,
		Project:     src.Project,
		ACSEndpoint: src.ACSEndpoint,
		Zone: CloudStackZoneSpec{
			Name: src.Zone.Name,
			ID:   src.Zone.ID,
			Network: Network{
				ID:          src.Zone.Network.ID,
				Type:        src.Zone.Network.Type,
				Name:        src.Zone.Network.Name,
				Gateway:     src.Zone.Network.Gateway,
				Netmask:     src.Zone.Network.Netmask,
				Offering:    src.Zone.Network.Offering,
				RoutingMode: src.Zone.Network.RoutingMode,
				VPC:         convertVPCFromV4(src.Zone.Network.VPC),
			},
		},
	}
}

func convertVPCToV4(src *VPC) *v1beta4.VPC {
	if src == nil {
		return nil
	}
	return &v1beta4.VPC{
		ID:       src.ID,
		Name:     src.Name,
		CIDR:     src.CIDR,
		Offering: src.Offering,
	}
}

func convertVPCFromV4(src *v1beta4.VPC) *VPC {
	if src == nil {
		return nil
	}
	return &VPC{
		ID:       src.ID,
		Name:     src.Name,
		CIDR:     src.CIDR,
		Offering: src.Offering,
	}
}

// CloudStackMachine helpers

func convertCloudStackMachineToV4(src *CloudStackMachine, dst *v1beta4.CloudStackMachine) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = v1beta4.CloudStackMachineSpec{
		Name:                 src.Spec.Name,
		ID:                   src.Spec.ID,
		InstanceID:           src.Spec.InstanceID,
		Offering:             v1beta4.CloudStackResourceIdentifier{ID: src.Spec.Offering.ID, Name: src.Spec.Offering.Name},
		Template:             v1beta4.CloudStackResourceIdentifier{ID: src.Spec.Template.ID, Name: src.Spec.Template.Name},
		DiskOffering:         v1beta4.CloudStackResourceDiskOffering{CloudStackResourceIdentifier: v1beta4.CloudStackResourceIdentifier{ID: src.Spec.DiskOffering.ID, Name: src.Spec.DiskOffering.Name}, CustomSize: src.Spec.DiskOffering.CustomSize, MountPath: src.Spec.DiskOffering.MountPath, Device: src.Spec.DiskOffering.Device, Filesystem: src.Spec.DiskOffering.Filesystem, Label: src.Spec.DiskOffering.Label},
		SSHKey:               src.Spec.SSHKey,
		Details:              src.Spec.Details,
		AffinityGroupIDs:     src.Spec.AffinityGroupIDs,
		Affinity:             src.Spec.Affinity,
		AffinityGroupRef:     src.Spec.AffinityGroupRef,
		ProviderID:           src.Spec.ProviderID,
		FailureDomainName:    src.Spec.FailureDomainName,
		UncompressedUserData: src.Spec.UncompressedUserData,
	}
	if src.Spec.Networks != nil {
		dst.Spec.Networks = make([]v1beta4.NetworkSpec, len(src.Spec.Networks))
		for i, n := range src.Spec.Networks {
			dst.Spec.Networks[i] = v1beta4.NetworkSpec{Name: n.Name, IP: n.IP, ID: n.ID}
		}
	}
	dst.Status = v1beta4.CloudStackMachineStatus{
		Addresses:                src.Status.Addresses,
		InstanceState:            src.Status.InstanceState,
		InstanceStateLastUpdated: src.Status.InstanceStateLastUpdated,
		Ready:                    src.Status.Ready,
		Status:                   src.Status.Status,
		Reason:                   src.Status.Reason,
	}
	return nil
}

func convertCloudStackMachineFromV4(src *v1beta4.CloudStackMachine, dst *CloudStackMachine) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = CloudStackMachineSpec{
		Name:                 src.Spec.Name,
		ID:                   src.Spec.ID,
		InstanceID:           src.Spec.InstanceID,
		Offering:             CloudStackResourceIdentifier{ID: src.Spec.Offering.ID, Name: src.Spec.Offering.Name},
		Template:             CloudStackResourceIdentifier{ID: src.Spec.Template.ID, Name: src.Spec.Template.Name},
		DiskOffering:         CloudStackResourceDiskOffering{CloudStackResourceIdentifier: CloudStackResourceIdentifier{ID: src.Spec.DiskOffering.ID, Name: src.Spec.DiskOffering.Name}, CustomSize: src.Spec.DiskOffering.CustomSize, MountPath: src.Spec.DiskOffering.MountPath, Device: src.Spec.DiskOffering.Device, Filesystem: src.Spec.DiskOffering.Filesystem, Label: src.Spec.DiskOffering.Label},
		SSHKey:               src.Spec.SSHKey,
		Details:              src.Spec.Details,
		AffinityGroupIDs:     src.Spec.AffinityGroupIDs,
		Affinity:             src.Spec.Affinity,
		AffinityGroupRef:     src.Spec.AffinityGroupRef,
		ProviderID:           src.Spec.ProviderID,
		FailureDomainName:    src.Spec.FailureDomainName,
		UncompressedUserData: src.Spec.UncompressedUserData,
	}
	if src.Spec.Networks != nil {
		dst.Spec.Networks = make([]NetworkSpec, len(src.Spec.Networks))
		for i, n := range src.Spec.Networks {
			dst.Spec.Networks[i] = NetworkSpec{Name: n.Name, IP: n.IP, ID: n.ID}
		}
	}
	dst.Status = CloudStackMachineStatus{
		Addresses:                src.Status.Addresses,
		InstanceState:            src.Status.InstanceState,
		InstanceStateLastUpdated: src.Status.InstanceStateLastUpdated,
		Ready:                    src.Status.Ready,
		Status:                   src.Status.Status,
		Reason:                   src.Status.Reason,
	}
	// Initialization dropped (v1beta4-only)
	return nil
}

// CloudStackMachineTemplate helpers

func convertCloudStackMachineTemplateToV4(src *CloudStackMachineTemplate, dst *v1beta4.CloudStackMachineTemplate) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Template.ObjectMeta = clusterv1beta2.ObjectMeta{
		Labels:      src.Spec.Template.ObjectMeta.Labels,
		Annotations: src.Spec.Template.ObjectMeta.Annotations,
	}
	tmp := &CloudStackMachine{Spec: src.Spec.Template.Spec}
	dstMachine := &v1beta4.CloudStackMachine{}
	if err := convertCloudStackMachineToV4(tmp, dstMachine); err != nil {
		return err
	}
	dst.Spec.Template.Spec = dstMachine.Spec
	return nil
}

func convertCloudStackMachineTemplateFromV4(src *v1beta4.CloudStackMachineTemplate, dst *CloudStackMachineTemplate) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Template.ObjectMeta = clusterv1beta1.ObjectMeta{
		Labels:      src.Spec.Template.ObjectMeta.Labels,
		Annotations: src.Spec.Template.ObjectMeta.Annotations,
	}
	srcMachine := &v1beta4.CloudStackMachine{Spec: src.Spec.Template.Spec}
	dstMachine := &CloudStackMachine{}
	if err := convertCloudStackMachineFromV4(srcMachine, dstMachine); err != nil {
		return err
	}
	dst.Spec.Template.Spec = dstMachine.Spec
	return nil
}

// CloudStackFailureDomain helpers

func convertCloudStackFailureDomainToV4(src *CloudStackFailureDomain, dst *v1beta4.CloudStackFailureDomain) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = convertFailureDomainSpecToV4(src.Spec)
	dst.Status = v1beta4.CloudStackFailureDomainStatus{Ready: src.Status.Ready}
	return nil
}

func convertCloudStackFailureDomainFromV4(src *v1beta4.CloudStackFailureDomain, dst *CloudStackFailureDomain) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = convertFailureDomainSpecFromV4(src.Spec)
	dst.Status = CloudStackFailureDomainStatus{Ready: src.Status.Ready}
	return nil
}

// CloudStackIsolatedNetwork helpers

func convertCloudStackIsolatedNetworkToV4(src *CloudStackIsolatedNetwork, dst *v1beta4.CloudStackIsolatedNetwork) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = v1beta4.CloudStackIsolatedNetworkSpec{
		Name:                 src.Spec.Name,
		ID:                   src.Spec.ID,
		ControlPlaneEndpoint: clusterv1beta2.APIEndpoint{Host: src.Spec.ControlPlaneEndpoint.Host, Port: src.Spec.ControlPlaneEndpoint.Port},
		FailureDomainName:    src.Spec.FailureDomainName,
		Gateway:              src.Spec.Gateway,
		Netmask:              src.Spec.Netmask,
		Offering:             src.Spec.Offering,
		VPC:                  convertVPCToV4(src.Spec.VPC),
	}
	dst.Status = v1beta4.CloudStackIsolatedNetworkStatus{
		PublicIPID:          src.Status.PublicIPID,
		LBRuleID:            src.Status.LBRuleID,
		RoutingMode:         src.Status.RoutingMode,
		FirewallRulesOpened: src.Status.FirewallRulesOpened,
		Ready:               src.Status.Ready,
	}
	return nil
}

func convertCloudStackIsolatedNetworkFromV4(src *v1beta4.CloudStackIsolatedNetwork, dst *CloudStackIsolatedNetwork) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = CloudStackIsolatedNetworkSpec{
		Name:                 src.Spec.Name,
		ID:                   src.Spec.ID,
		ControlPlaneEndpoint: clusterv1beta1.APIEndpoint{Host: src.Spec.ControlPlaneEndpoint.Host, Port: src.Spec.ControlPlaneEndpoint.Port},
		FailureDomainName:    src.Spec.FailureDomainName,
		Gateway:              src.Spec.Gateway,
		Netmask:              src.Spec.Netmask,
		Offering:             src.Spec.Offering,
		VPC:                  convertVPCFromV4(src.Spec.VPC),
	}
	dst.Status = CloudStackIsolatedNetworkStatus{
		PublicIPID:          src.Status.PublicIPID,
		LBRuleID:            src.Status.LBRuleID,
		RoutingMode:         src.Status.RoutingMode,
		FirewallRulesOpened: src.Status.FirewallRulesOpened,
		Ready:               src.Status.Ready,
	}
	return nil
}

// CloudStackAffinityGroup helpers

func convertCloudStackAffinityGroupToV4(src *CloudStackAffinityGroup, dst *v1beta4.CloudStackAffinityGroup) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = v1beta4.CloudStackAffinityGroupSpec{
		Type:              src.Spec.Type,
		Name:              src.Spec.Name,
		ID:                src.Spec.ID,
		FailureDomainName: src.Spec.FailureDomainName,
	}
	dst.Status = v1beta4.CloudStackAffinityGroupStatus{Ready: src.Status.Ready}
	return nil
}

func convertCloudStackAffinityGroupFromV4(src *v1beta4.CloudStackAffinityGroup, dst *CloudStackAffinityGroup) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = CloudStackAffinityGroupSpec{
		Type:              src.Spec.Type,
		Name:              src.Spec.Name,
		ID:                src.Spec.ID,
		FailureDomainName: src.Spec.FailureDomainName,
	}
	dst.Status = CloudStackAffinityGroupStatus{Ready: src.Status.Ready}
	return nil
}

// CloudStackMachineStateChecker helpers

func convertCloudStackMachineStateCheckerToV4(src *CloudStackMachineStateChecker, dst *v1beta4.CloudStackMachineStateChecker) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = v1beta4.CloudStackMachineStateCheckerSpec{InstanceID: src.Spec.InstanceID}
	dst.Status = v1beta4.CloudStackMachineStateCheckerStatus{Ready: src.Status.Ready}
	return nil
}

func convertCloudStackMachineStateCheckerFromV4(src *v1beta4.CloudStackMachineStateChecker, dst *CloudStackMachineStateChecker) error {
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = CloudStackMachineStateCheckerSpec{InstanceID: src.Spec.InstanceID}
	dst.Status = CloudStackMachineStateCheckerStatus{Ready: src.Status.Ready}
	return nil
}

// ConvertToV4/FromV4 are exported helpers for use by v1beta1/v2 spoke packages.

func ConvertCloudStackClusterToV4(src *CloudStackCluster, dst *v1beta4.CloudStackCluster) error {
	return convertCloudStackClusterToV4(src, dst)
}

func ConvertCloudStackClusterFromV4(src *v1beta4.CloudStackCluster, dst *CloudStackCluster) error {
	return convertCloudStackClusterFromV4(src, dst)
}

func ConvertCloudStackMachineToV4(src *CloudStackMachine, dst *v1beta4.CloudStackMachine) error {
	return convertCloudStackMachineToV4(src, dst)
}

func ConvertCloudStackMachineFromV4(src *v1beta4.CloudStackMachine, dst *CloudStackMachine) error {
	return convertCloudStackMachineFromV4(src, dst)
}

func ConvertCloudStackMachineTemplateToV4(src *CloudStackMachineTemplate, dst *v1beta4.CloudStackMachineTemplate) error {
	return convertCloudStackMachineTemplateToV4(src, dst)
}

func ConvertCloudStackMachineTemplateFromV4(src *v1beta4.CloudStackMachineTemplate, dst *CloudStackMachineTemplate) error {
	return convertCloudStackMachineTemplateFromV4(src, dst)
}

func ConvertCloudStackFailureDomainToV4(src *CloudStackFailureDomain, dst *v1beta4.CloudStackFailureDomain) error {
	return convertCloudStackFailureDomainToV4(src, dst)
}

func ConvertCloudStackFailureDomainFromV4(src *v1beta4.CloudStackFailureDomain, dst *CloudStackFailureDomain) error {
	return convertCloudStackFailureDomainFromV4(src, dst)
}

func ConvertCloudStackIsolatedNetworkToV4(src *CloudStackIsolatedNetwork, dst *v1beta4.CloudStackIsolatedNetwork) error {
	return convertCloudStackIsolatedNetworkToV4(src, dst)
}

func ConvertCloudStackIsolatedNetworkFromV4(src *v1beta4.CloudStackIsolatedNetwork, dst *CloudStackIsolatedNetwork) error {
	return convertCloudStackIsolatedNetworkFromV4(src, dst)
}

func ConvertCloudStackAffinityGroupToV4(src *CloudStackAffinityGroup, dst *v1beta4.CloudStackAffinityGroup) error {
	return convertCloudStackAffinityGroupToV4(src, dst)
}

func ConvertCloudStackAffinityGroupFromV4(src *v1beta4.CloudStackAffinityGroup, dst *CloudStackAffinityGroup) error {
	return convertCloudStackAffinityGroupFromV4(src, dst)
}

func ConvertCloudStackMachineStateCheckerToV4(src *CloudStackMachineStateChecker, dst *v1beta4.CloudStackMachineStateChecker) error {
	return convertCloudStackMachineStateCheckerToV4(src, dst)
}

func ConvertCloudStackMachineStateCheckerFromV4(src *v1beta4.CloudStackMachineStateChecker, dst *CloudStackMachineStateChecker) error {
	return convertCloudStackMachineStateCheckerFromV4(src, dst)
}
