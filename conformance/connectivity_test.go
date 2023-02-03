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
	"context"
	"fmt"
	"math/rand"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

var _ = Describe("Connectivity to remote services", func() {
	ctx := context.TODO()

	// Shared namespace
	namespace := fmt.Sprintf("mcs-conformance-%v", rand.Uint32())

	BeforeEach(func() {
		Expect(clients).ToNot(BeEmpty())

		// Set up the shared namespace
		for _, client := range clients {
			_, err := client.k8s.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		// Set up the remote service (the first cluster is considered to be the remote)
		_, err := clients[0].k8s.AppsV1().Deployments(namespace).Create(ctx, &helloDeployment, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = clients[0].k8s.CoreV1().Services(namespace).Create(ctx, &helloService, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		// Clean up the shared namespace
		for _, client := range clients {
			err := client.k8s.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		}
	})

	//	Context("with no exported service", func() {
	//		It("should be inaccessible", func() {
	//			By("attempting to access the remote service", func() {})
	//		})
	//	})

	Context("with an exported service", func() {
		It("should be accessible through DNS (after a potential delay)", Label("optional"), func() {
			AddReportEntry("spec", "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#dns")
			By("exporting the service", func() {
				// On the "remote" cluster
				_, err := clients[0].mcs.MulticlusterV1alpha1().ServiceExports(namespace).Create(ctx,
					&v1alpha1.ServiceExport{ObjectMeta: metav1.ObjectMeta{Name: helloService.Name}}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
			By("issuing a request from all clusters", func() {
				// Run on all clusters
				for _, client := range clients {
					_, err := client.k8s.CoreV1().Pods(namespace).Create(ctx, &requestPod, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					Eventually(func() error {
						pod, err := client.k8s.CoreV1().Pods(namespace).Get(ctx, requestPod.Name, metav1.GetOptions{})
						if err != nil {
							return err
						}
						if pod.Status.Phase != v1.PodRunning {
							return fmt.Errorf("pod is not running yet, current status %v", pod.Status.Phase)
						}
						return nil
					}, 20, 1).Should(Succeed())

					command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc %s.%s.svc.clusterset.local 42", helloService.Name, namespace)}
					Eventually(func() string {
						stdout, _, _ := execCmd(client.k8s, client.rest, requestPod.Name, namespace, command)
						return string(stdout)
					}, 20, 1).Should(ContainSubstring("pod ip"))
				}
			})
		})
	})
})
