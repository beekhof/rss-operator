# How Replication Works

## When Replication Happens

Replication changes (adding or removing a peer) only triggered by the operator
when the correct number of pods are available.

- If too few pods are available, we wait for the Stateful Set (STS) controller to make them available.[[1]](#fnote1)
- If too many pods are available, we wait for the Stateful Set controller to stop some.

## Seeding Replication

Once the correct number of pods are available, we check if the correct number
of primaries and secondaries (if any) are running.  If not, we use the configured
`sequence` command to obtainer the replication sequence number for each pod.

> The sequence numbers must be the only value sent to stdout by the
> script/program and is interpreted as unisgned integer.

If no sequence number is available, exit with any value other than 0.

The pod with the highest sequnce number is selected to host the first primary.
If multiple pods have the same sequence number, the pod with the lowest index
will be preferred - as it is the least likely to be stopped by the STS
controller during a scale-down event.

By default the first primary is started invoking the configured (and required)
`primary` command.  However, we also allow a `seed` command to be configured
since bringing up the frist memeber often involves many special cases.

In both cases, the configured command should indicate success by returning 0.
Any other value is treated as an an error.

## Adding Additional Primaries

Once the application has been successfully seeded, we once again look for the
pod with the highest remaining sequnce number (since it will be more up-to-
date and faster to become synchronized) and index upon which to invoke the
command for starting a `primary`.

To make life easier for the authors of the replication commands, we pass along
the DNS resolvable host names of any existing primaries.

Just like in the seeding case, the configured command should indicate success
by returning 0. Any other value is treated as an an error.

Each new primary is created in serial, waiting for the prior one to either
fail or start successfully.

## Stopping Primaries

Under normal circumstances, the replication operator allows the STS controller
to manage the shutdown of application primaries.  However when the configured
number of replicas exceeds the configured number of replicas, then potentially
there are copies of the application running on pods that the STS controller
has no need to terminate.  So the operator will look for primaries with the
_lowest_ sequnce number and/or _highest_ index and invoke the configured
`stop` command on them in series until only the required number of masters
remain.

## Secondaries

An application may wish to model different classes of functionality, such as
read-only peers or passive hot-standbies.  The replication controller supports
this with the concept of secondaries.

### Adding Secondaries

If

- a `secondary` command has been configured, and
- the application has the required number of primaries, and
- the configured replica count is higher than the configured primary count, then

we once again look for the pod with the highest remaining sequnce number
and/or index upon which to invoke the `secondary` command.

As for primaries, we pass along the DNS resolvable host names of any existing
primaries and the command should indicate success by returning 0. Any other
value is treated as an an error.

## Handling Failures

### Seed Failures

If the command for creating the initial application primary fails, the pod
with the next lowest index (but equal sequnce number) will be tried until one
succeeds.  If no further pods with the highest sequence number remain,
replication will be aborted and the cluster will enter a failed state.

The pod on which the command failed will be listed as failed and excluded from
hosting primaries and secondaries until after it has been cleaned up by the
subsequent replicaiton phases.

### Primary Failures

If the command for creating a subsequent application primary fails, the pod
with the next lowest sequence number and/or index will be tried until one
succeeds.  If no further pods remain, replication will be aborted and the
cluster will enter a failed state.

The pod on which the command failed will be listed as failed and excluded from
hosting primaries and secondaries until after it has been cleaned up by the
subsequent replicaiton phases.

Any existing primaries and secondaries will remain active.

### Secondary Failures

If the command for creating an application secondary fails, the pod
with the next lowest sequence number and/pr index will be tried until one
succeeds.  If no further pods remain, replication will be aborted and the
cluster will enter a failed state.

The pod on which the command failed will be listed as failed and excluded from
hosting primaries and secondaries until after it has been cleaned up by the
subsequent replicaiton phases.

Any existing primaries and secondaries will remain active.

### Stop Failures

The initial response to a failed replication command is to clean the pod up by
issuing the configured `stop` command.  If this completes successfully
(indicated by returning 0) then the pod is permitted to once again try to
become an application primary or secondary.

If the configured command fails, then the pod is deleted from Kubernetes and
the operator waits for the STS controller to respawn it.

## Footnotes

<a name="fnote1">[1]</a>In the future, we may decide to support scale-up
events in the case that there is already a quorate number of primaries.
Patches are accepted.
