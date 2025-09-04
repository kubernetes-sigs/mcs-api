/*
Copyright 2024 The Kubernetes Authors.

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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Local service not impacted", func() {
	helloDeployment := appsv1.Deployment{
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
							Args:  []string{"-v", "-v", "TCP-LISTEN:42,crlf,reuseaddr,fork", "SYSTEM:echo $(CLUSTER_ID)"},
							Env: []v1.EnvVar{
								{
									Name: "CLUSTER_ID",
									ValueFrom: &v1.EnvVarSource{
										ConfigMapKeyRef: &v1.ConfigMapKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: "cluster-info",
											},
											Key: "clusterID",
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

	var (
		namespace string

		ctx    = context.Background()
		reqPod *v1.Pod
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
		_, err = cluster1.k8s.CoreV1().ConfigMaps(namespace).Create(ctx, &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-info",
			},
			Data: map[string]string{
				"clusterID": "cluster1",
			},
		}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		_, err = cluster2.k8s.CoreV1().ConfigMaps(namespace).Create(ctx, &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-info",
			},
			Data: map[string]string{
				"clusterID": "cluster2",
			},
		}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		pod := requestPod
		_, err = cluster1.k8s.CoreV1().Pods(namespace).Create(ctx, &pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		dep := helloDeployment
		_, err = cluster1.k8s.AppsV1().Deployments(namespace).Create(ctx, &dep, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = cluster2.k8s.AppsV1().Deployments(namespace).Create(ctx, &dep, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		svc := helloService
		_, err = cluster1.k8s.CoreV1().Services(namespace).Create(ctx, &svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = cluster2.k8s.CoreV1().Services(namespace).Create(ctx, &svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Create(ctx, &helloServiceImport, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		deploymentAvailable := func(clients clusterClients) func(Gomega) {
			return func(g Gomega) {
				deployment, err := clients.k8s.AppsV1().Deployments(namespace).Get(ctx, dep.Name, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(deployment.Status.Conditions).To(ContainElement(Satisfy(func(cond appsv1.DeploymentCondition) bool {
					return cond.Type == appsv1.DeploymentAvailable &&
						cond.Status == v1.ConditionTrue
				})))
			}
		}
		Eventually(deploymentAvailable(cluster1), 30).Should(Succeed())
		Eventually(deploymentAvailable(cluster2), 30).Should(Succeed())

		exportService(ctx, cluster2, cluster1, namespace, svc.Name)

		Eventually(func() []string {
			svcImport, err := cluster1.mcs.MulticlusterV1alpha1().ServiceImports(namespace).Get(ctx, helloServiceImport.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return svcImport.Spec.IPs
		}).ShouldNot(BeEmpty())
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
	Specify("DNS resolves as expected", func() {
		checkAllClustersReachable := func(command []string, clusterIDs ...string) {
			clusters := map[string]int{}
			Eventually(func(g Gomega) {
				stdout, _, err := execCmd(cluster1.k8s, restcfg1, reqPod.Name, reqPod.Namespace, command)
				g.Expect(err).ToNot(HaveOccurred())
				clusterID := strings.TrimSpace(string(stdout))
				g.Expect(clusterID).To(BeElementOf(clusterIDs))
				clusters[clusterID]++
			}).MustPassRepeatedly(20).Within(time.Second * 10).Should(Succeed())
			Expect(clusters).To(HaveEach(Not(BeZero())))
		}

		By("verifying a local, unexported service is reachable via cluster.local")
		command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc %s.%s.svc.cluster.local 42", helloService.Name, namespace)}
		checkAllClustersReachable(command, "cluster1")

		By("verifying a remote, exported service is reachable via clusterset.local")
		command = []string{"sh", "-c", fmt.Sprintf("echo hi | nc %s.%s.svc.clusterset.local 42", helloServiceImport.Name, namespace)}
		checkAllClustersReachable(command, "cluster2")

		By("exporting the service from cluster1")
		exportService(ctx, cluster1, cluster1, namespace, helloService.Name)

		By("verifying a service exported from both local and remote clusters is reachable via clusterset.local")
		checkAllClustersReachable(command, "cluster1", "cluster2")

		By("verifying a local, exported service is reachable via cluster.local")
		command = []string{"sh", "-c", fmt.Sprintf("echo hi | nc %s.%s.svc.cluster.local 42", helloService.Name, namespace)}
		checkAllClustersReachable(command, "cluster1")
	})
})
