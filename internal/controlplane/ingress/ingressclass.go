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

package ingress

const (
	// IngressKey picks a specific "class" for the Ingress.
	// The controller only processes Ingresses with this annotation either
	// unset, or set to either the configured value or the empty string.
	IngressKey = "kubernetes.io/ingress.class"

	// DefaultControllerName defines the default controller name for Ingress NGINX
	DefaultControllerName = "k8s.io/ingate"

	// DefaultAnnotationValue defines the default annotation value for the InGate controller
	DefaultAnnotationValue = "ingate"
)

// Configuration defines the various aspects of IngressClass parsing
// and how the controller should behave in each case
type Configuration struct {
	// Controller defines the controller value this daemon watch to.
	// Defaults to "k8s.io/ingate" defined in flags
	Controller string
	// AnnotationValue defines the annotation value this Controller watch to, in case of the
	// ingressSpecName is not found but the annotation is.
	// The Annotation is deprecated and should not be used in future releases
	AnnotationValue string
	// WatchWithoutClass defines if Controller should watch to Ingress Objects that does
	// not contain an IngressClass configuration
	WatchWithoutClass bool
	// IgnoreIngressClass defines if Controller should ignore the IngressClass Object if no permissions are
	// granted on IngressClass
	IgnoreIngressClass bool
	// IngressClassByName defines if the Controller should watch for Ingress Classes by
	// .metadata.name together with .spec.Controller
	IngressClassByName bool
}
