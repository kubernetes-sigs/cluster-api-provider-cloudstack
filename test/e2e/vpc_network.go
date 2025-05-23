/*
Copyright 2023 The Kubernetes Authors.

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

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

// VPCNetworkSpec implements a test that verifies cluster creation in a VPC network.
func VPCNetworkSpec(ctx context.Context, inputGetter func() CommonSpecInput) {
	var (
		specName         = "vpc-network"
		input            CommonSpecInput
		namespace        *corev1.Namespace
		cancelWatches    context.CancelFunc
		clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
	)

	BeforeEach(func() {
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		input = inputGetter()
		Expect(input.E2EConfig).ToNot(BeNil(), "Invalid argument. input.E2EConfig can't be nil when calling %s spec", specName)
		Expect(input.ClusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. input.ClusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(input.BootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. input.BootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(input.ArtifactFolder, 0750)).To(Succeed(), "Invalid argument. input.ArtifactFolder can't be created for %s spec", specName)
		Expect(input.E2EConfig.Variables).To(HaveKey(KubernetesVersion))
		Expect(input.E2EConfig.Variables).To(HaveValidVersion(input.E2EConfig.GetVariable(KubernetesVersion)))

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("Should successfully create a cluster in a VPC network with default offering", func() {
		By("Creating a workload cluster in a VPC network with default offering")
		vpcName := input.E2EConfig.GetVariable(VPCName)
		networkName := input.E2EConfig.GetVariable(VPCNetworkName)
		setupClusterWithVpcAndVerifyOfferingName(ctx, input, namespace, clusterResources, specName, "vpc-network", vpcName, DefaultVPCOffering, networkName, DefaultVPCNetworkOffering)
		By("PASSED!")
	})

	It("Should successfully create a cluster in a VPC network with a custom offering", func() {
		By("Creating a workload cluster in a VPC network with a custom offering")
		vpcWithCustomOfferingName := input.E2EConfig.GetVariable(VPCWithCustomOfferingName)
		networkName := input.E2EConfig.GetVariable(VPCNetworkWithCustomOfferingName)
		customVpcOfferingName := input.E2EConfig.GetVariable(CustomVPCOfferingName)
		customVpcNetworkOfferingName := input.E2EConfig.GetVariable(CustomVPCNetworkOfferingName)
		setupClusterWithVpcAndVerifyOfferingName(ctx, input, namespace, clusterResources, specName, "custom-vpc-offering", vpcWithCustomOfferingName, customVpcOfferingName, networkName, customVpcNetworkOfferingName)
		By("PASSED!")
	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)
	})
}

func setupClusterWithVpcAndVerifyOfferingName(ctx context.Context, input CommonSpecInput, namespace *corev1.Namespace,
	clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult, specName string, flavor string, vpcName string,
	vpcOfferingName string, networkName string, networkOfferingName string,
) {
	clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
		ClusterProxy:    input.BootstrapClusterProxy,
		CNIManifestPath: input.E2EConfig.GetVariable(CNIPath),
		ConfigCluster: clusterctl.ConfigClusterInput{
			LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
			ClusterctlConfigPath:     input.ClusterctlConfigPath,
			KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
			InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
			Flavor:                   flavor,
			Namespace:                namespace.Name,
			ClusterName:              fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
			KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
			ControlPlaneMachineCount: pointer.Int64(1),
			WorkerMachineCount:       pointer.Int64(1),
		},
		WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
		WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
		WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
	}, clusterResources)

	csClient := CreateCloudStackClient(ctx, input.BootstrapClusterProxy.GetKubeconfigPath())

	Byf("Checking if VPC %s exists with offering %s", vpcName, vpcOfferingName)
	vpc, count, err := csClient.VPC.GetVPCByName(vpcName)
	Expect(err).To(BeNil())
	Expect(count).To(BeEquivalentTo(1))
	Expect(vpc.Vpcofferingname).To(BeEquivalentTo(vpcOfferingName))

	Byf("Checking if network %s exists with offering %s", networkName, networkOfferingName)
	network, count, err := csClient.Network.GetNetworkByName(networkName)
	Expect(err).To(BeNil())
	Expect(count).To(BeEquivalentTo(1))
	Expect(network.Networkofferingname).To(BeEquivalentTo(networkOfferingName))
}
