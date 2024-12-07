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
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Connectivity to remote services", func() {
	t := newTestDriver()

	Context("with no exported service", func() {
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

	Context("with an exported ClusterIP service", func() {
		It("should be accessible through DNS (after a potential delay)", Label(OptionalLabel, ConnectivityLabel, ClusterIPLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#dns")
			By("exporting the service", func() {
				// On the "remote" cluster
				t.createServiceExport(&clients[0])
			})
			By("issuing a request from all clusters", func() {
				// Run on all clusters
				command := []string{"sh", "-c", fmt.Sprintf("echo hi | nc %s.%s.svc.clusterset.local 42",
					t.helloService.Name, t.namespace)}

				// Run on all clusters
				for _, client := range clients {
					By(fmt.Sprintf("Executing command %q on cluster %q", strings.Join(command, " "), client.name))

					t.awaitCmdOutputContains(&client, command, "pod ip", 1, reportNonConformant(""))
				}
			})
		})
	})
})
