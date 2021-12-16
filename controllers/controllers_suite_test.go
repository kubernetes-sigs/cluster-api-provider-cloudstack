/*
Copyright 2021.

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
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrav1 "cluster.x-k8s.io/cluster-api-provider-capc/api/v1alpha3"
	csReconcilers "cluster.x-k8s.io/cluster-api-provider-capc/controllers"
	"cluster.x-k8s.io/cluster-api-provider-capc/pkg/mocks"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	//+kubebuilder:scaffold:imports
)

var (
	clusterAPIVersionRegex = regexp.MustCompile(`^(\W)sigs.k8s.io/cluster-api v(.+)`)
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
	mockClient        *cloudstack.CloudStackClient
	ClusterReconciler *csReconcilers.CloudStackClusterReconciler
	CS                *mocks.MockClient
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	// Check for ginkgo recover statements.
	cmd := exec.Command("../hack/testing_ginkgo_recover_statements.sh", "--contains")
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Println("Refusing to run tests without ginkgo recover set.")
		os.Exit(1)
	}

	ctx, cancel = context.WithCancel(context.TODO())

	// Setup mock CloudStack client.
	mockCtrl = gomock.NewController(GinkgoT())
	mockClient = cloudstack.NewMockClient(mockCtrl)
	CS = mocks.NewMockClient(mockCtrl)

	By("bootstrapping test environment")
	// Get the root of the current file to use in CRD paths.
	_, filename, _, _ := goruntime.Caller(0) //nolint
	root := path.Join(path.Dir(filename), "..")
	fmt.Println(root)

	crdPaths := []string{
		filepath.Join(root, "config", "crd", "bases"),
	}

	// Append CAPI CRDs path
	if capiPath := getFilePathToCAPICRDs(root); capiPath != "" {
		crdPaths = append(crdPaths, capiPath)
	}

	// Create the test environment.
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     crdPaths,
	}

	Ω(infrav1.AddToScheme(scheme.Scheme)).Should(Succeed())
	Ω(clusterv1.AddToScheme(scheme.Scheme)).Should(Succeed())

	cfg, err := testEnv.Start()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(cfg).ShouldNot(BeNil())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Ω(err).ShouldNot(HaveOccurred())
	Ω(k8sClient).ShouldNot(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Ω(err).ShouldNot(HaveOccurred())

	ClusterReconciler = &csReconcilers.CloudStackClusterReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
		CS:     CS,
	}
	Ω(ClusterReconciler.SetupWithManager(k8sManager)).Should(Succeed())

	go func() {
		defer GinkgoRecover()
		Ω(k8sManager.Start(ctrl.SetupSignalHandler())).Should(Succeed(), "failed to run manager")
	}()

}, 60)

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	Ω(testEnv.Stop()).Should(Succeed())
	mockCtrl.Finish()
})
