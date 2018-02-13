# RSS operator
unit/integration:
[![Build Status](https://quay.io/repository/beekhof/rss-operator/status)](https://quay.io/repository/beekhof/rss-operator)
e2e (Kubernetes stable):
[![Build Status](#)](#)
e2e (upgrade):
[![Build Status](#)](#)

### Project status: beta

Most major planned features have been completed and while no breaking API changes are currently planned, we reserve the right to address bugs and API changes in a backwards incompatible way before the project is declared stable. See [upgrade guide](./doc/user/upgrade/upgrade_guide.md) for safe upgrade process.

Currently user facing _rss_ cluster objects are created as [Kubernetes Custom Resources](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/), however, taking advantage of [User Aggregated API Servers](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/aggregated-api-servers.md) to improve reliability, validation and versioning is planned. The use of Aggregated API should be minimally disruptive to existing users but may change what Kubernetes objects are created or how users deploy the operator.

We expect to consider the operator stable soon; backwards incompatible changes will not be made once the project reaches stability.

### Overview

The RSS operator manages etcd clusters deployed to [Kubernetes][k8s-home] and automates tasks related to operating a cluster.

- [Create and destroy](#create-and-destroy-a-cluster)
- [Resize](#resize-a-cluster)
- [Failover](#failover)
- [Rolling upgrade](#upgrade-a-cluster)
- [Backup and Restore](#backup-and-restore-a-cluster)

There are [more spec examples (TODO)](./doc/user/spec_examples.md) on setting up clusters with different configurations

Read [Best Practices (TODO)](./doc/best_practices.md) for more information on how to better use RSS operator.

Read [RBAC docs (TODO)](./doc/user/rbac.md) for how to setup RBAC rules for RSS operator if RBAC is in place.

Read [Developer Guide](./doc/dev/developer_guide.md) for setting up development environment if you want to contribute.

See the [Resources and Labels](./doc/user/resource_labels.md) doc for an overview of the resources created by the rss-operator.

## Requirements

- Kubernetes 1.9+

## Demo

## Getting started

![RSS operator demo](https://raw.githubusercontent.com/beekhof/rss-operator/master/doc/gif/demo.gif)

### Deploy rss operator

See [instructions on how to install/uninstall RSS operator](doc/user/install_guide.md) .
$ kubectl create -f example/operator.yaml

### Create and destroy a cluster

```bash
$ kubectl create -f example/cluster.yaml
```

A 3 member cluster will be created.

```bash
$ kubectl get pods
NAME            READY     STATUS    RESTARTS   AGE
rss-example-0   1/1       Running   0          3m
rss-example-1   1/1       Running   0          3m
rss-example-2   1/1       Running   0          3m
rss-operator    1/1       Running   0          6m
```

In addition to the pod status, we can check the state of the application by examining the Kubernetes Custom Resource representing the cluster:

```bash
$ kubectl -n dummy get rss/example -o=jsonpath='{"Primaries: "}{.status.members.primary}{"\n"}{"Members:   "}{.status.members.ready}{"\n"}'
Primaries: [rss-example-0 rss-example-1 rss-example-2]
Members:   [rss-example-0 rss-example-1 rss-example-2]
```

See [client service](doc/user/client_service.md) for how to access clusters created by the RSS operator.

If you are working with [minikube locally](https://github.com/kubernetes/minikube#minikube) create a nodePort service and _TODO..._

Destroy etcd cluster:

```bash
$ kubectl delete -f example/cluster.yaml
```

### Resize a cluster

Create a cluster:

```
$ kubectl create -f example/cluster.yaml
```

In `example/cluster.yaml` the initial cluster size is 3.
Modify the file and change `size` from 3 to 5.

```
$ cat example/cluster.yaml
apiVersion: clusterlabs.org/v1alpha1
kind: ReplicatedStatefulSet
metadata:
  name: example
spec:
  replicas: 5
  pod:
    antiAffinity: true
    commands:
      sequence: 
        timeout: 20s
        command: ["/sequence.sh"]
      primary: 
        command: ["/start.sh"]
      seed: 
        command: ["/seed.sh"]
      status: 
        timeout: 2m
        command: ["/check.sh"]
      stop: 
        command: ["/stop.sh"]
    containers:
    - name: dummy
      image: quay.io/beekhof/dummy:latest
```

Apply the size change to the cluster CR:
```
$ kubectl apply -f example/cluster.yaml
```
The etcd cluster will scale to 5 members (5 pods):
```
$ kubectl get pods
NAME            READY     STATUS    RESTARTS   AGE
rss-example-0   1/1       Running   0          5m
rss-example-1   1/1       Running   0          5m
rss-example-2   1/1       Running   0          5m
rss-example-3   1/1       Running   0          2m
rss-example-4   1/1       Running   0          2m
rss-operator    1/1       Running   0          9m
```

Similarly we can decrease the size of cluster from 5 back to 3 by changing the size field again and reapplying the change.

```
$ cat example/cluster.yaml
apiVersion: clusterlabs.org/v1alpha1
kind: ReplicatedStatefulSet
metadata:
  name: example
spec:
  replicas: 3
  pod:
    antiAffinity: true
    commands:
      sequence: 
        timeout: 20s
        command: ["/sequence.sh"]
      primary: 
        command: ["/start.sh"]
      seed: 
        command: ["/seed.sh"]
      status: 
        timeout: 2m
        command: ["/check.sh"]
      stop: 
        command: ["/stop.sh"]
    containers:
    - name: dummy
      image: quay.io/beekhof/dummy:latest
```
```
$ kubectl apply -f example/cluster.yaml
```

We should see that the cluster will eventually reduce to 3 pods:

```
$ kubectl get pods
NAME            READY     STATUS    RESTARTS   AGE
rss-example-0   1/1       Running   0          8m
rss-example-1   1/1       Running   0          8m
rss-example-2   1/1       Running   0          8m
rss-operator    1/1       Running   0          11m
```

### Failover

If any members crash, the RSS operator will automatically recover the failure.
Let's walk through in the following steps.

Create a cluster:

```
$ kubectl create -f example/cluster.yaml
```

Wait until all three members are up. Simulate a member failure by deleting a pod:

```bash
$ kubectl delete pod rss-example-0 --now
```

The RSS operator will recover the failure by recreating the pod `rss-example-0 `:

```bash
$ kubectl get pods
NAME            READY     STATUS    RESTARTS   AGE
rss-example-0   1/1       Running   0          41s
rss-example-1   1/1       Running   0          10m
rss-example-2   1/1       Running   0          10m
rss-operator    1/1       Running   0          13m
```

Destroy dummy cluster:
```bash
$ kubectl delete -f example/cluster.yaml
```

### RSS operator recovery

If the RSS operator restarts, it can recover its previous state.
Let's walk through in the following steps.

```
$ kubectl create -f example/cluster.yaml
```

Wait until all three members are up. Then

```bash
$ kubectl delete -f example/operator.yaml
pod "rss-operator" deleted

$ kubectl delete pod rss-example-0  --now
pod "rss-example-0 " deleted
```

Then restart the RSS operator. It should recover itself and the clusters it manages.

```bash
$ kubectl create -f example/operator.yaml
pod "rss-operator" created

$ kubectl get pods
NAME            READY     STATUS    RESTARTS   AGE
rss-example-0   1/1       Running   0          4m
rss-example-1   1/1       Running   0          13m
rss-example-2   1/1       Running   0          13m
rss-operator    1/1       Running   0          4m
```

### Upgrade a cluster

TODO


### Backup and Restore a cluster

TODO

### Limitations

- The RSS operator only manages clusters created in the same namespace. Users need to create multiple operators in different namespaces to manage clusters in different namespaces.

- Lights-out recovery of the RSS operator currently requires shared storage. Backup and restore capability will be added in the future if there is sufficient interest. 


[k8s-home]: http://kubernetes.io
