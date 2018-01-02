FROM centos:centos7
RUN yum install -y httpd wget curl which bind-utils lsof docker kubernetes-client golang git

COPY *.sh peer-finder.go /
RUN go get k8s.io/apimachinery/pkg/util/sets/
CMD ["go", "run", "/peer-finder.go", "-on-change", "/on-change.sh"]
