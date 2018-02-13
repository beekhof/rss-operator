# Operation Guide

## Setup RBAC

Set up basic [RBAC rules](./rbac.md) for the replication operator:

```bash
$ example/rbac/create_role.sh
```

## Install the replciation operator

Create a deployment for the replication operator:

```bash
$ kubectl create -f example/operator.yaml
```

The replication operator will automatically create a Kubernetes Custom Resource Definition (CRD):

```bash
$ kubectl get customresourcedefinitions
NAME                                     AGE
replicatedstatefulsets.clusterlabs.org   16m
```

## Uninstall the replciation operator

Note that the clusters managed by the replciation operator will **NOT** be deleted even if the operator is uninstalled.
This is an intentional design to prevent accidental operator failure from killing all the clusters.
In order to delete all clusters, delete all cluster CR objects before uninstall the operator.

Cleanup the replciation operator:

```bash
kubectl delete -f example/operator.yaml
kubectl delete rss --all
kubectl delete endpoints rss-operator
kubectl delete clusterrole rss-operator
kubectl delete clusterrolebinding rss-operator
```

## Installation via Helm
**Disclaimer:** The following Helm chart is an external project not maintained by the rss-operator maintainers; so it may not be up to date.

One day we might make this operator available as a Helm
chart.  See the [etcd-operator](https://github.com/kubernetes/charts/tree/master/stable/etcd-operator) for an example.
