#!/bin/bash


_term() { 
  echo "Caught SIGTERM signal!" 
  exit 0
}

trap _term SIGTERM

count=1
echo "$$ Container starting..."
kubectl annotate --overwrite pods $HOSTNAME state=starting
while [ 1 = 1 ]; do
    kubectl annotate --overwrite pods $HOSTNAME state=running
    printf .
    sleep 30
    kubectl annotate --overwrite pods $HOSTNAME counter=$count
    count=$((count+1))
done
kubectl annotate --overwrite pods $HOSTNAME state=done
echo "$$ Container done."
