#!/bin/bash

count=1
echo "$$ Container starting..."
kubectl label --overwrite pods $HOSTNAME state=active
echo "$$ Container done."
