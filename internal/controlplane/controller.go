package controlplane

//need to parse flags
//configs for gw and ingress
//start prom collector
//start prom http server
//start controller

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	// Register core Kubernetes types
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	// Register Gateway API types
	utilruntime.Must(gatewayv1.AddToScheme(scheme))
	utilruntime.Must(gatewayv1beta1.AddToScheme(scheme))
}

const (
	inGateControllerName = "k8s.io/ingate"
)

func Start() error {

	// Register standard Kubernetes types (Pods, Deployments, etc)
	_ = clientgoscheme.AddToScheme(scheme)

	// Register Gateway API types (GatewayClass, Gateway, etc.)
	_ = gatewayv1.AddToScheme(scheme)

	// Create the ctrl runtime manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,  // All registered types
		HealthProbeBindAddress: ":9000", //needs a flag
		LeaderElection:         false,   //needs a flag
		LeaderElectionID:       inGateControllerName,
		Metrics: metricsserver.Options{
			BindAddress: ":8080", //needs a flag
		},
	})
	if err != nil {
		klog.ErrorS(err, "failed to construct InGate manager")
		return fmt.Errorf("failed to construct InGate manager: %w", err)
	}

	log.SetLogger(klog.Logger{})
	// Add health and readiness probes
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		klog.ErrorS(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		klog.ErrorS(err, "unable to set up ready check")
		return err
	}

	klog.Info("adding gateway class controller")
	// Create and Add Gateway Class reconciler to manager
	newGateWayClassReconciler := NewGatewayClassReconciler(mgr)

	err = newGateWayClassReconciler.SetupWithManager(mgr)
	if err != nil {
		return err
	}

	klog.Info("adding gateway controller")
	// Create and Add Gateway reconciler to manager
	newGateWayReconciler := NewGatewayReconciler(mgr)

	err = newGateWayReconciler.SetupWithManager(mgr)
	if err != nil {
		return err
	}

	klog.Info("Starting InGate Manager")
	return mgr.Start(ctrl.SetupSignalHandler())
}
