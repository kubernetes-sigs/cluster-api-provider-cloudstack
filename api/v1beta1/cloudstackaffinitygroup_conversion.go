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
	"context"
	conv "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *CloudStackAffinityGroup) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta2.CloudStackAffinityGroup)
	return Convert_v1beta1_CloudStackAffinityGroup_To_v1beta2_CloudStackAffinityGroup(src, dst, nil)
}

func (dst *CloudStackAffinityGroup) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta2.CloudStackAffinityGroup)
	return Convert_v1beta2_CloudStackAffinityGroup_To_v1beta1_CloudStackAffinityGroup(src, dst, nil)
}

//nolint:golint,revive,stylecheck
func Convert_v1beta1_CloudStackAffinityGroup_To_v1beta2_CloudStackAffinityGroup(in *CloudStackAffinityGroup, out *v1beta2.CloudStackAffinityGroup, s conv.Scope) error {
	out.ObjectMeta = in.ObjectMeta

	csCluster := &v1beta2.CloudStackCluster{}
	clusterName := in.ObjectMeta.Labels["cluster.x-k8s.io/cluster-name"]
	namespace := in.ObjectMeta.Namespace
	key := client.ObjectKey{Name: clusterName, Namespace: namespace}
	if err := v1beta2.K8sClient.Get(context.TODO(), key, csCluster); err != nil {
		return err
	}

	out.Spec = v1beta2.CloudStackAffinityGroupSpec{
		Type:              in.Spec.Type,
		Name:              in.Spec.Name,
		ID:                in.Spec.ID,
		FailureDomainName: csCluster.Spec.FailureDomains[0].Name,
	}

	out.Status = v1beta2.CloudStackAffinityGroupStatus{
		Ready: in.Status.Ready,
	}
	return nil
}

//nolint:golint,revive,stylecheck
func Convert_v1beta2_CloudStackAffinityGroup_To_v1beta1_CloudStackAffinityGroup(in *v1beta2.CloudStackAffinityGroup, out *CloudStackAffinityGroup, s conv.Scope) error {
	out.ObjectMeta = in.ObjectMeta

	out.Spec = CloudStackAffinityGroupSpec{
		Type: in.Spec.Type,
		Name: in.Spec.Name,
		ID:   in.Spec.ID,
	}

	out.Status = CloudStackAffinityGroupStatus{
		Ready: in.Status.Ready,
	}
	return nil
}
