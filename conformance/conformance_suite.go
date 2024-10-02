/*
Copyright 2023 The Kubernetes Authors.

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
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
	mcsclient "sigs.k8s.io/mcs-api/pkg/client/clientset/versioned"
)

type clusterClients struct {
	k8s  kubernetes.Interface
	mcs  mcsclient.Interface
	rest *rest.Config
}

var contexts string
var clients []clusterClients
var loadingRules *clientcmd.ClientConfigLoadingRules
var ctx = context.TODO()

func TestConformance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Conformance Suite")
}

func init() {
	loadingRules = clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	flag.StringVar(&loadingRules.ExplicitPath, "kubeconfig", "", "absolute path(s) to the kubeconfig file(s)")
	flag.StringVar(&contexts, "contexts", "", "comma-separated list of contexts to use")
}

var _ = BeforeSuite(func() {
	Expect(setupClients()).To(Succeed())
})

func setupClients() error {
	splitContexts := strings.Split(contexts, ",")
	clients = make([]clusterClients, len(splitContexts))
	for i, context := range splitContexts {
		overrides := clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
		overrides.CurrentContext = context
		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules, &overrides)
		restConfig, err := clientConfig.ClientConfig()
		if err != nil {
			return err
		}
		k8sClient, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return err
		}
		mcsClient, err := mcsclient.NewForConfig(restConfig)
		if err != nil {
			return err
		}
		clients[i] = clusterClients{k8s: k8sClient, mcs: mcsClient, rest: restConfig}
	}
	return nil
}

type testDriver struct {
	namespace    string
	helloService *corev1.Service
	requestPod   *corev1.Pod
}

func newTestDriver() *testDriver {
	t := &testDriver{}

	BeforeEach(func() {
		t.namespace = fmt.Sprintf("mcs-conformance-%v", rand.Uint32())
		t.helloService = newHelloService()
		t.requestPod = newRequestPod()
	})

	JustBeforeEach(func() {
		Expect(clients).ToNot(BeEmpty())

		// Set up the shared namespace
		for _, client := range clients {
			_, err := client.k8s.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: t.namespace},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		// Set up the remote service (the first cluster is considered to be the remote)
		t.deployHelloService(&clients[0], t.helloService)

		// Start the request pod on all clusters
		for _, client := range clients {
			t.startRequestPod(ctx, client)
		}
	})

	AfterEach(func() {
		// Clean up the shared namespace
		for _, client := range clients {
			err := client.k8s.CoreV1().Namespaces().Delete(ctx, t.namespace, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		}
	})

	return t
}

func (t *testDriver) createServiceExport(c *clusterClients) {
	_, err := c.mcs.MulticlusterV1alpha1().ServiceExports(t.namespace).Create(ctx,
		&v1alpha1.ServiceExport{ObjectMeta: metav1.ObjectMeta{Name: helloServiceName}}, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func (t *testDriver) deployHelloService(c *clusterClients, service *corev1.Service) {
	_, err := c.k8s.AppsV1().Deployments(t.namespace).Create(ctx, newHelloDeployment(), metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	_, err = c.k8s.CoreV1().Services(t.namespace).Create(ctx, service, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func (t *testDriver) awaitServiceImport(c *clusterClients, name string, verify func(*v1alpha1.ServiceImport) bool) *v1alpha1.ServiceImport {
	var serviceImport *v1alpha1.ServiceImport

	_ = wait.PollUntilContextTimeout(ctx, 100*time.Millisecond,
		20*time.Second, true, func(ctx context.Context) (bool, error) {
			defer GinkgoRecover()

			si, err := c.mcs.MulticlusterV1alpha1().ServiceImports(t.namespace).Get(ctx, name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) || errors.Is(err, context.DeadlineExceeded) ||
				(err != nil && strings.Contains(err.Error(), "rate limiter")) {
				return false, nil
			}

			Expect(err).ToNot(HaveOccurred(), "Error retrieving ServiceImport")

			serviceImport = si

			return verify == nil || verify(serviceImport), nil
		})

	return serviceImport
}

func (t *testDriver) awaitServiceExportCondition(c *clusterClients, condType string) {
	Eventually(func() bool {
		se, err := c.mcs.MulticlusterV1alpha1().ServiceExports(t.namespace).Get(ctx, helloServiceName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		return meta.FindStatusCondition(se.Status.Conditions, condType) != nil
	}, 20*time.Second, 100*time.Millisecond).Should(BeTrue(),
		reportNonConformant(fmt.Sprintf("The %s condition was not set", condType)))
}

func (t *testDriver) startRequestPod(ctx context.Context, client clusterClients) {
	_, err := client.k8s.CoreV1().Pods(t.namespace).Create(ctx, t.requestPod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		pod, err := client.k8s.CoreV1().Pods(t.namespace).Get(ctx, t.requestPod.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("pod is not running yet, current status %v", pod.Status.Phase)
		}

		return nil
	}, 20, 1).Should(Succeed())
}

func toMCSPorts(from []corev1.ServicePort) []v1alpha1.ServicePort {
	var mcsPorts []v1alpha1.ServicePort

	for _, port := range from {
		mcsPorts = append(mcsPorts, v1alpha1.ServicePort{
			Name:        port.Name,
			Protocol:    port.Protocol,
			Port:        port.Port,
			AppProtocol: port.AppProtocol,
		})
	}

	return sortMCSPorts(mcsPorts)
}

func sortMCSPorts(p []v1alpha1.ServicePort) []v1alpha1.ServicePort {
	slices.SortFunc(p, func(a, b v1alpha1.ServicePort) int {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return p
}

func requireTwoClusters() {
	if len(clients) < 2 {
		Skip("This test requires at least 2 clusters - skipping")
	}
}
