#!/bin/bash

count=1
echo "$$ Container starting..."
while [ 1 = 1 ]; do
    kubectl label --overwrite pods $HOSTNAME state=active
    printf .
    sleep 30
    count=$((count+1))
done
echo "$$ Container done."
