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
	"flag"
	"os"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"gopkg.in/ini.v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrastructurev1alpha4 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
	"gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/controllers"
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
	utilruntime.Must(infrastructurev1alpha4.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// Simple CloudStack API config reader. Works for now.
func readAPIConfig() (string, string, string) {
	cfg, err := ini.Load("/config/cloud-config")
	if err != nil {
		setupLog.Error(err, "could not read cloud-config")
		os.Exit(1)
	}
	g := cfg.Section("Global")
	return g.Key("api-url").Value(), g.Key("api-key").Value(), g.Key("secret-key").Value()
}

func main() {

	// Parse args and setup logger.
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", "localhost:8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Development: true})))

	// Setup CloudStack api client.
	// TODO: turn on ssl verification in production.
	apiUrl, apiKey, secretKey := readAPIConfig()
	cs := cloudstack.NewAsyncClient(apiUrl, apiKey, secretKey, false)

	// Create the controller manager.
	mgr, err := ctrl.NewManager(config.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "capc-leader-election-controller",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Register machine and cluster reconcilers with the controller manager.
	ctx := ctrl.SetupSignalHandler()
	if err = (&controllers.CloudStackClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		CS:     cs,
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create CloudStack cluster controller")
		os.Exit(1)
	}
	if err = (&controllers.CloudStackMachineReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		CS:     cs,
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create CloudStack machine controller")
		os.Exit(1)
	}

	// Add health and ready checks.
	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to setup health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to setup ready check")
		os.Exit(1)
	}

	// Start the controller manager.
	// +kubebuilder:scaffold:builder
	setupLog.Info("starting controller manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "could not start the controller manager")
		os.Exit(1)
	}
}
