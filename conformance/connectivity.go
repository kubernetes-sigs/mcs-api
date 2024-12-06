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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("", func() {
	t := newTestDriver()

	Context("Connectivity to a service that is not exported", func() {
		It("should be inaccessible", Label(RequiredLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#exporting-services")
			By("attempting to access the remote service", func() {
				By("issuing a request from all clusters", func() {
					command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc %s.%s.svc.clusterset.local 42",
						t.helloService.Name, t.namespace)}

					// Run on all clusters
					for _, client := range clients {
						// Repeat multiple times
						for i := 0; i < 20; i++ {
							Expect(t.execCmdOnRequestPod(&client, command)).NotTo(ContainSubstring("pod ip"), reportNonConformant(""))
						}
					}
				})
			})
		})
	})

	Context("Connectivity to an exported ClusterIP service", func() {
		It("should be accessible through DNS", Label(OptionalLabel, ConnectivityLabel, ClusterIPLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#dns")
			By("Exporting the service", func() {
				// On the "remote" cluster
				t.createServiceExport(&clients[0])
			})
			By("Issuing a request from all clusters", func() {
				// Run on all clusters
				command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc %s.%s.svc.clusterset.local 42",
					t.helloService.Name, t.namespace)}

				for _, client := range clients {
					By(fmt.Sprintf("Executing command %q on cluster %q", strings.Join(command, " "), client.name))

					t.awaitCmdOutputContains(&client, command, "pod ip", 1, reportNonConformant(""))
				}
			})
		})
	})

	Context("Connectivity to a ClusterIP service existing in two clusters but exported from one", func() {
		BeforeEach(func() {
			requireTwoClusters()
		})

		JustBeforeEach(func() {
			t.deployHelloService(&clients[1], newHelloService())
		})

		It("should only access the exporting cluster", Label(OptionalLabel, ConnectivityLabel, ClusterIPLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/blob/master/keps/sig-multicluster/1645-multi-cluster-services-api/README.md#exporting-services")

			By(fmt.Sprintf("Exporting the service on cluster %q", clients[0].name))

			t.createServiceExport(&clients[0])

			By(fmt.Sprintf("Awaiting service deployment pod IP on cluster %q", clients[0].name))

			servicePodIP := ""

			Eventually(func() string {
				pods, err := clients[0].k8s.CoreV1().Pods(t.namespace).List(context.TODO(), metav1.ListOptions{
					LabelSelector: metav1.FormatLabelSelector(newHelloDeployment().Spec.Selector),
				})
				Expect(err).ToNot(HaveOccurred())

				if len(pods.Items) > 0 {
					servicePodIP = pods.Items[0].Status.PodIP
				}

				return servicePodIP
			}, 20, 1).ShouldNot(BeEmpty(), "Service deployment pod was not allocated an IP")

			By(fmt.Sprintf("Retrieved service deployment pod IP %q", servicePodIP))

			command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc %s.%s.svc.clusterset.local 42",
				t.helloService.Name, t.namespace)}

			for _, client := range clients {
				By(fmt.Sprintf("Executing command %q on cluster %q", strings.Join(command, " "), client.name))

				t.awaitCmdOutputContains(&client, command, servicePodIP, 10, reportNonConformant(""))
			}
		})
	})
})
