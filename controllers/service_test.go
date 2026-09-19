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
	"testing"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/mcs-api/pkg/apis/v1beta1"
)

func TestServiceImportOwner(t *testing.T) {
	cases := []struct {
		name string
		refs []metav1.OwnerReference
		want string
	}{
		{
			name: "no owner references",
			refs: nil,
			want: "",
		},
		{
			name: "owned by a ServiceImport",
			refs: []metav1.OwnerReference{
				{APIVersion: v1beta1.GroupVersion.String(), Kind: serviceImportKind, Name: "my-import"},
			},
			want: "my-import",
		},
		{
			name: "owned by something else",
			refs: []metav1.OwnerReference{
				{APIVersion: "v1", Kind: "ReplicaSet", Name: "not-an-import"},
			},
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := serviceImportOwner(c.refs); got != c.want {
				t.Errorf("serviceImportOwner() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestServiceReconcileMissingServiceImport(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1beta1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "derived-svc",
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: v1beta1.GroupVersion.String(), Kind: serviceImportKind, Name: "gone-import"},
			},
		},
		Spec: v1.ServiceSpec{ClusterIPs: []string{"10.0.0.1"}},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(svc).Build()
	r := &ServiceReconciler{Client: cl, Log: logr.Discard()}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "derived-svc"},
	})
	if err != nil {
		t.Fatalf("Reconcile() error = %v, want nil when the owning ServiceImport does not exist", err)
	}
}
