/*
Copyright 2025 The Kubernetes Authors.

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

package controlplane

import (
	//builtin
	"fmt"
	//external
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	// Register core Kubernetes types
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	// Register Gateway API types
	utilruntime.Must(gatewayv1.Install(scheme))
}

const (
	inGateControllerName = "k8s.io/ingate"
)

func Start() error {

	logger := klog.NewKlogr()
	ctrl.SetLogger(logger)

	ctx := ctrl.SetupSignalHandler()

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

	err = newGateWayClassReconciler.SetupWithManager(ctx, mgr)
	if err != nil {
		return err
	}

	klog.Info("adding gateway controller")
	// Create and Add Gateway reconciler to manager
	newGateWayReconciler := NewGatewayReconciler(ctx, mgr)
	err = newGateWayReconciler.SetupWithManager(ctx, mgr)
	if err != nil {
		return err
	}

	klog.Info("Starting InGate Manager")
	if err := mgr.Start(ctx); err != nil {
		klog.Errorf("problem running manager %s", err.Error())
		return err
	}

	return nil
}
