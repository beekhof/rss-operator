#!/bin/bash

: ${OCF_ROOT=/usr/lib/ocf}
: ${OCF_FUNCTIONS_DIR=${OCF_ROOT}/lib/heartbeat}
. ${OCF_FUNCTIONS_DIR}/ocf-shellfuncs
. ${OCF_FUNCTIONS_DIR}/mysql-common.sh
. container-common.sh

OCF_RESKEY_enable_creation=true

ocf_log info "Seeding application"
if [ -e ${OCF_RESKEY_datadir}/grastate.dat ]; then
    sed -ie 's/^\(safe_to_bootstrap:\) 0/\1 1/' ${OCF_RESKEY_datadir}/grastate.dat
fi

mysql_common_prepare_dirs
mysql_common_start "--wsrep-cluster-address=gcomm://"
rc=$?

mysql -B < database.sql

handle_result "seed" $rc
