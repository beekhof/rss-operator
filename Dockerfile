FROM centos:centos7
RUN yum install -y httpd wget curl which bind-utils lsof docker kubernetes-client golang git glide make

ADD . $HOME/go/src/github.com/beekhof/galera-operator
WORKDIR $HOME/go/src/github.com/beekhof/galera-operator
RUN pwd
RUN ls -al
RUN make operator
CMD ["/usr/local/bin/rss-operator"]
