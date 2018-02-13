# Best practices

## Large-scale deployment

To run replicated clusters at large-scale, it is important to assign cluster pods to nodes with desired resources, such as SSD, high performance network. 

### Assign to nodes with desired resources

Kuberentes nodes can be attached with labels. Users can [assign pods to nodes with given labels](http://kubernetes.io/docs/user-guide/node-selection/). Similarly for the rss-operator, users can specify `Node Selector` in the pod policy to select nodes for member pods. For example, users can label a set of nodes with SSD with label `"disk"="SSD"`. To assign application pods to these node, users can specify `"disk"="SSD"` node selector in the cluster spec.

### Assign to dedicated node

Even with container isolation, not all resources are isolated. Thus, performance interference can still affect  clusters' performance unexpectedly. The only way to achieve predictable performance is to dedicate a node for each member pod.

Kuberentes node can be [tainted](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/taint-toleration-dedicated.md) with keys. Together with node selector feature, users can create nodes that are dedicated for only running a particular cluster. This is the **suggested way** to run high performance large scale clusters.

Use kubectl to taint the node with `kubectl taint nodes {app-name} dedicated`. Then only `{app-name}` pods, which tolerate this taint will be assigned to the nodes.
