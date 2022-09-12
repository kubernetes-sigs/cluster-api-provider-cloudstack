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
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/blang/semver"
	. "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/exec"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Test suite constants for e2e config variables.
const (
	KubernetesVersionManagement = "KUBERNETES_VERSION_MANAGEMENT"
	KubernetesVersion           = "KUBERNETES_VERSION"
	CNIPath                     = "CNI"
	CNIResources                = "CNI_RESOURCES"
	IPFamily                    = "IP_FAMILY"
	InvalidZoneName             = "CLOUDSTACK_INVALID_ZONE_NAME"
	InvalidDiskOfferingName     = "CLOUDSTACK_INVALID_DISK_OFFERING_NAME"
	InvalidNetworkName          = "CLOUDSTACK_INVALID_NETWORK_NAME"
	InvalidAccountName          = "CLOUDSTACK_INVALID_ACCOUNT_NAME"
	InvalidDomainName           = "CLOUDSTACK_INVALID_DOMAIN_NAME"
	InvalidTemplateName         = "CLOUDSTACK_INVALID_TEMPLATE_NAME"
	InvalidCPOfferingName       = "CLOUDSTACK_INVALID_CONTROL_PLANE_MACHINE_OFFERING"
)

const (
	ControlPlaneIndicator      = "control-plane"
	MachineDeploymentIndicator = "md"
	DataVolumePrefix           = "DATA-"
)

func Byf(format string, a ...interface{}) {
	By(fmt.Sprintf(format, a...))
}

func setupSpecNamespace(ctx context.Context, specName string, clusterProxy framework.ClusterProxy, artifactFolder string) (*corev1.Namespace, context.CancelFunc) {
	Byf("Creating a namespace for hosting the %q test spec", specName)
	namespace, cancelWatches := framework.CreateNamespaceAndWatchEvents(ctx, framework.CreateNamespaceAndWatchEventsInput{
		Creator:   clusterProxy.GetClient(),
		ClientSet: clusterProxy.GetClientSet(),
		Name:      fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
		LogFolder: filepath.Join(artifactFolder, "clusters", clusterProxy.GetName()),
	})

	return namespace, cancelWatches
}

func dumpSpecResourcesAndCleanup(ctx context.Context, specName string, clusterProxy framework.ClusterProxy, artifactFolder string, namespace *corev1.Namespace, cancelWatches context.CancelFunc, cluster *clusterv1.Cluster, intervalsGetter func(spec, key string) []interface{}, skipCleanup bool) {
	Byf("Dumping logs from the %q workload cluster", cluster.Name)

	// Dump all the logs from the workload cluster before deleting them.
	clusterProxy.CollectWorkloadClusterLogs(ctx, cluster.Namespace, cluster.Name, filepath.Join(artifactFolder, "clusters", cluster.Name))

	Byf("Dumping all the Cluster API resources in the %q namespace", namespace.Name)

	// Dump all Cluster API related resources to artifacts before deleting them.
	framework.DumpAllResources(ctx, framework.DumpAllResourcesInput{
		Lister:    clusterProxy.GetClient(),
		Namespace: namespace.Name,
		LogPath:   filepath.Join(artifactFolder, "clusters", clusterProxy.GetName(), "resources"),
	})

	if !skipCleanup {
		Byf("Deleting cluster %s/%s", cluster.Namespace, cluster.Name)
		// While https://github.com/kubernetes-sigs/cluster-api/issues/2955 is addressed in future iterations, there is a chance
		// that cluster variable is not set even if the cluster exists, so we are calling DeleteAllClustersAndWait
		// instead of DeleteClusterAndWait
		framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
			Client:    clusterProxy.GetClient(),
			Namespace: namespace.Name,
		}, intervalsGetter(specName, "wait-delete-cluster")...)

		Byf("Deleting namespace used for hosting the %q test spec", specName)
		framework.DeleteNamespace(ctx, framework.DeleteNamespaceInput{
			Deleter: clusterProxy.GetClient(),
			Name:    namespace.Name,
		})
	}
	cancelWatches()
}

// HaveValidVersion succeeds if version is a valid semver version.
func HaveValidVersion(version string) types.GomegaMatcher {
	return &validVersionMatcher{version: version}
}

type validVersionMatcher struct{ version string }

func (m *validVersionMatcher) Match(actual interface{}) (success bool, err error) {
	if _, err := semver.ParseTolerant(m.version); err != nil {
		return false, err
	}
	return true, nil
}

func (m *validVersionMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n%s\n%s", m.version, " to be a valid version ")
}

func (m *validVersionMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n%s\n%s", m.version, " not to be a valid version ")
}

