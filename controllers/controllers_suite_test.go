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
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	csReconcilers "sigs.k8s.io/cluster-api-provider-cloudstack/controllers"
	csCtrlrUtils "sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/mocks"

	"sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	//+kubebuilder:scaffold:imports
)

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

var ( // TestEnv vars...
	testEnv   *envtest.Environment
	k8sClient client.Client
	ctx       context.Context
	cancel    context.CancelFunc
)
var ( // Mock var.
	mockCtrl          *gomock.Controller
	mockCloudClient   *mocks.MockClient
	mockCSAPIClient   *cloudstack.CloudStackClient
	MachineReconciler *csReconcilers.CloudStackMachineReconciler
)

func TestAPIs(t *testing.T) {

	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {

	projectDir := os.Getenv("PROJECT_DIR")

	// Add ginkgo recover statements to controllers.
	cmd := exec.Command(projectDir+"/hack/testing_ginkgo_recover_statements.sh", "--add")
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Println(errors.Wrapf(err, "adding gingko statements"))
		os.Exit(1)
	}

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")

	crdPaths := []string{
		filepath.Join(projectDir, "config", "crd", "bases"),
	}

	// Append CAPI CRDs path
	if capiPath := getFilePathToCAPICRDs(projectDir); capiPath != "" {
		crdPaths = append(crdPaths, capiPath)
	}

	Ω(infrav1.AddToScheme(scheme.Scheme)).Should(Succeed())
	Ω(clusterv1.AddToScheme(scheme.Scheme)).Should(Succeed())

	// Create the test environment.
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     crdPaths,
	}
	var cfg *rest.Config
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

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	Ω(err).ShouldNot(HaveOccurred())

	logger := klogr.New()

	klog.InitFlags(nil) // Add klog options to flag set.
	flag.Lookup("v").Value.Set("1")
	flag.Parse()

	// Base reconciler shared across reconcilers.
	base := csCtrlrUtils.ReconcilerBase{
		K8sClient:  k8sManager.GetClient(),
		Scheme:     k8sManager.GetScheme(),
		CSClient:   mockCloudClient,
		BaseLogger: logger}

	ctrl.SetLogger(logger)

	// Setup each specific reconciler.
	// ClusterReconciler := &csReconcilers.CloudStackClusterReconciler{ReconcilerBase: base}
	// Ω(ClusterReconciler.SetupWithManager(k8sManager)).Should(Succeed())
	MachineReconciler = &csReconcilers.CloudStackMachineReconciler{ReconcilerBase: base}
	Ω(MachineReconciler.SetupWithManager(k8sManager)).Should(Succeed())
	// ZoneReconciler := &csReconcilers.CloudStackZoneReconciler{ReconcilerBase: base}
	// Ω(ZoneReconciler.SetupWithManager(k8sManager)).Should(Succeed())
	// IsoNetReconciler := &csReconcilers.CloudStackIsoNetReconciler{ReconcilerBase: base}
	// Ω(IsoNetReconciler.SetupWithManager(k8sManager)).Should(Succeed())
	// AffinityGReconciler := &csReconcilers.CloudStackAffinityGroupReconciler{ReconcilerBase: base}
	// Ω(AffinityGReconciler.SetupWithManager(k8sManager)).Should(Succeed())

	go func() {
		defer GinkgoRecover()
		Ω(k8sManager.Start(ctrl.SetupSignalHandler())).Should(Succeed(), "failed to run manager")
	}()
})

// Setup and teardown on a per test basis.
var _ = BeforeEach(func() {
	mockCtrl = gomock.NewController(GinkgoT())

	// Setup mock clients.
	mockCSAPIClient = cloudstack.NewMockClient(mockCtrl)
	mockCloudClient = mocks.NewMockClient(mockCtrl)
	MachineReconciler.CSClient = mockCloudClient

	dummies.SetDummyVars()
	setupClusterCRDs()
})

var _ = AfterEach(func() {
	// Finishint the mockCtrl checks expected calls on mock objects matched.
	mockCtrl.Finish()

	// Delete any CRDs left by test.
	cleanupCRDs()
})

var _ = AfterSuite(func() {
	projectDir := os.Getenv("PROJECT_DIR")
	// Add ginkgo recover statements to controllers.
	cmd := exec.Command(projectDir+"/hack/testing_ginkgo_recover_statements.sh", "--remove")
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Println(errors.Wrapf(err, "cleaning up gingko statements"))
		os.Exit(1)
	}
	cancel()
	By("tearing down the test environment")
	Ω(testEnv.Stop()).Should(Succeed())

})

// cleanupCRDs deletes all CRDs in the dummies CSClusterNamespace.
func cleanupCRDs() {
	nameSpaceFilter := &client.DeleteAllOfOptions{ListOptions: client.ListOptions{Namespace: dummies.ClusterNameSpace}}
	Ω(k8sClient.DeleteAllOf(ctx, &clusterv1.Cluster{}, nameSpaceFilter)).Should(Succeed())
	Ω(k8sClient.DeleteAllOf(ctx, &infrav1.CloudStackCluster{}, nameSpaceFilter)).Should(Succeed())
	Ω(k8sClient.DeleteAllOf(ctx, &infrav1.CloudStackMachine{}, nameSpaceFilter)).Should(Succeed())
	Ω(k8sClient.DeleteAllOf(ctx, &infrav1.CloudStackZone{}, nameSpaceFilter)).Should(Succeed())
	Ω(k8sClient.DeleteAllOf(ctx, &infrav1.CloudStackAffinityGroup{}, nameSpaceFilter)).Should(Succeed())
	Ω(k8sClient.DeleteAllOf(ctx, &infrav1.CloudStackIsolatedNetwork{}, nameSpaceFilter)).Should(Succeed())
}

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
