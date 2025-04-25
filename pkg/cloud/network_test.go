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
	"errors"
	"fmt"

	csapi "github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
)

var _ = ginkgo.Describe("Network", func() {
	var ( // Declare shared vars.
		mockCtrl   *gomock.Controller
		mockClient *csapi.CloudStackClient
		ns         *csapi.MockNetworkServiceIface
		rs         *csapi.MockResourcetagsServiceIface
		client     cloud.Client
	)

	ginkgo.BeforeEach(func() {
		// Setup new mock services.
		mockCtrl = gomock.NewController(ginkgo.GinkgoT())
		mockClient = csapi.NewMockClient(mockCtrl)
		ns = mockClient.Network.(*csapi.MockNetworkServiceIface)
		rs = mockClient.Resourcetags.(*csapi.MockResourcetagsServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient, nil)
		dummies.SetDummyVars()
	})

	ginkgo.AfterEach(func() {
		mockCtrl.Finish()
	})

	ginkgo.Context("for an existing network", func() {
		ginkgo.It("resolves network by ID", func() {
			ns.EXPECT().GetNetworkByName(dummies.ISONet1.Name, gomock.Any()).Return(nil, 0, nil)
			ns.EXPECT().GetNetworkByID(dummies.ISONet1.ID, gomock.Any()).Return(dummies.CAPCNetToCSAPINet(&dummies.ISONet1), 1, nil)

			gomega.Ω(client.ResolveNetwork(&dummies.ISONet1)).Should(gomega.Succeed())
		})

		ginkgo.It("resolves network by Name", func() {
			ns.EXPECT().GetNetworkByName(dummies.ISONet1.Name, gomock.Any()).Return(dummies.CAPCNetToCSAPINet(&dummies.ISONet1), 1, nil)

			gomega.Ω(client.ResolveNetwork(&dummies.ISONet1)).Should(gomega.Succeed())
		})

		ginkgo.It("When there exists more than one network with the same name", func() {
			ns.EXPECT().GetNetworkByName(dummies.ISONet1.Name, gomock.Any()).Return(dummies.CAPCNetToCSAPINet(&dummies.ISONet1), 2, nil)
			ns.EXPECT().GetNetworkByID(dummies.ISONet1.ID, gomock.Any()).Return(nil, 2, errors.New("There is more then one result for Network UUID"))
			err := client.ResolveNetwork(&dummies.ISONet1)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring(fmt.Sprintf("expected 1 Network with name %s, but got %d", dummies.ISONet1.Name, 2)))
		})
	})

	ginkgo.Context("Remove cluster tag from network", func() {
		ginkgo.It("Remove tag from network", func() {
			rtdp := &csapi.DeleteTagsParams{}
			createdByCAPCResponse := &csapi.ListTagsResponse{Tags: []*csapi.Tag{{Key: dummies.CSClusterTagKey, Value: "1"}}}
			rtlp := &csapi.ListTagsParams{}
			rs.EXPECT().NewDeleteTagsParams(gomock.Any(), gomock.Any()).Return(rtdp)
			rs.EXPECT().DeleteTags(rtdp).Return(&csapi.DeleteTagsResponse{}, nil)
			rs.EXPECT().NewListTagsParams().Return(rtlp)
			rs.EXPECT().ListTags(rtlp).Return(createdByCAPCResponse, nil)
			gomega.Ω(client.RemoveClusterTagFromNetwork(dummies.CSCluster, dummies.ISONet1)).Should(gomega.Succeed())
		})
	})
})
