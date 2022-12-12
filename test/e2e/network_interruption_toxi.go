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

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api-provider-cloudstack-staging/test/e2e/helpers"
	"sigs.k8s.io/cluster-api-provider-cloudstack-staging/test/e2e/toxiproxy"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

// NetworkInterruptionToxiSpec implements a test that verifies that an app deployed to the workload cluster works.
func NetworkInterruptionToxiSpec(ctx context.Context, inputGetter func() CommonSpecInput) {
	var (
		specName                          = "network-interruption-toxi"
		input                             CommonSpecInput
		namespace                         *corev1.Namespace
		cancelWatches                     context.CancelFunc
		clusterResources                  *clusterctl.ApplyClusterTemplateAndWaitResult
		cloudStackToxiProxyContext        *toxiproxy.Context
		clusterName                       = fmt.Sprintf("%s-%s", specName, util.RandomString(6))
		networkInterruptorShutdownChannel = make(chan bool, 2)
	)

	BeforeEach(func() {
		// ToxiProxy running in a docker container requires docker host networking, only available in linux.
		Expect(runtime.GOOS).To(Equal("linux"))

		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		input = inputGetter()
		Expect(input.E2EConfig).ToNot(BeNil(), "Invalid argument. input.E2EConfig can't be nil when calling %s spec", specName)
		Expect(input.ClusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. input.ClusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(input.BootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. input.BootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(input.ArtifactFolder, 0750)).To(Succeed(), "Invalid argument. input.ArtifactFolder can't be created for %s spec", specName)
		Expect(input.E2EConfig.Variables).To(HaveKey(KubernetesVersion))

		// Set up a toxiProxy for CloudStack
		cloudStackToxiProxyContext = toxiproxy.SetupForToxiProxyTestingACS(
			ctx,
			clusterName,
			input.BootstrapClusterProxy,
			input.E2EConfig,
			input.ClusterctlConfigPath,
		)

		// Set up a Namespace to host objects for this spec and create a watcher for the namespace events.
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)

	})

	It("Should be able to create a cluster despite a network interruption during that process", func() {
		By("Creating a workload cluster")

		flavor := clusterctl.DefaultFlavor
		if input.Flavor != nil {
			flavor = *input.Flavor
		}
		namespace := namespace.Name

		// While I'd prefer to closely synchronize the network interruption (ToxiProxy disable) to a particular point in the cluster provisioning
		//  process, doing so for this asynchronously running process would be harder and more impactful than I can tackle right now.  So I'm going
		//  to give CAPC a short period to get started with the provisioning, and then interrupt the network for a fixed time, and then restore it.
		//  CAPC should tolerate this and ultimately succeed.
		// To do this while ApplyClusterTemplateAndWait() is waiting, I'm going to use a concurrent goroutine and an interruptible version of sleep
		//  so it can be shut down cleanly.
		go networkInterruptor(cloudStackToxiProxyContext, networkInterruptorShutdownChannel)

		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy:    input.BootstrapClusterProxy,
			CNIManifestPath: input.E2EConfig.GetVariable(CNIPath),
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     cloudStackToxiProxyContext.ConfigPath,
				KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   flavor,
				Namespace:                namespace,
				ClusterName:              clusterName,
				KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(2),
			},
			WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		By("PASSED!")
	})

	AfterEach(func() {
		// Stop the networkInterruptor (in case it's still running because tests failed before it completed)
		networkInterruptorShutdownChannel <- true

		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)

		// Tear down the ToxiProxies
		toxiproxy.TearDownToxiProxyACS(ctx, input.BootstrapClusterProxy, cloudStackToxiProxyContext)
	})
}

func networkInterruptor(toxiProxyContext *toxiproxy.Context, shutdownChannel chan bool) {
	// Wait for ApplyClusterTemplateAndWait() to make some progress
	helpers.InterruptibleSleep(15*time.Second, time.Second, shutdownChannel)

	// Disable communications to ACS
	toxiProxyContext.Disable()

	// Leave the network disabled for some period of time
	helpers.InterruptibleSleep(30*time.Second, time.Second, shutdownChannel)

	// Restore communications to ACS
	toxiProxyContext.Enable()
}
