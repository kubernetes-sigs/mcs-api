#!/bin/bash

# Copyright 2020 The Kubernetes Authors.
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

cd $(dirname ${BASH_SOURCE})

set -e

export KUBECONFIG1=$(mktemp --suffix=".kubeconfig")
export KUBECONFIG2=$(mktemp --suffix=".kubeconfig")

function cleanup() {
    if [ -z "${NO_TEAR_DOWN}" ]; then
        ./down.sh
        rm ${KUBECONFIG1}
        rm ${KUBECONFIG2}
    else
        echo "KUBECONFIG1=${KUBECONFIG1}"
        echo "KUBECONFIG2=${KUBECONFIG2}"
    fi
}

trap cleanup EXIT

./up.sh
go run github.com/onsi/ginkgo/v2/ginkgo ../e2etest
