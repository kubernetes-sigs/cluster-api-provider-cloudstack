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
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *CloudStackMachine) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta3.CloudStackMachine)
	if err := Convert_v1beta2_CloudStackMachine_To_v1beta3_CloudStackMachine(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data
	restored := &v1beta3.CloudStackMachine{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}

	if len(restored.Spec.Networks) > 0 {
		dst.Spec.Networks = restored.Spec.Networks
	}

	return nil
}

func (dst *CloudStackMachine) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta3.CloudStackMachine)
	if err := Convert_v1beta3_CloudStackMachine_To_v1beta2_CloudStackMachine(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion, including Networks field
	err := utilconversion.MarshalData(src, dst)
	return err
}

// Convert_v1beta3_CloudStackMachineSpec_To_v1beta2_CloudStackMachineSpec handles the conversion from v1beta3 to v1beta2,
// ignoring the Networks field that doesn't exist in v1beta2
func Convert_v1beta3_CloudStackMachineSpec_To_v1beta2_CloudStackMachineSpec(in *v1beta3.CloudStackMachineSpec, out *CloudStackMachineSpec, s machineryconversion.Scope) error { // nolint
	// Use the auto-generated conversion function, which will handle all fields except Networks
	return autoConvert_v1beta3_CloudStackMachineSpec_To_v1beta2_CloudStackMachineSpec(in, out, s)
}
