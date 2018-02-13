## Resource Labels
The rss-operator creates the following Kubernetes resources for each rss cluster:
- Pods for the member nodes
- Services for clients and peer members

where each resource has the following labels:
- `app=<cluster-name>`
- `rss-operator-managed=true`
