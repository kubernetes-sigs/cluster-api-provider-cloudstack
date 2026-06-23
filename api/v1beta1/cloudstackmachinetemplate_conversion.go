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
	machineryconversion "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *CloudStackMachineTemplate) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackMachineTemplate)
	if err := Convert_v1beta1_CloudStackMachineTemplate_To_v1beta4_CloudStackMachineTemplate(src, dst, nil); err != nil {
		return err
	}

	// Restore the template machine-spec fields absent from v1beta1 from the annotation.
	restored := &v1beta4.CloudStackMachineTemplate{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	if len(restored.Spec.Template.Spec.Networks) > 0 {
		dst.Spec.Template.Spec.Networks = restored.Spec.Template.Spec.Networks
	}
	if restored.Spec.Template.Spec.FailureDomainName != "" {
		dst.Spec.Template.Spec.FailureDomainName = restored.Spec.Template.Spec.FailureDomainName
	}
	if restored.Spec.Template.Spec.UncompressedUserData != nil {
		dst.Spec.Template.Spec.UncompressedUserData = restored.Spec.Template.Spec.UncompressedUserData
	}
	return nil
}

func (dst *CloudStackMachineTemplate) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackMachineTemplate)
	if err := Convert_v1beta4_CloudStackMachineTemplate_To_v1beta1_CloudStackMachineTemplate(src, dst, nil); err != nil {
		return err
	}

	// Preserve v1beta4 hub data on down-conversion.
	return utilconversion.MarshalData(src, dst)
}

// Convert_v1beta1_CloudStackMachineTemplateSpec_To_v1beta4_CloudStackMachineTemplateSpec maps the
// v1beta1 `Spec` field to the v1beta4 `Template` field (the field was renamed between versions).
func Convert_v1beta1_CloudStackMachineTemplateSpec_To_v1beta4_CloudStackMachineTemplateSpec(in *CloudStackMachineTemplateSpec, out *v1beta4.CloudStackMachineTemplateSpec, s machineryconversion.Scope) error { // nolint
	return Convert_v1beta1_CloudStackMachineTemplateResource_To_v1beta4_CloudStackMachineTemplateResource(&in.Spec, &out.Template, s)
}

// Convert_v1beta4_CloudStackMachineTemplateSpec_To_v1beta1_CloudStackMachineTemplateSpec maps the
// v1beta4 `Template` field back to the v1beta1 `Spec` field.
func Convert_v1beta4_CloudStackMachineTemplateSpec_To_v1beta1_CloudStackMachineTemplateSpec(in *v1beta4.CloudStackMachineTemplateSpec, out *CloudStackMachineTemplateSpec, s machineryconversion.Scope) error { // nolint
	return Convert_v1beta4_CloudStackMachineTemplateResource_To_v1beta1_CloudStackMachineTemplateResource(&in.Template, &out.Spec, s)
}

// Convert_v1beta2_ObjectMeta_To_v1_ObjectMeta converts the trimmed CAPI ObjectMeta (labels and
// annotations only) to a full metav1.ObjectMeta. Other metav1.ObjectMeta fields are not
// representable in the CAPI contract metadata and are intentionally not round-tripped.
func Convert_v1beta2_ObjectMeta_To_v1_ObjectMeta(in *clusterv1.ObjectMeta, out *metav1.ObjectMeta, s machineryconversion.Scope) error { // nolint
	if in.Annotations != nil {
		out.Annotations = in.Annotations
	}
	if in.Labels != nil {
		out.Labels = in.Labels
	}
	return nil
}

// Convert_v1_ObjectMeta_To_v1beta2_ObjectMeta converts a full metav1.ObjectMeta to the trimmed CAPI
// ObjectMeta, keeping only labels and annotations.
func Convert_v1_ObjectMeta_To_v1beta2_ObjectMeta(in *metav1.ObjectMeta, out *clusterv1.ObjectMeta, s machineryconversion.Scope) error { // nolint
	if in.Annotations != nil {
		out.Annotations = in.Annotations
	}
	if in.Labels != nil {
		out.Labels = in.Labels
	}
	return nil
}
