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
	"fmt"
	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	conv "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"sigs.k8s.io/cluster-api/api/v1beta1"
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
	zoneID, zoneName, err := getAffinityGroupZone(in.ObjectMeta.Namespace, in.Spec.ID)
	if err != nil {
		return err
	}
	failureDomainName, err := GetDefaultFailureDomainName(in.ObjectMeta.Namespace, in.ObjectMeta.Labels[v1beta1.ClusterLabelName], zoneID, zoneName)
	if err != nil {
		return err
	}
	out.Spec = v1beta2.CloudStackAffinityGroupSpec{
		Type:              in.Spec.Type,
		Name:              in.Spec.Name,
		ID:                in.Spec.ID,
		FailureDomainName: failureDomainName,
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

func getAffinityGroupZone(namespace string, affinityGroupID string) (zoneID string, zoneName string, err error) {
	config, err := GetCloudStackConfig(namespace)
	if err != nil {
		return "", "", err
	}
	endpointCredentials := &corev1.Secret{}
	key := client.ObjectKey{Name: DefaultEndpointCredential, Namespace: namespace}
	if err := v1beta2.K8sClient.Get(context.TODO(), key, endpointCredentials); err != nil {
		return "", "", err
	}

	csClient := cloudstack.NewAsyncClient(fmt.Sprint(config["api-url"]), fmt.Sprint(config["api-key"]), fmt.Sprint(config["secret-key"]), fmt.Sprint(config["verify-ssl"]) == "true")

	parameters := csClient.VirtualMachine.NewListVirtualMachinesParams()
	parameters.SetAffinitygroupid(affinityGroupID)
	listVMResponse, err := csClient.VirtualMachine.ListVirtualMachines(parameters)
	if err != nil {
		return "", "", err
	}
	for _, vm := range listVMResponse.VirtualMachines {
		for _, ag := range vm.Affinitygroup {
			if ag.Id == affinityGroupID {
				return vm.Zoneid, vm.Zonename, nil
			}
		}
	}

	return "", "", errors.Errorf("zone id and name could not found for affinity group id %s", affinityGroupID)
}
