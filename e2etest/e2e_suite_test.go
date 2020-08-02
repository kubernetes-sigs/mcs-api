package e2etest

import (
	"context"
	"flag"
	"io/ioutil"
	"math/rand"
	"os"
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

	cluster1 clusterClients
	cluster2 clusterClients
)

type clusterClients struct {
	k8s kubernetes.Interface
	mcs mcsclient.Interface
}

func podLogs(ctx context.Context, k8s kubernetes.Interface, namespace, name string) (string, error) {
	logRequest := k8s.CoreV1().Pods(namespace).GetLogs(name, &v1.PodLogOptions{})
	logs, err := logRequest.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer logs.Close()
	data, err := ioutil.ReadAll(logs)
	if err != nil {
		return "", err
	}
	return string(data), nil
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
