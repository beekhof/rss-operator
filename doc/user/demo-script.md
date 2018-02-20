# Prep

unset PROMPT_COMMAND
unset PROMPT_COLOR
unset GOPATH

EMK="\033[1;30m"
EMR="\033[1;31m"
EMG="\033[1;32m"
EMY="\033[1;33m"
EMB="\033[1;34m"
EMM="\033[1;35m"
EMC="\033[1;36m"
EMW="\033[1;37m"
NONE="\033[0m"    # unsets color to term fg color
# Start of Line code
SOL="\033[G"

PS1="demo # "

function echo() {
	/bin/echo -e "${SOL}${EMB}###### ${*} ######${NONE}"
}


echo Context
go get -u github.com/beekhof/rss-operator
cd /go/src/github.com/beekhof/rss-operator
export NAMESPACE=demo


echo Create a test namespace
kubectl create ns $NAMESPACE

echo Setup RBAC roles
example/rbac/create_role.sh --namespace $NAMESPACE --role-name $NAMESPACE-operator --role-binding-name $NAMESPACE-operator

echo Deploy the operator
kubectl -n $NAMESPACE create -f example/operator.yaml

echo Wait for the operator to start
kubectl -n $NAMESPACE get pods -w

echo Define a cluster
kubectl -n $NAMESPACE create -f example/cluster.yaml

#echo Watch the operator create it
#kubectl -n $NAMESPACE logs -f po/rss-operator

echo Check the pods
kubectl -n $NAMESPACE get pods -w

echo Check the cluster status
kubectl -n $NAMESPACE get rss/example -o=jsonpath='{"Primaries: "}{.status.members.primary}{"\n"}{"Members:   "}{.status.members.ready}{"\n"}'

echo Test recovery
kubectl -n $NAMESPACE delete pod rss-example-0 --now

#echo Watch the cluster recover it
#kubectl -n $NAMESPACE logs -f po/rss-operator

echo Check the pods again
kubectl -n $NAMESPACE get pods -w

# Cleanup

kubectl -n $NAMESPACE delete crd,deploy,rs,rss,sts,svc,pods --all
kubectl delete clusterrole/$NAMESPACE-operator clusterrolebinding/$NAMESPACE-operator ns/$NAMESPACE
