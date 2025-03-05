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

package controllers_test

import (
	"context"
	"flag"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/fakes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	csReconcilers "sigs.k8s.io/cluster-api-provider-cloudstack/controllers"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/mocks"

	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	//+kubebuilder:scaffold:imports
)

func TestAPIs(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Controller Suite")
}

var (
	clusterAPIVersionRegex = regexp.MustCompile(`^(\W)sigs.k8s.io/cluster-api v(.+)`)
)

const (
	timeout             = 10 * time.Second
	pollInterval        = 1 * time.Second
	fakeEventBufferSize = 10
)

func envOr(envKey, defaultValue string) string {
	if value, ok := os.LookupEnv(envKey); ok {
		return value
	}
	return defaultValue
}

// Have to get the path to the installed CAPI to inject CAPI CRDs.
func getFilePathToCAPICRDs(root string) string {
	modBits, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}

	var clusterAPIVersion string
	for _, line := range strings.Split(string(modBits), "\n") {
		matches := clusterAPIVersionRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			clusterAPIVersion = matches[2]
		}
	}

	if clusterAPIVersion == "" {
		return ""
	}

	gopath := envOr("GOPATH", build.Default.GOPATH)
	return filepath.Join(gopath, "pkg", "mod", "sigs.k8s.io",
		fmt.Sprintf("cluster-api@v%s", clusterAPIVersion), "config", "crd", "bases")
}

var (
	// TestEnv vars...
	testEnv        *envtest.Environment
	k8sClient      client.Client
	ctx            context.Context
	cancel         context.CancelFunc
	k8sManager     manager.Manager
	cfg            *rest.Config
	logger         logr.Logger
	fakeCtrlClient client.Client
	fakeRecorder   *record.FakeRecorder

	// Mock Vars.
	mockCtrl        *gomock.Controller
	mockCloudClient *mocks.MockClient
	mockCSAPIClient *cloudstack.CloudStackClient

	// Reconcilers
	MachineReconciler       *csReconcilers.CloudStackMachineReconciler
	ClusterReconciler       *csReconcilers.CloudStackClusterReconciler
	FailureDomainReconciler *csReconcilers.CloudStackFailureDomainReconciler
	IsoNetReconciler        *csReconcilers.CloudStackIsoNetReconciler
	AffinityGReconciler     *csReconcilers.CloudStackAffinityGroupReconciler

	// CKS Reconcilers
	CksClusterReconciler *csReconcilers.CksClusterReconciler
	CksMachineReconciler *csReconcilers.CksMachineReconciler
)

var _ = ginkgo.BeforeSuite(func() {
	repoRoot := os.Getenv("REPO_ROOT")

	// Add ginkgo recover statements to controllers.
	cmd := exec.Command(repoRoot+"/hack/testing_ginkgo_recover_statements.sh", "--contains")
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Println(errors.Wrapf(err, "refusing to run test suite without ginkgo recover statements present"))
		os.Exit(1)
	}

	ginkgo.By("bootstrapping test environment")

	gomega.Ω(infrav1.AddToScheme(scheme.Scheme)).Should(gomega.Succeed())
	gomega.Ω(clusterv1.AddToScheme(scheme.Scheme)).Should(gomega.Succeed())
	gomega.Ω(fakes.AddToScheme(scheme.Scheme)).Should(gomega.Succeed())

	// Increase log verbosity.
	klog.InitFlags(nil)
	gomega.Ω(flag.Lookup("v").Value.Set("1")).Should(gomega.Succeed())
	flag.Parse()

	logger = klogr.New()
})

// A mock fo the CloudClient interface used in controller utils.
type MockCtrlrCloudClientImplementation struct {
	*csCtrlrUtils.ReconciliationRunner
	csCtrlrUtils.CloudClientExtension
}

// AsFailureDomainUser is a method used in the reconciliation runner to set up the CloudStack client. Using this here
// just sets the CSClient to a mock client.
func (m *MockCtrlrCloudClientImplementation) AsFailureDomainUser(
	*infrav1.CloudStackFailureDomainSpec) csCtrlrUtils.CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		m.CSUser = mockCloudClient

		return ctrl.Result{}, nil
	}
}

func (m *MockCtrlrCloudClientImplementation) RegisterExtension(r *csCtrlrUtils.ReconciliationRunner) csCtrlrUtils.CloudClientExtension {
	return &MockCtrlrCloudClientImplementation{ReconciliationRunner: r}
}

