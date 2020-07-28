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

package e2etest

const service = `apiVersion: v1
kind: Service
metadata:
  name: serve
spec:
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: serve
`

const deployment = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: serve
spec:
  replicas: 1
  selector:
    matchLabels:
      app: serve
  template:
    metadata:
      labels:
        app: serve
    spec:
      containers:
      - name: serve
        image: jeremyot/serve
        args:
        - "--message='hello from cluster 2 (Node: {{env \"NODE_NAME\"}} Pod: {{env \"POD_NAME\"}} Address: {{addr}})'"
        env:
          - name: NODE_NAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
          - name: POD_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
`
const serviceImport = `apiVersion: multicluster.x-k8s.io/v1alpha1
kind: ServiceImport
metadata:
  name: serve
spec:
  type: SuperclusterIP
  ports:
  - port: 80
    protocol: TCP
`
