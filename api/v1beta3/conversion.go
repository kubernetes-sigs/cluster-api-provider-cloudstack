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
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// CloudStackCluster

func (src *CloudStackCluster) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackCluster)
	if err := convertCloudStackClusterToV4(src, dst); err != nil {
		return err
	}
	// Restore v1beta4-only fields that were preserved as annotations on v1beta3.
	restored := &v1beta4.CloudStackCluster{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	if restored.Status.Initialization != nil {
		dst.Status.Initialization = restored.Status.Initialization
	}
	// Status.FailureDomains[].ControlPlane is *bool in the hub but bool in the spoke;
	// the down-conversion collapses nil→false, so restore the original hub values.
	if restored.Status.FailureDomains != nil {
		dst.Status.FailureDomains = restored.Status.FailureDomains
	}
	return nil
}

func (dst *CloudStackCluster) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackCluster)
	if err := convertCloudStackClusterFromV4(src, dst); err != nil {
		return err
	}
	// Preserve v1beta4-only fields as annotations on v1beta3
	return utilconversion.MarshalData(src, dst)
}

func (src *CloudStackClusterList) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackClusterList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := v1beta4.CloudStackCluster{}
		if err := convertCloudStackClusterToV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

func (dst *CloudStackClusterList) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackClusterList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := CloudStackCluster{}
		if err := convertCloudStackClusterFromV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

// CloudStackMachine

func (src *CloudStackMachine) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackMachine)
	if err := convertCloudStackMachineToV4(src, dst); err != nil {
		return err
	}
	// Restore v1beta4-only fields that were preserved as annotations on v1beta3
	restored := &v1beta4.CloudStackMachine{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	if restored.Status.Initialization != nil {
		dst.Status.Initialization = restored.Status.Initialization
	}
	return nil
}

func (dst *CloudStackMachine) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackMachine)
	if err := convertCloudStackMachineFromV4(src, dst); err != nil {
		return err
	}
	// Preserve v1beta4-only fields as annotations on v1beta3
	return utilconversion.MarshalData(src, dst)
}

func (src *CloudStackMachineList) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackMachineList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := v1beta4.CloudStackMachine{}
		if err := convertCloudStackMachineToV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

func (dst *CloudStackMachineList) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackMachineList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := CloudStackMachine{}
		if err := convertCloudStackMachineFromV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

// CloudStackMachineTemplate

func (src *CloudStackMachineTemplate) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackMachineTemplate)
	return convertCloudStackMachineTemplateToV4(src, dst)
}

func (dst *CloudStackMachineTemplate) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackMachineTemplate)
	return convertCloudStackMachineTemplateFromV4(src, dst)
}

func (src *CloudStackMachineTemplateList) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackMachineTemplateList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := v1beta4.CloudStackMachineTemplate{}
		if err := convertCloudStackMachineTemplateToV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

func (dst *CloudStackMachineTemplateList) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackMachineTemplateList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := CloudStackMachineTemplate{}
		if err := convertCloudStackMachineTemplateFromV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

// CloudStackFailureDomain

func (src *CloudStackFailureDomain) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackFailureDomain)
	return convertCloudStackFailureDomainToV4(src, dst)
}

func (dst *CloudStackFailureDomain) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackFailureDomain)
	return convertCloudStackFailureDomainFromV4(src, dst)
}

func (src *CloudStackFailureDomainList) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackFailureDomainList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := v1beta4.CloudStackFailureDomain{}
		if err := convertCloudStackFailureDomainToV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

func (dst *CloudStackFailureDomainList) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackFailureDomainList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := CloudStackFailureDomain{}
		if err := convertCloudStackFailureDomainFromV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

// CloudStackIsolatedNetwork

func (src *CloudStackIsolatedNetwork) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackIsolatedNetwork)
	return convertCloudStackIsolatedNetworkToV4(src, dst)
}

func (dst *CloudStackIsolatedNetwork) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackIsolatedNetwork)
	return convertCloudStackIsolatedNetworkFromV4(src, dst)
}

func (src *CloudStackIsolatedNetworkList) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackIsolatedNetworkList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := v1beta4.CloudStackIsolatedNetwork{}
		if err := convertCloudStackIsolatedNetworkToV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

func (dst *CloudStackIsolatedNetworkList) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackIsolatedNetworkList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := CloudStackIsolatedNetwork{}
		if err := convertCloudStackIsolatedNetworkFromV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

// CloudStackAffinityGroup

func (src *CloudStackAffinityGroup) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackAffinityGroup)
	return convertCloudStackAffinityGroupToV4(src, dst)
}

func (dst *CloudStackAffinityGroup) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackAffinityGroup)
	return convertCloudStackAffinityGroupFromV4(src, dst)
}

func (src *CloudStackAffinityGroupList) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackAffinityGroupList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := v1beta4.CloudStackAffinityGroup{}
		if err := convertCloudStackAffinityGroupToV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

func (dst *CloudStackAffinityGroupList) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackAffinityGroupList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := CloudStackAffinityGroup{}
		if err := convertCloudStackAffinityGroupFromV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

// CloudStackMachineStateChecker

func (src *CloudStackMachineStateChecker) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackMachineStateChecker)
	return convertCloudStackMachineStateCheckerToV4(src, dst)
}

func (dst *CloudStackMachineStateChecker) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackMachineStateChecker)
	return convertCloudStackMachineStateCheckerFromV4(src, dst)
}

func (src *CloudStackMachineStateCheckerList) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta4.CloudStackMachineStateCheckerList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := v1beta4.CloudStackMachineStateChecker{}
		if err := convertCloudStackMachineStateCheckerToV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}

func (dst *CloudStackMachineStateCheckerList) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta4.CloudStackMachineStateCheckerList)
	dst.ListMeta = src.ListMeta
	for i := range src.Items {
		dstItem := CloudStackMachineStateChecker{}
		if err := convertCloudStackMachineStateCheckerFromV4(&src.Items[i], &dstItem); err != nil {
			return err
		}
		dst.Items = append(dst.Items, dstItem)
	}
	return nil
}
