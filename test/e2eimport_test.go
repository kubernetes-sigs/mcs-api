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

package e2etest

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
	mcsclient "sigs.k8s.io/mcs-api/pkg/client/clientset/versioned"
	"sigs.k8s.io/mcs-api/pkg/controllers"
)

var (
	upscript    = flag.String("up", "", "The path to the script for kind setup")
	kubeconfig1 = flag.String("kubeconfig1", "", "The path to a kubeconfig for cluster 1")
	kubeconfig2 = flag.String("kubeconfig2", "", "The path to a kubeconfig for cluster 2")
	downscript  = flag.String("down", "", "The path to the script for kind teardown")
)

func TestMain(m *testing.M) {
	flag.Parse()
	if *upscript != "" {
		if !filepath.IsAbs(*upscript) {
			*upscript, _ = filepath.Abs(*upscript)
		}
		bash, err := exec.LookPath("bash")
		if err != nil {
			panic(err)
		}
		cmd := exec.Cmd{
			Path:   bash,
			Args:   []string{bash, *upscript},
			Stdout: os.Stderr,
			Stderr: os.Stderr,
		}
		if err := cmd.Run(); err != nil {
			log.Printf("Up failed (continuing incase clusters already exist): %v", err)
		}
		if *kubeconfig1 == "" {
			*kubeconfig1 = path.Join(path.Dir(*upscript), "c1.kubeconfig")
		}
		if *kubeconfig2 == "" {
			*kubeconfig2 = path.Join(path.Dir(*upscript), "c2.kubeconfig")
		}
	}
	code := m.Run()
	if *downscript != "" {
		if !filepath.IsAbs(*downscript) {
			*downscript, _ = filepath.Abs(*downscript)
		}
		bash, err := exec.LookPath("bash")
		if err != nil {
			panic(err)
		}
		cmd := exec.Cmd{
			Path:   bash,
			Args:   []string{bash, *downscript},
			Stdout: os.Stderr,
			Stderr: os.Stderr,
		}
		if err := cmd.Run(); err != nil {
			panic(err)
		}
	}
	os.Exit(code)
}

type clusterClients struct {
	k8s        kubernetes.Interface
	mcs        mcsclient.Interface
	kubeconfig string
}

func (c *clusterClients) apply(ns, config string) error {
	k, err := exec.LookPath("kubectl")
	if err != nil {
		return err
	}
	cmd := exec.Cmd{
		Path:   k,
		Args:   []string{k, "--kubeconfig", c.kubeconfig, "--namespace", ns, "apply", "-f", "-"},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  strings.NewReader(config),
	}
	return cmd.Run()
}

type clusters []clusterClients

func clients(t *testing.T) clusters {
	config1, err := clientcmd.BuildConfigFromFlags("", *kubeconfig1)
	if err != nil {
		t.Fatalf("Failed to load kubeconfig from %v: %v", *kubeconfig1, err)
	}
	config2, err := clientcmd.BuildConfigFromFlags("", *kubeconfig2)
	if err != nil {
		t.Fatalf("Failed to load kubeconfig from %v: %v", *kubeconfig2, err)
	}
	return clusters{
		{
			k8s:        kubernetes.NewForConfigOrDie(config1),
			mcs:        mcsclient.NewForConfigOrDie(config1),
			kubeconfig: *kubeconfig1,
		},
		{
			k8s:        kubernetes.NewForConfigOrDie(config2),
			mcs:        mcsclient.NewForConfigOrDie(config2),
			kubeconfig: *kubeconfig2,
		},
	}
}

func setup(t *testing.T) (clusters, string, func()) {
	c := clients(t)
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		t.Fatal("Failed to generate namespace name:", err)
	}
	name := fmt.Sprintf("test-ns-%s", strings.ToLower(base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)))
	ns := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if _, err := c[0].k8s.CoreV1().Namespaces().Create(context.Background(), &ns, metav1.CreateOptions{}); err != nil {
		t.Fatal("Failed to create namespace:", err)
	}
	if _, err := c[1].k8s.CoreV1().Namespaces().Create(context.Background(), &ns, metav1.CreateOptions{}); err != nil {
		t.Fatal("Failed to create namespace:", err)
	}
	return c, name, func() {
		if err := c[0].k8s.CoreV1().Namespaces().Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
			t.Log("Failed to delete namespace:", err)
		}
		if err := c[1].k8s.CoreV1().Namespaces().Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
			t.Log("Failed to delete namespace:", err)
		}
	}
}

