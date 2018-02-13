#!/bin/bash

echo "$$ Container stopping..."
rm -f /.running
kubectl label --overwrite pods $HOSTNAME state=stopping
echo "$$ Container stop done."
