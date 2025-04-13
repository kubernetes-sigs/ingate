package gatewayapi

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// GatewayReconciler reconciles a Gateway object
type GatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	klog.InfoS("reconciling Gateway")
	crtlResult := ctrl.Result{
		Requeue:      false,
		RequeueAfter: 0,
	}
	return crtlResult, nil
}

func NewGatewayReconciler(mgr ctrl.Manager) *GatewayReconciler {
	return &GatewayReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
}

func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	klog.InfoS("starting gateway controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.Gateway{}).
		Complete(r)
}
