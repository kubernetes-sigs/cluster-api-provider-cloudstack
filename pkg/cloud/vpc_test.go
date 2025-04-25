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

package cloud_test

import (
	"errors"
	"fmt"

	csapi "github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
)

var _ = ginkgo.Describe("VPC", func() {
	var (
		mockCtrl   *gomock.Controller
		mockClient *csapi.CloudStackClient
		vs         *csapi.MockVPCServiceIface
		rs         *csapi.MockResourcetagsServiceIface
		client     cloud.Client
	)

	ginkgo.BeforeEach(func() {
		// Setup new mock services.
		mockCtrl = gomock.NewController(ginkgo.GinkgoT())
		mockClient = csapi.NewMockClient(mockCtrl)
		vs = mockClient.VPC.(*csapi.MockVPCServiceIface)
		rs = mockClient.Resourcetags.(*csapi.MockResourcetagsServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient, nil)
		dummies.SetDummyVars()
	})

	ginkgo.AfterEach(func() {
		mockCtrl.Finish()
	})

	ginkgo.Context("for an existing VPC", func() {
		var dummyVPC infrav1.VPC

		ginkgo.BeforeEach(func() {
			dummyVPC = infrav1.VPC{
				ID:   "vpc-123",
				Name: "test-vpc",
				CIDR: "10.0.0.0/16",
			}
		})

		ginkgo.It("resolves VPC by ID", func() {
			dummyCSVPC := &csapi.VPC{
				Id:   dummyVPC.ID,
				Name: dummyVPC.Name,
				Cidr: dummyVPC.CIDR,
			}

			vs.EXPECT().GetVPCByID(dummyVPC.ID, gomock.Any()).Return(dummyCSVPC, 1, nil)

			gomega.Ω(client.ResolveVPC(&dummyVPC)).Should(gomega.Succeed())
			gomega.Ω(dummyVPC.Name).Should(gomega.Equal("test-vpc"))
		})

		ginkgo.It("resolves VPC by Name", func() {
			dummyVPC.ID = "" // Clear ID to test by name

			dummyCSVPC := &csapi.VPC{
				Id:   "vpc-123",
				Name: dummyVPC.Name,
				Cidr: dummyVPC.CIDR,
			}

			vs.EXPECT().GetVPCByName(dummyVPC.Name, gomock.Any()).Return(dummyCSVPC, 1, nil)

			gomega.Ω(client.ResolveVPC(&dummyVPC)).Should(gomega.Succeed())
			gomega.Ω(dummyVPC.ID).Should(gomega.Equal("vpc-123"))
		})

		ginkgo.It("returns error when VPC not found by ID", func() {
			vs.EXPECT().GetVPCByID(dummyVPC.ID, gomock.Any()).Return(nil, 0, nil)

			err := client.ResolveVPC(&dummyVPC)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring(fmt.Sprintf("no VPC found with ID %s", dummyVPC.ID)))
		})

		ginkgo.It("returns error when VPC not found by Name", func() {
			dummyVPC.ID = "" // Clear ID to test by name

			vs.EXPECT().GetVPCByName(dummyVPC.Name, gomock.Any()).Return(nil, 0, nil)

			err := client.ResolveVPC(&dummyVPC)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring(fmt.Sprintf("no VPC found with name %s", dummyVPC.Name)))
		})

		ginkgo.It("returns error when GetVPCByID fails", func() {
			expectedErr := errors.New("API error")
			vs.EXPECT().GetVPCByID(dummyVPC.ID, gomock.Any()).Return(nil, 0, expectedErr)

			err := client.ResolveVPC(&dummyVPC)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring(fmt.Sprintf("failed to get VPC with ID %s", dummyVPC.ID)))
		})

		ginkgo.It("returns error when GetVPCByName fails", func() {
			dummyVPC.ID = "" // Clear ID to test by name
			expectedErr := errors.New("API error")
			vs.EXPECT().GetVPCByName(dummyVPC.Name, gomock.Any()).Return(nil, 0, expectedErr)

			err := client.ResolveVPC(&dummyVPC)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring(fmt.Sprintf("failed to get VPC with name %s", dummyVPC.Name)))
		})

		ginkgo.It("handles nil VPC", func() {
			gomega.Ω(client.ResolveVPC(nil)).Should(gomega.Succeed())
		})

		ginkgo.It("handles empty VPC", func() {
			emptyVPC := &infrav1.VPC{}
			gomega.Ω(client.ResolveVPC(emptyVPC)).Should(gomega.Succeed())
		})
	})

	ginkgo.Context("for creating a VPC", func() {
		var (
			dummyFD  infrav1.CloudStackFailureDomain
			dummyVPC infrav1.VPC
		)

		ginkgo.BeforeEach(func() {
			dummyFD = infrav1.CloudStackFailureDomain{
				Spec: infrav1.CloudStackFailureDomainSpec{
					Zone: infrav1.CloudStackZoneSpec{
						ID: "zone-123",
					},
				},
			}
			dummyVPC = infrav1.VPC{
				Name: "test-vpc",
				CIDR: "10.0.0.0/16",
			}
		})

		ginkgo.It("creates a new VPC successfully", func() {
			offeringID := "offering-123"
			createVPCParams := &csapi.CreateVPCParams{}
			createVPCResponse := &csapi.CreateVPCResponse{
				Id: "vpc-123",
			}

			vs.EXPECT().GetVPCOfferingID(cloud.VPCOffering).Return(offeringID, 1, nil)
			vs.EXPECT().NewCreateVPCParams(dummyVPC.Name, dummyVPC.Name, offeringID, dummyFD.Spec.Zone.ID).Return(createVPCParams)
			vs.EXPECT().CreateVPC(createVPCParams).Return(createVPCResponse, nil)
			rs.EXPECT().NewCreateTagsParams(gomock.Any(), gomock.Any(), gomock.Any()).Return(&csapi.CreateTagsParams{})
			rs.EXPECT().CreateTags(gomock.Any()).Return(&csapi.CreateTagsResponse{}, nil)

			gomega.Ω(client.CreateVPC(&dummyFD, &dummyVPC)).Should(gomega.Succeed())
			gomega.Ω(dummyVPC.ID).Should(gomega.Equal("vpc-123"))
		})

		ginkgo.It("returns error when VPC offering cannot be fetched", func() {
			expectedErr := errors.New("failed to get VPC offering")
			vs.EXPECT().GetVPCOfferingID(cloud.VPCOffering).Return("", 0, expectedErr)

			err := client.CreateVPC(&dummyFD, &dummyVPC)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.Equal(expectedErr.Error()))
		})

		ginkgo.It("returns error when multiple VPC offerings found", func() {
			vs.EXPECT().GetVPCOfferingID(cloud.VPCOffering).Return("", 2, nil)

			err := client.CreateVPC(&dummyFD, &dummyVPC)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.Equal("found more than one vpc offering"))
		})

		ginkgo.It("returns error when CreateVPC fails", func() {
			offeringID := "offering-123"
			createVPCParams := &csapi.CreateVPCParams{}
			expectedErr := errors.New("API error")

			vs.EXPECT().GetVPCOfferingID(cloud.VPCOffering).Return(offeringID, 1, nil)
			vs.EXPECT().NewCreateVPCParams(dummyVPC.Name, dummyVPC.Name, offeringID, dummyFD.Spec.Zone.ID).Return(createVPCParams)
			vs.EXPECT().CreateVPC(createVPCParams).Return(nil, expectedErr)

			err := client.CreateVPC(&dummyFD, &dummyVPC)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.ContainSubstring(fmt.Sprintf("creating VPC with name %s", dummyVPC.Name)))
		})

		ginkgo.It("returns error when VPC is nil", func() {
			err := client.CreateVPC(&dummyFD, nil)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.Equal("VPC name must be specified"))
		})

		ginkgo.It("returns error when VPC name is empty", func() {
			emptyNameVPC := &infrav1.VPC{
				CIDR: "10.0.0.0/16",
			}
			err := client.CreateVPC(&dummyFD, emptyNameVPC)
			gomega.Ω(err).ShouldNot(gomega.Succeed())
			gomega.Ω(err.Error()).Should(gomega.Equal("VPC name must be specified"))
		})
	})
})
