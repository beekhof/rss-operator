# Application Cluster Lifecycle in Replication Operator

Consider the lifecycle of application cluster "A".

- Initially, "A" doesn't exist. 

  The operator considers this cluster has 0 members. Any cluster with 0
  members would be considered as non-existant.

- At some point of time, a user creates an object for "A".

  Operator would receive "ADDED" event and create this cluster.
  For the entire lifecycle, an application cluster can only be created once.

- Then user might update the spec of "A" 0 or more times. 

  Each time, the operator receives a "MODIFIED" event and (since we currently
  only allow sizes to be altered) will update the Stateful Set (STS) object
  and wait for the STS controller to reconcile the actual state with the
  desired state of given spec.

  Once this completes, we then manage any additional changes to the
  replication setup by creating or removing additional primaries and/or
  secondaries.


- Finally, a user deletes the object "A". 
  The operator will delete and recycle all resources of "A". For the entire
  lifecycle, an application cluster can only be deleted once.
