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

func (src *CloudStackMachineTemplate) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta3.CloudStackMachineTemplate)
	if err := Convert_v1beta1_CloudStackMachineTemplate_To_v1beta3_CloudStackMachineTemplate(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data
	restored := &v1beta3.CloudStackMachineTemplate{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	if restored.Spec.Template.FailureDomainName != "" {
		dst.Spec.Template.FailureDomainName = restored.Spec.Template.FailureDomainName
	}
	if restored.Spec.Template.UncompressedUserData != nil {
		dst.Spec.Template.UncompressedUserData = restored.Spec.Template.UncompressedUserData
	}
	return nil
}

func (dst *CloudStackMachineTemplate) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta3.CloudStackMachineTemplate)
	return Convert_v1beta3_CloudStackMachineTemplate_To_v1beta1_CloudStackMachineTemplate(src, dst, nil)
}

func Convert_v1beta1_CloudStackMachineTemplateSpec_To_v1beta3_CloudStackMachineTemplateSpec(in *CloudStackMachineTemplateSpec, out *v1beta3.CloudStackMachineTemplateSpec, s machineryconversion.Scope) error { // nolint
	return autoConvert_v1beta1_CloudStackMachineSpec_To_v1beta3_CloudStackMachineSpec(&in.Spec.Spec, &out.Template, s)
}

func Convert_v1beta3_CloudStackMachineTemplateSpec_To_v1beta1_CloudStackMachineTemplateSpec(in *v1beta3.CloudStackMachineTemplateSpec, out *CloudStackMachineTemplateSpec, s machineryconversion.Scope) error { // nolint
	return autoConvert_v1beta3_CloudStackMachineSpec_To_v1beta1_CloudStackMachineSpec(&in.Template, &out.Spec.Spec, s)
}
