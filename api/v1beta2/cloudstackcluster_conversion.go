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
	"sort"

	machineryconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *CloudStackCluster) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackCluster)
	if err := Convert_v1beta2_CloudStackCluster_To_v1beta4_CloudStackCluster(src, dst, nil); err != nil {
		return err
	}

	// Restore v1beta4-only data that v1beta2 cannot represent, from the conversion annotation.
	restored := &v1beta4.CloudStackCluster{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	if restored.Spec.SyncWithACS != nil {
		dst.Spec.SyncWithACS = restored.Spec.SyncWithACS
	}
	// Failure-domain names ARE representable in v1beta2, so restore only the missing per-domain
	// fields by matching name (avoids clobbering v1beta2-side edits).
	if len(restored.Spec.FailureDomains) > 0 {
		restoredByName := make(map[string]v1beta4.CloudStackFailureDomainSpec, len(restored.Spec.FailureDomains))
		for _, fd := range restored.Spec.FailureDomains {
			restoredByName[fd.Name] = fd
		}
		for i := range dst.Spec.FailureDomains {
			if r, ok := restoredByName[dst.Spec.FailureDomains[i].Name]; ok {
				restoreFailureDomainHubFields(&dst.Spec.FailureDomains[i], &r)
			}
		}
	}
	if restored.Status.FailureDomains != nil {
		dst.Status.FailureDomains = restored.Status.FailureDomains
	}
	if restored.Status.CloudStackClusterID != "" {
		dst.Status.CloudStackClusterID = restored.Status.CloudStackClusterID
	}
	if restored.Status.Initialization != nil {
		dst.Status.Initialization = restored.Status.Initialization
	}
	return nil
}

func (dst *CloudStackCluster) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackCluster)
	if err := Convert_v1beta4_CloudStackCluster_To_v1beta2_CloudStackCluster(src, dst, nil); err != nil {
		return err
	}

	// Preserve v1beta4 hub data on down-conversion.
	return utilconversion.MarshalData(src, dst)
}

// Convert_v1beta4_CloudStackClusterSpec_To_v1beta2_CloudStackClusterSpec drops the v1beta4-only
// SyncWithACS field; it is restored from the annotation on up-conversion.
func Convert_v1beta4_CloudStackClusterSpec_To_v1beta2_CloudStackClusterSpec(in *v1beta4.CloudStackClusterSpec, out *CloudStackClusterSpec, s machineryconversion.Scope) error { // nolint
	return autoConvert_v1beta4_CloudStackClusterSpec_To_v1beta2_CloudStackClusterSpec(in, out, s)
}

// Convert_v1beta2_CloudStackClusterStatus_To_v1beta4_CloudStackClusterStatus converts the
// v1beta2 failure-domain map to the v1beta4 slice, iterating in sorted name order so the result
// is deterministic.
func Convert_v1beta2_CloudStackClusterStatus_To_v1beta4_CloudStackClusterStatus(in *CloudStackClusterStatus, out *v1beta4.CloudStackClusterStatus, s machineryconversion.Scope) error { // nolint
	if err := autoConvert_v1beta2_CloudStackClusterStatus_To_v1beta4_CloudStackClusterStatus(in, out, s); err != nil {
		return err
	}
	if in.FailureDomains != nil {
		out.FailureDomains = make([]v1beta4.FailureDomain, 0, len(in.FailureDomains))
		names := make([]string, 0, len(in.FailureDomains))
		for name := range in.FailureDomains {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fd := in.FailureDomains[name]
			out.FailureDomains = append(out.FailureDomains, v1beta4.FailureDomain{
				Name:         name,
				ControlPlane: ptr.To(fd.ControlPlane),
				Attributes:   fd.Attributes,
			})
		}
	}
	return nil
}

// Convert_v1beta4_CloudStackClusterStatus_To_v1beta2_CloudStackClusterStatus converts the
// v1beta4 failure-domain slice to the v1beta2 map and drops the v1beta4-only CloudStackClusterID
// and Initialization fields (restored from the annotation on up-conversion).
func Convert_v1beta4_CloudStackClusterStatus_To_v1beta2_CloudStackClusterStatus(in *v1beta4.CloudStackClusterStatus, out *CloudStackClusterStatus, s machineryconversion.Scope) error { // nolint
	if err := autoConvert_v1beta4_CloudStackClusterStatus_To_v1beta2_CloudStackClusterStatus(in, out, s); err != nil {
		return err
	}
	if in.FailureDomains != nil {
		out.FailureDomains = make(FailureDomains, len(in.FailureDomains))
		for _, fd := range in.FailureDomains {
			cp := false
			if fd.ControlPlane != nil {
				cp = *fd.ControlPlane
			}
			out.FailureDomains[fd.Name] = FailureDomainSpec{ControlPlane: cp, Attributes: fd.Attributes}
		}
	}
	return nil
}
