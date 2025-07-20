# InGate as a generic Gateway API Controller

InGate was concieved originally to be a simple Gateway API controller,
allowing users to have a simple and yet comformant implementation of the API
and also to still make use of existing Ingress APIs.

During the implementation of InGate, and also checking other implementations,
it was realized that there's no easy way to reproduce existing code into
our own implementation.

But looking at the existing L7 routing implementations (eg.: kgateway, envoygateway,
and others) we can conclude that what differ one implementation from the other
is how the backend is implemented.

For every Gateway API implementation, we see a standard: a comformant way to
reconcile GatewayClass, then to reconcile Gateway resources, and the *Route
objects depending on what kind of backend is supported.

When evolving the implementation with new experimental features, like ListenerSet
there is a need of every implementation to do their own code changes, just to
be sure that whatever is desired by users on the API is reflected as a backend
configuration.

This proposal intends to define a well known controller, that can be used by
backend implementations to do their own configuration, while avoiding to
re-implement a Gateway API controller every time a new backend wants to support
it or a new feature is added to Gateway API resources.

It is worth noticing that this approach is not new on Kubernetes world, where
a CSI, for instance defines how a volume should be created but it is up to each
backend to program it, or Cluster API, that defines how a Kubernetes cluster should
be created, and then each Infrastructure Provider can define their own way of
provisioning and managing the infrastructure for that Kubernetes cluster.

# Exploring how backends relate with Gateway API

Before jumping into the proposal, it is important to understand what is the current
situation. Let's take 3 major backend implementations, and understand how a
Gateway API resource creation reflects into their configurations.

The 3 major backend selected for this proposal are: Envoy, NGINX and HAProxy.

## Gateway Class reconciliation
Gateway Class reconciliation is the simpler one. Per Gateway API definition, Ian,
the Infrastructure Provider is responsible for defining a type of Gateway, and
a common set of shared configuration between the Gateways that belong to this class.

From a backend perspective, this means that the controller should program just
backends that comply with these features. It should state something like "I just
want to program backends that support this cloud provider".

There is no difference between how each backend should be programmed, but instead
what Gateway and *Route resources should be reconciled, based on the desired class.

In fact, from the controller implementation perspective it means that a controller
should reconcile Gateway resources that matches the desired `controllerName` and
optionally use the `parametersRef` when doing this reconciliation (eg.: which
IP Pool to use when creating new Gateways, which class of LoadBalancer should
be used from the infrastructure provider when creating the service for a desired
Gateway, etc).

It is also always expected that the controller sets the `conditions` of the
GatewayClass to at least signal if it was accepted or not.

## Gateway reconciliation
Gateway resources represent a real exposed service. They will be used during a
`Route` reconciliation to define what is the desired entrypoint.

Let's take the following statement as a use case: "Chihiro, the `Cluster Operator`
wants to allow users/developers to expose their application through Gateway API".

But given the nature of Chihiro's cluster, some applications may be staging and
don't need a very expensive production level Load Balancer to expose the backend,
and the certificate used to expose the application can be a "non-valid" one, for
cost optimization. But at the same time, production applications should use a
different listener, with a different certificate.

From a backend perspective, this means that when creating a new Gateway that
will have routes attached to it, each gateway should:
* Create a new Service on Kubernetes with the parameters set from GatewayClass
* Create on the backend N new listeners for this Gateway, using the desired certificate
  * Each listener should have their own hostname or IP definition, and desired ports
  * Additionally, it may configure a set of trusted certificates to be used when
  communicating with backends
* May allow just a specific set of routes to be attached to this Gateway.

Let's take the following `Gateway` definition and how it would reflect on the backends
configuration:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: example-gateway
spec:
  gatewayClassName: example
  listeners:
  - name: prod-web-gw
    protocol: HTTP
    port: 80
    name: prod-web-gw
  - name: foo-https
    protocol: HTTPS
    port: 443
    hostname: foo.example.com
    tls:
      certificateRefs:
      - kind: Secret
        group: ""
        name: foo-cert
  - name: wildcard-https
    protocol: HTTPS
    port: 443
    hostname: "*.example.com"
    tls:
      certificateRefs:
      - kind: Secret
        group: ""
        name: wildcard-cert
