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

. ./util.sh

set -e

kind() {
  go -C ../controllers run sigs.k8s.io/kind "$@"
}

c1=${CLUSTER1:-c1}
c2=${CLUSTER2:-c2}
kubeconfig1=${KUBECONFIG1:-"$PWD/${c1}.kubeconfig"}
kubeconfig2=${KUBECONFIG2:-"$PWD/${c2}.kubeconfig"}
controller_image=${MCS_CONTROLLER_IMAGE:-"mcs-api-controller"}

k1="kubectl --kubeconfig ${kubeconfig1}"
k2="kubectl --kubeconfig ${kubeconfig2}"

if [ ! -z "${BUILD_CONTROLLER}" ] || [ -z "$(docker images mcs-api-controller -q)" ]; then
  pushd ../
  make docker-build
  popd
fi

coredns_version="1.11.4"
coredns_image="multicluster/coredns:latest"
coredns_path="/tmp/coredns-${coredns_version}"
if [ ! -d "${coredns_path}" ]; then
  pushd /tmp
  git clone --depth 1 https://github.com/coredns/coredns.git --branch v${coredns_version} --single-branch "${coredns_path}"
  popd
fi
pushd "${coredns_path}"
if ! grep -q -F 'multicluster:github.com/coredns/multicluster' "plugin.cfg"; then
  sed -i -e 's/^kubernetes:kubernetes$/&\nmulticluster:github.com\/coredns\/multicluster/' "plugin.cfg"
fi
docker run --rm \
    -v $PWD:/go/src/github.com/coredns/coredns -w /go/src/github.com/coredns/coredns \
        golang:1.23 make gen coredns GOFLAGS=-buildvcs=false
docker build -t "${coredns_image}" .
popd

kind create cluster --name "${c1}" --config "$PWD/${c1}.yaml"
kind create cluster --name "${c2}" --config "$PWD/${c2}.yaml"

kind get kubeconfig --name "${c1}" > "${kubeconfig1}"
kind get kubeconfig --name "${c2}" > "${kubeconfig2}"

kind load docker-image "${controller_image}" "${coredns_image}" --name "${c1}"
kind load docker-image "${controller_image}" "${coredns_image}" --name "${c2}"

echo "Configuring CoreDNS"
function update_coredns() {
  kubectl --kubeconfig ${1} patch clusterrole system:coredns --type json --patch-file coredns-rbac.json
  kubectl --kubeconfig ${1} get configmap -n kube-system coredns -o yaml | \
    sed -E -e 's/^(\s*)kubernetes.*cluster\.local.*$/\1multicluster clusterset.local\n&/' | \
    kubectl --kubeconfig ${1} replace -f-
  kubectl --kubeconfig ${1} rollout restart deploy -n kube-system coredns
}
update_coredns ${kubeconfig1}
update_coredns ${kubeconfig2}

function pod_cidrs() {
  kubectl --kubeconfig "${1}" get nodes -o jsonpath='{range .items[*]}{.spec.podCIDR}{"\n"}'
}

function add_routes() {
  unset IFS
  routes=$(kubectl --kubeconfig ${3} get node ${2} -o jsonpath='ip route add {.spec.podCIDR} via {.status.addresses[?(.type=="InternalIP")].address}')
  echo "Connecting cluster ${1} to ${2}"

  IFS=$'\n'
  for n in $(kind get nodes --name "${1}"); do
    for r in $routes; do
      eval "docker exec $n $r"
    done
  done
  unset IFS
}

waitfor pod_cidrs ${kubeconfig1}
waitfor pod_cidrs ${kubeconfig2}

echo "Connecting cluster networks..."
add_routes "${c1}" "${c2}-control-plane" "${kubeconfig2}"
add_routes "${c2}" "${c1}-control-plane" "${kubeconfig1}"
echo "Cluster networks connected"

${k1} apply -f ../config/crd -f ../config/rbac
${k2} apply -f ../config/crd -f ../config/rbac

${k1} create sa mcs-api-controller
${k1} create clusterrolebinding mcs-api-binding --clusterrole=mcs-derived-service-manager --serviceaccount=default:mcs-api-controller
${k1} run --image "${controller_image}" --image-pull-policy=Never mcs-api-controller --overrides='{ "spec": { "serviceAccount": "mcs-api-controller" }  }'

${k2} create sa mcs-api-controller
${k2} create clusterrolebinding mcs-api-binding --clusterrole=mcs-derived-service-manager --serviceaccount=default:mcs-api-controller
${k2} run --image "${controller_image}" --image-pull-policy=Never mcs-api-controller --overrides='{ "spec": { "serviceAccount": "mcs-api-controller" }  }'
