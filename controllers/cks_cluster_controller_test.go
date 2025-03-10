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
	"github.com/golang/mock/gomock"
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

var _ = ginkgo.Describe("CksCloudStackClusterReconciler", func() {
	ginkgo.Context("With k8s like test environment.", func() {
		ginkgo.BeforeEach(func() {
			dummies.SetDummyVars()
			SetupTestEnvironment()
			gomega.Ω(ClusterReconciler.SetupWithManager(ctx, k8sManager, controller.Options{})).Should(gomega.Succeed())  // Register CloudStack ClusterReconciler.
			gomega.Ω(FailureDomainReconciler.SetupWithManager(k8sManager, controller.Options{})).Should(gomega.Succeed()) // Register CloudStack FailureDomainReconciler.
			gomega.Ω(CksClusterReconciler.SetupWithManager(k8sManager)).Should(gomega.Succeed())                          // Register CloudStack Cks ClusterReconciler.
		})

		ginkgo.It("Should create a cluster in CKS.", func() {
			mockCloudClient.EXPECT().GetOrCreateCksCluster(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_, arg1, _ interface{}) {
				arg1.(*infrav1.CloudStackCluster).Status.CloudStackClusterID = "cluster-id-123"
			}).MinTimes(1).Return(nil)
			mockCloudClient.EXPECT().ResolveZone(gomock.Any()).AnyTimes()
			mockCloudClient.EXPECT().ResolveNetworkForZone(gomock.Any()).AnyTimes().Do(
				func(arg1 interface{}) {
					arg1.(*infrav1.CloudStackZoneSpec).Network.ID = "SomeID"
					arg1.(*infrav1.CloudStackZoneSpec).Network.Type = cloud.NetworkTypeShared
				}).MinTimes(1)

			gomega.Eventually(func() string {
				key := client.ObjectKeyFromObject(dummies.CSCluster)
				if err := k8sClient.Get(ctx, key, dummies.CSCluster); err != nil {
					return ""
				}
				return dummies.CSCluster.Status.CloudStackClusterID
			}, timeout).WithPolling(pollInterval).Should(gomega.Equal("cluster-id-123"))

		})
	})

	ginkgo.Context("With k8s like test environment.", func() {
		ginkgo.BeforeEach(func() {
			dummies.SetDummyVars()
			dummies.CSCluster.Status.CloudStackClusterID = "cluster-id-123"
			SetupTestEnvironment()
			gomega.Ω(ClusterReconciler.SetupWithManager(ctx, k8sManager, controller.Options{})).Should(gomega.Succeed())  // Register CloudStack ClusterReconciler.
			gomega.Ω(FailureDomainReconciler.SetupWithManager(k8sManager, controller.Options{})).Should(gomega.Succeed()) // Register CloudStack FailureDomainReconciler.
			gomega.Ω(CksClusterReconciler.SetupWithManager(k8sManager)).Should(gomega.Succeed())                          // Register CloudStack Cks ClusterReconciler.
		})

		ginkgo.It("Should delete the cluster in CKS.", func() {

			gomega.Ω(k8sClient.Delete(ctx, dummies.CSCluster)).Should(gomega.Succeed())

			gomega.Eventually(func() bool {
				csCluster := &infrav1.CloudStackCluster{}
				csClusterKey := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.CSCluster.Name}
				if err := k8sClient.Get(ctx, csClusterKey, csCluster); err != nil {
					return errors.IsNotFound(err)
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(gomega.BeTrue())
		})

	})

	ginkgo.Context("Without a k8s test environment.", func() {
		ginkgo.It("Should create a reconciliation runner with a Cloudstack Cluster as the reconciliation subject.", func() {
			reconRunner := controllers.NewCksClusterReconciliationRunner()
			gomega.Ω(reconRunner.ReconciliationSubject).ShouldNot(gomega.BeNil())
		})
	})
})
