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
	"crypto/sha256"
	"encoding/base64"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

// ServiceImportReconciler reconciles a ServiceImport object
type ServiceImportReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=serviceimports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=serviceimports/status,verbs=get;update;patch

func derivedName(name types.NamespacedName) string {
	hash := sha256.New()
	return "import-" + base64.RawURLEncoding.EncodeToString(hash.Sum([]byte(name.String())))[:10]
}

// Reconcile the changes.
func (r *ServiceImportReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	serviceName := derivedName(req.NamespacedName)
	r.Log.WithValues("serviceimport", req.NamespacedName, "derived", serviceName)
	return ctrl.Result{}, nil
}

// SetupWithManager wires up the controller.
func (r *ServiceImportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.ServiceImport{}).Complete(r)
}
