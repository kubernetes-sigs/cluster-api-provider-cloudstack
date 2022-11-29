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
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/blang/semver"
	. "github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
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

type CommonSpecInput struct {
	E2EConfig             *clusterctl.E2EConfig
	ClusterctlConfigPath  string
	BootstrapClusterProxy framework.ClusterProxy
	ArtifactFolder        string
	SkipCleanup           bool

	// Flavor, if specified is the template flavor used to create the cluster for testing.
	// If not specified, and the e2econfig variable IPFamily is IPV6, then "ipv6" is used,
	// otherwise the default flavor is used.
	Flavor *string
}

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

func DeployAppToWorkloadClusterAndWaitForDeploymentReady(ctx context.Context, workloadKubeconfigPath string, appName string, appConfigLink string, timeout int) error {
	applyArgs := []string{
		"-f", appConfigLink,
	}
	_, err := KubectlExec(ctx, "apply", workloadKubeconfigPath, applyArgs...)
	if err != nil {
		return err
	}

	waitArgs := []string{
		"--for=condition=ready", fmt.Sprintf("--timeout=%ds", timeout), "pods", "-l", fmt.Sprintf("app=%s", appName),
	}
	_, err = KubectlExec(ctx, "wait", workloadKubeconfigPath, waitArgs...)
	if err != nil {
		return err
	}
	return nil
}

func DownloadFromAppInWorkloadCluster(ctx context.Context, workloadKubeconfigPath string, appName string, port int, path string) (string, error) {
	runArgs := []string{
		// Required by below: container name is runArg zero.
		"dummy", "-i", "--restart=Never", "--image=dockerqa/curl:ubuntu-trusty", "--command", "--", "curl", "--silent", "--show-error", fmt.Sprintf("%s:%d%s", appName, port, path),
	}
	var result, err = KubectlExec(ctx, "run", workloadKubeconfigPath, runArgs...)
	if err != nil {
		return result, err
	}
	if result == "" {
		// A single retry to accommodate occasional cases where an empty string is returned, ostensibly
		//  because the service isn't fully ready.  Subsequent requests have always worked.
		fmt.Println("Retrying html download")
		time.Sleep(5 * time.Second)
		runArgs[0] = "dummy2" // Assumed: container name is runArg zero.
		result, err = KubectlExec(ctx, "run", workloadKubeconfigPath, runArgs...)
	}
	return result, err
}

func DownloadMetricsFromCAPCManager(ctx context.Context, bootstrapKubeconfigPath string) (string, error) {
	// Expose the CAPC manager metrics port via a K8S service
	runArgs := []string{
		"--port=8080", "--target-port=metrics", "--name=capc-controller-manager-metrics", "--namespace=capc-system", "deployment", "capc-controller-manager",
	}
	_, err := KubectlExec(ctx, "expose", bootstrapKubeconfigPath, runArgs...)
	Ω(err).ShouldNot(HaveOccurred())

	// Scrape the metrics from the service
	runArgs = []string{
		"-i", "--restart=Never", "dummy", "--image=dockerqa/curl:ubuntu-trusty", "--command", "--", "curl", "--silent", "capc-controller-manager-metrics.capc-system:8080/metrics",
	}
	result, err := KubectlExec(ctx, "run", bootstrapKubeconfigPath, runArgs...)
	Ω(err).ShouldNot(HaveOccurred())

	// Remove the metrics service
	runArgs = []string{
		"--namespace=capc-system", "service", "capc-controller-manager-metrics",
	}
	_, err = KubectlExec(ctx, "delete", bootstrapKubeconfigPath, runArgs...)
	Ω(err).ShouldNot(HaveOccurred())

	return result, nil
}

