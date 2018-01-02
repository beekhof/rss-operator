#!/bin/bash

: ${OCF_ROOT=/usr/lib/ocf}
: ${OCF_FUNCTIONS_DIR=${OCF_ROOT}/lib/heartbeat}
. ${OCF_FUNCTIONS_DIR}/ocf-shellfuncs
. ${OCF_FUNCTIONS_DIR}/mysql-common.sh
. container-common.sh

ocf_log info "Checking status..."
mysql_common_status err
handle_result "status" $?
