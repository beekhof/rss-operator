#!/bin/bash

echo "$$ Container stopping..."
kubectl label --overwrite pods $HOSTNAME state=stopping
echo "$$ Container stop done."
