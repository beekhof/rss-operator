#!/bin/bash

: ${OCF_ROOT=/usr/lib/ocf}
: ${OCF_FUNCTIONS_DIR=${OCF_ROOT}/lib/heartbeat}
. ${OCF_FUNCTIONS_DIR}/ocf-shellfuncs
. ${OCF_FUNCTIONS_DIR}/mysql-common.sh
. container-common.sh

# TODO: Do something sane if galera is already running (might be useful during recovery of the operator)
# SHOW STATUS LIKE 'wsrep_%';

function ocf_log() {
	: Swallow all logging... $*
	return
}

ocf_log info "Detecting replication version"
mysql_common_prepare_dirs

mysql_common_status info
if [ $? = 0 ]; then
	# Its alive
	mysql  -e "SHOW STATUS LIKE 'wsrep_last_%';" | grep wsrep_last_committed | awk '{print $2}'
else
	detect_last_commit 
fi
