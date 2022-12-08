/*
Copyright 2020 The Kubernetes Authors.

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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

// AffinityGroupSpec implements a test that verifies that an app deployed to the workload cluster works.
func AffinityGroupSpec(ctx context.Context, inputGetter func() CommonSpecInput) {
	var (
		specName         = "affinity-group"
		input            CommonSpecInput
		namespace        *corev1.Namespace
		cancelWatches    context.CancelFunc
		clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
		affinityIds      []string
	)

	BeforeEach(func() {
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		input = inputGetter()

		csClient := CreateCloudStackClient(ctx, input.BootstrapClusterProxy.GetKubeconfigPath())
		zoneName := input.E2EConfig.GetVariable("CLOUDSTACK_ZONE_NAME")
		numHosts := GetHostCount(csClient, zoneName)
		if numHosts < 3 {
			Skip("Too few ACS hosts to run conclusive affinity tests.  Please provision at least three for the zone.")
		}

		Expect(input.E2EConfig).ToNot(BeNil(), "Invalid argument. input.E2EConfig can't be nil when calling %s spec", specName)
		Expect(input.ClusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. input.ClusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(input.BootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. input.BootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(input.ArtifactFolder, 0750)).To(Succeed(), "Invalid argument. input.ArtifactFolder can't be created for %s spec", specName)

		Expect(input.E2EConfig.Variables).To(HaveKey(KubernetesVersion))

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("Should have host affinity group when affinity is pro", func() {
		affinityIds = executeTest(ctx, input, namespace, specName, clusterResources, "pro")
	})

	It("Should have host affinity group when affinity is anti", func() {
		affinityIds = executeTest(ctx, input, namespace, specName, clusterResources, "anti")
	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)

		csClient := CreateCloudStackClient(ctx, input.BootstrapClusterProxy.GetKubeconfigPath())
		err := CheckAffinityGroupsDeleted(csClient, affinityIds)
		if err != nil {
			Fail(err.Error())
		}
		By("PASSED!")
	})
}

func executeTest(ctx context.Context, input CommonSpecInput, namespace *corev1.Namespace, specName string, clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult, affinityType string) []string {
	clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
		ClusterProxy:    input.BootstrapClusterProxy,
		CNIManifestPath: input.E2EConfig.GetVariable(CNIPath),
		ConfigCluster: clusterctl.ConfigClusterInput{
			LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
			ClusterctlConfigPath:     input.ClusterctlConfigPath,
			KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
			InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
			Flavor:                   "affinity-group-" + affinityType,
			Namespace:                namespace.Name,
			ClusterName:              fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
			KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
			ControlPlaneMachineCount: pointer.Int64Ptr(3),
			WorkerMachineCount:       pointer.Int64Ptr(2),
		},
		WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
		WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
		WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
	}, clusterResources)

	csClient := CreateCloudStackClient(ctx, input.BootstrapClusterProxy.GetKubeconfigPath())
	return CheckAffinityGroup(csClient, clusterResources.Cluster.Name, affinityType)
}
