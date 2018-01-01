#!/bin/bash

: ${OCF_ROOT=/usr/lib/ocf}
: ${OCF_FUNCTIONS_DIR=${OCF_ROOT}/lib/heartbeat}
. ${OCF_FUNCTIONS_DIR}/ocf-shellfuncs
. ${OCF_FUNCTIONS_DIR}/mysql-common.sh

OCF_RESKEY_enable_creation=true

# It is common for some galera instances to store
# check user that can be used to query status
# in this file
if [ -f "/etc/sysconfig/clustercheck" ]; then
    . /etc/sysconfig/clustercheck
elif [ -f "/etc/default/clustercheck" ]; then
    . /etc/default/clustercheck
fi

kubectl label --overwrite pods $HOSTNAME state=active
echo "$$ Service seeded."

sed -ie 's/^\(safe_to_bootstrap:\) 0/\1 1/' ${OCF_RESKEY_datadir}/grastate.dat

mysql_common_prepare_dirs
mysql_common_start "--wsrep-cluster-address=gcomm://"
