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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/mcs-api/pkg/apis/v1beta1"
)

var _ = Describe("ServiceImport", func() {
	var (
		serviceImport      v1beta1.ServiceImport
		serviceName        types.NamespacedName
		derivedServiceName types.NamespacedName
	)
	ctx := context.Background()
	Context("should be ignored", func() {
		Specify("when headless", func() {
			Expect(shouldIgnoreImport(&v1beta1.ServiceImport{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNS,
					Name:      "headless",
				},
				Spec: v1beta1.ServiceImportSpec{
					Type: v1beta1.Headless,
					Ports: []v1beta1.ServicePort{
						{Port: 80},
					},
				},
			})).To(BeTrue())
		})
		Specify("when deleted", func() {
			Expect(shouldIgnoreImport(&v1beta1.ServiceImport{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         testNS,
					Name:              "deleted",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: v1beta1.ServiceImportSpec{
					Type: v1beta1.ClusterSetIP,
					Ports: []v1beta1.ServicePort{
						{Port: 80},
					},
				},
			})).To(BeTrue())
		})
	})
	Context("created", func() {
		BeforeEach(func() {
			serviceName = types.NamespacedName{Namespace: testNS, Name: fmt.Sprintf("svc-%v", rand.Uint64())}
			derivedServiceName = types.NamespacedName{Namespace: testNS, Name: derivedName(serviceName)}
			serviceImport = v1beta1.ServiceImport{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNS,
					Name:      serviceName.Name,
				},
				Spec: v1beta1.ServiceImportSpec{
					Type: v1beta1.ClusterSetIP,
					Ports: []v1beta1.ServicePort{
						{Port: 80},
					},
				},
			}
			Expect(k8s.Create(ctx, &serviceImport)).To(Succeed())
		})
		It("has derived service annotation", func() {
			Eventually(func() string {
				var s v1beta1.ServiceImport
				Expect(k8s.Get(ctx, serviceName, &s)).To(Succeed())
				return s.Annotations[DerivedServiceAnnotation]
			}, 10).Should(Equal(derivedName(serviceName)))
		}, 10)
		It("has derived service IP", func() {
			var s v1beta1.ServiceImport
			Eventually(func() string {
				Expect(k8s.Get(ctx, serviceName, &s)).To(Succeed())
				if len(s.Spec.IPs) > 0 {
					return s.Spec.IPs[0]
				}
				return ""
			}, 10).ShouldNot(BeEmpty())
		}, 15)
		It("created derived service", func() {
			var s v1.Service
			Eventually(func() error {
				return k8s.Get(ctx, derivedServiceName, &s)
			}, 10).Should(Succeed())
			Expect(len(s.OwnerReferences)).To(Equal(1))
			Expect(s.OwnerReferences[0].UID).To(Equal(serviceImport.UID))
		}, 15)
		It("removes derived service", func() {
			var s v1.Service
			Eventually(func() error {
				return k8s.Get(ctx, derivedServiceName, &s)
			}, 10).Should(Succeed())
			var imp v1beta1.ServiceImport
			Expect(k8s.Get(ctx, serviceName, &imp)).To(Succeed())
			Expect(k8s.Delete(ctx, &imp)).To(Succeed())
			Eventually(func() error {
				return k8s.Get(ctx, derivedServiceName, &s)
			}, 15).ShouldNot(Succeed())
		}, 15)
	})
	Context("created with IP", func() {
		BeforeEach(func() {
			serviceName = types.NamespacedName{Namespace: testNS, Name: fmt.Sprintf("svc-%v", rand.Uint64())}
			derivedServiceName = types.NamespacedName{Namespace: testNS, Name: derivedName(serviceName)}
			serviceImport = v1beta1.ServiceImport{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNS,
					Name:      serviceName.Name,
				},
				Spec: v1beta1.ServiceImportSpec{
					Type: v1beta1.ClusterSetIP,
					Ports: []v1beta1.ServicePort{
						{Port: 80},
					},
				},
			}
			Expect(k8s.Create(ctx, &serviceImport)).To(Succeed())
		})
		It("updates derived service IP", func() {
			var svcImport v1beta1.ServiceImport
			var s v1.Service
			Eventually(func() error {
				return k8s.Get(ctx, derivedServiceName, &s)
			}, 10).Should(Succeed())
			Eventually(func() string {
				Expect(k8s.Get(ctx, serviceName, &svcImport)).To(Succeed())
				if len(svcImport.Spec.IPs) > 0 {
					return svcImport.Spec.IPs[0]
				}
				return ""
			}, 10).Should(Equal(s.Spec.ClusterIP))
		}, 15)
	})
	Context("created with existing clustersetIP", func() {
		BeforeEach(func() {
			serviceName = types.NamespacedName{Namespace: testNS, Name: fmt.Sprintf("svc-%v", rand.Uint64())}
			derivedServiceName = types.NamespacedName{Namespace: testNS, Name: derivedName(serviceName)}
			serviceImport = v1beta1.ServiceImport{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNS,
					Name:      serviceName.Name,
				},
				Spec: v1beta1.ServiceImportSpec{
					Type: v1beta1.ClusterSetIP,
					Ports: []v1beta1.ServicePort{
						{Port: 80},
					},
					IPs: []string{"10.42.42.42"},
				},
			}
			Expect(k8s.Create(ctx, &serviceImport)).To(Succeed())
		})
		It("updates service loadbalancer status with service import IPs", func() {
			var svcImport v1beta1.ServiceImport
			var s v1.Service
			Eventually(func() error {
				return k8s.Get(ctx, derivedServiceName, &s)
			}, 10).Should(Succeed())
			Eventually(func() string {
				Expect(k8s.Get(ctx, serviceName, &svcImport)).To(Succeed())
				if len(svcImport.Spec.IPs) > 0 {
					return svcImport.Spec.IPs[0]
				}
				return ""
			}, 10).Should(Equal(s.Status.LoadBalancer.Ingress[0].IP))
		}, 15)
	})
})
