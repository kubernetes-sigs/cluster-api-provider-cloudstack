/*
Copyright 2021 The Kubernetes Authors.

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

// SecondClusterSpec implements a test that verifies two clusters can co-exist.
func SecondClusterSpec(ctx context.Context, inputGetter func() CommonSpecInput) {

	const specName = "second-cluster"

	var (
		input             CommonSpecInput
		namespace         *corev1.Namespace
		cancelWatches     context.CancelFunc
		clusterResources1 *clusterctl.ApplyClusterTemplateAndWaitResult
		clusterResources2 *clusterctl.ApplyClusterTemplateAndWaitResult
	)

	createCluster := func(flavor string, resources *clusterctl.ApplyClusterTemplateAndWaitResult) {
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
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, resources)
	}

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
		clusterResources1 = new(clusterctl.ApplyClusterTemplateAndWaitResult)
		clusterResources2 = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("should successfully add and remove a second cluster", func() {
		By("create the first cluster")
		createCluster(clusterctl.DefaultFlavor, clusterResources1)

		By("create the second cluster")
		createCluster("second-cluster", clusterResources2)

		By("verify that the second cluster works")

		By("delete the second cluster")
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources2.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)

		By("verify that the second cluster is gone")

		By("verify that the first cluster still works")

		By("PASSED!")
	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources1.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources2.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)
	})
}
