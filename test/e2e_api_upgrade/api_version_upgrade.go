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
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
)

// Test upgrading across API versions.
var _ = Describe("Upgrade Testing-ness", func() {
	var (
		specName              = "api-upgrade-test"
		namespace             *corev1.Namespace
		clusterResources      *clusterctl.ApplyClusterTemplateAndWaitResult
		cancelWatches         context.CancelFunc
		E2EConfig             *clusterctl.E2EConfig
		ClusterctlConfigPath  string
		BootstrapClusterProxy framework.ClusterProxy
		ArtifactFolder        string
		SkipCleanup           bool
	)

	BeforeEach(func() {
		E2EConfig = &clusterctl.E2EConfig{

			// Providers is a list of providers to be configured in the local repository that will be created for the e2e test.
			// It is required to provide following providers
			// - cluster-api
			// - bootstrap kubeadm
			// - control-plane kubeadm
			// - one infrastructure provider

		}
		// // ProviderConfig describes a provider to be configured in the local repository that will be created for the e2e test.
		// type ProviderConfig struct {
		// 	// Name is the name of the provider.
		// 	Name string `json:"name"`
		//
		// 	// Type is the type of the provider.
		// 	Type string `json:"type"`
		//
		// 	// Versions is a list of component YAML to be added to the local repository, one for each release.
		// 	// Please note that the first source will be used a a default release for this provider.
		// 	Versions []ProviderVersionSource `json:"versions,omitempty"`
		//
		// 	// Files is a list of files to be copied into the local repository for all the releases.
		// 	Files []Files `json:"files,omitempty"`
		// }

		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		Expect(E2EConfig).ToNot(BeNil(), "Invalid argument. E2EConfig can't be nil when calling %s spec", specName)
		Expect(clusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. ClusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(bootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. BootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(artifactFolder, 0o750)).To(Succeed(), "Invalid argument. ArtifactFolder can't be created for %s spec", specName)

		// E2EConfig.Variables[KubernetesVersion] =
		// Expect(E2EConfig.Variables).To(HaveKey(KubernetesVersion))

		// // Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		// namespace, cancelWatches = setupSpecNamespace(ctx, specName, BootstrapClusterProxy, ArtifactFolder)
		// clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("Should upgrade without a hitch.", func() {
		// k8sClient := BootstrapClusterProxy.GetClientSet()
		// k8sClient.RESTClient().Get()
		//
		//
		fmt.Println(ClusterctlConfigPath)
		// clusterctl.InitManagementClusterAndWatchControllerLogs(ctx, clusterctl.InitManagementClusterAndWatchControllerLogsInput{
		// 	ClusterProxy:            BootstrapClusterProxy,
		// 	ClusterctlConfigPath:    ClusterctlConfigPath,
		// 	InfrastructureProviders: E2EConfig.InfrastructureProviders(),
		// 	LogFolder:               filepath.Join(ArtifactFolder, "clusters", BootstrapClusterProxy.GetName()),
		// }, E2EConfig.GetIntervals(BootstrapClusterProxy.GetName(), "wait-controllers")...)
		// E2EConfig.Providers
		// root := os.Getenv("REPO_ROOT")

		// clusterctlInitInput := clusterctl.InitInput{
		// 	LogFolder:               filepath.Join(ArtifactFolder, "clusters", BootstrapClusterProxy.GetName()),
		// 	ClusterctlConfigPath:    "",
		// 	KubeconfigPath:          BootstrapClusterProxy.GetKubeconfigPath(),
		// 	// CoreProvider:            "",
		// 	// BootstrapProviders:      []string{},
		// 	// ControlPlaneProviders:   []string{},
		//     InfrastructureProviders: []string{"cloudstack"},
		// }
		// clusterctl.Init(ctx, clusterctlInitInput)
		fmt.Println(ClusterctlConfigPath)
		time.Sleep(time.Minute * 6)
		// clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
		// 	ClusterProxy:    BootstrapClusterProxy,
		// 	CNIManifestPath: E2EConfig.GetVariable(CNIPath),
		// 	ConfigCluster: clusterctl.ConfigClusterInput{
		// 		LogFolder:              filepath.Join(ArtifactFolder, "clusters", BootstrapClusterProxy.GetName()),
		// 		ClusterctlConfigPath:   ClusterctlConfigPath,
		// 		KubeconfigPath:         BootstrapClusterProxy.GetKubeconfigPath(),
		// 		InfrastructureProvider: clusterctl.DefaultInfrastructureProvider,
		// 		// Flavor:                   "affinity-group-" + affinityType, // TODO: create or pick flavor.
		// 		Namespace:                namespace.Name,
		// 		ClusterName:              fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
		// 		KubernetesVersion:        E2EConfig.GetVariable(KubernetesVersion),
		// 		ControlPlaneMachineCount: pointer.Int64Ptr(1),
		// 		WorkerMachineCount:       pointer.Int64Ptr(1),
		// 	},
		// 	WaitForClusterIntervals:      E2EConfig.GetIntervals(specName, "wait-cluster"),
		// 	WaitForControlPlaneIntervals: E2EConfig.GetIntervals(specName, "wait-control-plane"),
		// 	WaitForMachineDeployments:    E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
		// }, clusterResources)
	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		fmt.Println(ctx, specName, BootstrapClusterProxy, ArtifactFolder, namespace, cancelWatches,
			clusterResources.Cluster, SkipCleanup)
		// dumpSpecResourcesAndCleanup(ctx, specName, BootstrapClusterProxy, ArtifactFolder, namespace, cancelWatches,
		//           clusterResources.Cluster, E2EConfig.GetIntervals, SkipCleanup)

		// csClient := CreateCloudStackClient(ctx, BootstrapClusterProxy.GetKubeconfigPath())
		// err := CheckAffinityGroupsDeleted(csClient, affinityIds)

		var err error
		if err != nil {
			Fail(err.Error())
		}
		By("PASSED!")
	})
})
