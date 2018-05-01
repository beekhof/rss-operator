if [ x${CHAOS_MODULO} = x ]; then
	export CHAOS_MODULO=1
fi

# su [options...] [-] [user [args...]]
#
#      -c command, --command=command
#              Pass command to the shell with the -c option.
#
#       --session-command=command
#              Same as -c but do not create a new session (discouraged).
#
#       -f, --fast
#              Pass -f to the shell which may or may not be useful depending on the shell.
#
#       -g, --group=group
#              specify the primary group, this option is allowed for root user only
#
#       -G, --supp-group=group
#              Specify a supplemental group.  This option is available to the root user only.  The first specified supplementary group is also used as a primary group if the option --group is unspecified.
#
#       -, -l, --login
#              Starts the shell as login shell with an environment similar to a real login:
#
#                 o      clears all environment variables except for TERM
#
#                 o      initializes the environment variables HOME, SHELL, USER, LOGNAME, PATH
#
#                 o      changes to the target user's home directory
#
#                 o      sets argv[0] of the shell to '-' in order to make the shell a login shell
#
#       -m, -p, --preserve-environment
#              Preserves the whole environment, ie does not set HOME, SHELL, USER nor LOGNAME.  The option is ignored if the option --login is specified.
#
#       -s SHELL, --shell=SHELL
#              Runs the specified shell instead of the default.  The shell to run is selected according to the following rules in order:
#
#                 o      the shell specified with --shell
#
#                 o      The shell specified in the environment variable SHELL if the --preserve-environment option is used.
#
#                 o      the shell listed in the passwd entry of the target user
#
#                 o      /bin/sh
#
#              If the target user has a restricted shell (i.e. not listed in /etc/shells) the --shell option and the SHELL environment variables are ignored unless the calling user is root.
#
#       --help Display help text and exit.
#
#       --version
#              Display version information and exit.

function su() {
    options=""
    su_user="root"
    su_cmd="true"
    for arg in $*; do
        case $arg in
            -c|--command|--session-command) su_cmd="$1"; shift; shift;;
            -|-l|--login) options="$options --login"; shift;;
            -m|-p|--preserve-environment) options="$options --preserve-environment"; shift;;
            -s|--shell|-g|--group) options="$options $arg $1"; shift; shift;;
            -f|--fast) shift;;
            -G|--supp-group) shift; shift;;
            --help|--version) return 0;;
            *) if [ "$su_cmd" = "true" ]; then su_user=$arg; fi; shift;;
        esac
    done
    # Galera only wants to check if one directory is writable as mysql
    # In the container we're always mysql, so no need for sudo at all
    #
    # Anyone copying this that needs an actual bridge to sudo would want:
    #
    #     sudo $options $su_user "$su_cmd"
    #
    $su_cmd
}

function gcomm_from_args() {
	gcomm="gcomm://"

	while [ ! -z $2 ]; do
		gcomm="$gcomm$1,"
		shift
	done
	if [ ! -z $1 ]; then
		gcomm="$gcomm$1"
	fi
	echo $gcomm
}

function handle_result() {
	action=$1; shift
	if [ $1 = 0 ]; then
		kubectl label --overwrite pods $HOSTNAME state=$action
		ocf_log info "$action complete."
	else
		ocf_log info "$action failed: $1"
	fi

	exit $1
}

# Verbatim from the /usr/lib/ocf/resource.d/heartbeat/galera agent

