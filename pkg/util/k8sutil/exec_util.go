// Copyright 2016 The etcd-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8sutil

import (
	"bytes"
	"io"
	"net/url"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecOptions passed to ExecWithOptions
type ExecOptions struct {
	Command []string

	Namespace     string
	PodName       string
	ContainerName string

	Stdin         io.Reader
	CaptureStdout bool
	CaptureStderr bool
	// If false, whitespace in std{err,out} will be removed.
	PreserveWhitespace bool
}

// ExecWithOptions executes a command in the specified container,
// returning stdout, stderr and error. `options` allowed for
// additional parameters to be passed.
func ExecWithOptions(logger *logrus.Entry, cli kubernetes.Interface, options ExecOptions) (string, string, error) {
	logger.Infof("ExecWithOptions %+v", options)

	const tty = false
	config, _ := InClusterConfig()

	// // restClient := f.KubeClient.CoreV1().RESTClient()
	// restClient, err := restclient.RESTClientFor(config)
	// if err != nil {
	// 	return "", "", err
	// }

	req := cli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.PodName).
		Namespace(options.Namespace).
		SubResource("exec").
		Param("container", options.ContainerName)
	req.VersionedParams(&v1.PodExecOptions{
		Container: options.ContainerName,
		Command:   options.Command,
		Stdin:     options.Stdin != nil,
		Stdout:    options.CaptureStdout,
		Stderr:    options.CaptureStderr,
		TTY:       tty,
	}, scheme.ParameterCodec)

	var err error
	var stdout, stderr bytes.Buffer

	// Read/write code from davidvossel/kubevirt/pkg/virtctl/console/console.go

	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	var wg sync.WaitGroup
	wg.Add(2)

	resChan := make(chan error)
	readOutStop := make(chan struct{})
	readErrStop := make(chan struct{})

	go func() {
		err := execute("POST", req.URL(), config, options.Stdin, stdoutWriter, stderrWriter, tty)
		resChan <- err
	}()

	go func() {
		defer close(readOutStop)
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := stdoutReader.Read(buf)
			if err != nil && err != io.EOF {
				logger.Infof("non-EOF stdout read error: %v", err)
				return
			}
			if n == 0 && err == io.EOF {
				return
			}
			_, err = stdout.Write(buf[0:n])
			if err == io.EOF {
				return
			}
		}
	}()
	go func() {
		defer close(readErrStop)
		defer wg.Done()

		buf := make([]byte, 1024)
		for {
			n, err := stderrReader.Read(buf)
			if err != nil && err != io.EOF {
				logger.Infof("non-EOF stderr read error: %v", err)
				return
			}
			if n == 0 && err == io.EOF {
				return
			}
			_, err = stderr.Write(buf[0:n])
			if err == io.EOF {
				return
			}
		}
	}()

	logger.Infof("Waiting")
	wg.Wait()
	logger.Infof("WG all done")

	// Wait for them all to finish
	<-readOutStop
	<-readErrStop
	err = <-resChan

	logger.Infof("Have outputs")

	// logger.Infof("out: %v, err: %v", stdout.String(), stderr.String())

	if options.PreserveWhitespace {
		return stdout.String(), stderr.String(), err
	}
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}

// ExecCommandInContainerWithFullOutput executes a command in the
// specified container and return stdout, stderr and error
func ExecCommandInContainerWithFullOutput(logger *logrus.Entry, cli kubernetes.Interface, namespace string, podName string, containerName string, cmd ...string) (string, string, error) {
	return ExecWithOptions(logger, cli, ExecOptions{
		Command:       cmd,
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: containerName,

		Stdin:              nil,
		CaptureStdout:      true,
		CaptureStderr:      true,
		PreserveWhitespace: false,
	})
}

// ExecCommandInContainer executes a command in the specified container.
func ExecCommandInContainer(logger *logrus.Entry, cli kubernetes.Interface, namespace string, podName string, containerName string, cmd ...string) string {
	stdout, stderr, err := ExecCommandInContainerWithFullOutput(logger, cli, namespace, podName, containerName, cmd...)

	logger.Infof("Exec stderr: %q", stderr)
	if err != nil {
		logger.Errorf("failed to execute command in pod %v, container %v: %v",
			podName, containerName, err)
	}

	return stdout
}

func ExecCommandInPod(logger *logrus.Entry, cli kubernetes.Interface, namespace string, podName string, cmd ...string) string {
	pod, err := cli.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		logger.Errorf("failed to get pod %v: %v", podName, err)
	}
	if len(pod.Spec.Containers) <= 0 {
		logger.Errorf("No containers in %v", podName)
		return ""
	}
	return ExecCommandInContainer(logger, cli, namespace, podName, pod.Spec.Containers[0].Name, cmd...)
}

func ExecCommandInPodWithFullOutput(logger *logrus.Entry, cli kubernetes.Interface,
	namespace string, podName string, cmd ...string) (string, string, error) {
	pod, err := cli.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		logger.Errorf("failed to get pod %v: %v", podName, err)
	}
	if len(pod.Spec.Containers) <= 0 {
		logger.Errorf("No containers in %v", podName)
		return "", "", nil
	}
	return ExecCommandInContainerWithFullOutput(logger, cli, namespace, podName, pod.Spec.Containers[0].Name, cmd...)
}
