#!/bin/bash

: ${OCF_ROOT=/usr/lib/ocf}
: ${OCF_FUNCTIONS_DIR=${OCF_ROOT}/lib/heartbeat}
. ${OCF_FUNCTIONS_DIR}/ocf-shellfuncs
. ${OCF_FUNCTIONS_DIR}/mysql-common.sh
. container-common.sh

ocf_log info "Replicating state from $(gcomm_from_args $*)..."
OCF_RESKEY_enable_creation=false

for peer in $* ; do nslookup $peer; done

if [ ${CHAOS_LEVEL} -gt 2 -a $(( $RANDOM % ${CHAOS_LEVEL} )) = 0 ]; then
	ocf_log info "Monkeys everywhere!!"
	exit 1
fi

chown mysql:mysql $OCF_RESKEY_datadir
mysql_common_prepare_dirs
mysql_common_start "--wsrep-cluster-address=$(gcomm_from_args $*)"
handle_result "replication" $?
