#!/bin/bash

count=1
echo "$$ Container starting..."
kubectl label pods $HOSTNAME state=starting
while [ 1 = 1 ]; do
    kubectl label pods $HOSTNAME state=running
    printf .
    sleep 10
    kubectl label pods $HOSTNAME counter=$count
    count=$((count+1))
done
kubectl label pods $HOSTNAME state=done
echo "$$ Container done."
