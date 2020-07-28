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

package controllers

import (
	"crypto/sha256"
	"encoding/base32"
	"strings"

	"k8s.io/apimachinery/pkg/types"
)

const (
	DerivedServiceAnnotation = "multicluster.kubernetes.io/derived-service"
	serviceImportKind        = "ServiceImport"
)

var onlyPortName = "onlyport"

func derivedName(name types.NamespacedName) string {
	hash := sha256.New()
	return "import-" + strings.ToLower(base32.HexEncoding.EncodeToString(hash.Sum([]byte(name.String())))[:10])
}
