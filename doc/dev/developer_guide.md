# Developer guide

This document explains how to setup your dev environment. 

## Fetch dependency

We use [glide](https://github.com/Masterminds/glide) to manage dependency.
Install dependency if you haven't:

```
$ glide install --strip-vendor
```

## How to build

We provide a script to build binaries, build image, and push image to registry.

Required tools:
- Docker
- Go 1.8+
- git

Build in project root dir:

```
( under $GOPATH/src/github.com/beekhof/rss-operator/ )
$ make IMAGE=quay.io/your_username/rss-operator:latest publish
```
