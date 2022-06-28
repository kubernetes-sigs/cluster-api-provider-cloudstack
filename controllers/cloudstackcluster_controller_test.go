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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies"
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
		Ω(k8sClient.Create(ctx, dummies.CAPICluster)).Should(Succeed())
		Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(Succeed())
	})
	AfterEach(func() {
		Ω(k8sClient.Delete(ctx, dummies.CAPICluster)).Should(Succeed())
		Ω(k8sClient.Delete(ctx, dummies.CSCluster)).Should(Succeed())
	})

	It("Should create a cluster", func() {
		By("Fetching the CAPI cluster object that owns this CS cluster object")

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
