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

package v1beta2_test

import (
	"k8s.io/utils/ptr"
	capcv1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("CloudStackMachineConfig_CompressUserdata", func() {
	for _, tc := range []struct {
		Name    string
		Machine capcv1.CloudStackMachine
		Expect  bool
	}{
		{
			Name: "is true when uncompressed user data is nil",
			Machine: capcv1.CloudStackMachine{
				Spec: capcv1.CloudStackMachineSpec{
					UncompressedUserData: nil,
				},
			},
			Expect: true,
		},
		{
			Name: "is false when uncompressed user data is true",
			Machine: capcv1.CloudStackMachine{
				Spec: capcv1.CloudStackMachineSpec{
					UncompressedUserData: ptr.To(true),
				},
			},
			Expect: false,
		},
		{
			Name: "Is false when uncompressed user data is false",
			Machine: capcv1.CloudStackMachine{
				Spec: capcv1.CloudStackMachineSpec{
					UncompressedUserData: ptr.To(false),
				},
			},
			Expect: true,
		},
	} {
		tc := tc
		ginkgo.It(tc.Name, func() {
			result := tc.Machine.CompressUserdata()
			gomega.Expect(result).To(gomega.Equal(tc.Expect))
		})
	}
})
