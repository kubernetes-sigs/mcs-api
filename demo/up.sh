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

. ./udemo.sh

set -e
set -x

c1=c1
c2=c2

k1="kubectl --kubeconfig ${c1}.kubeconfig"
k2="kubectl --kubeconfig ${c2}.kubeconfig"

if [ -z "$(docker images mcs-api-controller -q)" ]; then
  pushd ../
  make -f kubebuilder.mk docker-build
  popd
fi

kind create cluster --name "${c1}" --config "yaml/${c1}.yaml"
kind create cluster --name "${c2}" --config "yaml/${c2}.yaml"

kind get kubeconfig --name "${c1}" > "${c1}".kubeconfig
kind get kubeconfig --name "${c2}" > "${c2}".kubeconfig

kind load docker-image mcs-api-controller --name "${c1}"
kind load docker-image mcs-api-controller --name "${c2}"

function pod_cidrs() {
  kubectl --kubeconfig ${1}.kubeconfig get nodes -o jsonpath='{range .items[*]}{.spec.podCIDR}{"\n"}'
}

function add_routes() {
  unset IFS
  routes=$(kubectl --kubeconfig ${2}.kubeconfig get nodes -o jsonpath='{range .items[*]}ip route add {.spec.podCIDR} via {.status.addresses[?(.type=="InternalIP")].address}{"\n"}')
  echo "Connecting cluster ${1} to ${2}"

  IFS=$'\n'
  for n in $(kind get nodes --name "${1}"); do
    for r in $routes; do
      eval "docker exec $n $r"
    done
  done
  unset IFS
}
waitfor pod_cidrs ${c1}
waitfor pod_cidrs ${c2}

echo "Connecting cluster networks..."
add_routes "${c1}" "${c2}"
add_routes "${c2}" "${c1}"
echo "Cluster networks connected"

${k1} apply -f ../config/crd -f ../config/rbac
${k2} apply -f ../config/crd -f ../config/rbac

${k1} create sa mcs-api-controller
${k1} create clusterrolebinding mcs-api-binding --clusterrole=mcs-derived-service-manager --serviceaccount=default:mcs-api-controller
${k1} run --image mcs-api-controller --serviceaccount=mcs-api-controller --image-pull-policy=Never mcs-api-controller

${k2} create sa mcs-api-controller
${k2} create clusterrolebinding mcs-api-binding --clusterrole=mcs-derived-service-manager --serviceaccount=default:mcs-api-controller
${k2} run --image mcs-api-controller --serviceaccount=mcs-api-controller --image-pull-policy=Never mcs-api-controller
