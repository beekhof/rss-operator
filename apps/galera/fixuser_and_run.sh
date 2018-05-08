# https://blog.openshift.com/jupyter-on-openshift-part-6-running-as-an-assigned-user-id/
#
# If runing with OpenShift's nonroot or anyuid SCC is not an option,
# then we at least want `whoami` and `id` to return something sane as
# most codebases assume that they will
# 

if [ `id -u` -ge 10000 -a "x$POD_USER" != x -a -w /etc/passwd ]; then
    cat /etc/passwd | sed -e "s/^$POD_USER:/builder:/" > /tmp/passwd
    echo "$POD_USER:x:`id -u`:`id -g`:,,,:/home/$POD_USER:/bin/bash" >> /tmp/passwd
    cat /tmp/passwd > /etc/passwd
    rm /tmp/passwd
fi

#
# Now run the configured command
#
echo Running: $*
shift
$*
