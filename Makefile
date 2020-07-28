# Copyright 2019 The Kubernetes Authors.
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

DOCKER ?= docker
# TOP is the current directory where this Makefile lives.
TOP := $(dir $(firstword $(MAKEFILE_LIST)))
# ROOT is the root of the mkdocs tree.
ROOT := $(abspath $(TOP))

all: controller generate verify

.PHONY: e2e-test
e2e-test: docker-build
	go test -v ./test -up ../demo/up.sh -down ../demo/down.sh

.PHONY: demo
demo: docker-build
	./demo/up.sh
	./demo/demo.sh
	./demo/down.sh

# Build manager binary and run static analysis.
.PHONY: controller
controller:
	$(MAKE) -f kubebuilder.mk manager

# Run generators for Deepcopy funcs and CRDs
.PHONY: generate
generate:
	./hack/update-codegen.sh
	$(MAKE) -f kubebuilder.mk generate
	$(MAKE) manifests

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests:
	$(MAKE) -f kubebuilder.mk manifests

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: generate docker-build
docker-build:
	$(MAKE) -f kubebuilder.mk docker-build

# Install CRD's and example resources to a pre-existing cluster.
.PHONY: install
install: manifests crd

# Install the CRD's to a pre-existing cluster.
.PHONY: crd
crd:
	$(MAKE) -f kubebuilder.mk install

# Remove installed CRD's and CR's.
.PHONY: uninstall
uninstall:
	./hack/delete-crds.sh

# Run static analysis.
.PHONY: verify
verify:
	./hack/verify-all.sh
