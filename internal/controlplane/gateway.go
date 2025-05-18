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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func NewGatewayReconciler(mgr ctrl.Manager) *GatewayReconciler {
	return &GatewayReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
}

func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	klog.Info("setting up gateway controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.Gateway{},
			builder.WithPredicates(predicate.NewPredicateFuncs(MatchControllerName(inGateControllerName)))).
		// Watch GatewayClass resources, which are linked to Gateway
		Watches(&gatewayv1.GatewayClass{},
			r.RetrieveGateClassResources(),
			builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
				klog.V(2).Infof("checking gateway class %s", object.GetName())
				return object.(*gatewayv1.GatewayClass).Spec.ControllerName == inGateControllerName
			}))).
		Complete(r)
}

func (r *GatewayReconciler) RetrieveGateClassResources() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {

		var reqs []reconcile.Request
		gwList := &gatewayv1.GatewayList{}
		if err := r.Client.List(ctx, gwList); err != nil {
			klog.Errorf("Unable to list Gateways %s", err)
			return nil
		}

		for _, gw := range gwList.Items {
			if gw.Spec.GatewayClassName != gatewayv1.ObjectName(a.GetName()) {
				klog.V(2).Infof("skipping gateway %s does not match class %s", gw.Name, gw.Spec.GatewayClassName)
				continue
			}

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: gw.Namespace,
					Name:      gw.Name,
				},
			}
			reqs = append(reqs, req)
			klog.Infof("Queueing gateway requests in namespace %s resource %s for gateway %s", gw.GetNamespace(), req.Name, gw.GetName())
		}
		return reqs
	})
}
