/*
Copyright 2023 The Kubernetes Authors.

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
	machineryconversion "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *CloudStackFailureDomain) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackFailureDomain)
	if err := Convert_v1beta2_CloudStackFailureDomain_To_v1beta4_CloudStackFailureDomain(src, dst, nil); err != nil {
		return err
	}

	// The v1beta2 failure-domain spec is missing fields added in later versions
	// (Project, network Gateway/Netmask/Offering/RoutingMode/VPC); restore them from the annotation.
	restored := &v1beta4.CloudStackFailureDomain{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	restoreFailureDomainHubFields(&dst.Spec, &restored.Spec)
	return nil
}

func (dst *CloudStackFailureDomain) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackFailureDomain)
	if err := Convert_v1beta4_CloudStackFailureDomain_To_v1beta2_CloudStackFailureDomain(src, dst, nil); err != nil {
		return err
	}

	// Preserve v1beta4 hub data on down-conversion.
	return utilconversion.MarshalData(src, dst)
}

// restoreFailureDomainHubFields copies the v1beta4-only failure-domain spec fields (Project and the
// extended network attributes) from restored into dst. v1beta2 cannot represent these fields, so
// they must be recovered from the conversion-data annotation to keep hub round-trips lossless.
func restoreFailureDomainHubFields(dst, restored *v1beta4.CloudStackFailureDomainSpec) {
	dst.Project = restored.Project
	dst.Zone.Network.Gateway = restored.Zone.Network.Gateway
	dst.Zone.Network.Netmask = restored.Zone.Network.Netmask
	dst.Zone.Network.Offering = restored.Zone.Network.Offering
	dst.Zone.Network.RoutingMode = restored.Zone.Network.RoutingMode
	dst.Zone.Network.VPC = restored.Zone.Network.VPC
}

// Convert_v1beta4_CloudStackFailureDomainSpec_To_v1beta2_CloudStackFailureDomainSpec drops the
// v1beta4-only Project field; it is restored from the annotation on up-conversion.
func Convert_v1beta4_CloudStackFailureDomainSpec_To_v1beta2_CloudStackFailureDomainSpec(in *v1beta4.CloudStackFailureDomainSpec, out *CloudStackFailureDomainSpec, s machineryconversion.Scope) error { // nolint
	return autoConvert_v1beta4_CloudStackFailureDomainSpec_To_v1beta2_CloudStackFailureDomainSpec(in, out, s)
}

// Convert_v1beta4_Network_To_v1beta2_Network drops the network fields absent from v1beta2
// (Gateway, Netmask, Offering, RoutingMode, VPC); they are restored from the annotation.
func Convert_v1beta4_Network_To_v1beta2_Network(in *v1beta4.Network, out *Network, s machineryconversion.Scope) error { // nolint
	return autoConvert_v1beta4_Network_To_v1beta2_Network(in, out, s)
}
