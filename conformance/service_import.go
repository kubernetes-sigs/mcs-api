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

package conformance

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

var _ = Describe("", func() {
	t := newTestDriver()

	BeforeEach(func() {
		t.createServiceExport()
	})

	Specify("Exporting a ClusterIP service should create a ServiceImport of type ClusterSetIP in the service's namespace in each cluster",
		Label(RequiredLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#importing-services")

			for i := range clients {
				serviceImport := t.awaitServiceImport(&clients[i], helloService.Name, nil)
				Expect(serviceImport).NotTo(BeNil(), reportNonConformant(fmt.Sprintf("ServiceImport was not found on cluster %d", i+1)))

				Expect(serviceImport.Spec.Type).To(Equal(v1alpha1.ClusterSetIP), reportNonConformant(
					fmt.Sprintf("ServiceImport on cluster %d has type %q", i+1, serviceImport.Spec.Type)))
			}
		})

	Specify("The ports for a ClusterSetIP ServiceImport should match those of the exported service",
		Label(RequiredLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#service-port")

			serviceImport := t.awaitServiceImport(&clients[0], helloService.Name, func(serviceImport *v1alpha1.ServiceImport) bool {
				return len(serviceImport.Spec.Ports) > 0
			})
			Expect(serviceImport).NotTo(BeNil(), "ServiceImport was not found")

			Expect(sortMCSPorts(serviceImport.Spec.Ports)).To(Equal(toMCSPorts(helloService.Spec.Ports)), reportNonConformant(""))
		})

	Specify("The SessionAffinity for a ClusterSetIP ServiceImport should match the exported service's SessionAffinity",
		Label(RequiredLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#session-affinity")

			serviceImport := t.awaitServiceImport(&clients[0], helloService.Name, nil)
			Expect(serviceImport).NotTo(BeNil(), "ServiceImport was not found")

			Expect(serviceImport.Spec.SessionAffinity).To(Equal(helloService.Spec.SessionAffinity), reportNonConformant(""))

			Expect(serviceImport.Spec.SessionAffinityConfig).To(Equal(helloService.Spec.SessionAffinityConfig), reportNonConformant(
				"The SessionAffinityConfig of the ServiceImport does not match the exported Service's SessionAffinityConfig"))
		})

	Specify("An IP should be allocated for a ClusterSetIP ServiceImport",
		Label(RequiredLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#clustersetip")

			serviceImport := t.awaitServiceImport(&clients[0], helloService.Name, func(serviceImport *v1alpha1.ServiceImport) bool {
				return len(serviceImport.Spec.IPs) > 0
			})
			Expect(serviceImport).NotTo(BeNil(), "ServiceImport was not found")

			Expect(serviceImport.Spec.IPs).ToNot(BeEmpty(), reportNonConformant(""))
			Expect(net.ParseIP(serviceImport.Spec.IPs[0])).ToNot(BeNil(),
				reportNonConformant(fmt.Sprintf("The value %q is not a valid IP", serviceImport.Spec.IPs[0])))
		})
})
