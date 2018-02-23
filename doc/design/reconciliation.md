# Cluster Membership Reconciliation

## Reconciliation

Reconciliation is the prerequisite for making changes to the replication setup
(starting new copies or removing old ones). It seeks to incorporate
information from kubernetes (the list of currently running Pods) with the
current operator's knowledge.  Additionally, it seeks to validate that the
each copy of the application is still in the expected state by utilizing the
configured `status` command.

### Resize

The existance and state of offline Pods is persisted between reconciliation
events in order to provide useful feedback to users and admins.  Excess
records are removed during once scale down events of the underlying
StatefulSet complete.

### Recovery

Should the operator fail or be stopped, its replacement uses the
reconciliation process to re-discover the current state of the application.

### Logging

Since knowing whether a Pod is active doesn't give the full picture of the
application's state, state is indicated in the logs by character suffixes
after the Pod's name.

- `~` Indicates the Pod is not online
- `!` Indicates the Pod is active but the application has failed
- `:` Indicates the Pod is active, the application is running and the sequence number is everything after the `:`
- `?` Indicates the Pod is active, the application is stopped and the sequence number is everything after the `?`

After the reconciliation the current membership is logged.  The application
below, for example, is in pretty bad shape:

```
[2018-02-23T00:26:33Z]  INFO  c=galera-demo o=rss-operator:	  current membership: [rss-galera-demo-0:7, rss-galera-demo-1~, rss-galera-demo-2!, rss-galera-demo-3?7]
```
