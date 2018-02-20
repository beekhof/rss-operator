# Resource Ownership and GC

## Resource Ownership

A replication cluster creates and manages pods, services and stateful sets.
Those resources are owned by the cluster. Two clusters own a completely
disjoint set of resources, even if they have the same name. For example,
replication cluster "A" was deleted and later replication cluster "A" was
created again; we consider these as two different clusters. Thus, the new
cluster should not mange the resources from the old one. The old resources
should be treated as garbage and to be collected.

As discussed in https://github.com/coreos/etcd-operator/issues/517, we
correlate owners (i.e. cluster) and their resources by making use of
`ObjectMeta.OwnerReferences` field.

For replication services, they will have only one owner -- it's managing
cluster.

## Options for Garbage Collection

We will talk about two strategies to do GC in the following. We are only
covering stateful sets (pods are managed by the STS controller), services,
although the algorithm applies to more resources.

### Lazy deletion

We should remove resources when deleting or creating a cluster.
We remove them via label convention:
- remove Stateful Sets with selector label{"a[[": $cluster_name, "rss-operator-managed": "true"}
- remove services with selector label{"app": $cluster_name, "rss-operator-managed": "true"}

As a side note on creating a cluster: If, right before creating a cluster, a
stateful set or svc was selected out, then it must hangs around as garbage.
This could lead to confusing, even harmful cases. So we must delete all
related garbage resources before we start the new cluster.


### Periodic full GC

Every interval (10 mins or so), we find out orphaned statefulsets and svcs.

Initially, our use case would be O(10) clusters, and thus total selected items would be at most O(100).

A simple full scanning algorithm:
- List statefulsets and services.
- Controller has knowledge of all current replication clusters.
- Find out statefulsets and services whose OwnerRef-ed cluster doesn't currently exist within the controller.
