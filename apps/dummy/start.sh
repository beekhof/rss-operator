#!/bin/bash

echo "$$ Replicating state from $*..." 1>&2
touch /.running
kubectl label --overwrite pods $HOSTNAME state=active
echo "$$ Replication complete."