```

This resource definition, tells us:
* Any request to this Gateway on port 80 should be accepted
* Any request to this gateway on port 443 to the hostname `foo.example.com` should
present the certificate contained on secret `foo-cert`
* Any request to port 443 to any hostname that contains the suffix `example.com` should
present the certificate contained on secret `wildcard-cert`, except if this
is `foo.example.com` that was defined before.

How would this look like on each backend configuration?

### NGINX
On NGINX the configuration above would be as the following snippet:
```
server {
    listen 80 default_server;
    server_name _;
    ... # Locations / Routes come here
}

server {
    listen 443 ssl;
    server_name foo.example.com;

    ssl_certificate /path/to/foo-cert.crt;
    ssl_certificate_key /path/to/foo-cert.key;

    ... # Locations / Routes come here
}

server {
    listen 443 ssl;
    server_name *.example.com;

    ssl_certificate /path/to/wildcard-cert.crt;
    ssl_certificate_key /path/to/wildcard-cert.key;


    ... # Locations / Routes come here
}
```

As we can see, a `Gateway` definition turns into a `server` definition on NGINX.

### HAProxy
For HAProxy the configuration may differ a bit, but as we can see each Gateway
can become its own definition:

```
frontend http-in
    bind *:80
frontend https-foo
    bind *:443 ssl crt /etc/haproxy/certs/foo-cert.pem crt-ignore-err all
    mode http
    acl is_something req_ssl_sni -i foo.example.com
    use_backend something_backend if is_something

frontend https-wildcard
    bind *:443 ssl crt /etc/haproxy/certs/wildcard-cert.pem crt-ignore-err all
    mode http
    acl is_wildcard req_ssl_sni -m end .ricardo.com
    use_backend wildcard_backend if is_wildcard
```

### Envoy
For envoy, the configuration is slightly more complicated but as we can see, the
listeners will define an array of filter chains that will expose the proper transport
configuration.

For the sake of exposition, we are using `static_resources` here but Envoy could
be programmed with the xDS endpoints, etc.

```yaml
static_resources:
  listeners:
    - name: listener_80
      address:
        socket_address: { address: 0.0.0.0, port_value: 80 }
      # Extra configs and routes here
    - name: listener_443
      address:
        socket_address: { address: 0.0.0.0, port_value: 443 }
      filter_chains:
        - filter_chain_match:
            server_names: ["foo.example.com"]
          transport_socket:
            name: envoy.transport_sockets.tls
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext
              common_tls_context:
                tls_certificates:
                  - certificate_chain:
                      filename: "/etc/envoy/foo-cert.crt"
                    private_key:
                      filename: "/etc/envoy/foo-cert.key"
          filters:
          # Your routes here
        - filter_chain_match:
            server_names: ["*.example.com"]
          transport_socket:
            name: envoy.transport_sockets.tls
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext
              common_tls_context:
                tls_certificates:
                  - certificate_chain:
                      filename: "/etc/envoy/wildcard-cert.crt"
                    private_key:
                      filename: "/etc/envoy/wildcard-cert.key"
          filters:
          # Your routes here
```

### Conclusion on Gateway
As we can see, for what matters on a Gateway reconciliation vs Backend programming,
the reconciliation should at least do the following operations:
* Persist the referred certificates on the backends
* Create the proper listener configuration that present the right certificates,
when requested, and do the proper routing based on the requested hostname.
* Later, during the `Route` reconciliation, each route should be attached to the
proper listener.

Additional configuration may apply to each backend/proxy, and the examples presented
on this section intend to show that they have common definitions for a Gateway reconciliation.

Also, we should have in mind that the configurations may vary for other different
types of listeners, like TCP or TLS Passthrough.

This means that from a reconciliation perspective, we need at least the following operations:
* Add/Update/Delete a listener
* Add/Update/Delete a certificate

## *Route reconciliation
The route reconciliation consist on programming the backend to send the traffic
to the right workload (on our case, Endpoints and EndpointSlices) based on criterias
like the requested Path, the requested Method, headers, etc.

It can also be used to re-program the request (eg.: add additional headers, rewrite
the request, etc).

Let's take a look into a simple example definition, and use the previous examples
from Gateway reconciliation.

The example exposed here will:
* Terminate the TLS on Gateway
* If the request is sent to `foo.example.com` and the path prefix contains `/login`
it will send to the endpoint IPs of `foo-svc:8080`. The exposed certificate will be
the one added to `foo.example.com` listener
* If the request is sent to `bar.example.com`, the exposed certificate will be the
one from `*.example.com`, and the following routing will be made:
  * In case the request contains a header `env:canary`, it will be send to the endpoints
  of `bar-svc-canary:8080`
  * Otherwise, it will be sent to `bar-svc:8080`.

For the sake of this example, here are the endpoints definitions:
* foo-svc: 10.0.0.10 and 10.0.0.20
* bar-svc-canary: 10.10.10.10 and 10.10.10.20
* bar-svc: 10.20.20.10 and 10.20.20.20 

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: foo-route
spec:
  parentRefs:
  - name: example-gateway
  hostnames:
  - "foo.example.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /login
    backendRefs:
    - name: foo-svc
      port: 8080
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: bar-route
spec:
  parentRefs:
  - name: example-gateway
  hostnames:
  - "bar.example.com"
  rules:
  - matches:
    - headers:
      - type: Exact
        name: env
        value: canary
    backendRefs:
    - name: bar-svc-canary
      port: 8080
  - backendRefs:
    - name: bar-svc
      port: 8080
```

