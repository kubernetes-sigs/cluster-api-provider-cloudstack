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

package controllers_test

import (
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CloudStackZoneReconciler", func() {
	BeforeEach(func() {
		SetupTestEnvironment()                                           // Must happen before setting up managers/reconcilers.
		Ω(ZoneReconciler.SetupWithManager(k8sManager)).Should(Succeed()) // Register CloudStack ZoneReconciler.
		dummies.SetDummyVars()
	})

	It("Should create a CloudStackIsolatedNetwork", func() {
		// Specify an Isolated Network for the Zone and create the CRD to kick of reconciliation.
		dummies.CSZone1.Spec.Network = dummies.ISONet1
		Ω(k8sClient.Create(ctx, dummies.CSZone1)).Should(Succeed())

		mockCloudClient.EXPECT().ResolveZone(gomock.Any()).AnyTimes()
		mockCloudClient.EXPECT().ResolveNetworkForZone(gomock.Any()).AnyTimes()

		// Test that the CloudStackCluster controller creates a CloudStackZone CRD.
		Eventually(func() bool {
			nameSpaceFilter := &client.DeleteAllOfOptions{ListOptions: client.ListOptions{Namespace: dummies.ClusterNameSpace}}
			isoNets := &infrav1.CloudStackIsolatedNetworkList{}
			if err := k8sClient.List(ctx, isoNets, nameSpaceFilter); err == nil {
				if len(isoNets.Items) == 1 {
					return true
				}
			}
			return false
		}, timeout).WithPolling(2 * time.Second).Should(BeTrue())
	})
})
