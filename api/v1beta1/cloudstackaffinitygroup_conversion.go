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
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *CloudStackAffinityGroup) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta3.CloudStackAffinityGroup)
	if err := Convert_v1beta1_CloudStackAffinityGroup_To_v1beta3_CloudStackAffinityGroup(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data
	restored := &v1beta3.CloudStackAffinityGroup{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	if restored.Spec.FailureDomainName != "" {
		dst.Spec.FailureDomainName = restored.Spec.FailureDomainName
	}
	return nil
}

func (dst *CloudStackAffinityGroup) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta3.CloudStackAffinityGroup)
	if err := Convert_v1beta3_CloudStackAffinityGroup_To_v1beta1_CloudStackAffinityGroup(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion
	if err := utilconversion.MarshalData(src, dst); err != nil {
		return err
	}
	return nil
}

func Convert_v1beta3_CloudStackAffinityGroupSpec_To_v1beta1_CloudStackAffinityGroupSpec(in *v1beta3.CloudStackAffinityGroupSpec, out *CloudStackAffinityGroupSpec, s machineryconversion.Scope) error { // nolint
	return autoConvert_v1beta3_CloudStackAffinityGroupSpec_To_v1beta1_CloudStackAffinityGroupSpec(in, out, s)
}
