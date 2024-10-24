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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

var _ = Describe("", func() {
	t := newTestDriver()

	JustBeforeEach(func() {
		t.createServiceExport(&clients[0])
	})

	Specify("Exporting a ClusterIP service should create a ServiceImport of type ClusterSetIP in the service's namespace in each cluster",
		Label(RequiredLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#importing-services")

			for i := range clients {
				serviceImport := t.awaitServiceImport(&clients[i], helloServiceName, nil)
				Expect(serviceImport).NotTo(BeNil(), reportNonConformant(fmt.Sprintf("ServiceImport was not found on cluster %q",
					clients[i].name)))

				Expect(serviceImport.Spec.Type).To(Equal(v1alpha1.ClusterSetIP), reportNonConformant(
					fmt.Sprintf("ServiceImport on cluster %q has type %q", clients[i].name, serviceImport.Spec.Type)))
			}
		})

	Specify("The ports for a ClusterSetIP ServiceImport should match those of the exported service",
		Label(RequiredLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#service-port")

			serviceImport := t.awaitServiceImport(&clients[0], helloServiceName, func(serviceImport *v1alpha1.ServiceImport) bool {
				return len(serviceImport.Spec.Ports) > 0
			})
			Expect(serviceImport).NotTo(BeNil(), "ServiceImport was not found")

			Expect(sortMCSPorts(serviceImport.Spec.Ports)).To(Equal(toMCSPorts(t.helloService.Spec.Ports)), reportNonConformant(""))
		})

	Specify("The SessionAffinity for a ClusterSetIP ServiceImport should match the exported service's SessionAffinity",
		Label(RequiredLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#session-affinity")

			serviceImport := t.awaitServiceImport(&clients[0], helloServiceName, nil)
			Expect(serviceImport).NotTo(BeNil(), "ServiceImport was not found")

			Expect(serviceImport.Spec.SessionAffinity).To(Equal(t.helloService.Spec.SessionAffinity), reportNonConformant(""))

			Expect(serviceImport.Spec.SessionAffinityConfig).To(Equal(t.helloService.Spec.SessionAffinityConfig), reportNonConformant(
				"The SessionAffinityConfig of the ServiceImport does not match the exported Service's SessionAffinityConfig"))
		})

	Specify("An IP should be allocated for a ClusterSetIP ServiceImport",
		Label(RequiredLabel), func() {
			AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#clustersetip")

			serviceImport := t.awaitServiceImport(&clients[0], t.helloService.Name, func(serviceImport *v1alpha1.ServiceImport) bool {
				return len(serviceImport.Spec.IPs) > 0
			})
			Expect(serviceImport).NotTo(BeNil(), "ServiceImport was not found")

			Expect(serviceImport.Spec.IPs).ToNot(BeEmpty(), reportNonConformant(""))
			Expect(net.ParseIP(serviceImport.Spec.IPs[0])).ToNot(BeNil(),
				reportNonConformant(fmt.Sprintf("The value %q is not a valid IP", serviceImport.Spec.IPs[0])))
		})

	Context("A service exported on two clusters", func() {
		var helloService2 *corev1.Service

		BeforeEach(func() {
			requireTwoClusters()

			helloService2 = newHelloService()
		})

		JustBeforeEach(func() {
			// Sleep a little before deploying on the second cluster to ensure the first cluster's ServiceExport timestamp
			// is older so conflict checking is deterministic.
			time.Sleep(100 * time.Millisecond)

			t.deployHelloService(&clients[1], helloService2)
			t.createServiceExport(&clients[1])
		})

		Context("", func() {
			BeforeEach(func() {
				helloService2.Spec.Ports = []corev1.ServicePort{
					t.helloService.Spec.Ports[0],
					{
						Name:     "stcp",
						Port:     142,
						Protocol: corev1.ProtocolSCTP,
					},
				}
			})

			Specify("should expose the union of the constituent service ports", Label(RequiredLabel), func() {
				AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#service-port")

				serviceImport := t.awaitServiceImport(&clients[0], t.helloService.Name, func(serviceImport *v1alpha1.ServiceImport) bool {
					return len(serviceImport.Spec.Ports) == 3
				})
				Expect(serviceImport).NotTo(BeNil(), "ServiceImport was not found")

				Expect(sortMCSPorts(serviceImport.Spec.Ports)).To(Equal(toMCSPorts(
					append(t.helloService.Spec.Ports, helloService2.Spec.Ports[1]))), reportNonConformant(""))
			})
		})

		Context("with conflicting ports", func() {
			BeforeEach(func() {
				helloService2.Spec.Ports = []corev1.ServicePort{t.helloService.Spec.Ports[0]}
				helloService2.Spec.Ports[0].Port = t.helloService.Spec.Ports[0].Port + 1
			})

			Specify("should apply the conflict resolution policy and report a Conflict condition on each ServiceExport",
				Label(RequiredLabel), func() {
					AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#service-port")

					t.awaitServiceExportCondition(&clients[0], v1alpha1.ServiceExportConflict)
					t.awaitServiceExportCondition(&clients[1], v1alpha1.ServiceExportConflict)

					serviceImport := t.awaitServiceImport(&clients[0], t.helloService.Name, func(serviceImport *v1alpha1.ServiceImport) bool {
						return len(serviceImport.Spec.Ports) == len(t.helloService.Spec.Ports)
					})
					Expect(serviceImport).NotTo(BeNil(), "ServiceImport was not found")

					Expect(sortMCSPorts(serviceImport.Spec.Ports)).To(Equal(toMCSPorts(t.helloService.Spec.Ports)),
						reportNonConformant("The service ports were not resolved correctly"))
				})
		})
	})
})
