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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ServiceExportPluralName is the plural name of ServiceExport
	ServiceExportPluralName = "serviceexports"
	// ServiceExportKindName is the kind name of ServiceExport
	ServiceExportKindName = "ServiceExport"
	// ServiceExportFullName is the full name of ServiceExport
	ServiceExportFullName = ServiceExportPluralName + "." + GroupName
)

// ServiceExportVersionedName is the versioned name of ServiceExport
var ServiceExportVersionedName = ServiceExportKindName + "/" + GroupVersion.Version

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName={svcex,svcexport}

// ServiceExport declares that the Service with the same name and namespace
// as this export should be consumable from other clusters.
type ServiceExport struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// spec defines the behavior of a ServiceExport.
	// +optional
	Spec ServiceExportSpec `json:"spec,omitempty"`
	// status describes the current state of an exported service.
	// Service configuration comes from the Service that had the same
	// name and namespace as this ServiceExport.
	// Populated by the multi-cluster service implementation's controller.
	// +optional
	Status ServiceExportStatus `json:"status,omitempty"`
}

// ServiceExportSpec describes an exported service extra information
type ServiceExportSpec struct {
	// exportedLabels describes the labels exported. It is optional for implementation.
	// +optional
	ExportedLabels map[string]string `json:"exportedLabels,omitempty"`
	// exportedAnnotations describes the annotations exported. It is optional for implementation.
	// +optional
	ExportedAnnotations map[string]string `json:"exportedAnnotations,omitempty"`
}

// ServiceExportStatus contains the current status of an export.
type ServiceExportStatus struct {
	// +optional
	// +patchStrategy=merge
	// +patchMergeKey=type
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

const (
	// ServiceExportValid means that the service referenced by this
	// service export has been recognized as valid by an mcs-controller.
	// This will be false if the service is found to be unexportable
	// (ExternalName, not found).
	ServiceExportValid = "Valid"
	// ServiceExportConflict means that there is a conflict between two
	// exports for the same Service. When "True", the condition message
	// should contain enough information to diagnose the conflict:
	// field(s) under contention, which cluster won, and why.
	// Users should not expect detailed per-cluster information in the
	// conflict message.
	ServiceExportConflict = "Conflict"
)

// +kubebuilder:object:root=true

// ServiceExportList represents a list of endpoint slices
type ServiceExportList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of endpoint slices
	// +listType=set
	Items []ServiceExport `json:"items"`
}
