FROM centos:centos7
RUN yum install -y curl which bind-utils docker kubernetes-client golang git
RUN yum install -y centos-release-openstack-ocata
RUN yum install -y mariadb-server-galera rsync resource-agents

COPY *.sh peer-finder.go /

RUN go get k8s.io/apimachinery/pkg/util/sets/
RUN rm -f /etc/my.cnf.d/auth_gssapi.cnf
CMD ["/run.sh", "-x"]
