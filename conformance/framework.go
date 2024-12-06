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
	"bytes"
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type (
	doOperationFunc func() (interface{}, error)
	checkResultFunc func(result interface{}) (bool, string, error)
)

// AwaitUntil repeats the given operation until the given checkResultFunc returns true, an error, or a
// timeout is reached; it then verifies that no error occurred.
func AwaitUntil(opMsg string, doOperation doOperationFunc, checkResult checkResultFunc) interface{} {
	result, errMsg, err := AwaitResultOrError(opMsg, doOperation, checkResult)
	Expect(err).NotTo(HaveOccurred(), errMsg)

	return result
}

// AwaitResultOrError repeats the given operation until the given checkResultFunc returns true, an error, or a
// timeout is reached. Transient errors returned by the operation itself are ignored.
func AwaitResultOrError(opMsg string, doOperation doOperationFunc, checkResult checkResultFunc) (interface{}, string, error) {
	var finalResult interface{}
	var lastMsg string

	err := wait.PollUntilContextTimeout(context.Background(), 500*time.Millisecond,
		10*time.Second, true, func(_ context.Context) (bool, error) {
			result, err := doOperation()
			if err != nil {
				if isTransientError(err, opMsg) {
					return false, nil
				}
				return false, err
			}

			ok, msg, err := checkResult(result)
			if err != nil {
				return false, err
			}

			if ok {
				finalResult = result
				return true, nil
			}

			lastMsg = msg
			return false, nil
		})

	errMsg := ""
	if err != nil {
		errMsg = "Failed to " + opMsg
		if lastMsg != "" {
			errMsg += ". " + lastMsg
		}
	}

	return finalResult, errMsg, err
}

// isTransientError identifies API errors which could be considered transient/recoverable
// due to server state.
func isTransientError(err error, opMsg string) bool {
	if errors.IsInternalError(err) ||
		errors.IsServerTimeout(err) ||
		errors.IsTimeout(err) ||
		errors.IsServiceUnavailable(err) ||
		errors.IsUnexpectedServerError(err) ||
		errors.IsTooManyRequests(err) {
		fmt.Fprintf(GinkgoWriter, "Transient failure when attempting to %s: %v", opMsg, err)
		return true
	}

	return false
}

func execCmd(k8s kubernetes.Interface, config *rest.Config, podName string, podNamespace string, command []string) ([]byte, []byte, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return []byte{}, []byte{}, err
	}

	return stdout.Bytes(), stderr.Bytes(), nil
}
