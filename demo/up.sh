#!/bin/bash

set -e
set -x

cd $(dirname ${BASH_SOURCE})

c1=c1
c2=c2

kind create cluster --name "${c1}"
kind create cluster --name "${c2}"

kind get kubeconfig --name "${c1}" > "${c1}".config
kind get kubeconfig --name "${c2}" > "${c2}".config

function waitfor() {
  for i in {1..30}; do
    if [ ! -z "$(${@})" ]; then
      break
    fi
    sleep 1
  done
  if [ -z "$(${@})" ]; then
    echo "No results for '${1}' after 30 attempts"
  fi
}

function pod_cidrs() {
  kubectl --kubeconfig ${1}.config get nodes -o jsonpath='{range .items[*]}{.spec.podCIDR}{"\n"}'
}

function add_routes() {
  unset IFS
  routes=$(kubectl --kubeconfig ${2}.config get nodes -o jsonpath='{range .items[*]}ip route add {.spec.podCIDR} via {.status.addresses[?(.type=="InternalIP")].address}{"\n"}')
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

kubectl --kubeconfig "${c1}.config" apply -f ../config/crd
kubectl --kubeconfig "${c2}.config" apply -f ../config/crd