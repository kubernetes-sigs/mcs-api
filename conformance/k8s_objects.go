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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const helloServiceName = "hello"

func newHelloService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: helloServiceName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": helloServiceName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "tcp",
					Port:     42,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "udp",
					Port:     42,
					Protocol: corev1.ProtocolUDP,
				},
			},
			SessionAffinity: corev1.ServiceAffinityClientIP,
			SessionAffinityConfig: &corev1.SessionAffinityConfig{
				ClientIP: &corev1.ClientIPConfig{TimeoutSeconds: ptr.To(int32(10))},
			},
		},
	}
}

func newHelloDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: helloServiceName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": helloServiceName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": helloServiceName},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "hello-tcp",
							Image: "alpine/socat:1.7.4.4",
							Args:  []string{"-v", "-v", "TCP-LISTEN:42,crlf,reuseaddr,fork", "SYSTEM:echo pod ip $(MY_POD_IP)"},
							Env: []corev1.EnvVar{
								{
									Name: "MY_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
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
							Env: []corev1.EnvVar{
								{
									Name: "MY_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
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
}

func newRequestPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "request",
			Labels: map[string]string{"app": "request"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "request",
					Image: "busybox",
					Args:  []string{"/bin/sh", "-ec", "while :; do echo '.'; sleep 5 ; done"},
				},
			},
		},
	}
}
