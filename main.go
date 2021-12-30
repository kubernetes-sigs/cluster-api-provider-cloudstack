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

package main

import (
	"cluster.x-k8s.io/cluster-api-provider-capc/pkg/cloud"
	"flag"
	"os"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrav1 "cluster.x-k8s.io/cluster-api-provider-capc/api/v1alpha3"
	"cluster.x-k8s.io/cluster-api-provider-capc/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(infrav1.AddToScheme(scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {

	// Parse args and setup logger.
	var cloudConfigFile string
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var watchingNamespace string
	var certDir string
	flag.StringVar(&cloudConfigFile, "cloud-config-file", "/config/cloud-config", "Overrides the default path to the cloud-config file that contains the CloudStack credentials.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", "localhost:8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&watchingNamespace, "namespace", "", "Namespace that the controller watches to reconcile cluster-api objects. If unspecified, the controller watches for cluster-api objects across all namespaces.")
	flag.StringVar(&certDir, "cert-dir", "", "Directory where webhook certs will be stored.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Setup CloudStack api client.
	client, err := cloud.NewClient(cloudConfigFile)
	if err != nil {
		if !strings.Contains(err.Error(), "Timeout") {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}
		setupLog.Info("cannot connect to CloudStack via client at startup time.  Pressing onward...")
	}
	setupLog.Info("CloudStack client initialized.")

	// Create the controller manager.
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "capc-leader-election-controller",
		Namespace:              watchingNamespace,
		CertDir:                certDir,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Register machine and cluster reconcilers with the controller manager.
	if err = (&controllers.CloudStackClusterReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Cluster"),
		Scheme: mgr.GetScheme(),
		CS:     client,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackCluster")
		os.Exit(1)
	}
	if err = (&controllers.CloudStackMachineReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Machine"),
		Scheme: mgr.GetScheme(),
		CS:     client,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackMachine")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	// Add health and ready checks.
	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Start the controller manager.
	if err = (&infrav1.CloudStackCluster{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CloudStackCluster")
		os.Exit(1)
	}
	if err = (&infrav1.CloudStackMachine{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CloudStackMachine")
		os.Exit(1)
	}
	if err = (&infrav1.CloudStackMachineTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CloudStackMachineTemplate")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
