#!/bin/bash

set -o errexit
set -o pipefail

OPERATOR_ROOT=$(dirname "${BASH_SOURCE}")/../..

print_usage() {
  echo "$(basename "$0") - Create Kubernetes RBAC role and role bindings for rss-operator
Usage: $(basename "$0") [options...]
Options:
  --role-name=STRING         Name of ClusterRole to create
                               (default=\"rss-operator\", environment variable: ROLE_NAME)
  --role-binding-name=STRING Name of ClusterRoleBinding to create
                               (default=\"rss-operator\", environment variable: ROLE_BINDING_NAME)
  --namespace=STRING         namespace to create role and role binding in. Must already exist.
                               (default=\"default\", environment vairable: NAMESPACE)
" >&2
}

ROLE_NAME="${ROLE_NAME:-rss-operator}"
ROLE_BINDING_NAME="${ROLE_BINDING_NAME:-rss-operator}"
NAMESPACE="${NAMESPACE:-default}"

while [ 1 = 1 ] ; do
  case "$1" in
       --role-name)
          ROLE_NAME="$2"; shift; shift;;
       --role-binding-name)
          ROLE_BINDING_NAME="$2"; shift; shift;;
       --namespace)
          NAMESPACE="$2"; shift; shift;;
       -h|--help)
          print_usage
          exit 0
          ;;
       --) shift; break;;
       "") break;;
       *)
          echo "Bad arg: $1"
          print_usage
          exit 1
          ;;
  esac
done

echo "Creating role with ROLE_NAME=${ROLE_NAME}, NAMESPACE=${NAMESPACE}"
sed -e "s/<ROLE_NAME>/${ROLE_NAME}/g" \
  -e "s/<NAMESPACE>/${NAMESPACE}/g" \
  "${OPERATOR_ROOT}/example/rbac/cluster-role-template.yaml" | \
  kubectl create -f -

echo "Creating role binding with ROLE_NAME=${ROLE_NAME}, ROLE_BINDING_NAME=${ROLE_BINDING_NAME}, NAMESPACE=${NAMESPACE}"
sed -e "s/<ROLE_NAME>/${ROLE_NAME}/g" \
  -e "s/<ROLE_BINDING_NAME>/${ROLE_BINDING_NAME}/g" \
  -e "s/<NAMESPACE>/${NAMESPACE}/g" \
  "${OPERATOR_ROOT}/example/rbac/cluster-role-binding-template.yaml" | \
  kubectl create -f -
