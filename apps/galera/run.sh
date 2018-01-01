#!/bin/bash

count=1
echo "$$ Container starting..."
kubectl label --overwrite pods $HOSTNAME state=starting
while [ 1 = 1 ]; do
    kubectl label --overwrite pods $HOSTNAME state=running
    printf .
    sleep 30
    kubectl label --overwrite pods $HOSTNAME counter=$count
    count=$((count+1))
done
kubectl label --overwrite pods $HOSTNAME state=done
echo "$$ Container done."