func SetupTestEnvironment() {
	repoRoot := os.Getenv("REPO_ROOT")
	crdPaths := []string{filepath.Join(repoRoot, "config", "crd", "bases"), filepath.Join(repoRoot, "test", "fakes")}

	// Append CAPI CRDs path
	if capiPath := getFilePathToCAPICRDs(repoRoot); capiPath != "" {
		crdPaths = append(crdPaths, capiPath)
	}
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     crdPaths,
	}
	var err error
	done := make(chan interface{})
	go func() {
		defer ginkgo.GinkgoRecover()
		cfg, err = testEnv.Start()
		close(done)
	}()
	gomega.Eventually(done).WithTimeout(time.Minute).Should(gomega.BeClosed())
	gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
	gomega.Ω(cfg).ShouldNot(gomega.BeNil())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
	gomega.Ω(k8sClient).ShouldNot(gomega.BeNil())
	k8sManager, _ = ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	gomega.Ω(err).ShouldNot(gomega.HaveOccurred())

	// Base reconciler shared across reconcilers.
	base := csCtrlrUtils.ReconcilerBase{
		K8sClient:            k8sManager.GetClient(),
		Scheme:               k8sManager.GetScheme(),
		CSClient:             mockCloudClient,
		BaseLogger:           logger,
		CloudClientExtension: &MockCtrlrCloudClientImplementation{},
	}

	// Setup each specific reconciler.
	ClusterReconciler = &csReconcilers.CloudStackClusterReconciler{ReconcilerBase: base}
	MachineReconciler = &csReconcilers.CloudStackMachineReconciler{ReconcilerBase: base}
	FailureDomainReconciler = &csReconcilers.CloudStackFailureDomainReconciler{ReconcilerBase: base}
	IsoNetReconciler = &csReconcilers.CloudStackIsoNetReconciler{ReconcilerBase: base}
	AffinityGReconciler = &csReconcilers.CloudStackAffinityGroupReconciler{ReconcilerBase: base}

	CksClusterReconciler = &csReconcilers.CksClusterReconciler{ReconcilerBase: base}
	CksMachineReconciler = &csReconcilers.CksMachineReconciler{ReconcilerBase: base}

	ctx, cancel = context.WithCancel(context.TODO())

	// Setup mock clients.
	mockCSAPIClient = cloudstack.NewMockClient(mockCtrl)
	mockCloudClient = mocks.NewMockClient(mockCtrl)

	// Set on reconcilers. The mock client wasn't available at suite startup, so set it now.
	ClusterReconciler.CSClient = mockCloudClient
	IsoNetReconciler.CSClient = mockCloudClient
	MachineReconciler.CSClient = mockCloudClient
	AffinityGReconciler.CSClient = mockCloudClient
	FailureDomainReconciler.CSClient = mockCloudClient

	CksClusterReconciler.CSClient = mockCloudClient
	CksMachineReconciler.CSClient = mockCloudClient

	setupClusterCRDs()

	// See reconciliation results. Left commented as it's noisy otherwise.
	// TODO: find a way to see controller output without the additional setup output.
	ctrl.SetLogger(logger)

	ginkgo.DeferCleanup(func() {
		// Cancelling the context shuts down any outstanding requests and the test environment.
		cancel()
		gomega.Ω(testEnv.Stop()).Should(gomega.Succeed())
		k8sManager = nil
	})
}

// Sets up a fake k8s controller runtime client with CAPI and CloudStack clusters.
func setupFakeTestClient() {
	dummies.SetDummyVars()

	// Make a fake k8s client with CloudStack and CAPI cluster.
	fakeCtrlClient = fake.NewClientBuilder().WithObjects(dummies.CSCluster, dummies.CAPICluster).Build()
	fakeRecorder = record.NewFakeRecorder(fakeEventBufferSize)
	// Setup mock clients.
	mockCSAPIClient = cloudstack.NewMockClient(mockCtrl)
	mockCloudClient = mocks.NewMockClient(mockCtrl)

	// Base reconciler shared across reconcilers.
	base := csCtrlrUtils.ReconcilerBase{
		K8sClient:            fakeCtrlClient,
		Scheme:               scheme.Scheme,
		CSClient:             mockCloudClient,
		BaseLogger:           logger,
		Recorder:             fakeRecorder,
		CloudClientExtension: &MockCtrlrCloudClientImplementation{},
	}

	ctx, cancel = context.WithCancel(context.TODO())

	// Setup each specific reconciler.
	ClusterReconciler = &csReconcilers.CloudStackClusterReconciler{ReconcilerBase: base}
	MachineReconciler = &csReconcilers.CloudStackMachineReconciler{ReconcilerBase: base}
	FailureDomainReconciler = &csReconcilers.CloudStackFailureDomainReconciler{ReconcilerBase: base}
	IsoNetReconciler = &csReconcilers.CloudStackIsoNetReconciler{ReconcilerBase: base}
	AffinityGReconciler = &csReconcilers.CloudStackAffinityGroupReconciler{ReconcilerBase: base}

	// Set on reconcilers. The mock client wasn't available at suite startup, so set it now.
	ClusterReconciler.CSClient = mockCloudClient
	IsoNetReconciler.CSClient = mockCloudClient
	MachineReconciler.CSClient = mockCloudClient
	FailureDomainReconciler.CSClient = mockCloudClient
	AffinityGReconciler.CSClient = mockCloudClient

	ginkgo.DeferCleanup(func() {
		cancel()
	})
}

