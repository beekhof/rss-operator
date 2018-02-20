# Disaster Recovery

## Loosing a Majority of Pods

In the event that more than half the configured Pods are lost, the underlying
application may have lost quorum.

So long as the configured `status` command reports the application is healthy
on the remaining Pods, the operator will follow its normal recovery proceedure
of waiting for the StatefulSet (STS) controller to respawn lost Pods and following the [replication logic](replication.md).

If the loss of quorum results in the applcation self-terminating or entering a
zombie state, the operator will execute the configured `stop` command on those
pods as part of the [reconciliation logic](reconciliation.md), wait for the
STS controller to respawn lost Pods and once again follow the [replication logic](replication.md).

There is no need to be concerned with backup and recovery processes as each
pod has their own persistent copy the data.

## Scaling up from Zero

One special case does occur however when scaling an application back up from zero.

The replication CRD stores a copy of the configured replica count whenever it
is greater than 0 and matches the actual number of running pods.

This allows it to ensure that the same number of copies (or more) are once
again created when the operator recreates the applicaiton cluster (perhaps
after a lights out failure or an upgrade proceedure).  This avoids any
possibile race conditions that might result in or from `Pod N` holding the
most recent copy of the data _AND_ the configured number of replicas being
less than N.
