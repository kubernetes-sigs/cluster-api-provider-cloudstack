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
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/blang/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

// UnmanagedK8SSpec implements a spec that creates a cluster and checks whether an entry is created in ACS.
func UnmanagedK8SSpec(ctx context.Context, inputGetter func() CommonSpecInput) {
	var (
		specName         = "k8s-unmanaged"
		input            CommonSpecInput
		namespace        *corev1.Namespace
		cancelWatches    context.CancelFunc
		clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
	)

	BeforeEach(func() {
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		input = inputGetter()

		csClient := CreateCloudStackClient(ctx, input.BootstrapClusterProxy.GetKubeconfigPath())
		version, err := GetACSVersion(csClient)

		if err != nil || version == "" {
			Skip("Failed to get CloudStack's version")
		}

		v, err := semver.ParseTolerant(strings.Join(strings.Split(version, ".")[0:3], "."))

		if err != nil {
			Skip("Failed to parse CloudStack version " + version)
		}

		expectedRange, _ := semver.ParseRange(">=4.19.0")

		if !expectedRange(v) {
			Skip("Cloudstack version " + version + " is less than 4.19.")
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

	It("Should create a workload cluster", func() {
		By("Creating a workload cluster")

		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))

		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy:    input.BootstrapClusterProxy,
			CNIManifestPath: input.E2EConfig.GetVariable(CNIPath),
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     input.ClusterctlConfigPath,
				KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   specName,
				Namespace:                namespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		By("checking unmanaged k8s resource is created on ACS")
		// Get details from ACS
		csClient := CreateCloudStackClient(ctx, input.BootstrapClusterProxy.GetKubeconfigPath())
		lkcp := csClient.Kubernetes.NewListKubernetesClustersParams()
		lkcp.SetListall(true)

		clusters, err := csClient.Kubernetes.ListKubernetesClusters(lkcp)

		if err != nil {
			Fail("Failed to get Kubernetes clusters from ACS")
		}

		var cluster *cloudstack.KubernetesCluster

		for _, d := range clusters.KubernetesClusters {
			if strings.HasPrefix(d.Name, fmt.Sprintf("%s - %s", clusterName, clusterName)) {
				cluster = d
			}
		}

		Expect(cluster).ShouldNot(BeNil(), "Couldn't find the external managed kubernetes in ACS")
		Expect(len(cluster.Virtualmachines)).Should(Equal(2), "Expected 2 VMs in the cluster, found %d", len(cluster.Virtualmachines))
		By("PASSED!")
	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)
	})
}
