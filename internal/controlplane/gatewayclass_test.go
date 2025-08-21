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

package controlplane_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	v1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kubernetes-sigs/ingate/internal/controlplane"
	"github.com/kubernetes-sigs/ingate/test/framework"
)

func TestGatewayClassReconcile(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()

	// For now on InGate we care just about "standard" APIs.
	corev1.AddToScheme(scheme)
	v1.Install(scheme)

	// TODO: right now we are hardcoding kubernetes version to latest, and "stable" GW API channel
	// we can make it configuratble further for other tests
	envtest, restconfig, err := framework.StartEnvTest(scheme, "", "")
	require.NoError(t, err)

	k8sClient, err := client.New(restconfig, client.Options{
		Scheme: scheme,
	})
	require.NoError(t, err)

	// managerClient is the client used by reconciliation/controller runtime
	// it is a cached client, and we will use it to assure that caching options are
	// also working properly (eg.: skipping a gateway class we don't care)
	var managerClient client.Client

	t.Cleanup(func() {
		// Ignore the error for now
		envtest.Stop()
	})

	t.Run("should configure the manager and reconciler properly", func(t *testing.T) {
		mgr, err := ctrl.NewManager(restconfig, ctrl.Options{
			Scheme: scheme,
		})
		require.NoError(t, err)
		// TODO: why are we repeating the manager here so much? :)
		require.NoError(t, controlplane.NewGatewayClassReconciler(mgr).SetupWithManager(ctx, mgr))
		go mgr.Start(ctx)
		require.True(t, mgr.GetCache().WaitForCacheSync(ctx))
		managerClient = mgr.GetClient()
	})

	t.Run("when reconciling a gatewayclass", func(t *testing.T) {
		t.Run("should skip if it doesn't match ingate class", func(t *testing.T) {
			newClass := &v1.GatewayClass{}
			newClass.SetName(fmt.Sprintf("class-%d", rand.IntN(10000)))
			newClass.Spec = v1.GatewayClassSpec{
				ControllerName: "k8s.io/not-ingate",
				Description:    ptr.To("Not Ingate"),
			}

			require.NoError(t, k8sClient.Create(ctx, newClass))

			t.Cleanup(func() {
				require.NoError(t, k8sClient.Delete(ctx, newClass))
			})
			// Check if the manager client already watched the change
			// TODO: once we establish the cache method, this test will fail and we need to
			// assert that the ignored class is actually not found at all
			require.Eventually(t, func() bool {
				cachedClass := &v1.GatewayClass{}
				cachedClass.SetName(newClass.GetName())
				// Try to get the object, ignore if it is not found
				require.NoError(t, client.IgnoreNotFound(managerClient.Get(ctx, client.ObjectKeyFromObject(cachedClass), cachedClass)))

				return cachedClass.Spec.Description != nil && *cachedClass.Spec.Description == "Not Ingate"
			}, 3*time.Second, time.Millisecond, "new gatewayclass wasn't watched by manager client")
		})

		t.Run("should add the accepted condition when the class matches", func(t *testing.T) {
			newClass := &v1.GatewayClass{}
			newClass.SetName(fmt.Sprintf("class-%d", rand.IntN(10000)))
			newClass.Spec = v1.GatewayClassSpec{
				ControllerName: "k8s.io/ingate",
				Description:    ptr.To("This is Ingate"),
			}

			require.NoError(t, k8sClient.Create(ctx, newClass))

			t.Cleanup(func() {
				require.NoError(t, k8sClient.Delete(ctx, newClass))
			})

			reconciledClass := &v1.GatewayClass{}
			reconciledClass.SetName(newClass.GetName())

			require.Eventually(t, func() bool {
				// Try to get the object, ignore if it is not found yet
				require.NoError(t, client.IgnoreNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(reconciledClass), reconciledClass)))

				acceptedCondition := framework.GetConditionPerType(string(v1.GatewayClassConditionStatusAccepted), reconciledClass.Status.Conditions)
				return acceptedCondition != nil &&
					acceptedCondition.Status == metav1.ConditionTrue &&
					acceptedCondition.Reason == "Accepted" &&
					acceptedCondition.ObservedGeneration == reconciledClass.Generation

			}, 3*time.Second, time.Millisecond, "new gatewayclass wasn't watched by manager client", "reconciledclass", reconciledClass)
		})

	})

}
