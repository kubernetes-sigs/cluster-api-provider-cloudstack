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

// import (
// 	"context"

// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"
// 	"sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies"
// 	"sigs.k8s.io/controller-runtime/pkg/client"
// )

// var _ = Describe("CloudStackZoneReconciler", func() {
// 	ctx := context.Background()

// 	BeforeEach(func() {
// 		dummies.SetDummyVars()
// 		// Create onwer CRDs.
// 		dummies.CAPICluster.Spec.InfrastructureRef.Name = dummies.CSCluster.Name
// 		setupClusterCRDs()
// 		Ω(k8sClient.Create(ctx, dummies.CSZone1)).Should(Succeed())
// 	})

// 	AfterEach(func() {
// 		// deleteAllOfKindInNameSpace(&clusterv1.Cluster{})
// 		// deleteAllOfKindInNameSpace(&infrav1.CloudStackCluster{})
// 		Ω(k8sClient.Delete(ctx, dummies.CSZone1)).Should(Succeed())
// 	})

// 	It("Should create a CloudStackIsolatedNetwork", func() {

// 		// Test that the CloudStackCluster controller creates a CloudStackZone CRD.
// 		Eventually(func() bool {
// 			key := client.ObjectKey{Namespace: dummies.CSCluster.Namespace, Name: dummies.CSZone1.Spec.Network.Name}
// 			if err := k8sClient.Get(ctx, key, dummies.CSISONet1); err != nil {
// 				return true
// 			}
// 			return false
// 		}, timeout).Should(BeTrue())

// 	})
// })
