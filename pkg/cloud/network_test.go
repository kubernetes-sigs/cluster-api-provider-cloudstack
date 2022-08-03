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

package cloud_test

import (
	csapi "github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
)

var _ = Describe("Network", func() {
	var ( // Declare shared vars.
		mockCtrl   *gomock.Controller
		mockClient *csapi.CloudStackClient
		ns         *csapi.MockNetworkServiceIface
		client     cloud.Client
	)

	BeforeEach(func() {
		// Setup new mock services.
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = csapi.NewMockClient(mockCtrl)
		ns = mockClient.Network.(*csapi.MockNetworkServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
		dummies.SetDummyVars()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("for an existing network", func() {
		It("resolves network by ID", func() {
			ns.EXPECT().GetNetworkByName(dummies.ISONet1.Name).Return(nil, 0, nil)
			ns.EXPECT().GetNetworkByID(dummies.ISONet1.ID).Return(dummies.CAPCNetToCSAPINet(&dummies.ISONet1), 1, nil)

			Ω(client.ResolveNetwork(&dummies.ISONet1)).Should(Succeed())
		})

		It("resolves network by Name", func() {
			ns.EXPECT().GetNetworkByName(dummies.ISONet1.Name).Return(dummies.CAPCNetToCSAPINet(&dummies.ISONet1), 1, nil)

			Ω(client.ResolveNetwork(&dummies.ISONet1)).Should(Succeed())
		})
	})
})