func DestroyOneMachine(client *cloudstack.CloudStackClient, clusterName string, machineType string) {
	matcher := clusterName + "-" + machineType

	Byf("Listing machines with %q", matcher)
	listResp, err := client.VirtualMachine.ListVirtualMachines(client.VirtualMachine.NewListVirtualMachinesParams())
	if err != nil {
		Fail("Failed to list machines: " + err.Error())
	}
	var vmToDestroy *cloudstack.VirtualMachine
	originalCount := 0
	for _, vm := range listResp.VirtualMachines {
		if strings.Contains(vm.Name, matcher) {
			originalCount++
			if vmToDestroy == nil {
				vmToDestroy = vm
			}
		}
	}

	Byf("Destroying machine %s", vmToDestroy.Name)
	destroyParams := client.VirtualMachine.NewDestroyVirtualMachineParams(vmToDestroy.Id)
	destroyParams.SetExpunge(true)
	_, err = client.VirtualMachine.DestroyVirtualMachine(destroyParams)
	if err != nil {
		Fail("Failed to destroy machine: " + err.Error())
	}
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

func GetHostCount(client *cloudstack.CloudStackClient, zoneName string) int {
	pz := client.Zone.NewListZonesParams()
	pz.SetName(zoneName)
	listZonesResponse, err := client.Zone.ListZones(pz)
	Expect(err).To(BeNil(), "error listing zones")
	Expect(listZonesResponse.Count).To(Equal(1), "multiple zones resolve to zone name %s", zoneName)
	zoneId := listZonesResponse.Zones[0].Id

	ph := client.Host.NewListHostsParams()
	ph.SetZoneid(zoneId)
	ph.SetHypervisor("KVM")
	ph.SetResourcestate("Enabled")
	ph.SetState("Up")
	listHostsResponse, err := client.Host.ListHosts(ph)
	Expect(err).To(BeNil(), "error listing hosts")
	return listHostsResponse.Count
}

func CheckAffinityGroup(client *cloudstack.CloudStackClient, clusterName string, affinityType string) []string {
	By("Listing all machines")
	p := client.VirtualMachine.NewListVirtualMachinesParams()
	p.SetListall(true)
	listResp, err := client.VirtualMachine.ListVirtualMachines(p)
	if err != nil {
		Fail("Failed to list machines: " + err.Error())
	}
	affinityTypeString := strings.Title(fmt.Sprintf("%sAffinity", affinityType))
	cpHostIdSet := make(map[string]bool)
	mdHostIdSet := make(map[string]bool)
	affinityIds := []string{}

	for _, vm := range listResp.VirtualMachines {
		if strings.Contains(vm.Name, clusterName) {
			By(vm.Name + " is in host " + vm.Hostname + " (" + vm.Hostid + ")")
			err := checkVMHostAssignments(vm, cpHostIdSet, mdHostIdSet, affinityType)
			if err != nil {
				Fail(err.Error())
			}

			for _, affinity := range vm.Affinitygroup {
				affinityIds = append(affinityIds, affinity.Id)
				affinity, _, _ := client.AffinityGroup.GetAffinityGroupByID(affinity.Id)
				if err != nil {
					Fail("Failed to get affinity group for " + affinity.Id + " : " + err.Error())
				}
				if !strings.Contains(affinity.Name, affinityTypeString) {
					Fail(affinity.Name + " does not contain " + affinityTypeString)
				}
				if affinityType == "pro" && affinity.Type != "host affinity" {
					Fail(affinity.Type + " does not match " + affinityType)
				}
				if affinityType == "anti" && affinity.Type != "host anti-affinity" {
					Fail(affinity.Type + " does not match " + affinityType)
				}
			}
		}
	}
	return affinityIds
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

func WaitForMachineRemediationAfterDestroy(ctx context.Context, proxy framework.ClusterProxy, cluster *clusterv1.Cluster, machineMatcher string, healthyMachineCount int, intervals []interface{}) {
	mgmtClusterClient := proxy.GetClient()
	workloadClusterClient := proxy.GetWorkloadCluster(ctx, cluster.Namespace, cluster.Name).GetClient()

	WaitForHealthyMachineCount(ctx, mgmtClusterClient, workloadClusterClient, cluster, machineMatcher, healthyMachineCount, intervals)
	Byf("Current number of healthy %s is %d", machineMatcher, healthyMachineCount)

	Byf("Destroying one %s", machineMatcher)
	csClient := CreateCloudStackClient(ctx, proxy.GetKubeconfigPath())
	DestroyOneMachine(csClient, cluster.Name, machineMatcher)

	Byf("Waiting for the destroyed %s to be unhealthy", machineMatcher)
	WaitForHealthyMachineCount(ctx, mgmtClusterClient, workloadClusterClient, cluster, machineMatcher, healthyMachineCount-1, intervals)

	Byf("Waiting for remediation of %s", machineMatcher)
	WaitForHealthyMachineCount(ctx, mgmtClusterClient, workloadClusterClient, cluster, machineMatcher, healthyMachineCount, intervals)
	Byf("%s machine remediated successfully", machineMatcher)
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
