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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

// InvalidDomainSpec implements a test that verifies that creating a new cluster fails when the specified Domain does not exist
func InvalidDomainSpec(ctx context.Context, inputGetter func() CommonSpecInput) {
	var (
		specName         = "invalid-domain"
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

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("Should fail due to the specified domain is not found", func() {
		logFolder := filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName())
		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))

		By("Configuring a cluster")
		workloadClusterTemplate := clusterctl.ConfigCluster(ctx, clusterctl.ConfigClusterInput{
			// pass reference to the management cluster hosting this test
			KubeconfigPath: input.BootstrapClusterProxy.GetKubeconfigPath(),
			// pass the clusterctl config file that points to the local provider repository created for this test,
			ClusterctlConfigPath: input.ClusterctlConfigPath,
			// select template
			Flavor: "invalid-domain",
			// define template variables
			Namespace:                namespace.Name,
			ClusterName:              clusterName,
			KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
			ControlPlaneMachineCount: pointer.Int64Ptr(1),
			WorkerMachineCount:       pointer.Int64Ptr(1),
			InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
			// setup clusterctl logs folder
			LogFolder: logFolder,
		})

		By("Applying...")
		Expect(input.BootstrapClusterProxy.Apply(ctx, workloadClusterTemplate)).ShouldNot(HaveOccurred())

		clusterResources.Cluster = framework.GetClusterByName(ctx, framework.GetClusterByNameInput{
			Getter:    input.BootstrapClusterProxy.GetClient(),
			Name:      clusterName,
			Namespace: namespace.Name,
		})

		By("Waiting for 'domain not found' error to occur")
		Eventually(func() (string, error) {
			err := filepath.Walk(logFolder, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if strings.Contains(path, "capc-controller-manager") && strings.Contains(path, "manager.log") {
					log, _ := os.ReadFile(path)
					if strings.Contains(string(log), "No match found for "+input.E2EConfig.GetVariable(InvalidDomainName)) {
						By("Found 'domain not found' error")
						return errors.New("expected error found")
					}
				}
				return nil
			})
			if err == nil {
				return "expected error not found", nil
			} else {
				return err.Error(), nil
			}
		}, input.E2EConfig.GetIntervals(specName, "wait-errors")...).Should(Equal(string("expected error found")))

		By("PASSED!")
	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)
	})
}
