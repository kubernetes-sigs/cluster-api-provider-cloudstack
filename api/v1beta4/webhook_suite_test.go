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

package v1beta4_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"time"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"

	//+kubebuilder:scaffold:imports
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var k8sClient client.Client
var testEnv *envtest.Environment
var webhookCtx context.Context
var webhookCancel context.CancelFunc

var _ = ginkgo.BeforeSuite(func() {
	webhookCtx, webhookCancel = context.WithCancel(context.TODO())

	ginkgo.By("bootstrapping webhook test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("../../", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("../../", "config", "webhook")},
		},
	}

	var cfg *rest.Config
	var err error
	done := make(chan any)
	go func() {
		defer ginkgo.GinkgoRecover()
		cfg, err = testEnv.Start()
		close(done)
	}()
	gomega.Eventually(done).WithTimeout(time.Minute).Should(gomega.BeClosed())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(cfg).NotTo(gomega.BeNil())

	scheme := runtime.NewScheme()
	err = infrav1.AddToScheme(scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	err = admissionv1beta1.AddToScheme(scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(k8sClient).NotTo(gomega.BeNil())

	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics: server.Options{
			BindAddress: "0",
		},
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	gomega.Ω((&infrav1.CloudStackCluster{}).SetupWebhookWithManager(mgr)).Should(gomega.Succeed())
	gomega.Ω((&infrav1.CloudStackMachine{}).SetupWebhookWithManager(mgr)).Should(gomega.Succeed())
	gomega.Ω((&infrav1.CloudStackMachineTemplate{}).SetupWebhookWithManager(mgr)).Should(gomega.Succeed())

	//+kubebuilder:scaffold:webhook

	go func() {
		defer ginkgo.GinkgoRecover()
		err = mgr.Start(webhookCtx)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}()

	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	gomega.Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}).Should(gomega.Succeed())
})

var _ = ginkgo.AfterSuite(func() {
	webhookCancel()
	ginkgo.By("tearing down the webhook test environment")
	err := testEnv.Stop()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
})
