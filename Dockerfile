FROM centos:centos7
RUN yum install -y which docker golang git make

ARG GOPATH
ARG HOME
ENV GOPATH=${GOPATH:-/root/go} HOME=${HOME:-/root}

ADD . $GOPATH/src/github.com/beekhof/galera-operator
WORKDIR $GOPATH/src/github.com/beekhof/galera-operator
RUN make install

WORKDIR $GOPATH/src/github.com/beekhof/
RUN rm -rf galera-operator

CMD ["/usr/local/bin/rss-operator", "-alsologtostderr"]
