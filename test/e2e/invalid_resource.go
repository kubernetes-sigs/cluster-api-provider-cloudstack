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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var (
	specName         = "invalid-resource"
	input            CommonSpecInput
	namespace        *corev1.Namespace
	cancelWatches    context.CancelFunc
	clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
)

// InvalidResourceSpec implements a test that verifies that creating a new cluster fails when the specified resource does not exist
func InvalidResourceSpec(ctx context.Context, inputGetter func() CommonSpecInput) {
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

	It("Should fail due to the specified account is not found [TC4a]", func() {
		testInvalidResource(ctx, input, "invalid-account", "could not find account "+input.E2EConfig.GetVariable(InvalidAccountName))
	})

	It("Should fail due to the specified domain is not found [TC4b]", func() {
		testInvalidResource(ctx, input, "invalid-domain", "domain not found for domain path")
	})

	It("Should fail due to the specified control plane offering is not found [TC7]", func() {
		testInvalidResource(ctx, input, "invalid-cp-offering", "No match found for "+input.E2EConfig.GetVariable(InvalidCPOfferingName))
	})

	It("Should fail due to the specified template is not found [TC6]", func() {
		testInvalidResource(ctx, input, "invalid-template", "No match found for "+input.E2EConfig.GetVariable(InvalidTemplateName))
	})

	It("Should fail due to the specified zone is not found [TC3]", func() {
		testInvalidResource(ctx, input, "invalid-zone", "No match found for "+input.E2EConfig.GetVariable(InvalidZoneName))
	})

	It("Should fail due to the specified disk offering is not found", func() {
		testInvalidResource(ctx, input, "invalid-disk-offering", "could not get DiskOffering ID from "+input.E2EConfig.GetVariable(InvalidDiskOfferingName))
	})

	It("Should fail due to the compute resources are not sufficient for the specified offering [TC8]", func() {
		testInvalidResource(ctx, input, "insufficient-compute-resources", "Unable to create a deployment for VM")
	})

	It("Should fail due to the specified disk offer is not customized but the disk size is specified", func() {
		testInvalidResource(ctx, input, "invalid-disk-offering-size-for-non-customized", "is not customized, disk size can not be specified")
	})

	It("Should fail due to the specified disk offer is customized but the disk size is not specified", func() {
		testInvalidResource(ctx, input, "invalid-disk-offering-size-for-customized", "is customized, disk size can not be 0 GB")
	})

	It("Should fail due to the public IP can not be found", func() {
		testInvalidResource(ctx, input, "invalid-ip", "no public addresses found in available networks")
	})

	Context("When starting with a healthy cluster", func() {
		var logFolder string

		BeforeEach(func() {
			logFolder = generateLogFolderPath()

			By("Creating a workload cluster")
			clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
				ClusterProxy:    input.BootstrapClusterProxy,
				CNIManifestPath: input.E2EConfig.GetVariable(CNIPath),
				ConfigCluster: clusterctl.ConfigClusterInput{
					LogFolder:                logFolder,
					ClusterctlConfigPath:     input.ClusterctlConfigPath,
					KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
					InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
					Flavor:                   "insufficient-compute-resources-for-upgrade",
					Namespace:                namespace.Name,
					ClusterName:              generateClusterName(),
					KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
					ControlPlaneMachineCount: pointer.Int64Ptr(1),
					WorkerMachineCount:       pointer.Int64Ptr(1),
				},
				WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
				WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
				WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
			}, clusterResources)
		})

		It("Should fail to upgrade worker machine due to insufficient compute resources", func() {
			By("Making sure the expected error didn't occur yet")
			expectedError := "Unable to create a deployment for VM"
			Expect(errorExistsInLog(logFolder, expectedError)).To(BeFalse())

			By("Increasing the machine deployment instance size")
			deployment := clusterResources.MachineDeployments[0]
			deployment.Spec.Template.Spec.InfrastructureRef.Name =
				strings.Replace(deployment.Spec.Template.Spec.InfrastructureRef.Name, "-md-0", "-upgrade-md-0", 1)
			upgradeMachineDeploymentInfrastructureRef(ctx, deployment)

			By("Checking for the expected error")
			waitForErrorInLog(logFolder, expectedError)
		})

		It("Should fail to upgrade control plane machine due to insufficient compute resources", func() {
			By("Making sure the expected error didn't occur yet")
			expectedError := "Unable to create a deployment for VM"
			Expect(errorExistsInLog(logFolder, expectedError)).To(BeFalse())

			By("Increasing the machine deployment instance size")
			cp := clusterResources.ControlPlane
			cp.Spec.MachineTemplate.InfrastructureRef.Name =
				strings.Replace(cp.Spec.MachineTemplate.InfrastructureRef.Name, "-control-plane", "-upgrade-control-plane", 1)
			upgradeControlPlaneInfrastructureRef(ctx, cp)

			By("Checking for the expected error")
			waitForErrorInLog(logFolder, expectedError)
		})
	})

	AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)
	})

}

