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

# Resolve the absolute path to this script directory.
cd $(dirname "${BASH_SOURCE[0]}")
demo_dir=$(realpath "$(pwd)")
scripts_dir=$(realpath "$(pwd)/../scripts")

. "${demo_dir}/udemo.sh"
. "${scripts_dir}/util.sh"

DEMO_AUTO_RUN=true

kubeconfig1="${KUBECONFIG1:-${scripts_dir}/c1.kubeconfig}"
kubeconfig2="${KUBECONFIG2:-${scripts_dir}/c2.kubeconfig}"

k1="kubectl --kubeconfig ${kubeconfig1}"
k2="kubectl --kubeconfig ${kubeconfig2}"

desc "Setup our demo namespace"
run "${k1} create ns demo"
run "${k2} create ns demo"

c1_pane=$(tmux split-window -h -d -P)

function cleanup() {
    tmux kill-pane -t "$c1_pane"
}
trap cleanup EXIT

tmux send -t "$c1_pane" "${k1} logs -f mcs-api-controller" Enter

desc "Create our service in each cluster"
run "${k1} apply -f ${script_dir}/yaml/dep1.yaml -f ${script_dir}/yaml/svc.yaml"
run "${k2} apply -f ${script_dir}/yaml/dep2.yaml -f ${script_dir}/yaml/svc.yaml"
run "${k1} get endpointslice -n demo"

desc "Lets look at some requests to the service in cluster 1"
run "${k1} -n demo run -i --rm --restart=Never --image=jeremyot/request:0a40de8 request -- --duration=5s --address=serve.demo.svc.cluster.local"

desc "Ok, looks normal. Let's import the service from our other cluster"
ep_1=$(${k1} get endpointslice -n demo -l 'kubernetes.io/service-name=serve' --template="{{(index .items 0).metadata.name}}")
ep_2=$(${k2} get endpointslice -n demo -l 'kubernetes.io/service-name=serve' --template="{{(index .items 0).metadata.name}}")

run "${k1} get endpointslice -n demo ${ep_1} -o yaml | ${script_dir}/edit-meta --metadata '{name: imported-${ep_1}, namespace: demo, labels: {multicluster.kubernetes.io/service-name: serve}}' > ${script_dir}/yaml/slice-1.tmp"
run "${k2} get endpointslice -n demo ${ep_2} -o yaml | ${script_dir}/edit-meta --metadata '{name: imported-${ep_2}, namespace: demo, labels: {multicluster.kubernetes.io/service-name: serve}}' > ${script_dir}/yaml/slice-2.tmp"
run "${k1} apply -f ${script_dir}/yaml/serviceimport.yaml -f ${script_dir}/yaml/slice-1.tmp -f ${script_dir}/yaml/slice-2.tmp"
run "${k2} apply -f ${script_dir}/yaml/serviceimport.yaml -f ${script_dir}/yaml/slice-1.tmp -f ${script_dir}/yaml/slice-2.tmp"
run "${k1} apply -f ${script_dir}/yaml/serviceimport-with-vip.yaml -f ${script_dir}/yaml/slice-1.tmp -f ${script_dir}/yaml/slice-2.tmp"
run "${k2} apply -f ${script_dir}/yaml/serviceimport-with-vip.yaml -f ${script_dir}/yaml/slice-1.tmp -f ${script_dir}/yaml/slice-2.tmp"

desc "See what we've created..."
run "${k1} get -n demo serviceimports"
run "${k1} get -n demo endpointslice"
run "${k1} get -n demo service"

function import_ip() {
    ${k1} get serviceimport -n demo -o go-template --template='{{index (index .items 0).spec.ips 0}}'
}

waitfor import_ip

vip=$(${k1} get serviceimport -n demo -o go-template --template='{{index (index .items 0).spec.ips 0}}')
desc "Now grab the multi-cluster VIP from the serviceimport..."
run "${k1} get serviceimport -n demo -o go-template --template='{{index (index .items 0).spec.ips 0}}{{\"\n\"}}'"
desc "...and connect to it"
run "${k1} -n demo run -i --rm --restart=Never --image=jeremyot/request:0a40de8 request -- --duration=10s --address=${vip}"
desc "We have a multi-cluster service!"
desc "See for yourself"
desc "Cluster 1: kubectl --kubeconfig ${kubeconfig1} -n demo"
desc "Cluster 2: kubectl --kubeconfig ${kubeconfig2} -n demo"
desc "(Enter to exit)"
read -s
