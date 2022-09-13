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

package e2eapiupgrade_test

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
)

// Test upgrading across API versions.
var _ = Describe("Upgrade Testing-ness", func() {
	var (
		specName              = "api-upgrade-test"
        // TODO: actually use these to create cluster resources.
		// namespace             *corev1.Namespace
		// clusterResources      *clusterctl.ApplyClusterTemplateAndWaitResult
		// cancelWatches         context.CancelFunc
        // skipCleanup = true
	)

	BeforeEach(func() {
        fmt.Printf("%+v\n", ctx)

		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		Expect(e2eConfig).ToNot(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(clusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. ClusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(bootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. BootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(artifactFolder, 0o750)).To(Succeed(), "Invalid argument. artifactFolder can't be created for %s spec", specName)
		Expect(e2eConfig.Variables).To(HaveKey(KubernetesVersion))

		// // Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		// namespace, cancelWatches = setupSpecNamespace(ctx, specName, bootstrapClusterProxy, artifactFolder)
		// clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("Should upgrade without a hitch.", func() {
		k8sClient := bootstrapClusterProxy.GetClientSet()
		k8sClient.RESTClient().Get()


        // Initialize with CAPC at v0.4.4.
		clusterctlInitInput := clusterctl.InitInput{
			LogFolder:               filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
			ClusterctlConfigPath:    "",
			KubeconfigPath:          bootstrapClusterProxy.GetKubeconfigPath(),
            InfrastructureProviders: []string{"cloudstack:v0.4.4"},
		}
		clusterctl.Init(ctx, clusterctlInitInput)

        // TODO: Apply v1beta1 types to cluster.


        // Upgrade to local latest.
        clusterctlUpgradeInput := clusterctl.UpgradeInput{
	    	LogFolder:               filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
	    	ClusterctlConfigPath:    clusterctlConfigPath,
			KubeconfigPath:          bootstrapClusterProxy.GetKubeconfigPath(),
        	Contract:             "v1beta2",
        }
		clusterctl.Upgrade(ctx, clusterctlUpgradeInput)

        // TODO: Verify upgrade did not cause any VM replacements and the resulting v1beta2 custom resource objects are as desired.
	})

	AfterEach(func() {
        // TODO: cleanup any remaining resources.
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		// dumpSpecResourcesAndCleanup(ctx, specName, bootstrapClusterProxy, artifactFolder, namespace, cancelWatches,
		//           clusterResources.Cluster, e2eConfig.GetIntervals, skipCleanup)

        // TODO: Remove this or use the csClient to verify nothing major has changed.
		// csClient := CreateCloudStackClient(ctx, BootstrapClusterProxy.GetKubeconfigPath())
		// err := CheckAffinityGroupsDeleted(csClient, affinityIds)

		var err error
		if err != nil {
			Fail(err.Error())
		}
		By("PASSED!")
	})
})


// TODO: Use this to apply cluster spec to CAPC.
// clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
// 	ClusterProxy:    bootstrapClusterProxy,
// 	CNIManifestPath: e2eConfig.GetVariable(CNIPath),
// 	ConfigCluster: clusterctl.ConfigClusterInput{
// 		LogFolder:              filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
// 		ClusterctlConfigPath:   clusterctlConfigPath,
// 		KubeconfigPath:         bootstrapClusterProxy.GetKubeconfigPath(),
// 		InfrastructureProvider: clusterctl.DefaultInfrastructureProvider,
// 		// Flavor:                   "affinity-group-" + affinityType, // TODO: create or pick flavor.
// 		Namespace:                namespace.Name,
// 		ClusterName:              fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
// 		KubernetesVersion:        e2eConfig.GetVariable(KubernetesVersion),
// 		ControlPlaneMachineCount: pointer.Int64Ptr(1),
// 		WorkerMachineCount:       pointer.Int64Ptr(1),
// 	},
// 	WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
// 	WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
// 	WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
// }, clusterResources)
