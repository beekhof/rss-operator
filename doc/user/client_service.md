## Client service

For every rss cluster created the rss-operator will create a services in the
same namespace with the name `<cluster-name>-svc`.

```
$ kubectl create -f apps/galera/deployment.yaml
$ kubectl get services
NAME             TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)          AGE
galera-demo-svc  ClusterIP      None           <none>        3306/TCP         5m
```


The service is of type `ClusterIP` and is only accessible from within the
Kubernetes cluster's network.  It primarily exists for the application to
perform its replication tasks, including connecting to peers via DNS
addressable names.

In order to function, it requires the list of ports used internally by the
application to be configured as part of the CRD.

```
...
spec:
  replicas: 4
  servicePorts:
  - name: galera
    protocol: TCP
    port: 3306
    targetPort: 3306
...
```

So for instance we can access the service from a pod in our cluster:

```
$ kubectl run --rm -i --tty fun --image quay.io/beekhof/galera:latest --restart=Never -- mysql -h galera-demo-svc -u galera -ppass -P 3306 menagerie -e 'select * from pet;'
+----------+-------+---------+------+------------+-------+
| name     | owner | species | sex  | birth      | death |
+----------+-------+---------+------+------------+-------+
| Puffball | Diane | hamster | f    | 1999-03-30 | NULL  |
+----------+-------+---------+------+------------+-------+
```

If accessing this service from a different namespace than that of the rss
cluster, use the FQDN `http://<cluster-name>-client.<cluster-
namespace>.svc.cluster.local:3306` .


### Testing

If for some reason the service isn't working (relialy or even at all), we can
peak behind the scenes by examining the associated endpoints.


```
$ kubectl get endpoints galera-demo-svc 
NAME              ENDPOINTS                                                             AGE
galera-demo-svc   10.233.65.49:3306,10.233.66.219:3306,10.233.67.55:3306 + 1 more...    5h

$ kubectl get endpoints/galera-demo-svc -o yaml
apiVersion: v1
kind: Endpoints
metadata:
  creationTimestamp: 2018-02-21T22:58:17Z
  labels:
    app: galera-demo
    kind: galera
    rss-operator-managed: "true"
  name: galera-demo-svc
  namespace: default
  resourceVersion: "15806750"
  selfLink: /api/v1/namespaces/test/endpoints/galera-demo-svc
  uid: b432a512-175a-11e8-938e-545200860101
subsets:
- addresses:
  - hostname: rss-galera-demo-1
    ip: 10.233.65.49
    nodeName: worker-2
    targetRef:
      kind: Pod
      name: rss-galera-demo-1
      namespace: default
      resourceVersion: "15806737"
      uid: b44691c1-175a-11e8-938e-545200860101
  - hostname: rss-galera-demo-0
    ip: 10.233.66.219
    nodeName: worker-1
    targetRef:
      kind: Pod
      name: rss-galera-demo-0
      namespace: default
      resourceVersion: "15806730"
      uid: b43dc2ca-175a-11e8-938e-545200860101
  - hostname: rss-galera-demo-3
    ip: 10.233.67.55
    nodeName: worker-4
    targetRef:
      kind: Pod
      name: rss-galera-demo-3
      namespace: default
      resourceVersion: "15806748"
      uid: b4649127-175a-11e8-938e-545200860101
  - hostname: rss-galera-demo-2
    ip: 10.233.68.3
    nodeName: worker-3
    targetRef:
      kind: Pod
      name: rss-galera-demo-2
      namespace: default
      resourceVersion: "15806740"
      uid: b4558724-175a-11e8-938e-545200860101
  ports:
  - name: galera
    port: 3306
    protocol: TCP

```

Make sure the correct number of Pods are present and that all the ports
required for your application are listed at the end.


### Accessing the service from outside the cluster

In order to access the client API of the application cluster from outside the
Kubernetes cluster, we can expose a new client service of type `LoadBalancer`.
If using a cloud provider like GKE/GCE or AWS, setting the type to
`LoadBalancer` will automatically create the load balancer with a publicly
accessible IP.

The spec for this service will use the label selector:

```
  selector:
    app: <cluster-name>
    rss-active-member: "true"
```

to load balance the client requests over the application pods in our cluster.
Since a pod being active doesn't always correspond to the application being
started inside it, the operator tags Pods with a `rss-active-member` label
which allows us to exclude Pods with stopped or failed copies of the
application.

So for our example cluster above we can create a service like so:
```
$ cat example-galera-client-service-lb.yaml

apiVersion: v1
kind: Service
metadata:
  labels:
    app: galera-demo
  name: galera-demo-external
spec:
  externalTrafficPolicy: Local
  ports:
  - name: galera
    port: 3306
    protocol: TCP
    targetPort: 3306
  publishNotReadyAddresses: true
  selector:
    app: galera-demo
    rss-active-member: "true"
  sessionAffinity: ClientIP
  type: LoadBalancer

$ kubectl create -f example-rss-client-service-lb.yaml
```

Wait until the load balancer is created and the service is assigned an `EXTERNAL-IP`:
```
$ kubectl get services
NAME                  TYPE           CLUSTER-IP     EXTERNAL-IP    PORT(S)          AGE
galera-demo-svc       ClusterIP      None           <none>         3306/TCP         5m
galera-demo-external  LoadBalancer   10.233.47.99   35.184.74.127  3306:30323/TCP   5m
```

The rss client API should now be accessible from outside the kubernetes cluster:
```
$ mysql -h 35.184.74.127 -u galera -ppass -P 3306 menagerie -e 'select * from pet;'
+----------+-------+---------+------+------------+-------+
| name     | owner | species | sex  | birth      | death |
+----------+-------+---------+------+------------+-------+
| Puffball | Diane | hamster | f    | 1999-03-30 | NULL  |
+----------+-------+---------+------+------------+-------+
```
