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
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/blang/semver"
	. "github.com/onsi/ginkgo"
	"gopkg.in/ini.v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
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
	KubernetesVersionManagement  = "KUBERNETES_VERSION_MANAGEMENT"
	KubernetesVersion            = "KUBERNETES_VERSION"
	CNIPath                      = "CNI"
	CNIResources                 = "CNI_RESOURCES"
	IPFamily                     = "IP_FAMILY"
	InvalidZoneName              = "CLOUDSTACK_INVALID_ZONE_NAME"
	InvalidNetworkName           = "CLOUDSTACK_INVALID_NETWORK_NAME"
	InvalidAccountName           = "CLOUDSTACK_INVALID_ACCOUNT_NAME"
	InvalidDomainName            = "CLOUDSTACK_INVALID_DOMAIN_NAME"
	InvalidTemplateName          = "CLOUDSTACK_INVALID_TEMPLATE_NAME"
	InvalidCPOfferingName        = "CLOUDSTACK_INVALID_CONTROL_PLANE_MACHINE_OFFERING"
	ExtremelyLargeCPOfferingName = "CLOUDSTACK_EXTREMELY_LARGE_CONTROL_PLANE_MACHINE_OFFERING"
	InvalidWorkerOfferingName    = "CLOUDSTACK_INVALID_WORKER_MACHINE_OFFERING"
)

const (
	ControlPlaneIndicator      = "control-plane"
	MachineDeploymentIndicator = "md"
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
		"-i", "--restart=Never", "dummy", "--image=dockerqa/curl:ubuntu-trusty", "--command", "--", "curl", "--silent", fmt.Sprintf("%s:%d%s", appName, port, path),
	}
	return KubectlExec(ctx, "run", workloadKubeconfigPath, runArgs...)
}

type cloudConfig struct {
	APIURL    string `ini:"api-url"`
	APIKey    string `ini:"api-key"`
	SecretKey string `ini:"secret-key"`
	VerifySSL bool   `ini:"verify-ssl"`
}

func DestroyOneMachine(clusterName string, machineType string) {
	client := createCloudStackClient()
	matcher := clusterName + "-" + machineType

	Byf("Listing machines with %q", matcher)
	listResp, err := client.VirtualMachine.ListVirtualMachines(client.VirtualMachine.NewListVirtualMachinesParams())
	if err != nil {
		Fail("Failed to list machines")
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
	stopParams := client.VirtualMachine.NewStopVirtualMachineParams(vmToDestroy.Id)
	stopParams.SetForced(true)
	_, err = client.VirtualMachine.StopVirtualMachine(stopParams)
	if err != nil {
		Fail("Failed to stop machine")
	}
	destroyParams := client.VirtualMachine.NewDestroyVirtualMachineParams(vmToDestroy.Id)
	destroyParams.SetExpunge(true)
	_, err = client.VirtualMachine.DestroyVirtualMachine(destroyParams)
	if err != nil {
		Fail("Failed to destroy machine")
	}
}

func CheckAffinityGroupsDeleted(affinityIds []string) error {
	client := createCloudStackClient()

	for _, affinityId := range affinityIds {
		affinity, count, _ := client.AffinityGroup.GetAffinityGroupByID(affinityId)
		if count > 0 {
			return errors.New("Affinity group " + affinity.Name + " still exists")
		}
	}
	return nil
}

func CheckAffinityGroup(clusterName string, affinityType string) []string {
	client := createCloudStackClient()

	By("Listing all machines")
	listResp, err := client.VirtualMachine.ListVirtualMachines(client.VirtualMachine.NewListVirtualMachinesParams())
	if err != nil {
		Fail("Failed to list machines")
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
					Fail("Failed to get affinity group for " + affinity.Id)
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

func CheckZones(clusterName string, zoneNames []string) []string {
	client := createCloudStackClient()

	By("Listing all machines")
	listResp, err := client.VirtualMachine.ListVirtualMachines(client.VirtualMachine.NewListVirtualMachinesParams())
	if err != nil {
		Fail("Failed to list machines")
	}
	cpZoneIdMap := make(map[string]int)
	mdZoneIdMap := make(map[string]int)
	zoneIds := []string{}

	for _, vm := range listResp.VirtualMachines {
		if strings.Contains(vm.Name, clusterName) {
			By(vm.Name + " is in zone " + vm.Zonename + " (" + vm.Zoneid + ")")
			zoneIds = append(zoneIds, vm.Zoneid)
			if !checkZoneNameInRange(vm, zoneNames) {
				Byf("vm %s is assigned in zone %s : valid zones %s", vm.Name, vm.Zonename, zoneNames)
			}
			err := checkZoneAssignments(vm, cpZoneIdMap, mdZoneIdMap)
			if err != nil {
				Fail(err.Error())
			}
		}
	}
	By("cpZoneIdMap")
	for key, value := range cpZoneIdMap {
		Byf("\t%s value is %v\n", key, value)
	}
	By("mdZoneIdMap")
	for key, value := range mdZoneIdMap {
		Byf("\t%s value is %v\n", key, value)
	}
	return zoneIds
}

func checkZoneAssignments(vm *cloudstack.VirtualMachine, cpZoneIdMap map[string]int, mdZoneIdMap map[string]int) error {
	if strings.Contains(vm.Name, ControlPlaneIndicator) {
		count, ok := cpZoneIdMap[vm.Zoneid]
		if !ok {
			cpZoneIdMap[vm.Zoneid] = 1
		} else {
			cpZoneIdMap[vm.Zoneid] = count + 1
		}
	}
	if strings.Contains(vm.Name, MachineDeploymentIndicator) {
		count, ok := mdZoneIdMap[vm.Zoneid]
		if !ok {
			mdZoneIdMap[vm.Zoneid] = 1
		} else {
			mdZoneIdMap[vm.Zoneid] = count + 1
		}
	}
	return nil
}

func checkZoneNameInRange(vm *cloudstack.VirtualMachine, zoneNames []string) bool {
	for _, zoneName := range zoneNames {
		if vm.Zonename == zoneName {
			return true
		}
	}
	return false
}

func CheckNetworkExists(networkName string) (bool, error) {
	client := createCloudStackClient()

	_, count, err := client.Network.GetNetworkByName(networkName)
	if err != nil {
		if strings.Contains(err.Error(), "No match found for") {
			return false, nil
		}
		return false, err
	} else if count > 1 {
		return false, errors.New(fmt.Sprintf("Expected 0-1 Network with name %s, but got %d.", networkName, count))
	}
	return count == 1, nil
}

func createCloudStackClient() *cloudstack.CloudStackClient {
	encodedSecret := os.Getenv("CLOUDSTACK_B64ENCODED_SECRET")
	secret, err := base64.StdEncoding.DecodeString(encodedSecret)
	if err != nil {
		Fail("Failed ")
	}
	cfg := &cloudConfig{VerifySSL: true}
	if rawCfg, err := ini.Load(secret); err != nil {
		Fail("Failed to load INI file")
	} else if g := rawCfg.Section("Global"); len(g.Keys()) == 0 {
		Fail("Global section not found")
	} else if err = rawCfg.Section("Global").StrictMapTo(cfg); err != nil {
		Fail("Error encountered while parsing Global section")
	}

	By("Creating a CloudStack client")
	client := cloudstack.NewAsyncClient(cfg.APIURL, cfg.APIKey, cfg.SecretKey, cfg.VerifySSL)
	return client
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
	DestroyOneMachine(cluster.Name, machineMatcher)

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
