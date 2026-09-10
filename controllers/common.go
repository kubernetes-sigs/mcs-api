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
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// DerivedServiceAnnotation is set on a ServiceImport to reference the
	// derived Service that represents the imported service for kube-proxy.
	DerivedServiceAnnotation = "multicluster.kubernetes.io/derived-service"
	serviceImportKind        = "ServiceImport"
)

func derivedName(name types.NamespacedName) string {
	hash := sha256.New()
	hash.Write([]byte(name.String()))
	return "derived-" + strings.ToLower(base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(hash.Sum(nil)))[:10]
}

// Start the controllers with the supplied config
func Start(ctx context.Context, cfg *rest.Config, setupLog logr.Logger, opts ctrl.Options) error {
	mgr, err := ctrl.NewManager(cfg, opts)
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	if err = (&ServiceImportReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ServiceImport"),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller %q: %w", "ServiceImport", err)
	}
	if err = (&ServiceReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Service"),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller %q: %w", "Service", err)
	}
	if err = (&EndpointSliceReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("EndpointSlice"),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller %q: %w", "EndpointSlice", err)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}
	return nil
}
