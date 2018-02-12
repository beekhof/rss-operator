package cluster

import (
	"flag"
	"fmt"
	"testing"
)

var podName, namespace, kubeconfig string

func init() {
	flag.StringVar(&podName, "pod", "", "pod to run test against")
	flag.StringVar(&namespace, "namespace", "", "namespace to which the pod belongs")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "kube config path, e.g. $HOME/.kube/config")
}

func TestErrAppend(t *testing.T) {
	errors := []error{}
	if len(errors) != 0 {
		t.Fatal("Initial size if not 0")
	}

	errors = appendNonNil(errors, nil)
	if len(errors) != 0 {
		t.Fatal("nil is not ignored")
	}

	errors = appendNonNil(errors, fmt.Errorf("Some error"))
	if len(errors) != 1 {
		t.Fatal("non-nil is ignored")
	}

	errors = appendNonNil(errors, nil)
	if len(errors) != 1 {
		t.Fatal("second nil is not ignored")
	}

	errors = appendNonNil(errors, fmt.Errorf(""))
	if len(errors) != 1 {
		t.Fatal("empty strings are not ignored")
	}
}
