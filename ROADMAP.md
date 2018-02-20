# Roadmap

This document defines a high level roadmap for the rss operator development.

The dates below should not be considered authoritative, but rather indicative of the projected timeline of the project.


### 2018

#### Documentation

- Need user facing documentation.  Much of the current documentation in
 [doc/user](doc/user) was carried over from the etcd-operator and is in need of
 revision.

- Need a Galera demo

#### Cleanups

- Remove lingering traces of etcd and galera references and terminology

- Check conformance with the [controller pattern](https://github.com/kubernetes/community/blob/master/contributors/devel/controllers.md)

- Check conformance with [logging conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md)

#### Features

- Metrics and logging?
  - Expose operator metrics
      - How many clusters it manages
      - How many actions it does
   - Expose the running status of the cluster
      - cluster size, version
   - Expose errors 
     -  bad version, bad cluster size, dead cluster

- Security

- Create a Rabbit MQ sample app

- Decide on an upgrade story

- Implement backup/restore (Great for demos where there is no shared storage)

- Review TLS code/docs for relevance

- Create a demo

#### Stability/Reliability

- Additional unit tests

- e2e testing
