FROM centos:centos7
RUN yum install -y curl which bind-utils docker kubernetes-client golang git sudo
RUN yum install -y centos-release-openstack-ocata
RUN yum install -y mariadb-server-galera rsync resource-agents

COPY *.sh database.sql peer-finder.go /

ENV GOPATH /go
RUN go get k8s.io/apimachinery/pkg/util/sets/
RUN rm -f /etc/my.cnf.d/auth_gssapi.cnf
CMD ["go", "run", "/peer-finder.go", "-on-change", "/on-change.sh"]
USER mysql
