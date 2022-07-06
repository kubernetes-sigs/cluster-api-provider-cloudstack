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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	csReconcilers "sigs.k8s.io/cluster-api-provider-cloudstack/controllers"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/mocks"

	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	//+kubebuilder:scaffold:imports
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var (
	clusterAPIVersionRegex = regexp.MustCompile(`^(\W)sigs.k8s.io/cluster-api v(.+)`)
)

const (
	timeout = time.Second * 30
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

	// Mock Vars.
	mockCtrl        *gomock.Controller
	mockCloudClient *mocks.MockClient
	mockCSAPIClient *cloudstack.CloudStackClient

	// Reconcilers
	MachineReconciler       *csReconcilers.CloudStackMachineReconciler
	ClusterReconciler       *csReconcilers.CloudStackClusterReconciler
	FailureDomainReconciler *csReconcilers.CloudStackFailureDomainReconciler
	ZoneReconciler          *csReconcilers.CloudStackZoneReconciler
	IsoNetReconciler        *csReconcilers.CloudStackIsoNetReconciler
	AffinityGReconciler     *csReconcilers.CloudStackAffinityGroupReconciler
)

var projectDir = os.Getenv("PROJECT_DIR")

var _ = BeforeSuite(func() {
	// Add ginkgo recover statements to controllers.
	cmd := exec.Command(projectDir+"/hack/testing_ginkgo_recover_statements.sh", "--contains")
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Println(errors.Wrapf(err, "refusing to run test suite without ginkgo recover statements present"))
		os.Exit(1)
	}

	By("bootstrapping test environment")

	Ω(infrav1.AddToScheme(scheme.Scheme)).Should(Succeed())
	Ω(clusterv1.AddToScheme(scheme.Scheme)).Should(Succeed())

	// Increase log verbosity.
	klog.InitFlags(nil)
	Ω(flag.Lookup("v").Value.Set("1")).Should(Succeed())
	flag.Parse()

	logger = klogr.New()
})

func SetupTestEnvironment() {

	crdPaths := []string{filepath.Join(projectDir, "config", "crd", "bases")}

	// Append CAPI CRDs path
	if capiPath := getFilePathToCAPICRDs(projectDir); capiPath != "" {
		crdPaths = append(crdPaths, capiPath)
	}
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     crdPaths,
	}
	var err error
	done := make(chan interface{})
	go func() {
		defer GinkgoRecover()
		cfg, err = testEnv.Start()
		close(done)
	}()
	Eventually(done).WithTimeout(time.Minute).Should(BeClosed())
	Ω(err).ShouldNot(HaveOccurred())
	Ω(cfg).ShouldNot(BeNil())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Ω(err).ShouldNot(HaveOccurred())
	Ω(k8sClient).ShouldNot(BeNil())
	k8sManager, _ = ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	Ω(err).ShouldNot(HaveOccurred())

	// Base reconciler shared across reconcilers.
	base := csCtrlrUtils.ReconcilerBase{
		K8sClient:  k8sManager.GetClient(),
		Scheme:     k8sManager.GetScheme(),
		CSClient:   mockCloudClient,
		BaseLogger: logger}

	// Setup each specific reconciler.
	ClusterReconciler = &csReconcilers.CloudStackClusterReconciler{ReconcilerBase: base}
	MachineReconciler = &csReconcilers.CloudStackMachineReconciler{ReconcilerBase: base}
	ZoneReconciler = &csReconcilers.CloudStackZoneReconciler{ReconcilerBase: base}
	FailureDomainReconciler = &csReconcilers.CloudStackFailureDomainReconciler{ReconcilerBase: base}
	IsoNetReconciler = &csReconcilers.CloudStackIsoNetReconciler{ReconcilerBase: base}
	AffinityGReconciler = &csReconcilers.CloudStackAffinityGroupReconciler{ReconcilerBase: base}

	ctx, cancel = context.WithCancel(context.TODO())

	// Setup mock clients.
	mockCSAPIClient = cloudstack.NewMockClient(mockCtrl)
	mockCloudClient = mocks.NewMockClient(mockCtrl)

	// Set on reconcilers. The mock client wasn't available at suite startup, so set it now.
	ClusterReconciler.CSClient = mockCloudClient
	ZoneReconciler.CSClient = mockCloudClient
	IsoNetReconciler.CSClient = mockCloudClient
	MachineReconciler.CSClient = mockCloudClient
	AffinityGReconciler.CSClient = mockCloudClient
	FailureDomainReconciler.CSClient = mockCloudClient

	setupClusterCRDs()

	// See reconciliation results. Left commented as it's noisy otherwise.
	// TODO: find a way to see controller output without the additional setup output.
	// ctrl.SetLogger(logger)

	DeferCleanup(func() {
		// Cancelling the context shuts down any outstanding requests and the test environment.
		Ω(testEnv.Stop()).Should(Succeed())
		cancel()
		k8sManager = nil
	})
}

