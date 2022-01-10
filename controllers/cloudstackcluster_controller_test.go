/*
Copyright 2022.

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

	infrav1 "github.com/aws/cluster-api-provider-cloudstack-staging/api/v1alpha3"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getCloudStackCluster() *infrav1.CloudStackCluster {
	return &infrav1.CloudStackCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
			Kind:       "CloudStackCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "cs-cluster-test1-",
			Namespace:    "default",
		},
		Spec: infrav1.CloudStackClusterSpec{
			Zone:    "zone",
			Network: "network",
		},
	}
}

func getCapiCluster() *clusterv1.Cluster {
	return &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "capi-cluster-test-",
			Namespace:    "default",
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: infrav1.GroupVersion.String(),
				Kind:       "CloudStackCluster",
				Name:       "somename",
			},
		},
	}

}

const (
	timeout = time.Second * 60
)

var _ = Describe("CloudStackClusterReconciler", func() {
	ctx := context.Background()

	BeforeEach(func() {
	})

	It("Should create a cluster", func() {
		By("Fetching a CS Cluster Object")
		// Create the CS Cluster object for the reconciler to fetch.
		csCluster := getCloudStackCluster()
		Ω(k8sClient.Create(ctx, csCluster)).Should(Succeed())
		// TODO: add deletion defer here.

		By("Fetching the CAPI cluster object that owns this CS cluster object")
		// Create the CAPI cluster (owner) object.
		capiCluster := getCapiCluster()
		capiCluster.Spec.InfrastructureRef.Name = csCluster.Name
		Ω(k8sClient.Create(ctx, capiCluster)).Should(Succeed())
		// TODO: add deletion defer here.

		key := client.ObjectKey{Namespace: csCluster.Namespace, Name: csCluster.Name}
		Eventually(func() error {
			return k8sClient.Get(ctx, key, csCluster)
		}, timeout).Should(BeNil())

		By("Setting the OwnerRef on the CloudStack cluster")
		Eventually(func() error {
			ph, err := patch.NewHelper(csCluster, k8sClient)
			Ω(err).ShouldNot(HaveOccurred())
			csCluster.OwnerReferences = append(csCluster.OwnerReferences, metav1.OwnerReference{
				Kind:       "Cluster",
				APIVersion: clusterv1.GroupVersion.String(),
				Name:       capiCluster.Name,
				UID:        "uniqueness",
			})
			return ph.Patch(ctx, csCluster, patch.WithStatusObservedGeneration{})
		}, timeout).Should(Succeed())

		CS.EXPECT().GetOrCreateCluster(gomock.Any()).MinTimes(1)
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, csCluster); err != nil {
				return false
			}
			return len(csCluster.Finalizers) > 0
		}, timeout).Should(BeTrue())

	})
})