func KubectlExec(ctx context.Context, command string, kubeconfigPath string, args ...string) (string, error) {
	execArgs := append([]string{command, "--kubeconfig", kubeconfigPath}, args...)
	runCmd := exec.NewCommand(
		exec.WithCommand("kubectl"),
		exec.WithArgs(execArgs...),
	)
	stdout, stderr, err := runCmd.Run(ctx)
	if err != nil {
		fmt.Println(string(stderr))
		return "", err
	}
	return string(stdout), nil
}

func GetK8sObject(ctx context.Context, resourceType, name, namespace, kubeconfigPath string, obj runtime.Object) error {
	getArgs := []string{"--ignore-not-found", "--namespace", namespace, resourceType, name, "-o", "json"}
	stdOut, err := KubectlExec(ctx, "get", kubeconfigPath, getArgs...)
	if err != nil {
		return fmt.Errorf("getting %s/%s/%s with kubectl: %v", resourceType, namespace, name, err)
	}
	if len(stdOut) == 0 {
		return fmt.Errorf("not found %s/%s/%s", resourceType, namespace, name)
	}
	if err = json.Unmarshal([]byte(stdOut), obj); err != nil {
		return fmt.Errorf("parsing %s/%s/%s response: %v", resourceType, namespace, name, err)
	}

	return nil
}

func CheckAffinityGroupsDeleted(client *cloudstack.CloudStackClient, affinityIds []string) error {
	if len(affinityIds) == 0 {
		return errors.New("affinityIds are empty")
	}

	for _, affinityId := range affinityIds {
		affinity, count, _ := client.AffinityGroup.GetAffinityGroupByID(affinityId)
		if count > 0 {
			return errors.New("Affinity group " + affinity.Name + " still exists")
		}
	}
	return nil
}

func CheckNetworkExists(client *cloudstack.CloudStackClient, networkName string) (bool, error) {
	_, count, err := client.Network.GetNetworkByName(networkName)
	if err != nil {
		if strings.Contains(err.Error(), "No match found for") {
			return false, nil
		}
		return false, err
	} else if count > 1 {
		return false, fmt.Errorf("Expected 0-1 Network with name %s, but got %d.", networkName, count)
	}
	return count == 1, nil
}

func CreateCloudStackClient(ctx context.Context, kubeConfigPath string) *cloudstack.CloudStackClient {
	By("Getting a CloudStack client secret")
	secret := &corev1.Secret{}
	name := "secret1"
	namepace := "default"
	if err := GetK8sObject(ctx, "secret", name, namepace, kubeConfigPath, secret); err != nil {
		Fail("Failed to get secret: " + err.Error())
	}

	By("Creating a CloudStack client")
	apiURL := string(secret.Data["api-url"])
	apiKey := string(secret.Data["api-key"])
	secretKey := string(secret.Data["secret-key"])
	verifySSL := string(secret.Data["verify-ssl"])
	if apiURL == "" || apiKey == "" || secretKey == "" {
		Fail(fmt.Sprintf("Invalid secret: %+v, %s, %s, %s", secret.Data, apiURL, apiKey, secretKey))
	}
	fmt.Sprintf("from secret: %s, %s, %s", apiURL, apiKey, secretKey)

	return cloudstack.NewClient(apiURL, apiKey, secretKey, strings.ToLower(verifySSL) == "true")
}

func checkVMHostAssignments(vm *cloudstack.VirtualMachine, cpHostIdSet map[string]bool, mdHostIdSet map[string]bool, affinityType string) error {
	if strings.Contains(vm.Name, ControlPlaneIndicator) {
		if len(cpHostIdSet) > 0 {
			_, ok := cpHostIdSet[vm.Hostid]
			if affinityType == "pro" && !ok {
				return errors.New(vm.Name + " is deployed in a different host: " + vm.Hostname + " when affinity type is " + affinityType)
			}
			if affinityType == "anti" && ok {
				return errors.New(vm.Name + " is deployed in the same host: " + vm.Hostname + " when affinity type is " + affinityType)
			}
		}
		cpHostIdSet[vm.Hostid] = true
	}
	if strings.Contains(vm.Name, MachineDeploymentIndicator) {
		if len(mdHostIdSet) > 0 {
			_, ok := mdHostIdSet[vm.Hostid]
			if affinityType == "pro" && !ok {
				return errors.New(vm.Name + " is deployed in a different host: " + vm.Hostname + " when affinity type is " + affinityType)
			}
			if affinityType == "anti" && ok {
				return errors.New(vm.Name + " is deployed in the same host: " + vm.Hostname + " when affinity type is " + affinityType)
			}
		}
		mdHostIdSet[vm.Hostid] = true
	}
	return nil
}

