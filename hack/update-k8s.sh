#!/usr/bin/env bash

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

# This script updates mcs-api dependencies to either the latest
# k8s.io dependencies, or to the version given as argument
# (with the @, e.g. @v0.32.5).

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")/..

for dir in "$SCRIPT_ROOT"{,/tools}; do
    awk '/[^.]k8s.io[/][^ ]+ v[.0-9]+$/ { print $1 "'"$1"'" }' "$dir/go.mod" |
	xargs -r -n 1 go -C "$dir" get
done

for mod in "$SCRIPT_ROOT"/go.mod "$SCRIPT_ROOT"/*/go.mod; do
    go -C "${mod%/*}" mod tidy
done

make generate
