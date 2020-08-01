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
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

const (
	clusterName = "test-cluster"
)

var (
	cfg             *rest.Config
	k8s             client.Client
	env             *envtest.Environment
	stopCh          chan struct{}
	clusterProvider *cluster.Provider
	testNS          string
)

var _ = BeforeSuite(func(done Done) {
	var err error
	log.SetLogger(zap.LoggerTo(GinkgoWriter, true))
	// Use Kind for a more up-to-date K8s
	clusterProvider = cluster.NewProvider()
	Expect(clusterProvider.Create(clusterName)).To(Succeed())
	Expect(err).ToNot(HaveOccurred())
	kubeconfig, err := clusterProvider.KubeConfig(clusterName, false)
	Expect(err).ToNot(HaveOccurred())

	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	Expect(err).ToNot(HaveOccurred())
	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
	Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(err).ToNot(HaveOccurred())
	existingCluster := true
	env = &envtest.Environment{
		CRDDirectoryPaths:  []string{filepath.Join("..", "..", "config", "crd")},
		UseExistingCluster: &existingCluster,
		Config:             cfg,
	}
	cfg, err = env.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	k8s, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8s).ToNot(BeNil())

	testNS = fmt.Sprintf("test-%v", rand.Uint64())
	Expect(k8s.Create(context.Background(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNS,
		},
	})).To(Succeed())

	opts := ctrl.Options{
		Scheme: scheme,
	}

	stopCh = make(chan struct{})
	go Start(cfg, log.Log, opts, stopCh)
	close(done)
}, 60)

var _ = AfterSuite(func() {
	if stopCh != nil {
		close(stopCh)
	}
	Expect(clusterProvider.Delete(clusterName, "")).To(Succeed())
	err := env.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}