// Setup and teardown on a per test basis.
var _ = ginkgo.BeforeEach(func() {
	dummies.SetDummyVars()
	mockCtrl = gomock.NewController(ginkgo.GinkgoT())
})

var _ = ginkgo.JustBeforeEach(func() {
	if k8sManager != nil { // Allow skipping a test environment for tests that don't need it.
		// Launch the k8s manager.
		// Needs to be in JustBeforeEach() so individual contexts can register controllers first.
		go func() {
			defer ginkgo.GinkgoRecover()
			gomega.Ω(k8sManager.Start(ctx)).Should(gomega.Succeed(), "failed to run manager")
		}()
	}
})

var _ = ginkgo.AfterEach(func() {
	// Finishing mockCtrl checks expected calls on mock objects matched.
	mockCtrl.Finish()
})

var _ = ginkgo.AfterSuite(func() {})

// setClusterReady patches the clsuter with ready status true.
func setClusterReady(client client.Client) {
	gomega.Eventually(func() error {
		ph, err := patch.NewHelper(dummies.CSCluster, client)
		gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
		dummies.CSCluster.Status.Ready = true
		return ph.Patch(ctx, dummies.CSCluster, patch.WithStatusObservedGeneration{})
	}, timeout).Should(gomega.Succeed())
}

// setupClusterCRDs creates a CAPI and CloudStack cluster with an appropriate ownership ref between them.
func setupClusterCRDs() {

	//  Create them.
	gomega.Ω(k8sClient.Create(ctx, dummies.CAPICluster)).Should(gomega.Succeed())
	gomega.Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(gomega.Succeed())

	// Fetch the CS Cluster that was created.
	key := client.ObjectKey{Namespace: dummies.CSCluster.Namespace, Name: dummies.CSCluster.Name}
	gomega.Eventually(func() error {
		return k8sClient.Get(ctx, key, dummies.CSCluster)
	}, timeout).Should(gomega.BeNil())

	// Set owner ref from CAPI cluster to CS Cluster and patch back the CS Cluster.
	gomega.Eventually(func() error {
		ph, err := patch.NewHelper(dummies.CSCluster, k8sClient)
		gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
		dummies.CSCluster.OwnerReferences = append(dummies.CSCluster.OwnerReferences, metav1.OwnerReference{
			Kind:       "Cluster",
			APIVersion: clusterv1.GroupVersion.String(),
			Name:       dummies.CAPICluster.Name,
			UID:        "uniqueness",
		})
		return ph.Patch(ctx, dummies.CSCluster, patch.WithStatusObservedGeneration{})
	}, timeout).Should(gomega.Succeed())
}

// setupMachineCRDs creates a CAPI and CloudStack machine with an appropriate ownership ref between them.
func setupMachineCRDs() {
	//  Create them.
	gomega.Ω(k8sClient.Create(ctx, dummies.CAPIMachine)).Should(gomega.Succeed())
	gomega.Ω(k8sClient.Create(ctx, dummies.CSMachine1)).Should(gomega.Succeed())

	// Fetch the CS Machine that was created.
	key := client.ObjectKey{Namespace: dummies.CSCluster.Namespace, Name: dummies.CSMachine1.Name}
	gomega.Eventually(func() error {
		return k8sClient.Get(ctx, key, dummies.CSMachine1)
	}, timeout).Should(gomega.BeNil())

	// Set owner ref from CAPI machine to CS machine and patch back the CS machine.
	gomega.Eventually(func() error {
		ph, err := patch.NewHelper(dummies.CSMachine1, k8sClient)
		gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
		dummies.CSMachine1.OwnerReferences = append(dummies.CSMachine1.OwnerReferences, metav1.OwnerReference{
			Kind:       "Machine",
			APIVersion: clusterv1.GroupVersion.String(),
			Name:       dummies.CAPIMachine.Name,
			UID:        "uniqueness",
		})
		return ph.Patch(ctx, dummies.CSMachine1, patch.WithStatusObservedGeneration{})
	}, timeout).Should(gomega.Succeed())
}

