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
	machineryconversion "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *CloudStackIsolatedNetwork) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackIsolatedNetwork)
	if err := Convert_v1beta1_CloudStackIsolatedNetwork_To_v1beta4_CloudStackIsolatedNetwork(src, dst, nil); err != nil {
		return err
	}

	// The v1beta1 isolated-network spec/status are missing fields added in later versions; restore
	// them from the annotation (spec FailureDomainName/Gateway/Netmask/Offering/VPC, status
	// RoutingMode/FirewallRulesOpened).
	restored := &v1beta4.CloudStackIsolatedNetwork{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	if restored.Spec.FailureDomainName != "" {
		dst.Spec.FailureDomainName = restored.Spec.FailureDomainName
	}
	dst.Spec.Gateway = restored.Spec.Gateway
	dst.Spec.Netmask = restored.Spec.Netmask
	dst.Spec.Offering = restored.Spec.Offering
	dst.Spec.VPC = restored.Spec.VPC
	dst.Status.RoutingMode = restored.Status.RoutingMode
	dst.Status.FirewallRulesOpened = restored.Status.FirewallRulesOpened
	return nil
}

func (dst *CloudStackIsolatedNetwork) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackIsolatedNetwork)
	if err := Convert_v1beta4_CloudStackIsolatedNetwork_To_v1beta1_CloudStackIsolatedNetwork(src, dst, nil); err != nil {
		return err
	}

	// Preserve v1beta4 hub data on down-conversion.
	return utilconversion.MarshalData(src, dst)
}

// Convert_v1beta4_CloudStackIsolatedNetworkSpec_To_v1beta1_CloudStackIsolatedNetworkSpec drops the
// spec fields absent from v1beta1; they are restored from the annotation on up-conversion.
func Convert_v1beta4_CloudStackIsolatedNetworkSpec_To_v1beta1_CloudStackIsolatedNetworkSpec(in *v1beta4.CloudStackIsolatedNetworkSpec, out *CloudStackIsolatedNetworkSpec, s machineryconversion.Scope) error { // nolint
	return autoConvert_v1beta4_CloudStackIsolatedNetworkSpec_To_v1beta1_CloudStackIsolatedNetworkSpec(in, out, s)
}

// Convert_v1beta4_CloudStackIsolatedNetworkStatus_To_v1beta1_CloudStackIsolatedNetworkStatus drops
// the status fields absent from v1beta1; they are restored from the annotation on up-conversion.
func Convert_v1beta4_CloudStackIsolatedNetworkStatus_To_v1beta1_CloudStackIsolatedNetworkStatus(in *v1beta4.CloudStackIsolatedNetworkStatus, out *CloudStackIsolatedNetworkStatus, s machineryconversion.Scope) error { // nolint
	return autoConvert_v1beta4_CloudStackIsolatedNetworkStatus_To_v1beta1_CloudStackIsolatedNetworkStatus(in, out, s)
}
