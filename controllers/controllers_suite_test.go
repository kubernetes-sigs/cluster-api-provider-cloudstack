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
	"fmt"
	"go/build"
	"k8s.io/client-go/rest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	csReconcilers "github.com/aws/cluster-api-provider-cloudstack/controllers"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/mocks"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {

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

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Ω(err).ShouldNot(HaveOccurred())

	base := csCtrlrUtils.ReconcilerBase{
		K8sClient:  k8sManager.GetClient(),
		Scheme:     k8sManager.GetScheme(),
		CSClient:   CS,
		BaseLogger: logr.Discard()}
	ClusterReconciler = &csReconcilers.CloudStackClusterReconciler{ReconcilerBase: base}
	Ω(ClusterReconciler.SetupWithManager(k8sManager)).Should(Succeed())

	go func() {
		defer GinkgoRecover()
		Ω(k8sManager.Start(ctrl.SetupSignalHandler())).Should(Succeed(), "failed to run manager")
	}()

})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	Ω(testEnv.Stop()).Should(Succeed())
	mockCtrl.Finish()
})
