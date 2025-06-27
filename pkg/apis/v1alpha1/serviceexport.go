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
	//
	// Deprecated: use ServiceExportConditionAccepted instead
	ServiceExportValid = "Valid"
	// ServiceExportConflict means that there is a conflict between two
	// exports for the same Service. When "True", the condition message
	// should contain enough information to diagnose the conflict:
	// field(s) under contention, which cluster won, and why.
	// Users should not expect detailed per-cluster information in the
	// conflict message.
	//
	// Deprecated: use ServiceExportConditionConflicted instead
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

const (
	// ServiceExportConditionAccepted is true when the Service Export is accepted.
	// This does not indicate whether or not the configuration has been exported
	// to a control plane / data plane.
	//
	//
	// Possible reasons for this condition to be true are:
	//
	// * "Accepted"
	//
	// Possible reasons for this condition to be False are:
	//
	// * "NoService"
	// * "InvalidServiceType"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	ServiceExportConditionAccepted = "Accepted"

	// ServiceExportReasonAccepted is used with the "Accepted" condition when the
	// condition is True.
	ServiceExportReasonAccepted = "Accepted"

	// ServiceExportReasonNoService is used with the "Accepted" condition when
	// the associated Service does not exist.
	ServiceExportReasonNoService = "NoService"

	// ServiceExportReasonInvalidServiceType is used with the "Accepted"
	// condition when the associated Service has an invalid type
	// (per the KEP at least the ExternalName type).
	ServiceExportReasonInvalidServiceType = "InvalidServiceType"
)

const (
	// ServiceExportConditionExported is true when the service is exported to some
	// control plane or data plane.
	//
	//
	// Possible reasons for this condition to be true are:
	//
	// * "Exported"
	//
	// Possible reasons for this condition to be False are:
	//
	// * "Pending"
	// * "Failed"
	//
	// Possible reasons for this condition to be Unknown are:
	//
	// * "Pending"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	ServiceExportConditionExported = "Exported"

	// ServiceExportReasonExported is used with the "Exported" condition
	// when the condition is True.
	ServiceExportReasonExported = "Exported"

	// ServiceExportReasonPending is used with the "Exported" condition
	// when the service is going to be exported.
	ServiceExportReasonPending = "Pending"

	// ServiceExportReasonFailed is used with the "Exported" condition
	// when the service failed to be exported.
	ServiceExportReasonFailed = "Failed"
)

const (
	// ServiceExportConditionConflicted indicates that the controller was unable
	// to resolve conflict for a ServiceExport. This condition must be at
	// least raised on the conflicting ServiceExport and is recommended to
	// be raised on all on all the constituent `ServiceExport`s if feasible.
	//
	//
	// Possible reasons for this condition to be true are:
	//
	// * "PortConflict"
	// * "TypeConflict"
	// * "SessionAffinityConflict"
	// * "SessionAffinityConfigConflict"
	// * "AnnotationsConflict"
	// * "LabelsConflict"
	//
	// When multiple conflicts occurs the above reasons may be combined
	// using commas.
	//
	// Possible reasons for this condition to be False are:
	//
	// * "NoConflicts"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	ServiceExportConditionConflicted = "Conflicted"

	// ServiceExportReasonPortConflict is used with the "Conflicted" condition
	// when the exported service have a conflict related to port configuration.
	// This includes when ports on resulting imported services would have
	// duplicated names (including unnamed/empty name) or duplicated
	// port/protocol pairs.
	ServiceExportReasonPortConflict = "PortConflict"

	// ServiceExportReasonTypeConflict is used with the "Conflicted" condition
	// when the exported service have a conflict related to type.
	ServiceExportReasonTypeConflict = "TypeConflict"

	// ServiceExportReasonSessionAffinityConflict is used with the "Conflicted"
	// condition when the exported service have a conflict related to session affinity.
	ServiceExportReasonSessionAffinityConflict = "SessionAffinityConflict"

	// ServiceExportReasonSessionAffinityConfigConflict is used with the
	// "Conflicted" condition when the exported service have a conflict related
	// to session affinity config.
	ServiceExportReasonSessionAffinityConfigConflict = "SessionAffinityConfigConflict"

	// ServiceExportReasonLabelsConflict is used with the "Conflicted"
	// condition when the exported service have a conflict related to labels.
	ServiceExportReasonLabelsConflict = "LabelsConflict"

	// ServiceExportReasonAnnotationsConflict is used with the "Conflicted"
	// condition when the exported service have a conflict related to annotations.
	ServiceExportReasonAnnotationsConflict = "AnnotationsConflict"

	// ServiceExportReasonNoConflicts is used with the "Conflicted" condition
	// when the condition is False.
	ServiceExportReasonNoConflicts = "NoConflicts"
)
