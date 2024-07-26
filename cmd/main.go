/*
Copyright 2023.

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

	rcsv1alpha1 "github.com/dana-team/rcs-ocm-deployer/api/v1alpha1"
	rcswebhooks "github.com/dana-team/rcs-ocm-deployer/internal/webhooks"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	placementctrl "github.com/dana-team/rcs-ocm-deployer/internal/placement/controller"
	syncctrl "github.com/dana-team/rcs-ocm-deployer/internal/sync/controller"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/go-logr/zapr"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	workv1 "open-cluster-management.io/api/work/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	runtimezap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(knativev1.AddToScheme(scheme))
	utilruntime.Must(clusterv1.Install(scheme))
	utilruntime.Must(clusterv1beta1.Install(scheme))
	utilruntime.Must(workv1.Install(scheme))
	utilruntime.Must(cappv1alpha1.AddToScheme(scheme))
	utilruntime.Must(rcsv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func initEcsLogger() {
	encoderConfig := ecszap.NewDefaultEncoderConfig()
	core := ecszap.NewCore(encoderConfig, os.Stdout, zap.DebugLevel)
	logger := zap.New(core, zap.AddCaller())
	logf.SetLogger(zapr.NewLogger(logger))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var ecsLogging bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&ecsLogging, "ecs-logging", true, "Display controller logs in ecs format.")

	flag.Parse()

	if ecsLogging {
		initEcsLogger()
	} else {
		ctrl.SetLogger(runtimezap.New())
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer:          webhook.NewServer(webhook.Options{Port: 9443}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "80807133.rcs.dana.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&syncctrl.SyncReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("sync-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Capp")
		os.Exit(1)
	}

	if err = (&placementctrl.PlacementReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("placement-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Capp")
		os.Exit(1)
	}

	hookServer := mgr.GetWebhookServer()
	decoder := admission.NewDecoder(scheme)
	hookServer.Register(rcswebhooks.ValidatorServingPath, &webhook.Admission{Handler: &rcswebhooks.CappValidator{
		Client:  mgr.GetClient(),
		Decoder: decoder,
	}})

	hookServer.Register(rcswebhooks.MutatorServingPath, &webhook.Admission{Handler: &rcswebhooks.CappMutator{
		Client:  mgr.GetClient(),
		Decoder: decoder,
	}})

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
