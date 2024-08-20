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

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

var (
	replicaCount = int32(1)
	helloService = v1.Service{
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
		},
	}
	helloServiceImport = v1alpha1.ServiceImport{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello",
		},
		Spec: v1alpha1.ServiceImportSpec{
			Type: v1alpha1.ClusterSetIP,
			Ports: []v1alpha1.ServicePort{
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
		},
	}
	helloDeployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCount,
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
							Args:  []string{"-v", "-v", "TCP-LISTEN:42,crlf,reuseaddr,fork", "SYSTEM:echo $(MY_POD_IP)"},
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
							Args:  []string{"-v", "-v", "UDP-LISTEN:42,crlf,reuseaddr,fork", "SYSTEM:echo $(MY_POD_IP)"},
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
	requestPod = v1.Pod{
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
)

var _ = Describe("Connectivity", func() {
	var (
		namespace string

		ctx           = context.Background()
		serviceImport *v1alpha1.ServiceImport
		podIPs        []string
		reqPod        *v1.Pod

		verifyConnectivity = func() {
			checkPodReachable := func(command []string) {
				ips := map[string]int{}
				Eventually(func(g Gomega) {
					stdout, _, err := execCmd(cluster1.k8s, restcfg1, reqPod.Name, reqPod.Namespace, command)
					g.Expect(err).ToNot(HaveOccurred())
					ip := strings.TrimSpace(string(stdout))
					g.Expect(ip).To(BeElementOf(podIPs))
					ips[ip]++
				}, "5m").MustPassRepeatedly(50).Should(Succeed())
				Expect(ips).To(HaveEach(Not(BeZero())))
			}

			Specify("UDP connects across clusters using the VIP", func() {
				command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc -uw1 %s 42", serviceImport.Spec.IPs[0])}
				checkPodReachable(command)
			})
			Specify("TCP connects across clusters using the VIP", func() {
				command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc %s 42", serviceImport.Spec.IPs[0])}
				checkPodReachable(command)
			})
			Specify("UDP connects across clusters using the DNS name", func() {
				command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc -uw1 %s.%s.svc.clusterset.local 42", serviceImport.Name, serviceImport.Namespace)}
				checkPodReachable(command)
			})
			Specify("TCP connects across clusters using the DNS name", func() {
				command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc %s.%s.svc.clusterset.local 42", serviceImport.Name, serviceImport.Namespace)}
				checkPodReachable(command)
			})
		}
	)
	BeforeEach(func() {
		namespace = fmt.Sprintf("mcse2e-conformance-%v", rand.Uint32())
		_, err := cluster1.k8s.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = cluster2.k8s.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		pod := requestPod
		_, err = cluster1.k8s.CoreV1().Pods(namespace).Create(ctx, &pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		dep := helloDeployment
		_, err = cluster2.k8s.AppsV1().Deployments(namespace).Create(ctx, &dep, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		svc := helloService
		_, err = cluster2.k8s.CoreV1().Services(namespace).Create(ctx, &svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		imp := helloServiceImport
		_, err = cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Create(ctx, &imp, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() string {
			rp, err := cluster1.k8s.CoreV1().Pods(namespace).Get(ctx, requestPod.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return rp.Name
		}).ShouldNot(BeEmpty())
		Eventually(func() string {
			pods, err := cluster2.k8s.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: metav1.FormatLabelSelector(helloDeployment.Spec.Selector),
			})
			Expect(err).ToNot(HaveOccurred())
			if len(pods.Items) > 0 {
				return pods.Items[0].Status.PodIP
			}
			return ""
		}, 30).ShouldNot(BeEmpty())
		pods, err := cluster2.k8s.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(helloDeployment.Spec.Selector),
		})
		Expect(err).ToNot(HaveOccurred())
		for _, pod := range pods.Items {
			podIPs = append(podIPs, pod.Status.PodIP)
		}

		exportService(ctx, cluster2, cluster1, namespace, svc.Name)

		Eventually(func() []string {
			svcImport, err := cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Get(ctx, helloServiceImport.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return svcImport.Spec.IPs
		}).ShouldNot(BeEmpty())
		serviceImport, err = cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Get(ctx, helloServiceImport.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		reqPod, err = cluster1.k8s.CoreV1().Pods(namespace).Get(ctx, requestPod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		By("Created all in " + namespace)
	})
	AfterEach(func() {
		if *noTearDown {
			By(fmt.Sprintf("Skipping teardown. Test namespace %q", namespace))
			By(fmt.Sprintf("Cluster 1: kubectl --kubeconfig %q -n %q", *kubeconfig1, namespace))
			By(fmt.Sprintf("Cluster 2: kubectl --kubeconfig %q -n %q", *kubeconfig2, namespace))
			return
		}
		Expect(cluster1.k8s.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})).To(Succeed())
		Expect(cluster2.k8s.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})).To(Succeed())
	})

	When("trying to reach a service exported from only the remote cluster", func() {
		verifyConnectivity()
	})

	When("trying to reach a service exported by both the local and remote clusters", func() {
		BeforeEach(func() {
			dep := helloDeployment
			_, err := cluster1.k8s.AppsV1().Deployments(namespace).Create(ctx, &dep, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			svc := helloService
			_, err = cluster1.k8s.CoreV1().Services(namespace).Create(ctx, &svc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() string {
				pods, err := cluster1.k8s.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
					LabelSelector: metav1.FormatLabelSelector(helloDeployment.Spec.Selector),
				})
				Expect(err).ToNot(HaveOccurred())
				if len(pods.Items) > 0 {
					return pods.Items[0].Status.PodIP
				}
				return ""
			}, 30).ShouldNot(BeEmpty())
			pods, err := cluster1.k8s.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: metav1.FormatLabelSelector(helloDeployment.Spec.Selector),
			})
			Expect(err).ToNot(HaveOccurred())
			for _, pod := range pods.Items {
				podIPs = append(podIPs, pod.Status.PodIP)
			}
			exportService(ctx, cluster1, cluster1, namespace, svc.Name)
		})

		verifyConnectivity()
	})
})
