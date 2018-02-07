package k8sutil

import (
	"flag"
	"testing"

	"github.com/beekhof/rss-operator/pkg/util"
	"github.com/sirupsen/logrus"

	"k8s.io/client-go/tools/clientcmd"
)

var podName, namespace, kubeconfig string

func init() {
	flag.StringVar(&podName, "pod", "", "pod to run test against")
	flag.StringVar(&namespace, "namespace", "", "namespace to which the pod belongs")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "kube config path, e.g. $HOME/.kube/config")
}

func TestPodExec(t *testing.T) {
	logger := util.GetLogger("test")
	flag.Parse()

	if namespace == "" || podName == "" {
		t.SkipNow()
	}
	t.Log("building config from", kubeconfig)
	logrus.Info("building config from", kubeconfig)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Skip("error building config")
	}

	cli := MustNewKubeClientFromConfig(config)
	// if err != nil {
	// 	t.Skip("error creating cli")
	// }

	input := "hi there"
	context := ExecContext{Logger: logger, Config: config, Cli: &cli}
	stdout, stderr, err := ExecWithOptions(&context, ExecOptions{
		Command:       []string{"ls", "-al"},
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: "",

		Stdin:              nil,
		StdinText:          &input,
		CaptureStdout:      true,
		CaptureStderr:      true,
		PreserveWhitespace: false,
	})
	util.LogOutput(logger, logrus.InfoLevel, "stdout", stdout)
	util.LogOutput(logger, logrus.InfoLevel, "stderr", stderr)
	t.Logf("Result: %v", err)
}
