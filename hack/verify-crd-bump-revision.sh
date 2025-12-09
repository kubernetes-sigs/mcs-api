#!/bin/bash

# Copyright 2025 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Check against master if PULL_BASE_SHA is not defined by prow
BASE_REF="${PULL_BASE_SHA:-master}"

crd_changed="$(git diff --name-only "${BASE_REF}" | grep -c "^config/crd/.*\.yaml$")"
version_label_changed="$(git diff -U0 "${BASE_REF}" -- "config/crd-base/" | grep -c "multicluster.x-k8s.io/crd-schema-revision")"

if [ "${crd_changed}" -gt 0 ] && [ "${version_label_changed}" -ne 4 ]; then
	echo "❌ CRDs were modified, but the CRD revision labels were not changed in 'config/crd-base/'. Please bump the CRDs revision."
	exit 1
fi
