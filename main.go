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

package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog"
	"k8s.io/klog/klogr"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"

	infrastructurev1beta1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/controllers"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
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
	utilruntime.Must(controlplanev1.AddToScheme(scheme))
	utilruntime.Must(infrastructurev1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type managerOpts struct {
	CloudConfigFile      string
	MetricsAddr          string
	EnableLeaderElection bool
	ProbeAddr            string
	WatchingNamespace    string
	WatchFilterValue     string
	CertDir              string
}

func setFlags() *managerOpts {
	opts := &managerOpts{}
	flag.StringVar(
		&opts.CloudConfigFile,
		"cloud-config-file",
		"/config/cloud-config",
		"Overrides the default path to the cloud-config file that contains the CloudStack credentials.")
	flag.StringVar(
		&opts.MetricsAddr,
		"metrics-bind-addr",
		"localhost:8080",
		"The address the metric endpoint binds to.")
	flag.StringVar(
		&opts.ProbeAddr,
		"health-probe-bind-address",
		":8081",
		"The address the probe endpoint binds to.")
	flag.BoolVar(
		&opts.EnableLeaderElection,
		"leader-elect",
		false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(
		&opts.WatchingNamespace,
		"namespace",
		"",
		"Namespace that the controller watches to reconcile cluster-api objects. If unspecified, "+
			"the controller watches for cluster-api objects across all namespaces.")
	flag.StringVar( // TODO: use filter per CAPI book instructions in upgrade to v1alpha4.
		&opts.WatchFilterValue,
		"watch-filter",
		"",
		fmt.Sprintf(
			"Label value that the controller watches to reconcile cluster-api objects. "+
				"Label key is always %s. If unspecified, the controller watches for all cluster-api objects.",
			clusterv1.WatchLabel))
	flag.StringVar(
		&opts.CertDir,
		"webhook-cert-dir",
		"/tmp/k8s-webhook-server/serving-certs/",
		"Specify the directory where webhooks will get tls certificates.")
	return opts
}

func main() {
	opts := setFlags()  // Add our options to flag set.
	klog.InitFlags(nil) // Add klog options to flag set.
	flag.Parse()

	ctrl.SetLogger(klogr.New())

	// Setup CloudStack api client.
	client, err := cloud.NewClient(opts.CloudConfigFile)
	if err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "timeout") {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}
		setupLog.Info("cannot connect to CloudStack via client at startup time.  Pressing onward...")
	}
	setupLog.Info("CloudStack client initialized.")

	// Create the controller manager.
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     opts.MetricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: opts.ProbeAddr,
		LeaderElection:         opts.EnableLeaderElection,
		LeaderElectionID:       "capc-leader-election-controller",
		Namespace:              opts.WatchingNamespace,
		CertDir:                opts.CertDir,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Set a random seed for randomly placing CloudStackMachines in Zones.
	rand.Seed(time.Now().Unix())

	// Register reconcilers with the controller manager.
	base := csCtrlrUtils.ReconcilerBase{
		Client:     mgr.GetClient(),
		BaseLogger: ctrl.Log.WithName("controllers"),
		Scheme:     mgr.GetScheme(),
		CS:         client}
	if err = (&controllers.CloudStackClusterReconciler{ReconcilerBase: base}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackCluster")
		os.Exit(1)
	}
	if err = (&controllers.CloudStackZoneReconciler{ReconcilerBase: base}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackZone")
		os.Exit(1)
	}
	if err = (&controllers.CloudStackMachineReconciler{ReconcilerBase: base}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackMachine")
		os.Exit(1)
	}
	if err = (&controllers.CloudStackMachineStateCheckerReconciler{ReconcilerBase: base}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackMachineStateChecker")
		os.Exit(1)
	}
	if err = (&controllers.CloudStackIsoNetReconciler{ReconcilerBase: base}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackIsoNetReconciler")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

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
