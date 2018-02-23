
echo Preparation
go get -u github.com/beekhof/rss-operator
cd go/src/github.com/beekhof/rss-operator
export NAMESPACE=demo

echo Create a test namespace
kubectl create ns $NAMESPACE

echo Setup RBAC roles
example/rbac/create_role.sh --namespace $NAMESPACE --role-name $NAMESPACE-operator --role-binding-name $NAMESPACE-operator

echo Deploy the operator
kubectl -n $NAMESPACE create -f example/operator.yaml

echo Wait for the operator to start
kubectl -n $NAMESPACE get pods -w

echo Define a Ceph backed storgae-class
kubectl -n $NAMESPACE create -f example/storage.yaml

echo Define a cluster
kubectl -n $NAMESPACE create -f apps/galera/cluster.yaml

echo Watch the pods come up
kubectl -n $NAMESPACE get pods -w

echo See how the operator created it
kubectl -n $NAMESPACE logs --tail 70 po/rss-operator | grep -v stdout

echo Check the cluster status
kubectl -n $NAMESPACE get rss/galera-demo -o=jsonpath='{"Primaries: "}{.status.members.primary}{"\n"}{"Members:   "}{.status.members.ready}{"\n"}'

echo Try accessing the internal service name
kubectl  -n $NAMESPACE run --rm -i --tty fun --image quay.io/beekhof/galera:latest --restart=Never -- mysql -h galera-demo-svc -u galera -ppass -P 3306 menagerie -e 'select * from pet;'

echo Display the replication status
kubectl -n $NAMESPACE run --rm -i --tty fun --image quay.io/beekhof/galera:latest --restart=Never -- mysql -h galera-demo-svc -u galera -ppass -P 3306  -e "SHOW STATUS LIKE 'wsrep_%';" 

echo Create an external facing load balancer
kubectl -n $NAMESPACE create -f apps/galera/client.yaml
kubectl -n $NAMESPACE get svc

echo Now test it works
mysql -h $(kubectl -n $NAMESPACE get svc | grep external | awk '{print $3}') -u galera -ppass -P 3306  menagerie -e 'select * from pet;'

echo Test recovery
kubectl -n $NAMESPACE delete pod rss-galera-demo-0 --now

echo Watch the pod be recovered
kubectl -n $NAMESPACE get pods -w

echo See how the operator recovered it
kubectl -n $NAMESPACE logs --tail 20 po/rss-operator


echo Cleanup

kubectl -n $NAMESPACE delete crd,deploy,rs,rss,sts,svc,pods --all
kubectl delete clusterrole/$NAMESPACE-operator clusterrolebinding/$NAMESPACE-operator ns/$NAMESPACE
