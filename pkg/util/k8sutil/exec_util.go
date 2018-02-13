// Copyright 2018 The rss-operator Authors
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
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/beekhof/rss-operator/pkg/util"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecOptions passed to ExecWithOptions
type ExecOptions struct {
	Command []string

	Namespace     string
	PodName       string
	ContainerName string

	Stdin         io.Reader
	StdinText     *string
	CaptureStdout bool
	CaptureStderr bool
	// If false, whitespace in std{err,out} will be removed.
	PreserveWhitespace bool

	Timeout time.Duration
}

type ExecContext struct {
	Logger *util.RssLogger
	Cli    *kubernetes.Interface
	Config *rest.Config
}

func GetOutput(pReader *io.PipeReader, result *bytes.Buffer, wg *sync.WaitGroup, tag string) {
	buf := make([]byte, 1024)
	logger := util.GetLogger(tag)
	go func() {
		defer wg.Done()
		for {
			n, err := pReader.Read(buf)
			if n > 0 {
				//logger.Infof("writing %v: %v", n, string(buf[0:n]))
				_, werr := result.Write(buf[0:n])
				if werr == io.EOF {
					logger.Infof("output EOF")
					return
				}
			}

			if err == io.EOF || err == io.ErrClosedPipe {
				//logger.Infof("EOF")
				return
			} else if err != nil {
				logger.Infof("non-EOF read error: %v", err)
				return
			}
			// }()
			// time.AfterFunc(time.Second, func() { ch <- false })
		}
	}()
}

func setupContext(context *ExecContext) error {
	var err error

	if context.Logger == nil {
		context.Logger = util.GetLogger("exec")
	}

	if context.Config == nil {
		// TODO: Kill this code. Make config mandatory
		kubeconfig := ""
		kArg := flag.Lookup("kubeconfig")

		if kArg != nil {
			kubeconfig = kArg.Value.String()
		}

		if kubeconfig != "" {
			context.Logger.Info("Using local config")
			context.Config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)

		} else {
			context.Config, err = InClusterConfig()
		}

		if err != nil {
			return fmt.Errorf("No config available: %v", err)
		}
	}

	if context.Cli == nil {
		cli := MustNewKubeClientFromConfig(context.Config)
		context.Cli = &cli
	}

	return nil
}

// ExecWithOptions executes a command in the specified container,
// returning stdout, stderr and error. `options` allowed for
// additional parameters to be passed.
func ExecWithOptions(context *ExecContext, options ExecOptions) (string, string, error) {
	const tty = false
	var stdout, stderr bytes.Buffer

	err := setupContext(context)
	if err != nil {
		return "", "", err
	}

	context.Config.Timeout = options.Timeout

	cli := *context.Cli
	if options.ContainerName == "" {
		pod, err := cli.CoreV1().Pods(options.Namespace).Get(options.PodName, metav1.GetOptions{})
		if err != nil {
			context.Logger.Errorf("failed to get pod %v: %v", options.PodName, err)
			return "", "", nil
		}
		if len(pod.Spec.Containers) <= 0 {
			context.Logger.Errorf("No containers in %v", options.PodName)
			return "", "", nil
		}
		options.ContainerName = pod.Spec.Containers[0].Name
		context.Logger.Debugf("Executing in container %v", pod.Spec.Containers[0].Name)
	}

	context.Logger.Debugf("ExecWithOptions %+v", options)

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

	// Read/write code from davidvossel/kubevirt/pkg/virtctl/console/console.go

	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	var wg sync.WaitGroup
	wg.Add(3)

	resChan := make(chan error)

	go func() {
		defer wg.Done()
		err := execute("POST", req.URL(), context.Config, options.Stdin, stdoutWriter, stderrWriter, tty)
		resChan <- err
	}()

	GetOutput(stdoutReader, &stdout, &wg, "stdout")
	GetOutput(stderrReader, &stderr, &wg, "stderr")

	// Wait for the result
	err = <-resChan

	// Ensure the output streams are closed and the GetOutput() 'defer' actions are called
	stderrReader.Close()
	stdoutReader.Close()

	// Now wait for IO to complete
	wg.Wait()

	// logger.Infof("out: %v, err: %v, raw: %v", stdout.Len(), stderr.Len(), stdout.String())

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
func ExecCommandInContainer(context *ExecContext, namespace string, podName string, containerName string, cmd ...string) (string, string, error) {
	return ExecWithOptions(context, ExecOptions{
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

func ExecCommandInPod(context *ExecContext,
	namespace string, podName string, cmd ...string) (string, string, error) {
	return ExecCommandInContainer(context, namespace, podName, "", cmd...)
}