### NGINX
On NGINX, the following configuration will be presented:

```
http {
    upstream foo-svc {
        server 10.0.0.10:8080;
        server 10.0.0.20:8080;
    }

    upstream bar-svc-canary {
        server 10.10.10.10:8080;
        server 10.10.10.20:8080;
    }

    upstream bar-svc {
        server 10.20.20.10:8080;
        server 10.20.20.20:8080;
    }

    server {
        listen 443 ssl;
        server_name foo.example.com;
        ssl_certificate /etc/nginx/foo.crt;
        ssl_certificate_key /etc/nginx/foo.key;

        location /login {
            proxy_pass http://foo-svc;
        }
    }

    server {
        listen 443 ssl;
        server_name *.example.com;
        ssl_certificate /etc/nginx/wildcard.crt;
        ssl_certificate_key /etc/nginx/wildcard.key;

        location / {
            if ($http_env = "canary") {
                proxy_pass http://bar-svc-canary;
                break;
            }
            proxy_pass http://bar-svc;
        }
    }
}
```

As we can see:
* The endpoints definition is part of the `upstream` block
* Every route definition is made inside the listener definition

### HAProxy

For HAProxy, the following summarized configuration can be used:
```
frontend https-in
    bind *:443 ssl crt /etc/haproxy/certs/
    acl host_foo hdr(host) -i foo.example.com
    acl path_login path_beg /login
    acl host_wildcard hdr_reg(host) -i ^[a-z0-9.-]+\.example\.com$
    acl header_canary req.hdr(env) -i canary

    use_backend foo_svc if host_foo path_login
    use_backend bar_svc_canary if host_wildcard header_canary
    use_backend bar_svc if host_wildcard

backend foo_svc
    balance roundrobin
    server foo1 10.0.0.10:8080 check
    server foo2 10.0.0.20:8080 check

backend bar_svc_canary
    balance roundrobin
    server canary1 10.10.10.10:8080 check
    server canary2 10.10.10.20:8080 check

backend bar_svc
    balance roundrobin
    server bar1 10.20.20.10:8080 check
    server bar2 10.20.20.20:8080 check
```
* Each backend definition can be made on `backend` blocks
* The routing is made using ACLs inside the frontend block

Worth noticing the HAProxy configuration can be optimized, but the idea here is
to exemplify that the `Route` definitions with the `backendRefs` will define a
similar block of configurations: The backends, and the attached routes

### Envoy
For envoy, something similar happens:

```yaml
static_resources:
  listeners:
    - name: listener_443
      address:
        socket_address: { address: 0.0.0.0, port_value: 443 }
      filter_chains:
        - transport_socket:
            name: envoy.transport_sockets.tls
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext
              common_tls_context:
                tls_certificates:
                  - certificate_chain: { filename: "/etc/envoy/certs/foo.crt" }
                    private_key: { filename: "/etc/envoy/certs/foo.key" }
                  - certificate_chain: { filename: "/etc/envoy/certs/wildcard.crt" }
                    private_key: { filename: "/etc/envoy/certs/wildcard.key" }
          filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                stat_prefix: ingress
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: foo
                      domains: ["foo.example.com"]
                      routes:
                        - match: { prefix: "/login" }
                          route: { cluster: foo-svc }

                    - name: wildcard
                      domains: ["*.example.com"]
                      routes:
                        - match:
                            prefix: "/"
                            headers:
                              - name: env
                                exact_match: "canary"
                          route: { cluster: bar-svc-canary }

                        - match: { prefix: "/" }
                          route: { cluster: bar-svc }
                http_filters:
                  - name: envoy.filters.http.router

  clusters:
    - name: foo-svc
      type: strict_dns
      load_assignment:
        cluster_name: foo-svc
        endpoints:
          - lb_endpoints:
              - endpoint: { address: { socket_address: { address: 10.0.0.10, port_value: 8080 } } }
              - endpoint: { address: { socket_address: { address: 10.0.0.20, port_value: 8080 } } }

    - name: bar-svc-canary
      type: strict_dns
      load_assignment:
        cluster_name: bar-svc-canary
        endpoints:
          - lb_endpoints:
              - endpoint: { address: { socket_address: { address: 10.10.10.10, port_value: 8080 } } }
              - endpoint: { address: { socket_address: { address: 10.10.10.20, port_value: 8080 } } }

    - name: bar-svc
      type: strict_dns
      load_assignment:
        cluster_name: bar-svc
        endpoints:
          - lb_endpoints:
              - endpoint: { address: { socket_address: { address: 10.20.20.10, port_value: 8080 } } }
              - endpoint: { address: { socket_address: { address: 10.20.20.20, port_value: 8080 } } }
```

As we can see, each backend definition to be used will generate a new cluster inside
the `clusters` array.

And every new route will be attached to a filter of the relative listener, that
was previously defined on the `Gateway` definitions

### Conclusions on Route reconciliation
For route reconciliation, we can see that the same pattern happens for the 3 major
backends:
* For the workload IP / endpoint definitions, it will generate a specific block that
contains just "who should be contacted" (and how, timeouts, etc)
* For the routes, they will be attached to the listener based on the desired rules,
and direct the traffic to the proper backends.

This means that for a Route reconciliation, we need the following operations:
* Add/Update/Delete a group of backend/upstreams (based on Endpoints/EndpointSlices)
* Update a Listener to attach/update/detach a route, and point to a backend.

# Consuming InGate as a library

To make it possible for backends to consume InGate as a library, a proposal would be
to make InGate request, during its instantiation, structures that comply with some interfaces.

Given the following interfaces definitions (not final, just examples)

```go
type ListenerController interface {
  AddListener(ctx context.Context, name string, cfg *ListenerConfiguration) error
  RemoveListener(ctx context.Context, name string)
  UpdateListener(ctx context.Context, name string, cfg *ListenerConfiguration) error
}

type CertificateController interface {
  AddCertificate(ctx context.Context, name string, certificate *CertificateConfiguration) error
  RemoveCertificate(ctx context.Context, name string)
}

type RouteController interface {
  AddRoute(ctx context.Context, name string, listenerName string, cfg *RouteConfiguration) error
  RemoveRoute(ctx context.Context, name string, listenerName string) error
  UpdateRoute(ctx context.Context, name string, listenerName string, cfg *RouteConfiguration) error
}

type BackendController interface {
  AddBackend(ctx context.Context, name string, endpoints []*Endpoint) error
  UpdateBackend(ctx context.Context, name string, endpoints []*Endpoint) error
  RemoveBackend(ctx context.Context, name string) error
}
```

we could have a definition of an InGate controller as the following:

```go
type IngateConfiguration struct {
  listenerController ListenerController
  certificateController CertificateController
  routeController RouteController
  backendController BackendController
}

type GatewayReconciler struct {
 // ....
 listenerController ListenerController
 certificateController CertificateController
}

func NewGatewayReconciler(ctx context.Context, mgr ctrl.Manager, cfg IngateConfiguration) *GatewayReconciler {
	return &GatewayReconciler{
    // ....
		listenerController: cfg.ListenerController,
    certificateController: cfg.CertificateController,
	}
}
```

And inside the Gateway Reconciler, once we detect there's a need to update a
certificate, the implementation should look like:

```go
func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
  // .... detected a new certificate should be added
  err := r.certificateController.AddCertificate(ctx, "something", certificateConfig) 
}
```

This is a very raw example, but the idea is that InGate decides when to call each
backend implementation method, but leaves for the backend implementation the real
configuration to be made