func testInvalidResource(ctx context.Context, input CommonSpecInput, flavor string, expectedError string) {
	logFolder := generateLogFolderPath()
	clusterName := generateClusterName()

	By("Configuring a cluster")
	workloadClusterTemplate := clusterctl.ConfigCluster(ctx, clusterctl.ConfigClusterInput{
		KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
		ClusterctlConfigPath:     input.ClusterctlConfigPath,
		Flavor:                   flavor,
		Namespace:                namespace.Name,
		ClusterName:              clusterName,
		KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
		ControlPlaneMachineCount: pointer.Int64Ptr(1),
		WorkerMachineCount:       pointer.Int64Ptr(1),
		InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
		LogFolder:                logFolder,
	})

	By("Applying...")
	Expect(input.BootstrapClusterProxy.Apply(ctx, workloadClusterTemplate)).ShouldNot(HaveOccurred())

	clusterResources.Cluster = framework.GetClusterByName(ctx, framework.GetClusterByNameInput{
		Getter:    input.BootstrapClusterProxy.GetClient(),
		Name:      clusterName,
		Namespace: namespace.Name,
	})

	waitForErrorInLog(logFolder, expectedError)

	By("PASSED!")
}

func generateLogFolderPath() string {
	return filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName())
}

func generateClusterName() string {
	return fmt.Sprintf("%s-%s", specName, util.RandomString(6))
}

// errorExistsInLog looks for a specific error message in the CAPC controller log files.  Because the logs may contain
// entries from previous test runs, or from previous tests in the same run, the function also requires that the log
// line contains the namespace and cluster names.
func errorExistsInLog(logFolder string, expectedError string) (bool, error) {
	expectedErrorFound := errors.New("expected error found")
	controllerLogPath := filepath.Join(logFolder, "controllers", "capc-controller-manager")

	err := filepath.Walk(controllerLogPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, "manager.log") {
			log, _ := os.ReadFile(path)
			logLines := strings.Split(string(log), "\n")
			for _, line := range logLines {
				if strings.Contains(line, expectedError) &&
					strings.Contains(line, clusterResources.Cluster.Namespace) {
					Byf("Found %q error", expectedError)
					return expectedErrorFound
				}
			}
		}

		return nil
	})

	if err == nil {
		return false, nil
	} else if err == expectedErrorFound {
		return true, nil
	}
	return false, err
}

func waitForErrorInLog(logFolder string, expectedError string) {
	Byf("Waiting for %q error to occur", expectedError)
	Eventually(func() (bool, error) {
		return errorExistsInLog(logFolder, expectedError)
	}, input.E2EConfig.GetIntervals(specName, "wait-errors")...).Should(BeTrue())
}

// upgradeMachineDeploymentInfrastructureRef updates a machine deployment infrastructure ref and returns immediately.
// The logic was borrowed from framework.UpgradeMachineDeploymentInfrastructureRefAndWait.
func upgradeMachineDeploymentInfrastructureRef(ctx context.Context, deployment *v1beta1.MachineDeployment) {
	By("Patching the machine deployment infrastructure ref")
	mgmtClient := input.BootstrapClusterProxy.GetClient()

	// Create a new infrastructure ref based on the existing one
	infraRef := deployment.Spec.Template.Spec.InfrastructureRef
	newInfraObjName := createNewInfrastructureRef(ctx, infraRef)

	// Patch the new infra object's ref to the machine deployment
	patchHelper, err := patch.NewHelper(deployment, mgmtClient)
	Expect(err).ToNot(HaveOccurred())
	infraRef.Name = newInfraObjName
	deployment.Spec.Template.Spec.InfrastructureRef = infraRef
	Expect(patchHelper.Patch(ctx, deployment)).To(Succeed())
}

// upgradeControlPlane upgrades a control plane deployment infrastructure ref and returns immediately.
func upgradeControlPlaneInfrastructureRef(ctx context.Context, controlPlane *controlplanev1.KubeadmControlPlane) {
	By("Patching the control plane infrastructure ref")
	mgmtClient := input.BootstrapClusterProxy.GetClient()

	// Create a new infrastructure ref based on the existing one
	infraRef := controlPlane.Spec.MachineTemplate.InfrastructureRef
	newInfraObjName := createNewInfrastructureRef(ctx, infraRef)

	// Patch the control plane to use the new infrastructure ref
	patchHelper, err := patch.NewHelper(controlPlane, mgmtClient)
	Expect(err).ToNot(HaveOccurred())
	infraRef.Name = newInfraObjName
	controlPlane.Spec.MachineTemplate.InfrastructureRef = infraRef
	Expect(patchHelper.Patch(ctx, controlPlane)).To(Succeed())
}

// createNewInfrastructureRef creates a new infrastructure ref that's based on an existing one, but has a new name.  The
// new name is returned.
func createNewInfrastructureRef(ctx context.Context, sourceInfrastructureRef corev1.ObjectReference) string {
	mgmtClient := input.BootstrapClusterProxy.GetClient()

	// Retrieve the existing infrastructure ref object
	infraObj := &unstructured.Unstructured{}
	infraObj.SetGroupVersionKind(sourceInfrastructureRef.GroupVersionKind())
	key := client.ObjectKey{
		Namespace: clusterResources.Cluster.Namespace,
		Name:      sourceInfrastructureRef.Name,
	}
	Expect(mgmtClient.Get(ctx, key, infraObj)).NotTo(HaveOccurred())

	// Creates a new infrastructure ref object
	newInfraObj := infraObj
	newInfraObjName := fmt.Sprintf("%s-%s", sourceInfrastructureRef.Name, util.RandomString(6))
	newInfraObj.SetName(newInfraObjName)
	newInfraObj.SetResourceVersion("")
	Expect(mgmtClient.Create(ctx, newInfraObj)).NotTo(HaveOccurred())
	return newInfraObjName
}
