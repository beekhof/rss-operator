#!/bin/bash

echo "$$ Replicating state from $*..." 1>&2
kubectl label --overwrite pods $HOSTNAME state=active

: ${OCF_ROOT=/usr/lib/ocf}
: ${OCF_FUNCTIONS_DIR=${OCF_ROOT}/lib/heartbeat}
. ${OCF_FUNCTIONS_DIR}/ocf-shellfuncs
. ${OCF_FUNCTIONS_DIR}/mysql-common.sh

OCF_RESKEY_enable_creation=false

# It is common for some galera instances to store
# check user that can be used to query status
# in this file
if [ -f "/etc/sysconfig/clustercheck" ]; then
    . /etc/sysconfig/clustercheck
elif [ -f "/etc/default/clustercheck" ]; then
    . /etc/default/clustercheck
fi

kubectl label --overwrite pods $HOSTNAME state=active

mysql_common_prepare_dirs

set -x
mysql_common_start "--wsrep-cluster-address=gcomm://$1"
set +x

echo "$$ Replication complete."
