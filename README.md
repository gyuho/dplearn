## deep

[![Build Status](https://img.shields.io/travis/gyuho/deephardway.svg?style=flat-square)](https://travis-ci.org/gyuho/deephardway)
[![Build Status](https://semaphoreci.com/api/v1/gyuho/deephardway/branches/master/shields_badge.svg)](https://semaphoreci.com/gyuho/deephardway)
[![Godoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/gyuho/deephardwayhardway)

### What is `deep`?

It is a set of small projects on [Deep Learning](https://en.wikipedia.org/wiki/Deep_learning).

### System Overview

- `frontend` implements user-facing UI on web browsers, sends user requests to `backend`.
- `backend/web` schedules user requests on the queue service.
- `backend/deep` fetches the list of jobs, process, and the writes results back to the queue.
- `backend/web` gets notified with [watch API](https://godoc.org/github.com/coreos/etcd/clientv3#Watcher) when the job is done, and returns results back to users.

Notes:

- `pkg/etcd-queue` implements the queue service with single-node [etcd](https://github.com/coreos/etcd) cluster.
- It's a **single node** cluster, with no fault tolerance.
- In production, I would deploy separate 5-node etcd cluster.
- For Tensorflow, I would deploy [Tensorflow/serving](https://tensorflow.github.io/serving/).


### Development

Root [`Dockerfile`](./Dockerfile) contains *everything* needed for
development, built upon https://gcr.io/tensorflow/tensorflow. Container
registry can be found at https://gcr.io/deephardway/github-deep.