// Sets up a fake k8s controller runtime client with CAPI and CloudStack clusters.
func setupFakeTestClient() {
	dummies.SetDummyVars()

	// Make a fake k8s client with CloudStack and CAPI cluster.
	fakeCtrlClient = fake.NewClientBuilder().WithObjects(dummies.CSCluster, dummies.CAPICluster).Build()

	// Setup mock clients.
	mockCSAPIClient = cloudstack.NewMockClient(mockCtrl)
	mockCloudClient = mocks.NewMockClient(mockCtrl)

	// Base reconciler shared across reconcilers.
	base := csCtrlrUtils.ReconcilerBase{
		K8sClient:  fakeCtrlClient,
		Scheme:     scheme.Scheme,
		CSClient:   mockCloudClient,
		BaseLogger: logger}

	ctx, cancel = context.WithCancel(context.TODO())

	// Setup each specific reconciler.
	ClusterReconciler = &csReconcilers.CloudStackClusterReconciler{ReconcilerBase: base}
	MachineReconciler = &csReconcilers.CloudStackMachineReconciler{ReconcilerBase: base}
	ZoneReconciler = &csReconcilers.CloudStackZoneReconciler{ReconcilerBase: base}
	FailureDomainReconciler = &csReconcilers.CloudStackFailureDomainReconciler{ReconcilerBase: base}
	IsoNetReconciler = &csReconcilers.CloudStackIsoNetReconciler{ReconcilerBase: base}
	AffinityGReconciler = &csReconcilers.CloudStackAffinityGroupReconciler{ReconcilerBase: base}

	// Set on reconcilers. The mock client wasn't available at suite startup, so set it now.
	ClusterReconciler.CSClient = mockCloudClient
	ZoneReconciler.CSClient = mockCloudClient
	IsoNetReconciler.CSClient = mockCloudClient
	MachineReconciler.CSClient = mockCloudClient
	FailureDomainReconciler.CSClient = mockCloudClient
	AffinityGReconciler.CSClient = mockCloudClient

	DeferCleanup(func() {
		cancel()
	})
}

// Setup and teardown on a per test basis.
var _ = BeforeEach(func() {
	dummies.SetDummyVars()
	mockCtrl = gomock.NewController(GinkgoT())
})

var _ = JustBeforeEach(func() {
	if k8sManager != nil { // Allow skipping a test environment for tests that don't need it.
		// Launch the k8s manager.
		// Needs to be in JustBeforeEach() so individual contexts can register controllers first.
		go func() {
			defer GinkgoRecover()
			Ω(k8sManager.Start(ctx)).Should(Succeed(), "failed to run manager")
		}()
	}
})

var _ = AfterEach(func() {
	// Finishing mockCtrl checks expected calls on mock objects matched.
	mockCtrl.Finish()
})

var _ = AfterSuite(func() {})

// setClusterReady patches the clsuter with ready status true.
func setClusterReady() {
	Eventually(func() error {
		ph, err := patch.NewHelper(dummies.CSCluster, k8sClient)
		Ω(err).ShouldNot(HaveOccurred())
		dummies.CSCluster.Status.Ready = true
		return ph.Patch(ctx, dummies.CSCluster, patch.WithStatusObservedGeneration{})
	}, timeout).Should(Succeed())
}

// setupClusterCRDs creates a CAPI and CloudStack cluster with an appropriate ownership ref between them.
func setupClusterCRDs() {

	//  Create them.
	Ω(k8sClient.Create(ctx, dummies.CAPICluster)).Should(Succeed())
	Ω(k8sClient.Create(ctx, dummies.CSCluster)).Should(Succeed())

	// Fetch the CS Cluster that was created.
	key := client.ObjectKey{Namespace: dummies.CSCluster.Namespace, Name: dummies.CSCluster.Name}
	Eventually(func() error {
		return k8sClient.Get(ctx, key, dummies.CSCluster)
	}, timeout).Should(BeNil())

	// Set owner ref from CAPI cluster to CS Cluster and patch back the CS Cluster.
	Eventually(func() error {
		ph, err := patch.NewHelper(dummies.CSCluster, k8sClient)
		Ω(err).ShouldNot(HaveOccurred())
		dummies.CSCluster.OwnerReferences = append(dummies.CSCluster.OwnerReferences, metav1.OwnerReference{
			Kind:       "Cluster",
			APIVersion: clusterv1.GroupVersion.String(),
			Name:       dummies.CAPICluster.Name,
			UID:        "uniqueness",
		})
		return ph.Patch(ctx, dummies.CSCluster, patch.WithStatusObservedGeneration{})
	}, timeout).Should(Succeed())
}

// setupMachineCRDs creates a CAPI and CloudStack machine with an appropriate ownership ref between them.
func setupMachineCRDs() {
	//  Create them.
	Ω(k8sClient.Create(ctx, dummies.CAPIMachine)).Should(Succeed())
	Ω(k8sClient.Create(ctx, dummies.CSMachine1)).Should(Succeed())

	// Fetch the CS Machine that was created.
	key := client.ObjectKey{Namespace: dummies.CSCluster.Namespace, Name: dummies.CSMachine1.Name}
	Eventually(func() error {
		return k8sClient.Get(ctx, key, dummies.CSMachine1)
	}, timeout).Should(BeNil())

	// Set owner ref from CAPI machine to CS machine and patch back the CS machine.
	Eventually(func() error {
		ph, err := patch.NewHelper(dummies.CSMachine1, k8sClient)
		Ω(err).ShouldNot(HaveOccurred())
		dummies.CSMachine1.OwnerReferences = append(dummies.CSMachine1.OwnerReferences, metav1.OwnerReference{
			Kind:       "Machine",
			APIVersion: clusterv1.GroupVersion.String(),
			Name:       dummies.CAPIMachine.Name,
			UID:        "uniqueness",
		})
		return ph.Patch(ctx, dummies.CSMachine1, patch.WithStatusObservedGeneration{})
	}, timeout).Should(Succeed())
}
