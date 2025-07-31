package tunables

import (
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type tunables struct {
	logger       logr.Logger
	gatewayClass string
}

func NewTunables(logger logr.Logger, gwclass string) *tunables {
	return &tunables{
		logger:       logger,
		gatewayClass: gwclass,
	}
}

// TransformGatewayClass is a cache function that will:
// * Drop a Gateway Class from cache in case it doesn't belong to a class managed
// by InGate
// * Drop ManagedFields to save some memory
func (t *tunables) TransformGatewayClass() cache.TransformFunc {
	return func(i any) (any, error) {
		logger := t.logger.WithName("cache-transform")
		gwclass, ok := i.(*gatewayv1.GatewayClass)
		if !ok {
			logger.Info("ignoring object as it is not a gateway class")
			return nil, nil
		}
		// Drop the object from cache if we don't care about it
		if gwclass.Spec.ControllerName != gatewayv1.GatewayController(t.gatewayClass) {
			logger.Info("ignoring object with unknown class", "name", gwclass.GetName(), "class", gwclass.Spec.ControllerName)
			return nil, nil
		}
		// Clean managed fields for some memory economy
		gwclass.SetManagedFields(nil)
		return gwclass, nil
	}
}
