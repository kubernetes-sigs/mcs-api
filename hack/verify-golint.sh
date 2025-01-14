#!/bin/bash

# Copyright 2014 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

KUBE_ROOT=$(dirname "${BASH_SOURCE}")/..

cd "${KUBE_ROOT}"

PACKAGES=($(go list ./... | sed sXsigs.k8s.io/mcs-apiX..X))
bad_files=()
for package in "${PACKAGES[@]}"; do
  out=$(go -C tools run golang.org/x/lint/golint -min_confidence=0.9 "${package}" 2>&1 |
        sed 'sX^../XX;/should not use dot imports/d;/exported const OptionalLabel/d;/^go: downloading/d' ||:)
  if [[ -n "${out}" ]]; then
    bad_files+=("${out}")
  fi
done
if [[ "${#bad_files[@]}" -ne 0 ]]; then
  echo "!!! golint problems: "
  for err in "${bad_files[@]}"; do
    echo "$err"
  done
  exit 1
fi

# ex: ts=2 sw=2 et filetype=sh
