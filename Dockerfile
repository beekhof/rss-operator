FROM centos:centos7
RUN yum install -y httpd wget curl which docker golang git make

ADD . /root/go/src/github.com/beekhof/galera-operator
WORKDIR /root/go/src/github.com/beekhof/galera-operator
RUN pwd
RUN ls -al
RUN make operator
CMD ["/usr/local/bin/rss-operator"]
