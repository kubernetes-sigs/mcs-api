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
	"flag"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"math/rand"
	"os"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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

//func podLogs(ctx context.Context, k8s kubernetes.Interface, namespace, name string) (string, error) {
//	logRequest := k8s.CoreV1().Pods(namespace).GetLogs(name, &v1.PodLogOptions{})
//	logs, err := logRequest.Stream(ctx)
//	if err != nil {
//		return "", err
//	}
//	defer logs.Close()
//	data, err := ioutil.ReadAll(logs)
//	if err != nil {
//		return "", err
//	}
//	return string(data), nil
//}

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
