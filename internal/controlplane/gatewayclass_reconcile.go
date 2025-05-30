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
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// GatewayClassReconciler reconciles a Gateway Class object
type GatewayClassReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *GatewayClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	klog.Info("starting reconcile gateway class")
	var gwc gatewayv1.GatewayClass

	if err := r.Get(ctx, req.NamespacedName, &gwc); err != nil {
		// Could not get GatewayClass (maybe deleted)
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	klog.Infof("reconciling gateway class %s", gwc.Name)
	// Only manage GatewayClasses with our specific controllerName
	if gwc.Spec.ControllerName != inGateControllerName {
		klog.Infof("gateway class does not match controller %s/%s", gwc.Namespace, gwc.Name)
		return reconcile.Result{}, nil
	}

	// Gateway Class is being deleted
	if gwc.GetDeletionTimestamp() != nil {
		klog.Infof("gateway class is being deleted %s/%s", gwc.Namespace, gwc.Name)
		return reconcile.Result{}, nil
	}

	// Update status to GW Class Accepted True
	gwc.Status.Conditions = []metav1.Condition{
		{
			Type:               string(gatewayv1.GatewayClassConditionStatusAccepted),
			Status:             metav1.ConditionTrue,
			Reason:             "Accepted",
			Message:            "Gateway Class has been accepted by the InGate Controller.",
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: gwc.GetGeneration(),
		},
	}

	klog.Infof("accepted gateway class %s", gwc.Name)

	if err := r.Status().Update(ctx, &gwc); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
