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

package crd

import _ "embed" // import embed to be able to use go:embed

var (
	// ServiceExportCRD is the embedded YAML for the ServiceExport CRD
	//go:embed multicluster.x-k8s.io_serviceexports.yaml
	ServiceExportCRD []byte
	// ServiceImportCRD is the embedded YAML for the ServiceImport CRD
	//go:embed multicluster.x-k8s.io_serviceimports.yaml
	ServiceImportCRD []byte
)

const (
	// CustomResourceDefinitionSchemaVersionKey is key to label which holds the CRD schema version
	CustomResourceDefinitionSchemaVersionKey = "multicluster.x-k8s.io/crd-schema-version"
)
