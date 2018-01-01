#!/bin/bash

count=1
echo "$$ Replicating state from $*..." 1>&2
kubectl label --overwrite pods $HOSTNAME state=active
echo "$$ Replication complete."
