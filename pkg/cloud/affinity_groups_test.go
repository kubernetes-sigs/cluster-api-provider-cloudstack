/*
Copyright 2022.

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

package cloud_test

import (
	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1alpha3"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AffinityGroup", func() {
	var ( // Declare shared vars.
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient
		ags        *cloudstack.MockAffinityGroupServiceIface
		cluster    *infrav1.CloudStackCluster
		client     cloud.Client
	)

	BeforeEach(func() {
		// Setup new mock services.
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)
		ags = mockClient.AffinityGroup.(*cloudstack.MockAffinityGroupServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("for non-existent affinity group", func() {
		XIt("creates an affinity group", func() {
			ags.EXPECT().GetAffinityGroupByName("FakeAG").Return(&cloudstack.AffinityGroup{}, 1, nil)

			Ω(client.GetOrCreateAffinityGroup(cluster, cloud.AffinityGroup{Name: "FakeAG", AntiAffinity: true})).ShouldNot(Succeed())
		})
	})
})
