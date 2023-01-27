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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
							Image: "alpine/socat",
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
							Image: "alpine/socat",
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
		pods1         *v1.PodList
		pods2         *v1.PodList
		reqPod        *v1.Pod
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
		dep1 := helloDeployment
		_, err = cluster1.k8s.AppsV1().Deployments(namespace).Create(ctx, &dep1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		dep2 := helloDeployment
		_, err = cluster2.k8s.AppsV1().Deployments(namespace).Create(ctx, &dep2, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		svc1 := helloService
		_, err = cluster1.k8s.CoreV1().Services(namespace).Create(ctx, &svc1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		svc2 := helloService
		_, err = cluster2.k8s.CoreV1().Services(namespace).Create(ctx, &svc2, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		imp1 := helloServiceImport
		_, err = cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Create(ctx, &imp1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() string {
			rp, err := cluster1.k8s.CoreV1().Pods(namespace).Get(ctx, requestPod.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return rp.Name
		}).ShouldNot(BeEmpty())
		Eventually(func() string {
			pods, err := cluster1.k8s.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: metav1.FormatLabelSelector(helloDeployment.Spec.Selector),
			})
			Expect(err).ToNot(HaveOccurred())
			return pods.Items[0].Status.PodIP
		}, 30).ShouldNot(BeEmpty())
		pods1, err = cluster1.k8s.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(helloDeployment.Spec.Selector),
		})
		Eventually(func() string {
			pods, err := cluster2.k8s.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: metav1.FormatLabelSelector(helloDeployment.Spec.Selector),
			})
			Expect(err).ToNot(HaveOccurred())
			return pods.Items[0].Status.PodIP
		}, 30).ShouldNot(BeEmpty())
		Expect(err).ToNot(HaveOccurred())
		pods2, err = cluster2.k8s.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(helloDeployment.Spec.Selector),
		})
		Expect(err).ToNot(HaveOccurred())
		var slicesv1 *discoveryv1.EndpointSliceList
		var slicesv1b1 *discoveryv1beta1.EndpointSliceList
		Eventually(func() int {
			eps := 0
			slicesv1, err = cluster2.k8s.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labels.Set{discoveryv1.LabelServiceName: helloService.Name}.AsSelector().String(),
			})
			if err != nil && errors.IsNotFound(err) {
				slicesv1b1, err = cluster2.k8s.DiscoveryV1beta1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
					LabelSelector: labels.Set{discoveryv1beta1.LabelServiceName: helloService.Name}.AsSelector().String(),
				})
				Expect(err).ToNot(HaveOccurred())
				for _, s := range slicesv1b1.Items {
					eps += len(s.Endpoints)
				}
			} else {
				Expect(err).ToNot(HaveOccurred())
				for _, s := range slicesv1.Items {
					eps += len(s.Endpoints)
				}
			}
			return eps
		}, 30).Should(Equal(1))
		if slicesv1b1 != nil {
			importedSlice := slicesv1b1.Items[0]
			importedSlice.ObjectMeta = metav1.ObjectMeta{
				Name: importedSlice.Name,
				Labels: map[string]string{
					v1alpha1.LabelServiceName: helloServiceImport.Name,
				},
			}
			_, err = cluster1.k8s.DiscoveryV1beta1().EndpointSlices(namespace).Create(ctx, &importedSlice, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() []string {
				svcImport, err := cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Get(ctx, helloServiceImport.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return svcImport.Spec.IPs
			}).ShouldNot(BeEmpty())
			serviceImport, err = cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Get(ctx, helloServiceImport.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() string {
				updatedSlice, err := cluster1.k8s.DiscoveryV1beta1().EndpointSlices(namespace).Get(ctx, importedSlice.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return updatedSlice.Labels[discoveryv1beta1.LabelServiceName]
			}).ShouldNot(BeEmpty())
		} else {
			importedSlice := slicesv1.Items[0]
			importedSlice.ObjectMeta = metav1.ObjectMeta{
				Name: importedSlice.Name,
				Labels: map[string]string{
					v1alpha1.LabelServiceName: helloServiceImport.Name,
				},
			}
			_, err = cluster1.k8s.DiscoveryV1().EndpointSlices(namespace).Create(ctx, &importedSlice, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() []string {
				svcImport, err := cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Get(ctx, helloServiceImport.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return svcImport.Spec.IPs
			}).ShouldNot(BeEmpty())
			serviceImport, err = cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Get(ctx, helloServiceImport.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() string {
				updatedSlice, err := cluster1.k8s.DiscoveryV1().EndpointSlices(namespace).Get(ctx, importedSlice.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return updatedSlice.Labels[discoveryv1beta1.LabelServiceName]
			}).ShouldNot(BeEmpty())
		}
		reqPod, err = cluster1.k8s.CoreV1().Pods(namespace).Get(ctx, requestPod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		By("Created all in " + namespace)
	})
	AfterEach(func() {
		if *noTearDown {
			By(fmt.Sprintf("Skipping tearndown. Test namespace %q", namespace))
			By(fmt.Sprintf("Cluster 1: kubectl --kubeconfig %q -n %q", *kubeconfig1, namespace))
			By(fmt.Sprintf("Cluster 2: kubectl --kubeconfig %q -n %q", *kubeconfig2, namespace))
			return
		}
		Expect(cluster1.k8s.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}))
		Expect(cluster2.k8s.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}))
	})
	Specify("UDP connects across clusters using the VIP", func() {
		successfulChecks := 0
		command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc -uw1 %s 42", serviceImport.Spec.IPs[0])}
		for i := 0; i <= 59; i++ {
			stout, _, err := execCmd(cluster1.k8s, restcfg1, reqPod.Name, reqPod.Namespace, command)
			Expect(err).ToNot(HaveOccurred())
			ip := strings.TrimSpace(string(stout))
			if ip == strings.TrimSpace(pods2.Items[0].Status.PodIP) {
				successfulChecks++
			}
		}
		Eventually(func() int {
			return successfulChecks
		}).Should(BeNumerically(">=", 50))
	})
	Specify("TCP connects across clusters using the VIP", func() {
		successfulChecks := 0
		command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc %s 42", serviceImport.Spec.IPs[0])}
		for i := 0; i <= 59; i++ {
			stout, _, err := execCmd(cluster1.k8s, restcfg1, reqPod.Name, reqPod.Namespace, command)
			Expect(err).ToNot(HaveOccurred())
			ip := strings.TrimSpace(string(stout))
			if ip == strings.TrimSpace(pods2.Items[0].Status.PodIP) {
				successfulChecks++
			}
		}
		Eventually(func() int {
			return successfulChecks
		}).Should(BeNumerically(">=", 50))
	})
	Specify("TCP connects across clusters using the cluster.local", func() {
		successfulChecks := 0
		command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc hello.%s.svc.cluster.local 42", namespace)}
		for i := 0; i <= 59; i++ {
			stout, _, err := execCmd(cluster1.k8s, restcfg1, reqPod.Name, reqPod.Namespace, command)
			Expect(err).ToNot(HaveOccurred())
			ip := strings.TrimSpace(string(stout))
			if ip == strings.TrimSpace(pods1.Items[0].Status.PodIP) {
				successfulChecks++
			}
		}
		Eventually(func() int {
			return successfulChecks
		}).Should(BeNumerically(">=", 50))
	})
	Specify("TCP connects across clusters using the clusterset.local", func() {
		successfulChecks := 0
		command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc hello.%s.svc.clusterset.local 42", namespace)}
		for i := 0; i <= 59; i++ {
			stout, _, err := execCmd(cluster1.k8s, restcfg1, reqPod.Name, reqPod.Namespace, command)
			Expect(err).ToNot(HaveOccurred())
			ip := strings.TrimSpace(string(stout))
			if ip == strings.TrimSpace(pods2.Items[0].Status.PodIP) {
				successfulChecks++
			}
		}
		Eventually(func() int {
			return successfulChecks
		}).Should(BeNumerically(">=", 50))
	})
	Specify("TCP connects to both clusters using the clusterset.local", func() {
		successfulChecks := []bool{false, false}
		Eventually(func() int {
			slices, err := cluster1.k8s.DiscoveryV1beta1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labels.Set{discoveryv1beta1.LabelServiceName: helloService.Name}.AsSelector().String(),
			})
			Expect(err).ToNot(HaveOccurred())
			eps := 0
			for _, s := range slices.Items {
				eps += len(s.Endpoints)
			}
			return eps
		}, 30).Should(Equal(1))
		slices, err := cluster1.k8s.DiscoveryV1beta1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labels.Set{discoveryv1beta1.LabelServiceName: helloService.Name}.AsSelector().String(),
		})
		Expect(err).ToNot(HaveOccurred())
		importedSlice := slices.Items[0]
		importedSlice.ObjectMeta = metav1.ObjectMeta{
			Name: importedSlice.Name,
			Labels: map[string]string{
				v1alpha1.LabelServiceName: helloServiceImport.Name,
			},
		}
		_, err = cluster1.k8s.DiscoveryV1beta1().EndpointSlices(namespace).Update(ctx, &importedSlice, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() string {
			updatedSlice, err := cluster1.k8s.DiscoveryV1beta1().EndpointSlices(namespace).Get(ctx, importedSlice.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return updatedSlice.Labels[discoveryv1beta1.LabelServiceName]
		}).ShouldNot(BeEmpty())
		Expect(err).ToNot(HaveOccurred())
		command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc hello.%s.svc.clusterset.local 42", namespace)}
		for i := 0; i <= 59; i++ {
			stout, _, err := execCmd(cluster1.k8s, restcfg1, reqPod.Name, reqPod.Namespace, command)
			Expect(err).ToNot(HaveOccurred())
			ip := strings.TrimSpace(string(stout))
			if ip == strings.TrimSpace(pods1.Items[0].Status.PodIP) {
				successfulChecks[0] = true
			} else if ip == strings.TrimSpace(pods2.Items[0].Status.PodIP) {
				successfulChecks[1] = true
			}
		}
		Eventually(func() []bool {
			return successfulChecks
		}).Should(Equal(([]bool{true, true})))
	})
})
