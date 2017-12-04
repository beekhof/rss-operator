FROM centos:centos7
RUN yum install -y httpd wget curl which bind-utils lsof docker gdb
RUN yum install -y resource-agents pcs pacemaker
COPY *.sh /
#RUN yum install -y galera

