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
	"context"
	"k8s.io/client-go/util/retry"

	//external
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// GatewayReconciler reconciles a Gateway object
type GatewayReconciler struct {
	client.Client
	scheme *runtime.Scheme
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var gw gatewayv1.Gateway

	klog.Infof("starting reconcile of gateway %s", req.String())

	if err := r.Get(ctx, req.NamespacedName, &gw); err != nil {
		// Could not get GatewayClass (maybe deleted)
		klog.Infof("gateway not found in namespace %s ", req.NamespacedName)
		if apierrors.IsNotFound(err) {
			klog.Infof("gateway class %s not found", gw.Name)
			return reconcile.Result{}, client.IgnoreNotFound(err)
		}

		return reconcile.Result{}, err
	}

	klog.Infof("reconciling gateway %s", gw.Name)
	// Only manage GatewayClasses with our specific controllerName
	gwc := &gatewayv1.GatewayClass{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: string(gw.Spec.GatewayClassName)}, gwc); err != nil {
		klog.Infof("GatewayClassName does not match %s ", req.NamespacedName)
		return reconcile.Result{}, nil
	}

	if string(gwc.Spec.ControllerName) != inGateControllerName {
		klog.Infof("Nothing to do, GatewayClass %s does not have matching controller name %s", gwc.Spec.ControllerName, inGateControllerName)
		return reconcile.Result{}, nil
	}

	// Update status to Accepted=True
	gwStatusCondition := []metav1.Condition{
		{
			Type:               string(gatewayv1.GatewayClassConditionStatusAccepted),
			Status:             metav1.ConditionTrue,
			Reason:             "Accepted",
			Message:            "Gateway has been accepted by the InGate Controller.",
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: gw.GetGeneration(),
		},
	}

	gw.Status.Conditions = gwStatusCondition
	klog.Infof("accepted gateway %s", gw.Name)
	err := r.Status().Update(ctx, &gw)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Infof("gateway class %s not found", gwc.Name)
			return reconcile.Result{}, err
		}
		if apierrors.IsConflict(err) {
			klog.Infof("gateway class %s conflict, requeuing", gwc.Name)
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				gwRetry := &gatewayv1.Gateway{}
				if err := r.Get(ctx, req.NamespacedName, gwRetry); err != nil {
					klog.Warningf("error getting gateway %s/%s retrying", gw.Namespace, gw.Name)
					return err // Return non-nil error to retry.RetryOnConflict
				}

				return r.Status().Update(ctx, gwRetry)
			})
			if err != nil {
				klog.Warningf("failed to update gateway on retry %s", gwc.Name)
				return reconcile.Result{}, err
			}
		} //end conflict loop
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
