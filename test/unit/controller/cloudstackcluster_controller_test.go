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
	"context"
	"time"

	"github.com/aws/cluster-api-provider-cloudstack-staging/test/unit/dummies"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout = time.Second * 30
)

var _ = Describe("CloudStackClusterReconciler", func() {
	ctx := context.Background()

	BeforeEach(func() {
		dummies.SetDummyVars()
	})

	It("Should create a cluster", func() {
		By("Fetching a CS Cluster Object")
		// TODO make the tests work with other CRDs. This stage can't be reached yet.
		// CS.EXPECT().GetOrCreateCluster(gomock.Any()).MinTimes(1)

		// Create the CS Cluster object for the reconciler to fetch.
		Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(Succeed())
		// TODO: add deletion defer here.

		By("Fetching the CAPI cluster object that owns this CS cluster object")
		// Create the CAPI cluster (owner) object.
		dummies.CAPICluster.Spec.InfrastructureRef.Name = dummies.CSCluster.Name
		Ω(k8sClient.Create(ctx, dummies.CAPICluster)).Should(Succeed())
		// TODO: add deletion defer here.

		key := client.ObjectKey{Namespace: dummies.CSCluster.Namespace, Name: dummies.CSCluster.Name}
		Eventually(func() error {
			return k8sClient.Get(ctx, key, dummies.CSCluster)
		}, timeout).Should(BeNil())

		By("Setting the OwnerRef on the CloudStack cluster")
		Eventually(func() error {
			ph, err := patch.NewHelper(dummies.CSCluster, k8sClient)
			Ω(err).ShouldNot(HaveOccurred())
			dummies.CSCluster.OwnerReferences = append(dummies.CSCluster.OwnerReferences, metav1.OwnerReference{
				Kind:       "Cluster",
				APIVersion: clusterv1.GroupVersion.String(),
				Name:       dummies.CAPICluster.Name,
				UID:        "uniqueness",
			})
			return ph.Patch(ctx, dummies.CSCluster, patch.WithStatusObservedGeneration{})
		}, timeout).Should(Succeed())

		// Test that the CloudStackCluster controller creates a CloudStackZone CRD.
		Eventually(func() bool {
			key := client.ObjectKey{Namespace: dummies.CSCluster.Namespace, Name: dummies.CSCluster.Spec.Zones[0].Name}
			if err := k8sClient.Get(ctx, key, dummies.CSCluster); err != nil {
				return true
			}
			return false
		}, timeout).Should(BeTrue())

	})
})
