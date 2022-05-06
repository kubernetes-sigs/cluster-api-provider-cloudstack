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
	"fmt"

	csapi "github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/aws/cluster-api-provider-cloudstack/test/dummies"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Cluster", func() {
	var (
		client     cloud.Client
		mockCtrl   *gomock.Controller
		mockClient *csapi.CloudStackClient
		zs         *csapi.MockZoneServiceIface
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = csapi.NewMockClient(mockCtrl)
		zs = mockClient.Zone.(*csapi.MockZoneServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
		dummies.SetDummyVars()
		dummies.SetDummyDomainAndAccount()
		dummies.SetDummyCSApiResponse()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("an existing abstract dummies.CSCluster", func() {
		It("handles zone not found.", func() {
			expectedErr := fmt.Errorf("Not found")
			zs.EXPECT().GetZoneID(dummies.Zone1.Name).Return("", -1, expectedErr)
			zs.EXPECT().GetZoneByID(dummies.Zone1.ID).Return(nil, -1, expectedErr)

			err := client.ResolveZone(dummies.CSZone1)
			Expect(errors.Cause(err)).To(MatchError(expectedErr))
		})

		It("handles multiple zone IDs returned", func() {
			zs.EXPECT().GetZoneID(dummies.Zone1.Name).Return(dummies.Zone1.ID, 2, nil)
			zs.EXPECT().GetZoneByID(dummies.Zone1.ID).Return(nil, -1, fmt.Errorf("Not found"))

			Ω(client.ResolveZone(dummies.CSZone1)).Should(MatchError(And(
				ContainSubstring("expected 1 Zone with name "+dummies.Zone1.Name+", but got 2"),
				ContainSubstring("could not get Zone by ID "+dummies.Zone1.ID+": Not found"))))
		})
	})
})
