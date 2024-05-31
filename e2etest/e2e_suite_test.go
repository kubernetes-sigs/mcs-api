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
	"bytes"
	"context"
	"flag"
	"math/rand"
	"os"
	"strconv"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
	mcsclient "sigs.k8s.io/mcs-api/pkg/client/clientset/versioned"
)

var (
	kubeconfig1 = flag.String("kubeconfig1", os.Getenv("KUBECONFIG1"), "The path to a kubeconfig for cluster 1")
	kubeconfig2 = flag.String("kubeconfig2", os.Getenv("KUBECONFIG2"), "The path to a kubeconfig for cluster 2")
	noTearDown  = flag.Bool("no-tear-down", tryParseBool(os.Getenv("NO_TEAR_DOWN")), "Don't tear down after test (useful for debugging failures).")
	cluster1    clusterClients
	cluster2    clusterClients
	restcfg1, _ = clientcmd.BuildConfigFromFlags("", *kubeconfig1)
	//restcfg2, _ = clientcmd.BuildConfigFromFlags("", *kubeconfig2)
)

func tryParseBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

type clusterClients struct {
	k8s kubernetes.Interface
	mcs mcsclient.Interface
}

func TestE2E(t *testing.T) {
	flag.Parse()
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	rand.Seed(GinkgoRandomSeed())

	Expect(*kubeconfig1).ToNot(BeEmpty(), "either --kubeconfig1 or KUBECONFIG1 must be set")
	Expect(*kubeconfig2).ToNot(BeEmpty(), "either --kubeconfig2 or KUBECONFIG2 must be set")

	restcfg1, err := clientcmd.BuildConfigFromFlags("", *kubeconfig1)
	Expect(err).ToNot(HaveOccurred())
	restcfg2, err := clientcmd.BuildConfigFromFlags("", *kubeconfig2)
	Expect(err).ToNot(HaveOccurred())

	cluster1 = clusterClients{
		k8s: kubernetes.NewForConfigOrDie(restcfg1),
		mcs: mcsclient.NewForConfigOrDie(restcfg1),
	}
	cluster2 = clusterClients{
		k8s: kubernetes.NewForConfigOrDie(restcfg2),
		mcs: mcsclient.NewForConfigOrDie(restcfg2),
	}
})

func execCmd(k8s kubernetes.Interface, config *restclient.Config, podName string, podNamespace string, command []string) ([]byte, []byte, error) {
	req := k8s.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace(podNamespace).SubResource("exec")
	req.VersionedParams(&v1.PodExecOptions{
		Command: command,
		Stdin:   false,
		Stdout:  true,
		Stderr:  true,
		TTY:     true,
	}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return []byte{}, []byte{}, err
	}
	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return []byte{}, []byte{}, err
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}

func exportService(ctx context.Context, fromCluster, toCluster clusterClients, namespace string, svcName string) {
	_, err := fromCluster.mcs.MulticlusterV1alpha1().ServiceExports(namespace).Create(ctx, &v1alpha1.ServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Name: svcName,
		},
	}, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	var slices *discoveryv1.EndpointSliceList
	Eventually(func() int {
		eps := 0
		slices, err = fromCluster.k8s.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labels.Set{discoveryv1.LabelServiceName: svcName}.AsSelector().String(),
		})
		Expect(err).ToNot(HaveOccurred())
		for _, s := range slices.Items {
			eps += len(s.Endpoints)
		}
		return eps
	}, 30).Should(Equal(1))
	importedSlice := slices.Items[0] // This direct indexing is ok because we just asserted above that there is exactly one element here
	importedSlice.ObjectMeta = metav1.ObjectMeta{
		GenerateName: svcName + "-",
		Labels: map[string]string{
			v1alpha1.LabelServiceName: svcName,
		},
	}

	createdSlice, err := toCluster.k8s.DiscoveryV1().EndpointSlices(namespace).Create(ctx, &importedSlice, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() string {
		updatedSlice, err := toCluster.k8s.DiscoveryV1().EndpointSlices(namespace).Get(ctx, createdSlice.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return updatedSlice.Labels[discoveryv1.LabelServiceName]
	}).ShouldNot(BeEmpty())
}