function detect_last_commit() {
	local last_commit
	local recover_args="--defaults-file=$OCF_RESKEY_config \
	                    --pid-file=$OCF_RESKEY_pid \
	                    --socket=$OCF_RESKEY_socket \
	                    --datadir=$OCF_RESKEY_datadir \
	                    --user=$OCF_RESKEY_user"
	local recovery_file_regex='s/.*WSREP\:.*position\s*recovery.*--log_error='\''\([^'\'']*\)'\''.*/\1/p'
	local recovered_position_regex='s/.*WSREP\:\s*[R|r]ecovered\s*position.*\:\(.*\)\s*$/\1/p'

	# codership/galera#354
	# Some ungraceful shutdowns can leave an empty gvwstate.dat on
	# disk. This will prevent galera to join the cluster if it is
	# configured to attempt PC recovery. Removing that file makes the
	# node fall back to the normal, unoptimized joining process.
	if [ -f ${OCF_RESKEY_datadir}/gvwstate.dat ] && \
	   [ ! -s ${OCF_RESKEY_datadir}/gvwstate.dat ]; then
	    ocf_log warn "empty ${OCF_RESKEY_datadir}/gvwstate.dat detected, removing it to prevent PC recovery failure at next restart"
	    rm -f ${OCF_RESKEY_datadir}/gvwstate.dat
	fi

	ocf_log info "attempting to detect last commit version by reading ${OCF_RESKEY_datadir}/grastate.dat"
	last_commit="$(cat ${OCF_RESKEY_datadir}/grastate.dat | sed -n 's/^seqno.\s*\(.*\)\s*$/\1/p')"
	if [ -z "$last_commit" ] || [ "$last_commit" = "-1" ]; then
	    local tmp=$(mktemp)
	    chown $OCF_RESKEY_user:$OCF_RESKEY_group $tmp

	    # if we pass here because grastate.dat doesn't exist,
	    # try not to bootstrap from this node if possible
	    if [ ! -f ${OCF_RESKEY_datadir}/grastate.dat ]; then
	        set_no_grastate
	    fi

	    ocf_log info "now attempting to detect last commit version using 'mysqld_safe --wsrep-recover'"

	    ${OCF_RESKEY_binary} $recover_args --wsrep-recover --log-error=$tmp 1>&2

	    last_commit="$(cat $tmp | sed -n $recovered_position_regex | tail -1)"
	    if [ -z "$last_commit" ]; then
	        # Galera uses InnoDB's 2pc transactions internally. If
	        # server was stopped in the middle of a replication, the
	        # recovery may find a "prepared" XA transaction in the
	        # redo log, and mysql won't recover automatically

	        local recovery_file="$(cat $tmp | sed -n $recovery_file_regex)"
	        if [ -e $recovery_file ]; then
	            cat $recovery_file | grep -q -E '\[ERROR\]\s+Found\s+[0-9]+\s+prepared\s+transactions!' 2>/dev/null
	            if [ $? -eq 0 ]; then
	                # we can only rollback the transaction, but that's OK
	                # since the DB will get resynchronized anyway
	                ocf_log warn "local node <${NODENAME}> was not shutdown properly. Rollback stuck transaction with --tc-heuristic-recover"
	                ${OCF_RESKEY_binary} $recover_args --wsrep-recover \
	                                     --tc-heuristic-recover=rollback --log-error=$tmp 2>/dev/null

	                last_commit="$(cat $tmp | sed -n $recovered_position_regex | tail -1)"
	                if [ ! -z "$last_commit" ]; then
	                    ocf_log warn "State recovered. force SST at next restart for full resynchronization"
	                    rm -f ${OCF_RESKEY_datadir}/grastate.dat
	                    # try not to bootstrap from this node if possible
	                    set_no_grastate
	                fi
	            fi
	        fi
	    fi
	    rm -f $tmp
	fi

	if [ ! -z "$last_commit" ]; then
	    ocf_log info "Last commit version found:  $last_commit"
	    set_last_commit $last_commit
	    return $OCF_SUCCESS
	else
	    ocf_exit_reason "Unable to detect last known write sequence number"
	    clear_last_commit
	    return $OCF_ERR_GENERIC
	fi
}

# OCF and agent Overrides

NODENAME=$HOSTNAME

function ocf_log() {

    if
      [ $# -lt 2 ]
    then
      ocf_log err "Not enough arguments [$#] to ocf_log."
    fi
    __OCF_PRIO="$1"
    shift
    __OCF_MSG="$*"

    case "${__OCF_PRIO}" in
      crit) __OCF_PRIO="CRIT";;
      err)  __OCF_PRIO="ERROR";;
      warn) __OCF_PRIO="WARNING";;
      info) __OCF_PRIO="INFO";;
      debug)__OCF_PRIO="DEBUG";;
      *)    __OCF_PRIO=`echo ${__OCF_PRIO}| tr '[a-z]' '[A-Z]'`;;
    esac

    if [ "${__OCF_PRIO}" != "DEBUG" ]; then
            echo "$HOSTNAME[$$] ${__OCF_PRIO}: $__OCF_MSG" 2>&1
    fi
}

function ocf_exit_reason() {
    # No argument is likely not intentional.
    # Just one argument implies a printf format string of just "%s".
    # "Least surprise" in case some interpolated string from variable
    # expansion or other contains a percent sign.
    # More than one argument: first argument is going to be the format string.
    case $# in
    0)      ocf_log err "Not enough arguments to ocf_log_exit_msg." ;;
    1)      fmt="%s" ;;

    *)      fmt=$1
            shift
            case $fmt in
            *%*) : ;; # ok, does look like a format string
            *) ocf_log err "Does not look like format string: [$fmt]";;
            esac ;;
    esac

    msg=$(printf "${fmt}" "$@")

	kubectl annotate --overwrite pods $HOSTNAME last_error="$msg"
}

function set_no_grastate() {
	return
}

function set_last_commit() {
	echo "$1"
}

function clear_last_commit() {
	echo "0"
}


# It is common for some galera instances to store
# check user that can be used to query status
# in this file
if [ -f "/etc/sysconfig/clustercheck" ]; then
    . /etc/sysconfig/clustercheck
elif [ -f "/etc/default/clustercheck" ]; then
    . /etc/default/clustercheck
fi

