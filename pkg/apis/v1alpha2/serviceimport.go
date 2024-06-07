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

// ServiceImport describes a service imported from clusters in a ClusterSet.
type ServiceImport struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// spec defines the behavior of a ServiceImport.
	// +optional
	Spec ServiceImportSpec `json:"spec,omitempty"`
	// status contains information about the exported services that form
	// the multi-cluster service referenced by this ServiceImport.
	// +optional
	Status ServiceImportStatus `json:"status,omitempty"`
}

// ServiceImportType designates the type of a ServiceImport
type ServiceImportType string

const (
	// ClusterSetIP services are only accessible via the ClusterSet IP.
	ClusterSetIP ServiceImportType = "ClusterSetIP"
	// Headless services allow backend pods to be addressed directly.
	Headless ServiceImportType = "Headless"
)

// ServiceImportSpec describes an imported service and the information necessary to consume it.
type ServiceImportSpec struct{}

// ServiceImportStatus describes derived state of an imported service.
type ServiceImportStatus struct {
	// +listType=atomic
	Ports []v1.ServicePort `json:"ports"`
	// ips will be used as the VIP(s) for this service when type is ClusterSetIP.
	// +kubebuilder:validation:MaxItems:=1
	// +optional
	IPs []string `json:"ips,omitempty"`
	// type defines the type of this service.
	// Must be "ClusterSetIP" or "Headless".
	// The "ClusterSetIP" type reflects exported Service(s) with type ClusterIP
	// and the "Headless" type reflects exported Service(s) with type Headless.
	// A ServiceImport with type ClusterSetIP SHOULD populate `.status.ips`
	// with virtual IP address(es) where the service can be reached from within
	// the importing cluster.
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

// ClusterStatus contains service configuration mapped to a specific source cluster
type ClusterStatus struct {
	// cluster is the name of the exporting cluster. Must be a valid RFC-1123 DNS
	// label.
	Cluster string `json:"cluster"`
}

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
