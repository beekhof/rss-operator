#!/bin/bash

touch /.running
kubectl label --overwrite pods $HOSTNAME state=active
echo "$$ Service seeded."