func waitFor(t *testing.T, f func() error) {
	var err error
	timeout := time.Minute
	interval := time.Millisecond * 10
	after := time.After(timeout)
	tick := time.Tick(interval)
	for {
		select {
		case <-tick:
			err = f()
			if err == nil {
				return
			}
		case <-after:
			t.Fatalf("Operation failed to complete after %v: %v", timeout, err)
		}
	}
}

func TestDerivedServiceCreation(t *testing.T) {
	const serviceName = "serve"
	c, ns, cleanup := setup(t)
	defer cleanup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	t.Log(ns, c)
	if err := c[1].apply(ns, service); err != nil {
		t.Fatal("Failed to create Service:", err)
	}
	if err := c[1].apply(ns, deployment); err != nil {
		t.Fatal("Failed to create Deployment:", err)
	}
	if err := c[0].apply(ns, serviceImport); err != nil {
		t.Fatal("Failed to create ServiceImport:", err)
	}

	slices, err := c[1].k8s.DiscoveryV1beta1().EndpointSlices(ns).List(ctx, metav1.ListOptions{LabelSelector: labels.Set{discoveryv1beta1.LabelServiceName: "serve"}.AsSelector().String()})
	if err != nil {
		t.Fatal("Failed to retrieve endpoint slices:", err)
	}
	for _, slice := range slices.Items {
		slice.ObjectMeta = metav1.ObjectMeta{
			Namespace: ns,
			Name:      slice.Name,
			Labels: map[string]string{
				v1alpha1.LabelServiceName: serviceName,
			},
		}
		if _, err := c[0].k8s.DiscoveryV1beta1().EndpointSlices(ns).Create(ctx, &slice, metav1.CreateOptions{}); err != nil {
			t.Fatal("Failed to copy endpoint slice:", err)
		}
	}
	var serviceImport v1alpha1.ServiceImport
	waitFor(t, func() error {
		imp, err := c[0].mcs.MulticlusterV1alpha1().ServiceImports(ns).Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting serviceimport %q: %w", types.NamespacedName{Namespace: ns, Name: serviceName}, err)
		}
		serviceImport = *imp
		return nil
	})
	derivedServiceName := serviceImport.Annotations[controllers.DerivedServiceAnnotation]
	var derivedService v1.Service
	waitFor(t, func() error {
		svc, err := c[0].k8s.CoreV1().Services(ns).Get(ctx, derivedServiceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting service %q: %w", types.NamespacedName{Namespace: ns, Name: derivedServiceName}, err)
		}
		derivedService = *svc
		return nil
	})
	waitFor(t, func() error {
		derivedSlices, err := c[0].k8s.DiscoveryV1beta1().EndpointSlices(ns).List(ctx, metav1.ListOptions{LabelSelector: labels.Set{discoveryv1beta1.LabelServiceName: derivedServiceName}.AsSelector().String()})
		if err != nil {
			return fmt.Errorf("error listing endpoint slices in namespace %q matching selector %q: %w", ns, labels.Set{discoveryv1beta1.LabelServiceName: derivedServiceName}.AsSelector().String(), err)
		}
		if len(derivedSlices.Items) != len(slices.Items) {
			return fmt.Errorf("Not enough slices. got %v, want %v", len(derivedSlices.Items), len(slices.Items))
		}
		return nil
	})
	if derivedService.OwnerReferences[0].UID != serviceImport.UID {
		t.Fatalf("Derived service UID did not mat ServiceImport. got %v, want %v", derivedService.OwnerReferences[0].UID, serviceImport.UID)
	}
	waitFor(t, func() error {
		svc, err := c[0].k8s.CoreV1().Services(ns).Get(ctx, derivedServiceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting service %q: %w", types.NamespacedName{Namespace: ns, Name: derivedServiceName}, err)
		}
		svcImp, err := c[0].mcs.MulticlusterV1alpha1().ServiceImports(ns).Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting serviceimport %q: %w", types.NamespacedName{Namespace: ns, Name: serviceName}, err)
		}
		if len(svcImp.Spec.IPs) != 1 || svc.Spec.ClusterIP != svcImp.Spec.IPs[0] {
			return fmt.Errorf("serviceimport ip not set. got %v, want derived clusterip %v", svcImp.Spec.IPs, svc.Spec.ClusterIP)
		}
		return nil
	})
}
