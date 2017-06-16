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

- **Why is the queue service needed?** Users requests are concurrent, while worker has only limited computing power. Requests should be serialized into the queue, so that worker performance is maximized for each queue item.
- **How is this deployed?** I have limited budget on public serving, thus everything is run in *one container*. In production, [etcd](https://github.com/coreos/etcd) can be distributed for higher availability, and [Tensorflow/serving](https://tensorflow.github.io/serving/) can serve the pre-trained models.


### Development Workflow

To update dependencies:

```bash
# update 'Gopkg.toml',
# this is necessary, whether run with/without container
# since docker build copies 'vendor' directory to the container image
./scripts/dep/go-dep.sh

# update 'cmd/gen-package-json' and then
./scripts/dep/package-json.sh

# not needed if run inside container
# docker build fetches all frontend dependencies
./scripts/dep/frontend.sh
```

To update [`Dockerfile`](Dockerfile):

```bash
# after updating '*/Dockerfile.yaml', 'cmd/gen-dockerfiles'
./scripts/docker/gen.sh
```

To build Docker container image:

```bash
./scripts/docker/build-cpu.sh
./scripts/docker/build-gpu.sh
```

To run tests:

```bash
./scripts/tests/go.sh
ETCD_EXEC=/etcd ./scripts/tests/python.sh

# run tests inside container
./scripts/docker/tests-cpu.sh
./scripts/docker/tests-gpu.sh
```

To run [IPython Notebook](https://ipython.org/notebook.html) locally:

```bash
./scripts/docker/ipython-cpu.sh
./scripts/docker/ipython-gpu.sh

# Add 'source activate r' to run with R kernel.
# It uses Tensorflow base image, so need to
# manually configure the R Anaconda workspace.
ACTIVATE_COMMAND="source activate r &&" ./scripts/docker/ipython-cpu.sh
```

To run `deephardway` application (backend, web UI, worker) locally:

```bash
./scripts/docker/deephardway-cpu.sh
./scripts/docker/deephardway-gpu.sh
```

To deploy `deephardway` application to Google Cloud Platform:

```bash
GCP_KEY_PATH=/etc/gcp-key-deephardway.json ./scripts/gcp/create-instance.sh

# this creates a Google Cloud Platform Compute Engine VM with a start-up
# script that provisions GPUs, init systems, reverse proxy, and others
# see ./scripts/gcp/ubuntu-gpu.ansible.sh for more detail
```
