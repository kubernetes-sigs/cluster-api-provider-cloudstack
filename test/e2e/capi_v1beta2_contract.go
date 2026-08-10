/*
Copyright 2024 The Kubernetes Authors.

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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CAPIV1Beta2ContractSpec verifies CAPC satisfies the CAPI v1beta2 infrastructure contract:
//   - CloudStackCluster.status.initialization.provisioned == true after provisioning
//   - CloudStackMachine.status.initialization.provisioned == true for running machines
//   - CloudStackCluster.status.failureDomains is a JSON array (slice), not an object/map
func CAPIV1Beta2ContractSpec(ctx context.Context, inputGetter func() CommonSpecInput) {
	var (
		specName         = "capi-v1beta2-contract"
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

		namespace, cancelWatches = setupSpecNamespace(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
	})

	It("Should have CAPI v1beta2 contract fields populated after cluster provisioning", func() {
		By("Creating a workload cluster")
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy:    input.BootstrapClusterProxy,
			CNIManifestPath: input.E2EConfig.MustGetVariable(CNIPath),
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     input.ClusterctlConfigPath,
				KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   clusterctl.DefaultFlavor,
				Namespace:                namespace.Name,
				ClusterName:              fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
				KubernetesVersion:        input.E2EConfig.MustGetVariable(KubernetesVersion),
				ControlPlaneMachineCount: ptr.To(int64(1)),
				WorkerMachineCount:       ptr.To(int64(1)),
			},
			WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		mgmtClient := input.BootstrapClusterProxy.GetClient()
		clusterName := clusterResources.Cluster.Name

		By("Verifying CloudStackCluster status.initialization.provisioned == true")
		verifyCloudStackClusterInitialization(ctx, mgmtClient, clusterName, namespace.Name)

		By("Verifying CloudStackCluster status.failureDomains is a JSON array")
		verifyCloudStackClusterFailureDomainsIsArray(ctx, mgmtClient, clusterName, namespace.Name)

		By("Verifying CloudStackMachine status.initialization.provisioned == true for each machine")
		verifyCloudStackMachinesInitialization(ctx, mgmtClient, namespace.Name)

		By("Verifying CloudStackCluster fields survive round-trip conversion through all spoke API versions")
		verifyCloudStackClusterConversionRoundTrip(ctx, mgmtClient, clusterName, namespace.Name)
	})

	It("Should reject invalid objects submitted via spoke API versions, verifying webhook fires through conversion", func() {
		mgmtClient := input.BootstrapClusterProxy.GetClient()

		By("Rejecting a CloudStackCluster with no failure domains submitted as infrastructure.cluster.x-k8s.io/v1beta3")
		verifyWebhookRejectsInvalidCloudStackCluster(ctx, mgmtClient, namespace.Name)

		By("Rejecting a CloudStackMachineTemplate with no offering submitted as infrastructure.cluster.x-k8s.io/v1beta3")
		verifyWebhookRejectsInvalidCloudStackMachineTemplate(ctx, mgmtClient, namespace.Name)
	})

	AfterEach(func() {
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ClusterctlConfigPath, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)
	})
}

// assertInitializationProvisioned asserts that status.initialization.provisioned is true.
// Both CloudStackCluster and CloudStackMachine carry this field under the CAPI v1beta2 contract.
func assertInitializationProvisioned(status map[string]interface{}, subject string) {
	initialization, ok := status["initialization"].(map[string]interface{})
	Expect(ok).To(BeTrue(), "%s status.initialization must be present (CAPI v1beta2 contract)", subject)

	provisioned, ok := initialization["provisioned"].(bool)
	Expect(ok).To(BeTrue(), "%s status.initialization.provisioned must be a boolean", subject)
	Expect(provisioned).To(BeTrue(), "%s status.initialization.provisioned must be true", subject)
}

func verifyCloudStackClusterInitialization(ctx context.Context, mgmtClient client.Client, clusterName, namespace string) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta4",
		Kind:    "CloudStackCluster",
	})
	err := mgmtClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: namespace}, obj)
	Expect(err).NotTo(HaveOccurred(), "fetching CloudStackCluster %s/%s", namespace, clusterName)

	status, ok := obj.Object["status"].(map[string]interface{})
	Expect(ok).To(BeTrue(), "CloudStackCluster %s/%s status must be a map", namespace, clusterName)

	assertInitializationProvisioned(status, fmt.Sprintf("CloudStackCluster %s/%s", namespace, clusterName))
}

func verifyCloudStackClusterFailureDomainsIsArray(ctx context.Context, mgmtClient client.Client, clusterName, namespace string) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta4",
		Kind:    "CloudStackCluster",
	})
	err := mgmtClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: namespace}, obj)
	Expect(err).NotTo(HaveOccurred(), "fetching CloudStackCluster %s/%s", namespace, clusterName)

	status, ok := obj.Object["status"].(map[string]interface{})
	Expect(ok).To(BeTrue(), "CloudStackCluster %s/%s status must be a map", namespace, clusterName)

	fds, present := status["failureDomains"]
	if !present {
		return // omitempty — no failure domains, skip
	}

	_, isArray := fds.([]interface{})
	Expect(isArray).To(BeTrue(),
		"CloudStackCluster status.failureDomains must be a JSON array per CAPI v1beta2 contract, got %T", fds)
}

func verifyCloudStackMachinesInitialization(ctx context.Context, mgmtClient client.Client, namespace string) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta4",
		Kind:    "CloudStackMachineList",
	})
	err := mgmtClient.List(ctx, list, client.InNamespace(namespace))
	Expect(err).NotTo(HaveOccurred(), "listing CloudStackMachines in %s", namespace)
	Expect(list.Items).NotTo(BeEmpty(), "expected at least one CloudStackMachine in %s", namespace)

	for _, machine := range list.Items {
		name := machine.GetName()
		status, ok := machine.Object["status"].(map[string]interface{})
		Expect(ok).To(BeTrue(), "CloudStackMachine %s status must be a map", name)

		assertInitializationProvisioned(status, fmt.Sprintf("CloudStackMachine %s", name))
	}
}

// verifyCloudStackClusterConversionRoundTrip fetches the CloudStackCluster via each spoke
// API version (v1beta3, v1beta2, v1beta1) using the Go client and asserts that key fields
// survive the hub→spoke conversion without corruption.
func verifyCloudStackClusterConversionRoundTrip(ctx context.Context, mgmtClient client.Client, clusterName, namespace string) {
	// Baseline from the hub (v1beta4) — already validated as ready by this point.
	hub := &unstructured.Unstructured{}
	hub.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta4",
		Kind:    "CloudStackCluster",
	})
	err := mgmtClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: namespace}, hub)
	Expect(err).NotTo(HaveOccurred(), "fetching CloudStackCluster hub (v1beta4)")

	hubCPE := hub.Object["spec"].(map[string]interface{})["controlPlaneEndpoint"].(map[string]interface{})
	hubHost := hubCPE["host"].(string)
	hubPort := hubCPE["port"] // float64 from JSON unmarshalling

	for _, version := range []string{"v1beta3", "v1beta2", "v1beta1"} {
		spoke := &unstructured.Unstructured{}
		spoke.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "infrastructure.cluster.x-k8s.io",
			Version: version,
			Kind:    "CloudStackCluster",
		})
		err := mgmtClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: namespace}, spoke)
		Expect(err).NotTo(HaveOccurred(), "fetching CloudStackCluster via %s", version)

		spokeCPE, ok := spoke.Object["spec"].(map[string]interface{})["controlPlaneEndpoint"].(map[string]interface{})
		Expect(ok).To(BeTrue(), "controlPlaneEndpoint must survive conversion to %s", version)
		Expect(spokeCPE["host"]).To(Equal(hubHost),
			"controlPlaneEndpoint.host must survive conversion to %s", version)
		Expect(spokeCPE["port"]).To(Equal(hubPort),
			"controlPlaneEndpoint.port must survive conversion to %s", version)

		// status.ready is present on all spoke versions; must remain true.
		if spokeStatus, ok := spoke.Object["status"].(map[string]interface{}); ok {
			if ready, present := spokeStatus["ready"]; present {
				Expect(ready).To(BeTrue(), "status.ready must be true in %s response", version)
			}
		}
	}
}

// verifyWebhookRejectsInvalidCloudStackCluster submits a CloudStackCluster with no
// failureDomains via the v1beta3 spoke API. The API server converts it to v1beta4 via the
// conversion webhook; the v1beta4 validating webhook must then reject it.
func verifyWebhookRejectsInvalidCloudStackCluster(ctx context.Context, mgmtClient client.Client, namespace string) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "infrastructure.cluster.x-k8s.io/v1beta3",
			"kind":       "CloudStackCluster",
			"metadata": map[string]interface{}{
				"name":      "webhook-reject-" + util.RandomString(6),
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"controlPlaneEndpoint": map[string]interface{}{
					"host": "1.2.3.4",
					"port": int64(6443),
				},
				// failureDomains intentionally omitted — ValidateCreate requires at least one.
			},
		},
	}
	err := mgmtClient.Create(ctx, obj)
	Expect(err).To(HaveOccurred(),
		"CloudStackCluster with no failureDomains submitted as v1beta3 must be rejected")
	Expect(err.Error()).To(ContainSubstring("failureDomains"),
		"rejection error must identify the failing field")
}

// verifyWebhookRejectsInvalidCloudStackMachineTemplate submits a CloudStackMachineTemplate
// with no offering via the v1beta3 spoke API. After hub conversion, the v1beta4 validating
// webhook must reject it because Offering.Name and Offering.ID are both empty.
func verifyWebhookRejectsInvalidCloudStackMachineTemplate(ctx context.Context, mgmtClient client.Client, namespace string) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "infrastructure.cluster.x-k8s.io/v1beta3",
			"kind":       "CloudStackMachineTemplate",
			"metadata": map[string]interface{}{
				"name":      "webhook-reject-" + util.RandomString(6),
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						// offering omitted → EnsureAtLeastOneFieldExists fires on Offering.
						"template": map[string]interface{}{
							"name": "some-template",
						},
					},
				},
			},
		},
	}
	err := mgmtClient.Create(ctx, obj)
	Expect(err).To(HaveOccurred(),
		"CloudStackMachineTemplate with no offering submitted as v1beta3 must be rejected")
	Expect(err.Error()).To(ContainSubstring("offering"),
		"rejection error must identify the failing field")
}
