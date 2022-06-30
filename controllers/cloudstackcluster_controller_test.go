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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CloudStackClusterReconciler", func() {
	BeforeEach(func() {
		// Register the CloudStack ClusterReconciler only.
		Ω(ClusterReconciler.SetupWithManager(k8sManager)).Should(Succeed())
	})

	It("Should create a CloudStackZone", func() {
		// Test that the CloudStackCluster controller creates a CloudStackZone CRD.
		tempZone := &infrav1.CloudStackZone{}
		Eventually(func() bool {
			key := client.ObjectKey{Namespace: dummies.CSCluster.Namespace, Name: dummies.CSCluster.Spec.Zones[0].Name}
			if err := k8sClient.Get(ctx, key, tempZone); err != nil {
				return true
			}
			return false
		}, timeout).Should(BeTrue())
	})
})
