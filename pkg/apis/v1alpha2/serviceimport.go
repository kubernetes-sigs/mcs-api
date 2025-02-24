/*
Copyright 2020 The Kubernetes Authors.

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

package v1alpha2

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName={svcimport,svcim}

// ServiceImport describes a service imported from clusters in a ClusterSet and
// the information necessary to consume it. ServiceImport is managed by an MCS
// controller and should be updated automatically to show derived state as IP
// addresses or ServiceExport resources change.
type ServiceImport struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +listType=atomic
	Ports []ServicePort `json:"ports"`
	// ips will be used as the VIP(s) for this service when type is ClusterSetIP.
	// +kubebuilder:validation:MaxItems:=2
	// +optional
	IPs []string `json:"ips,omitempty"`
	// type defines the type of this service.
	// Must be "ClusterSetIP" or "Headless".
	// The "ClusterSetIP" type reflects exported Service(s) with type ClusterIP
	// and the "Headless" type reflects exported Service(s) with type Headless.
	// A ServiceImport with type ClusterSetIP SHOULD populate `.ips`
	// with virtual IP address(es) where the service can be reached from within
	// the importing cluster. These addresses MAY be endpoints directly reachable
	// over a "flat" network between clusters, or MAY direct traffic through
	// topology such as an intermediate east-west gateway.
	// If exported Services of the same name and namespace in a given ClusterSet
	// have differing types, a "Conflict" status condition SHOULD be reported in
	// ServiceExport status.
	// +kubebuilder:validation:Enum=ClusterSetIP;Headless
	Type ServiceImportType `json:"type"`
	// sessionAffinity is used to maintain client IP based session affinity.
	// Supports "ClientIP" and "None". Defaults to "None".
	// Reflects the `.spec.sessionAffinity` configuration of the underlying
	// exported Service. Ignored when exported Service type is Headless. If
	// exported Services of the same name and namespace in a given ClusterSet have
	// differing session affinity configuration, a "Conflict" status condition
	// SHOULD be reported in ServiceExport status.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies
	// +optional
	SessionAffinity v1.ServiceAffinity `json:"sessionAffinity,omitempty"`
	// sessionAffinityConfig contains session affinity configuration.
	// Reflects the `.spec.sessionAffinityConfig` configuration of the underlying
	// exported Service. Ignored when exported Service type is "Headless". If
	// exported Services of the same name and namespace in a given ClusterSet have
	// differing session affinity configuration, a "Conflict" status condition
	// SHOULD be reported in ServiceExport status.
	// +optional
	SessionAffinityConfig *v1.SessionAffinityConfig `json:"sessionAffinityConfig,omitempty"`
	// clusters is the list of exporting clusters from which this service
	// was derived.
	// +optional
	// +patchStrategy=merge
	// +patchMergeKey=cluster
	// +listType=map
	// +listMapKey=cluster
	Clusters []ClusterStatus `json:"clusters,omitempty"`
}

// ServicePort contains information on service's port.
type ServicePort struct {
	// The name of this port within the service. This must be a DNS_LABEL.
	// All ports within a ServiceSpec must have unique names. When considering
	// the endpoints for a Service, this must match the 'name' field in the
	// EndpointPort.
	// Optional if only one ServicePort is defined on this service.
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// The IP protocol for this port. Supports "TCP", "UDP", and "SCTP".
	// Default is TCP.
	// +default="TCP"
	// +optional
	Protocol v1.Protocol `json:"protocol,omitempty" protobuf:"bytes,2,opt,name=protocol,casttype=Protocol"`

	// The application protocol for this port.
	// This is used as a hint for implementations to offer richer behavior for protocols that they understand.
	// This field follows standard Kubernetes label syntax.
	// Valid values are either:
	//
	// * Un-prefixed protocol names - reserved for IANA standard service names (as per
	// RFC-6335 and https://www.iana.org/assignments/service-names).
	//
	// * Kubernetes-defined prefixed names:
	//   * 'kubernetes.io/h2c' - HTTP/2 prior knowledge over cleartext as described in https://www.rfc-editor.org/rfc/rfc9113.html#name-starting-http-2-with-prior-
	//   * 'kubernetes.io/ws'  - WebSocket over cleartext as described in https://www.rfc-editor.org/rfc/rfc6455
	//   * 'kubernetes.io/wss' - WebSocket over TLS as described in https://www.rfc-editor.org/rfc/rfc6455
	//
	// * Other protocols should use implementation-defined prefixed names such as
	// mycompany.com/my-custom-protocol.
	// +optional
	AppProtocol *string `json:"appProtocol,omitempty" protobuf:"bytes,6,opt,name=appProtocol"`

	// The port that will be exposed by this service.
	Port int32 `json:"port" protobuf:"varint,3,opt,name=port"`
}

// ServiceImportType designates the type of a ServiceImport
type ServiceImportType string

const (
	// ClusterSetIP services are only accessible via the ClusterSet IP.
	ClusterSetIP ServiceImportType = "ClusterSetIP"
	// Headless services allow backend pods to be addressed directly.
	Headless ServiceImportType = "Headless"
)

// ClusterStatus contains service configuration mapped to a specific source cluster
type ClusterStatus struct {
	// cluster is the name of the exporting cluster.
	Cluster ClusterName `json:"cluster"`
}

// ClusterName is the name of a cluster from which a service has been exported.
// Must be a valid RFC-1123 DNS label, which must consist of lower case
// alphanumeric characters, '-' or '.', and must start and end with an
// alphanumeric character.
//
// This validation is based off of the corresponding Kubernetes validation:
// https://github.com/kubernetes/apimachinery/blob/02cfb53916346d085a6c6c7c66f882e3c6b0eca6/pkg/util/validation/validation.go#L208
//
// Valid values include:
//
// * "example"
// * "example-01"
// * "01-example.com"
// * "foo.example.com"
//
// Invalid values include:
//
// * "example.com/bar" - "/" is an invalid character
//
// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=253
type ClusterName string

// +kubebuilder:object:root=true

// ServiceImportList represents a list of endpoint slices
type ServiceImportList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of endpoint slices
	// +listType=set
	Items []ServiceImport `json:"items"`
}
