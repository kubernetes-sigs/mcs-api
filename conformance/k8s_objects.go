/*
Copyright 2023 The Kubernetes Authors.

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

package conformance

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var helloService = v1.Service{
	ObjectMeta: metav1.ObjectMeta{
		Name: "hello",
	},
	Spec: v1.ServiceSpec{
		Selector: map[string]string{
			"app": "hello",
		},
		Ports: []v1.ServicePort{
			{
				Name:     "tcp",
				Port:     42,
				Protocol: v1.ProtocolTCP,
			},
			{
				Name:     "udp",
				Port:     42,
				Protocol: v1.ProtocolUDP,
			},
		},
		SessionAffinity: v1.ServiceAffinityClientIP,
		SessionAffinityConfig: &v1.SessionAffinityConfig{
			ClientIP: &v1.ClientIPConfig{TimeoutSeconds: ptr.To(int32(10))},
		},
	},
}

var helloDeployment = appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name: "hello",
	},
	Spec: appsv1.DeploymentSpec{
		Replicas: ptr.To(int32(1)),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "hello",
			},
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": "hello"},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "hello-tcp",
						Image: "alpine/socat:1.7.4.4",
						Args:  []string{"-v", "-v", "TCP-LISTEN:42,crlf,reuseaddr,fork", "SYSTEM:echo pod ip $(MY_POD_IP)"},
						Env: []v1.EnvVar{
							{
								Name: "MY_POD_IP",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "status.podIP",
									},
								},
							},
						},
					},
					{
						Name:  "hello-udp",
						Image: "alpine/socat:1.7.4.4",
						Args:  []string{"-v", "-v", "UDP-LISTEN:42,crlf,reuseaddr,fork", "SYSTEM:echo pod ip $(MY_POD_IP)"},
						Env: []v1.EnvVar{
							{
								Name: "MY_POD_IP",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "status.podIP",
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

var requestPod = v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:   "request",
		Labels: map[string]string{"app": "request"},
	},
	Spec: v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  "request",
				Image: "busybox",
				Args:  []string{"/bin/sh", "-ec", "while :; do echo '.'; sleep 5 ; done"},
			},
		},
	},
}
