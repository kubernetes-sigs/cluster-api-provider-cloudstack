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

package e2e_api_upgrade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
)

// Test upgrading across API versions.
func APIUpgradeSpec(ctx context.Context, inputGetter func() CommonSpecInput) {
	var (
		specName         = "api-upgrade-test"
		namespace        *corev1.Namespace
		clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
		input            CommonSpecInput
	)

	BeforeEach(func() {
		ctx = context.TODO()

		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		input = inputGetter()
		Expect(input.E2EConfig).ToNot(BeNil(), "Invalid argument. input.E2EConfig can't be nil when calling %s spec", specName)
		Expect(input.ClusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. input.ClusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(input.BootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. input.BootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(input.ArtifactFolder, 0o750)).To(Succeed(), "Invalid argument. input.ArtifactFolder can't be created for %s spec", specName)

		Expect(input.E2EConfig.Variables).To(HaveKey(KubernetesVersion))

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)

        
	})

	It("Should upgrade without a hitch.", func() {
        k8sClient := input.BootstrapClusterProxy.GetClientSet()
        k8sClient.RESTClient().Get()


	    fmt.Println(input.ClusterctlConfigPath)
	    // clusterctl.InitManagementClusterAndWatchControllerLogs(ctx, clusterctl.InitManagementClusterAndWatchControllerLogsInput{
	    // 	ClusterProxy:            input.BootstrapClusterProxy,
	    // 	ClusterctlConfigPath:    input.ClusterctlConfigPath,
	    // 	InfrastructureProviders: input.E2EConfig.InfrastructureProviders(),
	    // 	LogFolder:               filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
	    // }, input.E2EConfig.GetIntervals(input.BootstrapClusterProxy.GetName(), "wait-controllers")...)
        //input.E2EConfig.Providers
        // root := os.Getenv("REPO_ROOT")

        clusterctlInitInput := clusterctl.InitInput{
        	LogFolder:               filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
        	ClusterctlConfigPath:    "",
        	KubeconfigPath:          input.BootstrapClusterProxy.GetKubeconfigPath(),
        	// CoreProvider:            "",
        	// BootstrapProviders:      []string{},
        	// ControlPlaneProviders:   []string{},
            InfrastructureProviders: []string{"cloudstack"},
        }
        clusterctl.Init(ctx, clusterctlInitInput)
	    fmt.Println(input.ClusterctlConfigPath)
        time.Sleep(time.Minute * 6)
		// clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
		// 	ClusterProxy:    input.BootstrapClusterProxy,
		// 	CNIManifestPath: input.E2EConfig.GetVariable(CNIPath),
		// 	ConfigCluster: clusterctl.ConfigClusterInput{
		// 		LogFolder:              filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
		// 		ClusterctlConfigPath:   input.ClusterctlConfigPath,
		// 		KubeconfigPath:         input.BootstrapClusterProxy.GetKubeconfigPath(),
		// 		InfrastructureProvider: clusterctl.DefaultInfrastructureProvider,
		// 		// Flavor:                   "affinity-group-" + affinityType, // TODO: create or pick flavor.
		// 		Namespace:                namespace.Name,
		// 		ClusterName:              fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
		// 		KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
		// 		ControlPlaneMachineCount: pointer.Int64Ptr(1),
		// 		WorkerMachineCount:       pointer.Int64Ptr(1),
		// 	},
		// 	WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
		// 	WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
		// 	WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
		// }, clusterResources)
	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
        fmt.Println(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, 
            clusterResources.Cluster, input.SkipCleanup)
		// dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, 
  //           clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)


        
		// csClient := CreateCloudStackClient(ctx, input.BootstrapClusterProxy.GetKubeconfigPath())
		// err := CheckAffinityGroupsDeleted(csClient, affinityIds)

		var err error
		if err != nil {
			Fail(err.Error())
		}
		By("PASSED!")
	})
}
