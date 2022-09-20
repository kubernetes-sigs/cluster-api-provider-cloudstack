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
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/fakes/etcdcluster"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CloudStackFailureDomainReconciler", func() {
	Context("With k8s like test environment.", func() {
		BeforeEach(func() {
			dummies.SetDummyVars()
			SetupTestEnvironment()                                                    // Must happen before setting up managers/reconcilers.
			Ω(FailureDomainReconciler.SetupWithManager(k8sManager)).Should(Succeed()) // Register CloudStack FailureDomainReconciler.
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

		})

		It("Should set failure domain Status.Ready to true.", func() {
			assertFailuredomainCreated()
		})
		It("Should delete failure domain if no VM under this failure domain.", func() {
			assertFailuredomainCreated()
			Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

			assertFailuredomainNotExisted()
		})
		It("Should patch its machine deployment when worker machine exists in deleted failure domain.", func() {
			assertFailuredomainCreated()
			setupCAPIMachineDeploymentCRD(dummies.CAPIMachineDeployment)
			setupCAPIMachineAndCSMachineCRDs(dummies.CSMachine1, dummies.CAPIMachine1)
			setMachineOwnerReference(dummies.CSMachine1, dummies.MachineSetOwnerRef)
			labelMachineFailuredomain(dummies.CSMachine1, dummies.CSFailureDomain1)
			labelMachineDeploymentName(dummies.CSMachine1, dummies.CAPIMachineDeployment.Name)
			setMachineAnnotation(dummies.CSMachine1, "cluster.x-k8s.io/cloned-from-name", "capi-test-md-0-123")
			Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

			tempMd := &clusterv1.MachineDeployment{}
			Eventually(func() bool {
				key := client.ObjectKeyFromObject(dummies.CAPIMachineDeployment)
				if err := k8sClient.Get(ctx, key, tempMd); err == nil {
					_, found := tempMd.Spec.Template.Annotations["cluster.x-k8s.io/restartedAt"]
					return found
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())
		})
		It("Should patch kubeadmControlPlane when control plane machine exists in deleted failure domain.", func() {
			assertFailuredomainCreated()
			setKubeadmControlPlaneCRD(dummies.KubeadmControlPlane)
			setupCAPIMachineAndCSMachineCRDs(dummies.CSMachine1, dummies.CAPIMachine1)
			setMachineOwnerReference(dummies.CSMachine1, dummies.KubeadmControlPlaneOwnerRef)
			labelMachineFailuredomain(dummies.CSMachine1, dummies.CSFailureDomain1)

			Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

			tempKCP := &controlplanev1.KubeadmControlPlane{}
			Eventually(func() bool {
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.KubeadmControlPlaneOwnerRef.Name}
				if err := k8sClient.Get(ctx, key, tempKCP); err == nil {
					return tempKCP.Spec.RolloutAfter != nil
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())
		})

		It("Should patch etcdadmCluster when etcd node exists in deleted failure domain.", func() {
			assertFailuredomainCreated()
			setCSMachineTemplateCRD(dummies.CSMachineTemplate1)
			setupEtcdadmClusterCRD(dummies.EtcdadmCluster)
			preTemplateName := dummies.EtcdadmCluster.Spec.InfrastructureTemplate.Name
			setupCAPIMachineAndCSMachineCRDs(dummies.CSMachine1, dummies.CAPIMachine1)
			setMachineOwnerReference(dummies.CSMachine1, dummies.EtcdadmClusterOwnerRef)
			setMachineAnnotation(dummies.CSMachine1, "cluster.x-k8s.io/cloned-from-name", "test-machinetemplate-1")
			labelMachineFailuredomain(dummies.CSMachine1, dummies.CSFailureDomain1)
			Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

			etcdadmCluster := &etcdcluster.EtcdadmCluster{}
			Eventually(func() bool {
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.EtcdClusterName}
				if err := k8sClient.Get(ctx, key, etcdadmCluster); err == nil {
					return etcdadmCluster.Spec.InfrastructureTemplate.Name != preTemplateName
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())
		})

		It("Should not patch kubeadmControlPlane when etcd and cp machines both exist in deleted failure domain.", func() {
			assertFailuredomainCreated()
			setCSMachineTemplateCRD(dummies.CSMachineTemplate1)
			setupEtcdadmClusterCRD(dummies.EtcdadmCluster)
			setupCAPIMachineAndCSMachineCRDs(dummies.CSMachine1, dummies.CAPIMachine1)
			setMachineOwnerReference(dummies.CSMachine1, dummies.EtcdadmClusterOwnerRef)
			setMachineAnnotation(dummies.CSMachine1, "cluster.x-k8s.io/cloned-from-name", "test-machinetemplate-1")
			labelMachineFailuredomain(dummies.CSMachine1, dummies.CSFailureDomain1)

			assertFailuredomainCreated()
			setKubeadmControlPlaneCRD(dummies.KubeadmControlPlane)
			setupCAPIMachineAndCSMachineCRDs(dummies.CSMachine2, dummies.CAPIMachine2)
			setMachineOwnerReference(dummies.CSMachine2, dummies.KubeadmControlPlaneOwnerRef)
			labelMachineFailuredomain(dummies.CSMachine2, dummies.CSFailureDomain1)

			Ω(k8sClient.Delete(ctx, dummies.CSFailureDomain1))

			tempKCP := &controlplanev1.KubeadmControlPlane{}
			Consistently(func() bool {
				key := client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: dummies.KubeadmControlPlaneOwnerRef.Name}
				if err := k8sClient.Get(ctx, key, tempKCP); err == nil {
					return tempKCP.Spec.RolloutAfter == nil
				}
				return false
			}, timeout).WithPolling(pollInterval).Should(BeTrue())
		})
	})
})

func assertFailuredomainCreated() {
	tempfd := &infrav1.CloudStackFailureDomain{}
	Eventually(func() bool {
		key := client.ObjectKeyFromObject(dummies.CSFailureDomain1)
		if err := k8sClient.Get(ctx, key, tempfd); err == nil {
			return tempfd.Status.Ready
		}
		return false
	}, timeout).WithPolling(pollInterval).Should(BeTrue())
}

func assertFailuredomainNotExisted() {
	tempfd := &infrav1.CloudStackFailureDomain{}
	Eventually(func() bool {
		key := client.ObjectKeyFromObject(dummies.CSFailureDomain1)
		if err := k8sClient.Get(ctx, key, tempfd); err != nil {
			return true
		}
		return false
	}, timeout).WithPolling(pollInterval).Should(BeTrue())
}
