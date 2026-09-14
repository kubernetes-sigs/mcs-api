# Multi-Cluster Service API tutorials

These tutorials describe the API objects and the decisions an implementation
needs to make. They do not assume a particular multi-cluster service
controller, networking plugin, cloud, or DNS provider.

Before starting, install the MCS API CRDs and a compatible implementation in
each cluster that will participate in the ClusterSet. Follow the
implementation's instructions for connecting clusters and for exposing
ServiceImport objects through multi-cluster DNS or another discovery
mechanism.

## How the API fits together

- A Service is the ordinary, namespaced Kubernetes service that selects
  application endpoints.
- A ServiceExport declares that the same-name service in that namespace may be
  consumed from other clusters.
- A ServiceImport represents the combined service in a consuming cluster.
  Implementations normally create and update it; consumers should not create
  it by hand.
- An implementation decides how endpoints are discovered, how traffic is
  routed, and how health or locality affects the result. The MCS API defines
  the objects and their intent, not a particular failover or load-balancing
  algorithm.

The examples use v1beta1, which is the storage version in the CRDs shipped
by this repository. Keep the Service and ServiceExport names and
namespaces aligned.

## 1. Export one stateful service to other clusters

Use this pattern when the authoritative state lives in one cluster but clients
in other clusters need a stable service name. Keep the database or other
stateful component in the exporting cluster and make its replication and
write policy explicit; MCS does not replicate application data.

In the cluster hosting the service, create the ordinary Service and then
export it:

~~~yaml
apiVersion: v1
kind: Service
metadata:
  name: orders
  namespace: payments
spec:
  selector:
    app: orders
  ports:
    - name: grpc
      port: 8080
      targetPort: grpc
---
apiVersion: multicluster.x-k8s.io/v1beta1
kind: ServiceExport
metadata:
  name: orders
  namespace: payments
~~~

Apply the objects in the exporting cluster. In a consuming cluster, wait for
the implementation to publish a ServiceImport named orders in the
payments namespace:

~~~sh
kubectl get serviceimport orders -n payments
kubectl describe serviceimport orders -n payments
~~~

Use the implementation's documented discovery name to call the service. If it
supports the MCS DNS convention, the name is commonly
orders.payments.svc.clusterset.local.

## 2. Combine the same stateless service across regions

Use this pattern when each region can serve the same request independently and
the implementation should present one logical service. Deploy the same
namespace, service name, selector, and port contract in each cluster. Export
the service in every participating cluster:

~~~yaml
apiVersion: multicluster.x-k8s.io/v1beta1
kind: ServiceExport
metadata:
  name: catalog
  namespace: storefront
~~~

The Service named catalog must already exist in each cluster, with the
local deployment selecting the same labels and exposing the same port. The
implementation combines the exported endpoints into the ServiceImport for
storefront/catalog.

Configure region preference and health checking through the implementation or
traffic layer. A ServiceExport by itself does not promise active-passive
failover, equal distribution, or cross-region latency awareness. Test both
normal routing and the failure behavior before directing production traffic to
the combined service.

## 3. Use a combined service during a blue-green upgrade

Use this pattern to make two independently managed clusters serve the same
logical service while a release is promoted. The blue and green clusters each
run a compatible Service named frontend in the web namespace and each
create the same export:

~~~yaml
apiVersion: v1
kind: Service
metadata:
  name: frontend
  namespace: web
spec:
  selector:
    app: frontend
  ports:
    - name: http
      port: 80
      targetPort: http
---
apiVersion: multicluster.x-k8s.io/v1beta1
kind: ServiceExport
metadata:
  name: frontend
  namespace: web
~~~

Run the rollout in this order:

1. Keep the blue export serving and deploy the green version behind the same
   service contract.
2. Create the green ServiceExport and confirm the combined ServiceImport
   reports both exporting clusters.
3. Shift traffic using the implementation's supported locality, weight, or
   gateway controls, then exercise the service from representative clients.
4. After the green version is healthy, remove the blue export or drain it
   using the implementation's documented procedure.
5. Keep the service name and namespace stable for clients throughout the
   transition.

Because routing controls are implementation-specific, record the exact
promotion and rollback commands for the implementation you operate. If the
green deployment fails, restore the previous routing choice before removing
the blue export.

## Troubleshooting checklist

- Confirm the Service and ServiceExport have the same name and namespace.
- Check the ServiceExport status and the implementation controller logs in
  every participating cluster.
- Confirm the consuming cluster has a ServiceImport and that its status lists
  the expected exporting clusters.
- Compare the exported service port contract with the local workloads.
- Test the implementation's discovery and health behavior separately from
  application-level readiness.
