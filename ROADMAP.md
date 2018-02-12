# Roadmap

This document defines a high level roadmap for the rss operator development.

The dates below should not be considered authoritative, but rather indicative of the projected timeline of the project.


### 2018

#### Cleanups

- Remove last traces of etcd and galera references and terminology

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


#### Stability/Reliability

- Additional unit tests
- e2e testing
