#!/bin/bash

kubectl label --overwrite pods $HOSTNAME state=active
echo "$$ Service seeded."