func WaitForHealthyMachineCount(ctx context.Context, mgmtClient client.Client, workloadClient client.Client, cluster *clusterv1.Cluster, mhcMatcher string, healthyMachineCount int, intervals []interface{}) {
	machineHealthChecks := framework.GetMachineHealthChecksForCluster(ctx, framework.GetMachineHealthChecksForClusterInput{
		Lister:      mgmtClient,
		ClusterName: cluster.Name,
		Namespace:   cluster.Namespace,
	})

	for _, mhc := range machineHealthChecks {
		Expect(mhc.Spec.UnhealthyConditions).NotTo(BeEmpty())
		if !strings.Contains(mhc.Name, mhcMatcher) {
			continue
		}

		Eventually(func() (bool, error) {
			machines := framework.GetMachinesByMachineHealthCheck(ctx, framework.GetMachinesByMachineHealthCheckInput{
				Lister:             mgmtClient,
				ClusterName:        cluster.Name,
				MachineHealthCheck: mhc,
			})

			count := 0
			for _, machine := range machines {
				if machine.Status.NodeRef == nil {
					continue
				}
				node := &corev1.Node{}
				err := workloadClient.Get(ctx, k8stypes.NamespacedName{Name: machine.Status.NodeRef.Name, Namespace: machine.Status.NodeRef.Namespace}, node)
				if err != nil {
					continue
				}
				if !HasMatchingUnhealthyConditions(mhc, node.Status.Conditions) {
					count++
				}
			}
			return count == healthyMachineCount, nil
		}, intervals...).Should(BeTrue())
	}
}

func HasMatchingUnhealthyConditions(machineHealthCheck *clusterv1.MachineHealthCheck, nodeConditions []corev1.NodeCondition) bool {
	for _, unhealthyCondition := range machineHealthCheck.Spec.UnhealthyConditions {
		for _, nodeCondition := range nodeConditions {
			if nodeCondition.Type == unhealthyCondition.Type && nodeCondition.Status == unhealthyCondition.Status {
				return true
			}
		}
	}
	return false
}

func ClusterExists(ctx context.Context, mgmtClient client.Client, cluster *clusterv1.Cluster) bool {
	key := client.ObjectKey{
		Namespace: cluster.GetNamespace(),
		Name:      cluster.GetName(),
	}
	return !apierrors.IsNotFound(mgmtClient.Get(ctx, key, &clusterv1.Cluster{}))
}

func IsClusterReady(ctx context.Context, mgmtClient client.Client, cluster *clusterv1.Cluster) bool {
	key := client.ObjectKey{
		Namespace: cluster.GetNamespace(),
		Name:      cluster.GetName(),
	}
	c := &clusterv1.Cluster{}
	err := mgmtClient.Get(ctx, key, c)

	if apierrors.IsNotFound(err) {
		return false
	}
	Expect(err).To(BeNil(), "Failed to get cluster status")
	return c.Status.ControlPlaneReady && c.Status.InfrastructureReady
}

func CheckDiskOfferingOfVmInstances(client *cloudstack.CloudStackClient, clusterName string, diskOfferingName string) {
	Byf("Listing machines with %q", clusterName)
	listResp, err := client.VirtualMachine.ListVirtualMachines(client.VirtualMachine.NewListVirtualMachinesParams())
	if err != nil {
		Fail("Failed to list machines: " + err.Error())
	}
	for _, vm := range listResp.VirtualMachines {
		if strings.Contains(vm.Name, clusterName) {
			Expect(vm.Diskofferingname).To(Equal(diskOfferingName))
		}
	}
}

func CheckVolumeSizeofVmInstances(client *cloudstack.CloudStackClient, clusterName string, volumeSize int64) {
	Byf("Listing machines with %q", clusterName)
	listResp, err := client.VirtualMachine.ListVirtualMachines(client.VirtualMachine.NewListVirtualMachinesParams())
	if err != nil {
		Fail("Failed to list machines: " + err.Error())
	}
	for _, vm := range listResp.VirtualMachines {
		if strings.Contains(vm.Name, clusterName) {
			p := client.Volume.NewListVolumesParams()
			p.SetVirtualmachineid(vm.Id)
			volResp, err := client.Volume.ListVolumes(p)
			if err != nil {
				Fail(fmt.Sprintf("Failed to list volumes for VM instance %s", vm.Id))
			}
			isVolumeSizeChecked := false
			for _, vol := range volResp.Volumes {
				if strings.Contains(vol.Name, DataVolumePrefix) {
					if vol.Size != volumeSize {
						Fail(fmt.Sprintf("Expected %d volume size but got %d volume size for VM instance %s", volumeSize, vol.Size, vm.Id))
					}
					isVolumeSizeChecked = true
				}
			}
			if !isVolumeSizeChecked {
				Fail(fmt.Sprintf("Could not find any volumes with a prefix %s", DataVolumePrefix))
			}
		}
	}
}