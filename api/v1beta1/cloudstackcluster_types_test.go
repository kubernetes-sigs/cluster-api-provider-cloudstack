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

package v1beta1_test

import (
	capcv1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloudStackCluster types", func() {

	Context("When retrieving a zone from ZoneStatusMap using GetOne", func() {
		It("Should return nil if ZoneStatusMap is empty", func() {
			zones := capcv1.ZoneStatusMap{}
			Ω(zones.GetOne()).Should(BeNil())
		})

		It("Should return a zone if ZoneStatusMap is not empty", func() {
			zones := capcv1.ZoneStatusMap{dummies.Zone1.ID: dummies.Zone1}
			Ω(zones.GetOne()).ShouldNot(BeNil())
		})
	})

	Context("When retrieving a zone from ZoneStatusMap using GetByName", func() {
		It("Should return nil if ZoneStatusMap is empty", func() {
			zones := capcv1.ZoneStatusMap{}
			Ω(zones.GetByName(dummies.Zone1.Name)).Should(BeNil())
		})

		It("Should return a zone if ZoneStatusMap has a zone having the name passed into GetByName", func() {
			zones := capcv1.ZoneStatusMap{dummies.Zone1.ID: dummies.Zone1}
			Ω(zones.GetByName(dummies.Zone1.Name)).ShouldNot(BeNil())
		})

		It("Should return nil if ZoneStatusMap does not have a zone having the name passed into GetByName", func() {
			zones := capcv1.ZoneStatusMap{dummies.Zone2.ID: dummies.Zone2}.DeepCopy()
			Ω(zones.GetByName(dummies.Zone1.Name)).Should(BeNil())
		})
	})
})
