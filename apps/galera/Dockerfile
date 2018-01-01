FROM centos:centos7
RUN yum install -y httpd wget curl which bind-utils lsof docker kubernetes-client golang git
RUN yum install -y centos-release-openstack-ocata
RUN yum install -y mariadb-server-galera rsync  resource-agents

COPY run.sh sequence.sh start.sh stop.sh seed.sh check.sh peer-finder.go /
RUN go get k8s.io/apimachinery/pkg/util/sets/
RUN rm -f /etc/my.cnf.d/auth_gssapi.cnf
CMD ["/run.sh", "-x"]
