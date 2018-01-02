FROM centos:centos7
RUN yum install -y httpd wget curl which bind-utils lsof docker kubernetes-client golang git

COPY run.sh sequence.sh start.sh stop.sh seed.sh check.sh on-change.sh peer-finder.go /
RUN go get k8s.io/apimachinery/pkg/util/sets/
CMD ["/run.sh", "-x"]
