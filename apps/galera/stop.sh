#!/bin/bash

: ${OCF_ROOT=/usr/lib/ocf}
: ${OCF_FUNCTIONS_DIR=${OCF_ROOT}/lib/heartbeat}
. ${OCF_FUNCTIONS_DIR}/ocf-shellfuncs
. ${OCF_FUNCTIONS_DIR}/mysql-common.sh
. container-common.sh

ocf_log info "Stopping galera..."

if [ ${CHAOS_LEVEL} -gt 2 -a $(( $RANDOM % ${CHAOS_LEVEL} )) = 0 ]; then
	ocf_log info "Monkeys everywhere!!"
	exit 1
fi

mysql_common_stop
handle_result "stop" $?

