## Cluster status reporting

An application cluster permanently fails when all its members are dead and no
Pod is able to act as a seed. A user might update the cluster CRD with invalid
input or format. The operator needs to notify users about these bad events.

A common way to do this in the Kuberentes world is through status field, like
pod.status. The replication operator will write out cluster status similar to
pod status.

```go

type ReplicatedStatefulSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ClusterSpec   `json:"spec"`
	Status            ClusterStatus `json:"status"`
}

Type ClusterStatus struct {
	// Phase is the cluster running phase
	Phase  ClusterPhase `json:"phase"`
	Reason string       `json:"reason,omitempty"`

	// ControlPuased indicates the operator pauses the control of the cluster.
	ControlPaused bool `json:"controlPaused,omitempty"`

	// Condition keeps track of all cluster conditions, if they exist.
	Conditions []ClusterCondition `json:"conditions,omitempty"`

	// Size is the current size of the cluster
	Replicas        int `json:"replicas"`
	RestoreReplicas int `json:"restoreReplicas"`

	// ServiceName is the LB service for accessing galera nodes.
	ServiceName string `json:"serviceName,omitempty"`

	// ClientPort is the port for galera client to access.
	// It's the same on client LB service and galera nodes.
	ClientPort int `json:"clientPort,omitempty"`

	// Members are the galera members in the cluster
	Members MembersStatus `json:"members"`

    ...
}
```

The replication operator keeps the truth of the cluster status, and updates
the CRD after each cluster reconciliation atomically. Note that users can:

 - modify CRD item concurrently with the operator. 
 - update CRD and overwrite the status filed with empty or bad input

The operator MUST handle these two cases gracefully. Thus, the operator MUST
NOT overwrite the user input on spec field. The operator CANNOT trust the
existing status filed in CRD.

To not overwrite the user input, the replication operator will set a resource
version, and do a compare and swap on resource version to update the status.
If the compare and swap fails, the operator knows that the user updated the
spec concurrently, and it will retry later after get the new spec.

To not be affected by the potential empty or broken status accidentally
written by the user, with one exception[[1]](#fnote1), the replication operator
will never read the status from CRD. It always keeps the source of truth in
memory.

In summary, the CRD:

- receives spec updates (or initialize the spec) with a resource version
- collects cluster status during reconciliation
- atomically updates cluster status with known resource version after
  reconciliation if there is a status change
   - retries if resource version does not match

## Footnotes

<a name="fnote1">[1]</a>Reading the previous replica count is the one
exception to this.  Setting it too high could be considered an avenue to DoS
as the cluster will block until the configured replica count meets or exceeds
the bogus value.  Setting it too low may cause data loss, but may also be a
useful recovery capability when used responsibly.
