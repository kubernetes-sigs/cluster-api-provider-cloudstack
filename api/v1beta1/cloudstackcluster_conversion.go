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

package v1beta1

import (
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *CloudStackCluster) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackCluster)
	if err := Convert_v1beta1_CloudStackCluster_To_v1beta4_CloudStackCluster(src, dst, nil); err != nil {
		return err
	}

	// v1beta1 models failure domains as Zones and cannot represent the full v1beta4 failure-domain
	// spec, the hub status failure-domain list, SyncWithACS, CloudStackClusterID, or Initialization.
	// Restore them from the conversion annotation.
	restored := &v1beta4.CloudStackCluster{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	if restored.Spec.FailureDomains != nil {
		dst.Spec.FailureDomains = restored.Spec.FailureDomains
	}
	if restored.Spec.SyncWithACS != nil {
		dst.Spec.SyncWithACS = restored.Spec.SyncWithACS
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
	if err := Convert_v1beta4_CloudStackCluster_To_v1beta1_CloudStackCluster(src, dst, nil); err != nil {
		return err
	}

	// Preserve v1beta4 hub data on down-conversion.
	return utilconversion.MarshalData(src, dst)
}
