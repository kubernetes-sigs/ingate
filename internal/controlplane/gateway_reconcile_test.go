package controlplane

import (
	"context"
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
	"testing"
)

var valid = &gatewayv1.Gateway{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "ingate",
		Namespace: "default",
	},
	Spec: gatewayv1.GatewaySpec{
		GatewayClassName: "ingate",
	},
}

// deleting gateway
var deleted = &gatewayv1.Gateway{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "ingate-deleted",
		Namespace: "default",
	},
	Spec: gatewayv1.GatewaySpec{
		GatewayClassName: "ingate",
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
	utilruntime.Must(gatewayv1.AddToScheme(scheme))
	utilruntime.Must(gatewayv1beta1.AddToScheme(scheme))

	testClient := fake.NewClientBuilder().
		WithScheme(scheme).WithObjects(valid, deleted, orphan, noClass).
		WithStatusSubresource(&gatewayv1.Gateway{}).
		Build()

	r := &GatewayReconciler{
		Client: testClient,
		Scheme: scheme,
	}

	//valid gateway
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

	//deleting gateway
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
