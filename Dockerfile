FROM centos:centos7
RUN yum install -y which docker golang git make

ADD . /root/go/src/github.com/beekhof/galera-operator
WORKDIR /root/go/src/github.com/beekhof/galera-operator
RUN make install
WORKDIR /root/go/src/github.com/beekhof/
RUN rm -rf galera-operator
CMD ["/usr/local/bin/rss-operator"]
