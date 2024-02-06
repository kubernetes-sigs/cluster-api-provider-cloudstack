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
	"context"
	"fmt"
	"os"

	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"

	flag "github.com/spf13/pflag"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	goflag "flag"

	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	cgrecord "k8s.io/client-go/tools/record"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	logsv1 "k8s.io/component-base/logs/api/v1"
	"sigs.k8s.io/cluster-api/util/flags"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"

	infrav1b1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	infrav1b2 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	infrav1b3 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers"
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	tlsOptions = flags.TLSOptions{}
	logOptions = logs.NewOptions()
)

func init() {
	klog.InitFlags(nil)

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme))
	utilruntime.Must(infrav1b1.AddToScheme(scheme))
	utilruntime.Must(infrav1b2.AddToScheme(scheme))
	utilruntime.Must(infrav1b3.AddToScheme(scheme))
	utilruntime.Must(controlplanev1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type managerOpts struct {
	CloudConfigFile      string
	MetricsAddr          string
	EnableLeaderElection bool
	ProbeAddr            string
	WatchNamespace       string
	WatchFilterValue     string
	ProfilerAddr         string
	WebhookCertDir       string
	WebhookPort          int

	CloudStackClusterConcurrency       int
	CloudStackMachineConcurrency       int
	CloudStackAffinityGroupConcurrency int
	CloudStackFailureDomainConcurrency int
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
		&opts.WatchNamespace,
		"namespace",
		"",
		"Namespace that the controller watches to reconcile cluster-api objects. If unspecified, "+
			"the controller watches for cluster-api objects across all namespaces.")
	flag.StringVar(
		&opts.WatchFilterValue,
		"watch-filter",
		"",
		fmt.Sprintf(
			"Label value that the controller watches to reconcile cluster-api objects. "+
				"Label key is always %s. If unspecified, the controller watches for all cluster-api objects.",
			clusterv1.WatchLabel))
	flag.StringVar(
		&opts.ProfilerAddr,
		"profiler-address",
		"",
		"Bind address to expose the pprof profiler (e.g. localhost:6060)")
	flag.IntVar(
		&opts.WebhookPort,
		"webhook-port",
		9443,
		"The webhook server port the manager will listen on.")
	flag.StringVar(
		&opts.WebhookCertDir,
		"webhook-cert-dir",
		"/tmp/k8s-webhook-server/serving-certs/",
		"Specify the directory where webhooks will get tls certificates.")
	flag.IntVar(
		&opts.CloudStackClusterConcurrency,
		"cloudstackcluster-concurrency",
		10,
		"Maximum concurrent reconciles for CloudStackCluster resources",
	)
	flag.IntVar(
		&opts.CloudStackMachineConcurrency,
		"cloudstackmachine-concurrency",
		10,
		"Maximum concurrent reconciles for CloudStackMachine resources",
	)
	flag.IntVar(
		&opts.CloudStackAffinityGroupConcurrency,
		"cloudstackaffinitygroup-concurrency",
		5,
		"Maximum concurrent reconciles for CloudStackAffinityGroup resources",
	)
	flag.IntVar(
		&opts.CloudStackFailureDomainConcurrency,
		"cloudstackfailuredomain-concurrency",
		5,
		"Maximum concurrent reconciles for CloudStackFailureDomain resources",
	)

	return opts
}

func main() {
	opts := setFlags() // Add our options to flag set.
	logsv1.AddFlags(logOptions, flag.CommandLine)
	flags.AddTLSOptions(flag.CommandLine, &tlsOptions)
	flag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine) // Merge klog's goflag flags into the pflags.
	flag.Parse()

	if err := logsv1.ValidateAndApply(logOptions, nil); err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctrl.SetLogger(klogr.New())

	tlsOptionOverrides, err := flags.GetTLSOptionOverrideFuncs(tlsOptions)
	if err != nil {
		setupLog.Error(err, "unable to add TLS settings to the webhook server")
		os.Exit(1)
	}

	var watchNamespaces []string
	if opts.WatchNamespace != "" {
		setupLog.Info("Watching cluster-api objects only in namespace for reconciliation", "namespace", opts.WatchNamespace)
		watchNamespaces = []string{opts.WatchNamespace}
	}

	// Machine and cluster operations can create enough events to trigger the event recorder spam filter
	// Setting the burst size higher ensures all events will be recorded and submitted to the API
	broadcaster := cgrecord.NewBroadcasterWithCorrelatorOptions(cgrecord.CorrelatorOptions{
		BurstSize: 100,
	})

	// Define user agent for the controller
	restConfig := ctrl.GetConfigOrDie()
	restConfig.UserAgent = "cluster-api-provider-cloudstack-controller"

	// Create the controller manager.
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     opts.MetricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: opts.ProbeAddr,
		LeaderElection:         opts.EnableLeaderElection,
		LeaderElectionID:       "capc-leader-election-controller",
		PprofBindAddress:       opts.ProfilerAddr,
		Cache: cache.Options{
			Namespaces: watchNamespaces,
		},
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&corev1.ConfigMap{},
					&corev1.Secret{},
				},
			},
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    opts.WebhookPort,
			CertDir: opts.WebhookCertDir,
			TLSOpts: tlsOptionOverrides,
		}),
		EventBroadcaster: broadcaster,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Register reconcilers with the controller manager.
	base := utils.ReconcilerBase{
		K8sClient:        mgr.GetClient(),
		Recorder:         mgr.GetEventRecorderFor("capc-controller-manager"),
		Scheme:           mgr.GetScheme(),
		WatchFilterValue: opts.WatchFilterValue,
	}

	ctx := ctrl.SetupSignalHandler()
	setupReconcilers(ctx, base, *opts, mgr)
	infrav1b3.K8sClient = base.K8sClient

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
	if err = (&infrav1b3.CloudStackCluster{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CloudStackCluster")
		os.Exit(1)
	}
	if err = (&infrav1b3.CloudStackMachine{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CloudStackMachine")
		os.Exit(1)
	}
	if err = (&infrav1b3.CloudStackMachineTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CloudStackMachineTemplate")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupReconcilers(ctx context.Context, base utils.ReconcilerBase, opts managerOpts, mgr manager.Manager) {
	if err := (&controllers.CloudStackClusterReconciler{ReconcilerBase: base}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.CloudStackClusterConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackCluster")
		os.Exit(1)
	}
	if err := (&controllers.CloudStackMachineReconciler{ReconcilerBase: base}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.CloudStackMachineConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackMachine")
		os.Exit(1)
	}
	if err := (&controllers.CloudStackIsoNetReconciler{ReconcilerBase: base}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackIsoNetReconciler")
		os.Exit(1)
	}
	if err := (&controllers.CloudStackAffinityGroupReconciler{ReconcilerBase: base}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.CloudStackAffinityGroupConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackAffinityGroup")
		os.Exit(1)
	}
	if err := (&controllers.CloudStackFailureDomainReconciler{ReconcilerBase: base}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.CloudStackFailureDomainConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudStackFailureDomain")
		os.Exit(1)
	}
}
