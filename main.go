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
	"time"

	"k8s.io/klog/v2/klogr"

	flag "github.com/spf13/pflag"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	goflag "flag"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"

	infrav1b1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	infrav1b2 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers"
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme))
	utilruntime.Must(infrav1b1.AddToScheme(scheme))
	utilruntime.Must(infrav1b2.AddToScheme(scheme))
	utilruntime.Must(controlplanev1.AddToScheme(scheme))
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
	opts := setFlags()                                // Add our options to flag set.
	klog.InitFlags(nil)                               // Add klog options to flag set.
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine) // Merge klog's gofloag flags into the pflags.
	flag.Parse()

	ctrl.SetLogger(klogr.New())

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
	base := utils.ReconcilerBase{
		K8sClient:  mgr.GetClient(),
		BaseLogger: ctrl.Log.WithName("controllers"),
		Scheme:     mgr.GetScheme()}

	setupReconcilers(base, mgr)
	infrav1b2.K8sClient = base.K8sClient

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
	if err = (&infrav1b2.CloudStackCluster{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CloudStackCluster")
		os.Exit(1)
	}
	if err = (&infrav1b2.CloudStackMachine{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CloudStackMachine")
		os.Exit(1)
	}
	if err = (&infrav1b2.CloudStackMachineTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CloudStackMachineTemplate")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupReconcilers(base utils.ReconcilerBase, mgr manager.Manager) {
	if err := (&controllers.CloudStackClusterReconciler{ReconcilerBase: base}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackCluster")
		os.Exit(1)
	}
	if err := (&controllers.CloudStackMachineReconciler{ReconcilerBase: base}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackMachine")
		os.Exit(1)
	}
	// if err := (&controllers.CloudStackMachineStateCheckerReconciler{ReconcilerBase: base}).SetupWithManager(mgr); err != nil {
	// 	setupLog.Error(err, "unable to create controller", "controller", "CloudStackMachineStateChecker")
	// 	os.Exit(1)
	// }
	if err := (&controllers.CloudStackIsoNetReconciler{ReconcilerBase: base}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackIsoNetReconciler")
		os.Exit(1)
	}
	// if err := (&controllers.CloudStackAffinityGroupReconciler{ReconcilerBase: base}).SetupWithManager(mgr); err != nil {
	// 	setupLog.Error(err, "unable to create controller", "controller", "CloudStackAffinityGroup")
	// 	os.Exit(1)
	// }
	if err := (&controllers.CloudStackFailureDomainReconciler{ReconcilerBase: base}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackFailureDomain")
		os.Exit(1)
	}
}
