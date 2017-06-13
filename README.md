## deephardway

[![Build Status](https://img.shields.io/travis/gyuho/deephardway.svg?style=flat-square)](https://travis-ci.org/gyuho/deephardway)
[![Build Status](https://semaphoreci.com/api/v1/gyuho/deephardway/branches/master/shields_badge.svg)](https://semaphoreci.com/gyuho/deephardway)
[![Godoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/gyuho/deephardwayhardway)

Learn Deep Learning The Hard Way.

It is a set of small projects on [Deep Learning](https://en.wikipedia.org/wiki/Deep_learning).

### System Overview

<img src="./architecture.png" alt="architecture" width="620">

- [`frontend`](https://github.com/gyuho/deephardway/tree/master/frontend) implements user-facing UI, sends user requests to [`backend/*`](https://github.com/gyuho/deephardway/tree/master/backend).
- [`backend/web`](https://github.com/gyuho/deephardway/tree/master/backend/web) schedules user requests on [`pkg/etcd-queue`](https://github.com/gyuho/deephardway/tree/master/pkg/etcd-queue) service on top of [etcd](https://github.com/coreos/etcd).
- [`backend/etcd-python`](https://github.com/gyuho/deephardway/tree/master/backend/etcd-python) fetches the list of jobs, and the writes results back to the queue.
- [`backend/worker`](https://github.com/gyuho/deephardway/tree/master/backend/worker) processes/computes the list of jobs, and writes results back to queue.
- [`backend/web`](https://github.com/gyuho/deephardway/tree/master/backend/web) gets notified with [watch API](https://godoc.org/github.com/coreos/etcd/clientv3#Watcher) when the job is done, and returns results back to users.

Notes:

- **Why is the queue service needed?** Users requests are concurrent, while worker has only limited computing power. Requests are serialized into the queue, so that worker performance can be maximized for each job.
- **How is this deployed?** This is a simple project for learning purposes, mostly running in my local machine. With limited budget on public serving, everything is running in one container. In production, `etcd` should be distributed as a separate cluster, and [Tensorflow/serving](https://tensorflow.github.io/serving/) should be considered. Root [`Dockerfile`](./Dockerfile) container image could be broken into smaller ones.
