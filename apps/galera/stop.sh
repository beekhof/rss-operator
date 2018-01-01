#!/bin/bash

echo "$$ Galera stopping..."

: ${OCF_ROOT=/usr/lib/ocf}
: ${OCF_FUNCTIONS_DIR=${OCF_ROOT}/lib/heartbeat}
. ${OCF_FUNCTIONS_DIR}/ocf-shellfuncs
. ${OCF_FUNCTIONS_DIR}/mysql-common.sh

mysql_common_stop
kubectl label --overwrite pods $HOSTNAME state=stopping

echo "$$ Galera stop done."

