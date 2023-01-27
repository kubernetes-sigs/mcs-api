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

package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

var _ = Describe("EndpointSlice", func() {
	ctx := context.Background()
	Context("should be ignored", func() {
		Specify("when not multi-cluster", func() {
			Expect(shouldIgnoreEndpointSlice(&v1beta1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNS,
					Name:      "no-mc-service",
				},
				AddressType: v1beta1.AddressTypeIPv4,
			})).To(BeTrue())
		})
		Specify("when deleted", func() {
			Expect(shouldIgnoreEndpointSlice(&v1beta1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         testNS,
					Name:              "deleted",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				AddressType: v1beta1.AddressTypeIPv4,
			})).To(BeTrue())
		})
	})
	Context("created with mc label", func() {
		var (
			serviceName        types.NamespacedName
			derivedServiceName types.NamespacedName
			sliceName          types.NamespacedName
			epSlice            v1beta1.EndpointSlice
		)
		BeforeEach(func() {
			serviceName = types.NamespacedName{Namespace: testNS, Name: fmt.Sprintf("svc-%v", rand.Uint64())}
			derivedServiceName = types.NamespacedName{Namespace: testNS, Name: derivedName(serviceName)}
			sliceName = types.NamespacedName{Namespace: testNS, Name: fmt.Sprintf("slice-%v", rand.Uint64())}
			epSlice = v1beta1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNS,
					Name:      sliceName.Name,
					Labels: map[string]string{
						v1alpha1.LabelServiceName: serviceName.Name,
					},
				},
				AddressType: v1beta1.AddressTypeIPv4,
			}
			Expect(k8s.Create(ctx, &epSlice)).To(Succeed())
		})
		It("has correct label", func() {
			Eventually(func() string {
				var eps v1beta1.EndpointSlice
				Expect(k8s.Get(ctx, sliceName, &eps)).Should(Succeed())
				return eps.Labels[v1beta1.LabelServiceName]
			}).Should(Equal(derivedServiceName.Name))
		})
	})
	Context("created with wrong label", func() {
		var (
			serviceName        types.NamespacedName
			derivedServiceName types.NamespacedName
			sliceName          types.NamespacedName
			epSlice            v1beta1.EndpointSlice
		)
		BeforeEach(func() {
			serviceName = types.NamespacedName{Namespace: testNS, Name: fmt.Sprintf("svc-%v", rand.Uint64())}
			derivedServiceName = types.NamespacedName{Namespace: testNS, Name: derivedName(serviceName)}
			sliceName = types.NamespacedName{Namespace: testNS, Name: fmt.Sprintf("slice-%v", rand.Uint64())}
			epSlice = v1beta1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNS,
					Name:      sliceName.Name,
					Labels: map[string]string{
						v1alpha1.LabelServiceName: serviceName.Name,
						v1beta1.LabelServiceName:  serviceName.Name,
					},
				},
				AddressType: v1beta1.AddressTypeIPv4,
			}
			Expect(k8s.Create(ctx, &epSlice)).To(Succeed())
		})
		It("has correct label", func() {
			Eventually(func() string {
				var eps v1beta1.EndpointSlice
				Expect(k8s.Get(ctx, sliceName, &eps)).Should(Succeed())
				return eps.Labels[v1beta1.LabelServiceName]
			}).Should(Equal(derivedServiceName.Name))
		})
	})
})
