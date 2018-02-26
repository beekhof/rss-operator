# Why the Replicated Stateful Set Operator Exists

As we all know, the accepted best practice for cloud applications is to create
stateless services. However, some legacy workflows don't fit this model and may
be non-trivial to rewrite or replace. Additionally, many applications will
eventually end up persisting something somewhere, be it to a database, a key-
value store, or a disk.  Some past OSP summit speakers have advocated that
databases and key-value stores be treated like filesystems and exist as a
separate cluster to Kubernetes.  While attractive in that we get to continue
using existing best practices for recovering after lights out, this approach
misses out on the benefits of containerisation and one-stop management of the
application stack.

Other community members simplify the architecture by advocating a single copy of
mysql, relying on Kubernetes restarts and shared storage to solve the problem.
At best this leads to long recovery times for large databases, at worst the
current lack of fencing could lead to stalled recovery or disk corruption.

So we want replication to avoid long recovery times but what about lights-out
recovery?

Some rare applications will allow you start copies in any order and do the
reconciliation for you, however most require the most up-to-date copy to start
first.

It seems apparent that Stateful Sets were originally created for this use
case, at first blush it looks like pod 0 will always start first and by virtue
of being the last to stop, will have the most recent copy of the database.
Unfortunately due to the lack of a feedback mechanism in the scheduler, it is
trivial to arrange failure scenarios that allow pod 0 to [fall behind](https://www.dropbox.com/s/mqsuvbtr3lj5hxb/Stateful%20Set%20Fencing.pdf?dl=0) -
particularly during scale down events. Additionally, like the single-copy-
architecture, the current lack of fencing can lead to stalled recovery and an
inability to scale up or down in bare metal Kubernetes deployment.

More recently, the CoreOS folks successfully showed that some replicated
applications can be modelled with their [operator pattern](https://coreos.com/blog/introducing-operators.html).

The etcd and Prometheus implementations are based on ephemeral storage,
requiring periodic backups to (currently) AWS and a full restore after lights-
out.  Apart from the reliance on AWS for cold starts (possible security and/or
compliance issues), databases (particularly those in OSP) can grow rather large
resulting in extended recovery times.

The Replicated Stateful Set Operator is therefor a new operator based on
Stateful Sets and persistent storage. Using Stateful Sets allows us to reuse
core functionality for much of the upgrade and restart logic (the CoreOS folks
implemented their own), allowing the operator to focus on figuring out which
copy has the most recent version of the data and arranging it to start first.

To do this, we allow the admin to specify hooks for:
 
- detecting an application's sequence number seeding the application (first
- primary) starting subsequent primaries stopping the application

During a lights-out recovery, the operator will:

- wait for the Stateful Set to launch the application's containers (CMD for
  the container should not actually start the application)

- use Kubernetes APIs to execute the `sequence number` hook on each container
  (an integer response to stdout is required)

- determine the container with the largest value, and use Kubernetes APIs to
  execute the `seed` hook there

- use Kubernetes APIs to execute the `primary` hook on the remaining copies

To make life easier for container authors, we automatically create an internal
service so that the peers are addressable as '{pod_name}.{service_name}' and the
list of active primaries is passed to the `primary` hook on the command line.

With an eye to the future, we also allow for status checks and the ability to
have copies that are not primaries, whatever that may mean to the app (perhaps
read-only replicas?).
