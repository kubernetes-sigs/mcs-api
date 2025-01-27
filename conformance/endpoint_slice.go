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
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

// K8sEndpointSliceManagedByName is the name used for endpoint slices managed by the Kubernetes controller
const K8sEndpointSliceManagedByName = "endpointslice-controller.k8s.io"

// ServiceExportEndpointSliceHook is a hook meant to customize the ServiceExport
// resource before its creation in EndpointSlice related tests to integrate with
// MCS-API implementations requiring user to explicitly opt-in to enable
// EndpointSlice synchronization from remote clusters.
var ServiceExportEndpointSliceHook = func(serviceExport *v1alpha1.ServiceExport) *v1alpha1.ServiceExport {
	return serviceExport
}

// ServiceEndpointSliceHook is a hook meant to customize the Service
// resource in EndpointSlice related tests to integrate with MCS-API implementations
// requiring user to explicitly opt-in to enable EndpointSlice synchronization from
// remote clusters.
var ServiceEndpointSliceHook func(service *corev1.Service) *corev1.Service

var _ = Describe("", Label(OptionalLabel, EndpointSliceLabel), func() {
	t := newTestDriver()

	JustBeforeEach(func() {
		if ServiceEndpointSliceHook != nil {
			Expect(
				clients[0].k8s.CoreV1().Services(t.helloService.Namespace).Update(
					context.TODO(), ServiceEndpointSliceHook(t.helloService), metav1.UpdateOptions{},
				),
			).ToNot(HaveOccurred(), "Error Updating Services in EndpointSlice hook")
		}
		t.createServiceExport(&clients[0], ServiceExportEndpointSliceHook(newHelloServiceExport()))
	})

	Specify("Exporting a service should create an MCS EndpointSlice in the service's namespace in each cluster with the "+
		"required MCS labels. Unexporting should delete the EndpointSlice.", func() {
		AddReportEntry(SpecRefReportEntry, "https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#using-endpointslice-objects-to-track-endpoints")

		endpointSlices := make([]*discoveryv1.EndpointSlice, len(clients))

		for i, client := range clients {
			eps := t.awaitMCSEndpointSlice(&client)
			Expect(eps).ToNot(BeNil(), reportNonConformant(fmt.Sprintf(
				"an MCS EndpointSlice was not found on cluster %q. An MCS EndpointSlice is identified by the presence "+
					"of at least one of the required MCS labels, whose names are prefixed with \"multicluster.kubernetes.io\". "+
					"If the MCS implementation does not use MCS EndpointSlices, you can specify a Ginkgo label filter using "+
					"the %q label where appropriate to skip this test.",
				client.name, EndpointSliceLabel)))

			endpointSlices[i] = eps

			Expect(eps.Labels).To(HaveKeyWithValue(v1alpha1.LabelServiceName, t.helloService.Name),
				reportNonConformant(fmt.Sprintf("the MCS EndpointSlice %q does not contain the %q label referencing the service name",
					eps.Name, v1alpha1.LabelServiceName)))

			Expect(eps.Labels).To(HaveKey(v1alpha1.LabelSourceCluster),
				reportNonConformant(fmt.Sprintf("the MCS EndpointSlice %q does not contain the %q label",
					eps.Name, v1alpha1.LabelSourceCluster)))

			Expect(eps.Labels).To(HaveKey(discoveryv1.LabelManagedBy),
				reportNonConformant(fmt.Sprintf("the MCS EndpointSlice %q does not contain the %q label",
					eps.Name, discoveryv1.LabelManagedBy)))

			if !skipVerifyEndpointSliceManagedBy {
				Expect(eps.Labels[discoveryv1.LabelManagedBy]).ToNot(Equal(K8sEndpointSliceManagedByName),
					reportNonConformant(fmt.Sprintf("the MCS EndpointSlice's %q label must not reference %q",
						discoveryv1.LabelManagedBy, K8sEndpointSliceManagedByName)))
			}
		}

		By("Unexporting the service")

		t.deleteServiceExport(&clients[0])

		for i, client := range clients {
			Eventually(func() bool {
				_, err := client.k8s.DiscoveryV1().EndpointSlices(t.namespace).Get(ctx, endpointSlices[i].Name, metav1.GetOptions{})
				return apierrors.IsNotFound(err)
			}, 20*time.Second, 100*time.Millisecond).Should(BeTrue(),
				reportNonConformant(fmt.Sprintf("the EndpointSlice was not deleted on unexport from cluster %d", i+1)))
		}
	})
})

func (t *testDriver) awaitMCSEndpointSlice(c *clusterClients) *discoveryv1.EndpointSlice {
	var endpointSlice *discoveryv1.EndpointSlice

	hasLabel := func(eps *discoveryv1.EndpointSlice, label string) bool {
		_, exists := eps.Labels[label]
		return exists
	}

	_ = wait.PollUntilContextTimeout(ctx, 100*time.Millisecond,
		20*time.Second, true, func(ctx context.Context) (bool, error) {
			defer GinkgoRecover()

			list, err := c.k8s.DiscoveryV1().EndpointSlices(t.namespace).List(ctx, metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred(), "Error retrieving EndpointSlices")

			for i := range list.Items {
				eps := &list.Items[i]

				if hasLabel(eps, v1alpha1.LabelServiceName) || hasLabel(eps, v1alpha1.LabelSourceCluster) {
					endpointSlice = eps
					return true, nil
				}
			}

			return false, nil
		})

	return endpointSlice
}
