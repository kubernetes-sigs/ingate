package controller

//need to parse flags
//configs for gw and ingress
//start prom collector
//start prom http server
//start controller

import (
	"fmt"

	gwapi "github.com/kubernetes-sigs/ingate/internal/controlplane/gatewayapi"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	controllerName = "k8s.io/ingate"
)

func Start() error {

	//Create the ctrl runtime manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
	if err != nil {
		klog.ErrorS(err, "Failed to start InGate manager")
		return fmt.Errorf("failed to construct manager: %w", err)
	}

	//Start the gateway controller and reconciler 
	newGateWayReconciler := gwapi.NewGatewayReconciler(mgr)

	err = newGateWayReconciler.SetupWithManager(mgr)
	if err != nil {
		return err
	}

	klog.InfoS("Starting InGate Manager")
	return mgr.Start(ctrl.SetupSignalHandler())
}
