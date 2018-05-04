#!/bin/bash

: ${OCF_ROOT=/usr/lib/ocf}
: ${OCF_FUNCTIONS_DIR=${OCF_ROOT}/lib/heartbeat}
. ${OCF_FUNCTIONS_DIR}/ocf-shellfuncs
. ${OCF_FUNCTIONS_DIR}/mysql-common.sh
. container-common.sh

set -x
whoami
OCF_RESKEY_enable_creation=true

load_sql=0
ocf_log info "Seeding application"
if [ -e ${OCF_RESKEY_datadir}/grastate.dat ]; then
    sed -ie 's/^\(safe_to_bootstrap:\) 0/\1 1/' ${OCF_RESKEY_datadir}/grastate.dat
else
	load_sql=1
fi

if [ ${CHAOS_MODULO} -gt 2 -a $(( $RANDOM % ${CHAOS_MODULO} )) = 0 ]; then
	ocf_log info "Monkeys everywhere!!"
	exit 1
fi

mysql_common_prepare_dirs
mysql_common_start "--wsrep-cluster-address=gcomm://"
handle_failure "seed" $?

#MYSQL_PWD="$MYSQL_PASSWORD" mysql -h 127.0.0.1 -u $MYSQL_USER -D $MYSQL_DATABASE -e 'SELECT 1'

# Leverage knowledge from https://github.com/sclorg/mysql-container/blob/master/root-common/usr/share/container-scripts/mysql/common.sh
MYSQL_VERSION=$(mysqladmin -V | sed 's/Linux.*//' | tr -d 'a-zA-Z,_-' | awk '{print $2}')
mysql_flags="$MYSQL_OPTIONS_LOCAL -u root"
function log_info() {
    ocf_log info $*
}

$MYSQL $mysql_flags -D "${MYSQL_DATABASE}" -e 'select 1'
if [ $? = 0 ]; then
    # The database has already been created and initialized
    handle_result "seed" $rc
fi

if [ -v MYSQL_USER ]; then
    log_info "Creating user specified by MYSQL_USER (${MYSQL_USER}) ..."
    $MYSQL $mysql_flags <<EOSQL
    CREATE USER '${MYSQL_USER}'@'%' IDENTIFIED BY '${MYSQL_PASSWORD}';
EOSQL
    handle_failure "seed: create user" $?
fi


if [ -v MYSQL_DATABASE ]; then
    log_info "Creating database ${MYSQL_DATABASE} ..."
    $MYSQL $mysql_flags <<EOSQL
    CREATE DATABASE ${MYSQL_DATABASE};
EOSQL
    handle_failure "seed: create database" $?

    if [ -v MYSQL_USER ]; then
	log_info "Granting privileges to user ${MYSQL_USER} for ${MYSQL_DATABASE} ..."
	$MYSQL $mysql_flags <<EOSQL
      GRANT ALL ON \`${MYSQL_DATABASE}\`.* TO '${MYSQL_USER}'@'%' ;
      FLUSH PRIVILEGES ;
EOSQL
	handle_failure "seed: grant privileges" $?
    fi
fi

if [ -v MYSQL_ROOT_PASSWORD ]; then
    log_info "Setting password for MySQL root user ..."
    # for 5.6 and lower we use the trick that GRANT creates a user if not exists
    # because IF NOT EXISTS clause does not exist in that versions yet
    if [[ "$MYSQL_VERSION" > "5.6" ]] ; then
	$MYSQL $mysql_flags <<EOSQL
        CREATE USER IF NOT EXISTS 'root'@'%';
EOSQL
    fi
    $MYSQL $mysql_flags <<EOSQL
    GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' IDENTIFIED BY '${MYSQL_ROOT_PASSWORD}' WITH GRANT OPTION;
EOSQL
else
    if [ "$MYSQL_VERSION" \> "5.6" ] ; then
	mysql $mysql_flags <<EOSQL
      DROP USER IF EXISTS 'root'@'%';
      FLUSH PRIVILEGES;
EOSQL
    else
	# In 5.6 and lower, We do GRANT and DROP USER to emulate a DROP USER IF EXISTS statement
	# http://bugs.mysql.com/bug.php?id=19166
	mysql $mysql_flags <<EOSQL
      GRANT USAGE ON *.* TO 'root'@'%';
      DROP USER 'root'@'%';
      FLUSH PRIVILEGES;
EOSQL
    fi
fi

# TODO: Drop root@localhost user or leave for the admin to decide?
log_info 'Initialization finished'

# TODO: Remove - Tiny DB for testing
$MYSQL  $mysql_flags -D "${MYSQL_DATABASE}" -B < database.sql

handle_result "seed" 0
