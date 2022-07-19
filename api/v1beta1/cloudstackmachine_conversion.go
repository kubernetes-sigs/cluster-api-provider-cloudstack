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
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

const CloudstackMachineAnnotationPrefix string = "cloudstackmachine.infrastructure.cluster.x-k8s.io/"

func (src *CloudStackMachine) ConvertTo(dstRaw conversion.Hub) error { // nolint
	dst := dstRaw.(*v1beta2.CloudStackMachine)
	err := Convert_v1beta1_CloudStackMachine_To_v1beta2_CloudStackMachine(src, dst, nil)
	if err != nil {
		return err
	}
	// need to save zoneId and zoneName to v1beta2s annotation in metadata
	dst.ObjectMeta.Annotations[CloudstackMachineAnnotationPrefix+"zoneid"] = src.Spec.ZoneID
	dst.ObjectMeta.Annotations[CloudstackMachineAnnotationPrefix+"zonename"] = src.Spec.ZoneName

	// need to save failuredomainname in v1beta1's annotation to v1beta2's spec
	if len(src.ObjectMeta.Annotations[CloudstackMachineAnnotationPrefix+"failuredomainname"]) > 0 {
		dst.Spec.FailureDomainName = src.ObjectMeta.Annotations[CloudstackMachineAnnotationPrefix+"failuredomainname"]
	} else {
		defaultDomainName, err := GetDefaultFailureDomainName(src.ObjectMeta.Labels["cluster.x-k8s.io/cluster-name"], src.Spec.ZoneID, src.Spec.ZoneName)
		if err != nil {
			return err
		}
		dst.Spec.FailureDomainName = defaultDomainName
	}
	return nil
}

func (dst *CloudStackMachine) ConvertFrom(srcRaw conversion.Hub) error { // nolint
	src := srcRaw.(*v1beta2.CloudStackMachine)
	err := Convert_v1beta2_CloudStackMachine_To_v1beta1_CloudStackMachine(src, dst, nil)
	if err != nil {
		return err
	}
	// need to restore zoneid and zonename from v1beta2's annotation in metadata
	dst.Spec.ZoneID = src.ObjectMeta.Annotations[CloudstackMachineAnnotationPrefix+"zoneid"]
	dst.Spec.ZoneName = src.ObjectMeta.Annotations[CloudstackMachineAnnotationPrefix+"zonename"]

	// need to save failuredomainname in v1beta1's annotation
	dst.ObjectMeta.Annotations[CloudstackMachineAnnotationPrefix+"failuredomainname"] = src.Spec.FailureDomainName
	return nil
}
