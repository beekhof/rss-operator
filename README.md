# Replicated Stateful Set (RSS) Operator

[![Build Status](https://quay.io/repository/beekhof/rss-operator/status)](https://quay.io/repository/beekhof/rss-operator)

### Project status: beta

Most major planned features have been completed and while no breaking API
changes are currently planned, we reserve the right to address bugs and API
changes in a backwards incompatible way before the project is declared stable.
See [upgrade guide](./doc/user/upgrade/upgrade_guide.md) for safe upgrade
process.

We expect to consider the operator stable soon; backwards incompatible changes
will not be made once the project reaches stability.

See the [roadmap](./ROADMAP.md).


### Overview

The replication operator manages application clusters deployed to [Kubernetes][k8s-home]
and automates tasks related to seeding, and operating a cluster. It has been
adapted from the [etcd operator](https://github.com/CoreOS/etcd-operator)
from the CoreOS folks.

- [Create and destroy](#create-and-destroy-a-cluster)
- [Resize](#resize-a-cluster)
- [Failover](#failover)

There are [examples](./apps) of different applications and specs for driving them

Read [why](./doc/Rationale.md) the operator exists and [how](./doc/design/replication.md) replication is managed.

Read [RBAC docs](./doc/user/rbac.md) for how to setup RBAC rules for the replication operator if RBAC is in place.

Read [Developer Guide](./doc/dev/developer_guide.md) for setting up development environment if you want to contribute.

See the [Resources and Labels](./doc/user/resource_labels.md) doc for an overview of the resources created by the rss-operator.

## Requirements

- Kubernetes 1.9+

## Demo

[![asciicast](https://asciinema.org/a/164903.png)](https://asciinema.org/a/164903)

## Getting started

### Deploy replication operator

See [instructions on how to install/uninstall replication operator](./doc/user/install_guide.md) .

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

See [client service](doc/user/client_service.md) for how to access clusters created by the replication operator.

If you are working with [minikube locally](https://github.com/kubernetes/minikube#minikube) create a nodePort service and _TODO..._

Destroy a replicated cluster:

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
  reconcileInterval: 30s
  pod:
    antiAffinity: true
    commands:
      sequence: 
        command: ["/sequence.sh"]
      primary: 
        command: ["/start.sh"]
      seed: 
        command: ["/seed.sh"]
      status: 
        timeout: 60s
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
The replicated cluster will scale to 5 members (5 pods):
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
  reconcileInterval: 30s
  pod:
    antiAffinity: true
    commands:
      sequence: 
        command: ["/sequence.sh"]
      primary: 
        command: ["/start.sh"]
      seed: 
        command: ["/seed.sh"]
      status: 
        timeout: 60s
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

If any members crash, the replication operator will automatically recover the failure.
Let's walk through in the following steps.

Create a cluster:

```
$ kubectl create -f example/cluster.yaml
```

Wait until all three members are up. Simulate a member failure by deleting a pod:

```bash
$ kubectl delete pod rss-example-0 --now
```

The replication operator will recover the failure by recreating the pod `rss-example-0 `:

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

### Replication operator recovery

If the replication operator restarts, it can recover its previous state.
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

Then restart the replication operator. It should recover itself and the clusters it manages.

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


### Limitations

- The replication operator only manages clusters created in the same namespace. Users need to create multiple operators in different namespaces to manage clusters in different namespaces.

- Lights-out recovery of the replication operator currently requires shared storage. Backup and restore capability will be added in the future if there is sufficient interest. 


[k8s-home]: http://kubernetes.io
