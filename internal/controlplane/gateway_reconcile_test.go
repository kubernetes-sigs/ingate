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
	// builtin
	"context"
	"testing"

	// external
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

var valid = &gatewayv1.Gateway{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "ingate",
		Namespace: "default",
	},
	Spec: gatewayv1.GatewaySpec{
		GatewayClassName: "k8s.io/ingate",
	},
}

// deleting gateway
var deleted = &gatewayv1.Gateway{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "ingate-deleted",
		Namespace: "default",
	},
	Spec: gatewayv1.GatewaySpec{
		GatewayClassName: "k8s.io/ingate",
	},
}

// gateway not owned by InGate
var orphan = &gatewayv1.Gateway{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "ingate-orphan",
		Namespace: "default",
	},
	Spec: gatewayv1.GatewaySpec{
		GatewayClassName: "",
	},
}

// gateway with non-existent gateway class
var noClass = &gatewayv1.Gateway{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "ingate-no-class",
		Namespace: "default",
	},
	Spec: gatewayv1.GatewaySpec{
		GatewayClassName: "does-not-exist",
	},
}

func Test_Gateway_Reconciler(t *testing.T) {
	scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gatewayv1.Install(scheme))
	utilruntime.Must(gatewayv1beta1.Install(scheme))

	testClient := fake.NewClientBuilder().
		WithScheme(scheme).WithObjects(valid, deleted, orphan, noClass).
		WithStatusSubresource(&gatewayv1.Gateway{}).
		Build()

	r := &GatewayReconciler{
		Client: testClient,
		scheme: scheme,
	}

	// valid gateway
	t.Run("valid gateway", func(t *testing.T) {
		result, err := r.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: client.ObjectKey{
				Name:      "ingate",
				Namespace: "default",
			},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	})

	// deleting gateway
	t.Run("deleting gateway", func(t *testing.T) {
		result, err := r.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: client.ObjectKey{
				Name:      "ingate-deleted",
				Namespace: "default",
			},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	})

	// gateway not owned by InGate
	t.Run("gateway not owned by InGate", func(t *testing.T) {
		result, err := r.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: client.ObjectKey{
				Name:      "ingate-orphan",
				Namespace: "default",
			},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	})

	// gateway with non-existent gateway class
	t.Run("gateway with non-existent gateway class", func(t *testing.T) {
		result, err := r.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: client.ObjectKey{
				Name:      "ingate-no-class",
				Namespace: "default",
			},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	})

	// gateway does not exist
	t.Run("gateway not owned by InGate", func(t *testing.T) {
		result, err := r.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: client.ObjectKey{
				Name:      "ingate-non-existent",
				Namespace: "default",
			},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	})
}
