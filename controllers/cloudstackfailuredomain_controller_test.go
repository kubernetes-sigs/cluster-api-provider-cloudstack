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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CloudStackFailureDomainReconciler", func() {
	Context("With k8s like test environment.", func() {
		BeforeEach(func() {
			dummies.SetDummyVars()
			SetupTestEnvironment()                                                    // Must happen before setting up managers/reconcilers.
			Ω(FailureDomainReconciler.SetupWithManager(k8sManager)).Should(Succeed()) // Register CloudStack FailureDomainReconciler.
		})

		It("Should set failure domain Status.Ready to true.", func() {
			// Modify failure domain name the same way the cluster controller would.
			dummies.CSFailureDomain1.Name = dummies.CSFailureDomain1.Name + "-" + dummies.CSCluster.Name

			Ω(k8sClient.Create(ctx, dummies.ACSEndpointSecret1))
			Ω(k8sClient.Create(ctx, dummies.CSFailureDomain1))

			mockCloudClient.EXPECT().ResolveZone(gomock.Any()).MinTimes(1)

			mockCloudClient.EXPECT().ResolveNetworkForZone(gomock.Any()).AnyTimes().Do(
				func(arg1 interface{}) {
					arg1.(*infrav1.CloudStackZoneSpec).Network.ID = "SomeID"
					arg1.(*infrav1.CloudStackZoneSpec).Network.Type = cloud.NetworkTypeShared
				}).MinTimes(1)

			tempfd := &infrav1.CloudStackFailureDomain{}
			Eventually(func() bool {
				key := client.ObjectKeyFromObject(dummies.CSFailureDomain1)
				if err := k8sClient.Get(ctx, key, tempfd); err == nil {
					return tempfd.Status.Ready
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())
		})

		It("Should set owner reference to secret if the cluster names match and it's not owned", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "acsendpointsecret1",
					Labels: map[string]string{
						clusterv1.ClusterLabelName: "cluster1",
					},
				},
			}
			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster1",
				},
			}

			Ω(controllers.IsSetOwnerReferenceNeeded(secret, cluster.Name)).Should(BeTrue())
		})

		It("Should not set owner reference to secret if the cluster names don't match", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "acsendpointsecret1",
					Labels: map[string]string{
						clusterv1.ClusterLabelName: "cluster1",
					},
				},
			}
			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster2",
				},
			}

			Ω(controllers.IsSetOwnerReferenceNeeded(secret, cluster.Name)).Should(BeFalse())
		})

		It("Should not set owner reference to secret if the cluster name label is missing", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "acsendpointsecret1",
				},
			}
			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster1",
				},
			}

			Ω(controllers.IsSetOwnerReferenceNeeded(secret, cluster.Name)).Should(BeFalse())
		})

		It("Should not set owner reference to secret if it's already owned by a cluster", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "acsendpointsecret1",
					Labels: map[string]string{
						clusterv1.ClusterLabelName: "cluster1",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "Cluster",
							Name: "cluster2",
						},
					},
				},
			}
			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster1",
				},
			}

			Ω(controllers.IsSetOwnerReferenceNeeded(secret, cluster.Name)).Should(BeFalse())
		})

		It("Should set owner reference to secret if it's not owned by a cluster", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "acsendpointsecret1",
					Labels: map[string]string{
						clusterv1.ClusterLabelName: "cluster1",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "Machine",
							Name: "machine1",
						},
					},
				},
			}
			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster1",
				},
			}

			Ω(controllers.IsSetOwnerReferenceNeeded(secret, cluster.Name)).Should(BeTrue())
		})
	})
})