func setCSMachineOwnerCRD(owner *fakes.CloudStackMachineOwner, specReplicas, statusReplicas, statusReadyReplicas *int32, statusReady *bool) {
	owner.Spec.Replicas = specReplicas
	gomega.Ω(k8sClient.Create(ctx, owner)).Should(gomega.Succeed())
	key := client.ObjectKey{Namespace: owner.Namespace, Name: owner.Name}
	gomega.Eventually(func() error {
		return k8sClient.Get(ctx, key, owner)
	}, timeout).Should(gomega.BeNil())

	gomega.Eventually(func() error {
		owner.Status.Ready = statusReady
		owner.Status.Replicas = statusReplicas
		owner.Status.ReadyReplicas = statusReadyReplicas
		return k8sClient.Status().Update(ctx, owner)
	}, timeout).Should(gomega.BeNil())
}

// setCAPIMachineAndCSMachineCRDs creates a CAPI and CloudStack machine with an appropriate ownership ref between them.
func setCAPIMachineAndCSMachineCRDs(CSMachine *infrav1.CloudStackMachine, CAPIMachine *clusterv1.Machine) {
	//  Create them.
	gomega.Ω(k8sClient.Create(ctx, CAPIMachine)).Should(gomega.Succeed())
	gomega.Ω(k8sClient.Create(ctx, CSMachine)).Should(gomega.Succeed())

	// Fetch the CS Machine that was created.
	key := client.ObjectKey{Namespace: dummies.CSCluster.Namespace, Name: CSMachine.Name}
	gomega.Eventually(func() error {
		return k8sClient.Get(ctx, key, CSMachine)
	}, timeout).Should(gomega.BeNil())

	// Fetch the CAPI Machine that was created.
	key = client.ObjectKey{Namespace: dummies.ClusterNameSpace, Name: CAPIMachine.Name}
	gomega.Eventually(func() error {
		return k8sClient.Get(ctx, key, CAPIMachine)
	}, timeout).Should(gomega.BeNil())

	// Set ownerReference to CAPI machine in CS machine and patch back the CS machine.
	gomega.Eventually(func() error {
		ph, err := patch.NewHelper(dummies.CSMachine1, k8sClient)
		gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
		dummies.CSMachine1.OwnerReferences = append(dummies.CSMachine1.OwnerReferences, metav1.OwnerReference{
			Kind:       "Machine",
			APIVersion: clusterv1.GroupVersion.String(),
			Name:       CAPIMachine.Name,
			UID:        "uniqueness",
		})
		return ph.Patch(ctx, CSMachine, patch.WithStatusObservedGeneration{})
	}, timeout).Should(gomega.Succeed())
}

func setMachineOwnerReference(CSMachine *infrav1.CloudStackMachine, ownerRef metav1.OwnerReference) {
	key := client.ObjectKey{Namespace: dummies.CSCluster.Namespace, Name: CSMachine.Name}
	gomega.Eventually(func() error {
		return k8sClient.Get(ctx, key, CSMachine)
	}, timeout).Should(gomega.BeNil())

	// Set ownerReference to CAPI machine in CS machine and patch back the CS machine.
	gomega.Eventually(func() error {
		ph, err := patch.NewHelper(CSMachine, k8sClient)
		gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
		CSMachine.OwnerReferences = append(CSMachine.OwnerReferences, ownerRef)
		return ph.Patch(ctx, CSMachine, patch.WithStatusObservedGeneration{})
	}, timeout).Should(gomega.Succeed())
}

// labelMachineFailuredomain add cloudstackfailuredomain info in the labels.
func labelMachineFailuredomain(CSMachine *infrav1.CloudStackMachine, CSFailureDomain1 *infrav1.CloudStackFailureDomain) {
	key := client.ObjectKey{Namespace: dummies.CSCluster.Namespace, Name: CSMachine.Name}
	gomega.Eventually(func() error {
		return k8sClient.Get(ctx, key, CSMachine)
	}, timeout).Should(gomega.BeNil())

	// set cloudstack failuredomain in machine labels.
	gomega.Eventually(func() error {
		ph, err := patch.NewHelper(CSMachine, k8sClient)
		gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
		CSMachine.Labels["cloudstackfailuredomain.infrastructure.cluster.x-k8s.io/name"] = CSFailureDomain1.Name
		return ph.Patch(ctx, CSMachine, patch.WithStatusObservedGeneration{})
	}, timeout).Should(gomega.Succeed())
}
