#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -x
DOCKER_REPO_ROOT="/go/src/github.com/beekhof/rss-operator"
IMAGE=${IMAGE:-"gcr.io/coreos-k8s-scale-testing/codegen"}

docker run --rm \
  -v "$PWD":"$DOCKER_REPO_ROOT" \
  -w "$DOCKER_REPO_ROOT" \
  "$IMAGE" \
  "/go/src/k8s.io/code-generator/generate-groups.sh"  \
  "all" \
  "github.com/beekhof/rss-operator/pkg/generated" \
  "github.com/beekhof/rss-operator/pkg/apis" \
  "galera:v1alpha1" \
  --go-header-file "./hack/k8s/codegen/boilerplate.go.txt" \
  $@
