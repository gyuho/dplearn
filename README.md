## dplearn

[![Go Report Card](https://goreportcard.com/badge/github.com/gyuho/dplearn?style=flat-square)](https://goreportcard.com/report/github.com/gyuho/dplearn)
[![Build Status](https://img.shields.io/travis/gyuho/dplearn.svg?style=flat-square)](https://travis-ci.org/gyuho/dplearn)
[![Build Status](https://semaphoreci.com/api/v1/gyuho/dplearn/branches/master/shields_badge.svg)](https://semaphoreci.com/gyuho/dplearn)
[![Godoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/gyuho/dplearnhardway)

Learn Deep Learning The Hard Way.

It is a set of small projects on [Deep Learning](https://en.wikipedia.org/wiki/Deep_learning).


### System Overview

<img src="./architecture.png" alt="architecture" width="620">

- [`frontend`](https://github.com/gyuho/dplearn/tree/master/frontend) implements user-facing UI, sends user requests to [`backend/*`](https://github.com/gyuho/dplearn/tree/master/backend).
- [`backend/web`](https://github.com/gyuho/dplearn/tree/master/backend/web) schedules user requests on [`pkg/etcd-queue`](https://github.com/gyuho/dplearn/tree/master/pkg/etcd-queue).
- [`backend/worker`](https://github.com/gyuho/dplearn/tree/master/backend/worker) processes jobs from queue, and writes back the results.
- `backend/web` gets notified with [Watch API](https://godoc.org/github.com/coreos/etcd/clientv3#Watcher) when the job is done, and returns results back to users.
- Data serialization from `frontend` to `backend/web` is defined in [`backend/web.Request`](https://github.com/gyuho/dplearn/blob/master/backend/web/handler.go) and [`frontend/app/request.service.Request`](https://github.com/gyuho/dplearn/blob/master/frontend/app/request.service.ts).
- Data serialization from `backend/web` to `frontend` is defined in [`pkg/etcd-queue.Item`](https://github.com/gyuho/dplearn/blob/master/pkg/etcd-queue/item.go) and [`frontend/app/request.service.Item`](https://github.com/gyuho/dplearn/blob/master/frontend/app/request.service.ts).
- Data serialization between `backend/web` and `backend/worker` is defined in [`pkg/etcd-queue.Item`](https://github.com/gyuho/dplearn/blob/master/pkg/etcd-queue/item.go) and [`backend/worker/worker.py`](https://github.com/gyuho/dplearn/blob/master/backend/worker/worker.py).

Notes:

- **Why is the queue service needed?** To process concurrent users requests. Worker has limited resources. Requests can be serialized into the queue, so that worker performance is maximized for each task.
- **Why Go?** To natively use [`embedded etcd`](https://github.com/coreos/etcd/tree/master/embed).
- **Why etcd?** It has *really great* [Watch API implementation](https://godoc.org/github.com/coreos/etcd/clientv3#Watcher). `pkg/etcd-queue` uses Watch API to stream updates to `backend/worker` and `frontend`. This minimizes TCP socket creation and slow TCP starts (e.g. streaming vs. polling). *TODO: use streaming to broadcast more detailed job status.*
- **How is this deployed?** Everything is run in *one container*, due to limited budget on public serving. In production, I would use: [Tensorflow/serving](https://tensorflow.github.io/serving/) to serve the pre-trained models, distributed [`etcd`](https://github.com/coreos/etcd) for higher availability and master-worker election.


### Development Workflow

To update dependencies:

```bash
# update 'Gopkg.toml' and then (for local development)
./scripts/dep/go.sh

# update 'cmd/gen-package-json' and then (for local development)
./scripts/dep/frontend.sh

# Docker builds pull down all dependencies from scratch
```

To update [`Dockerfile`](Dockerfile):

```bash
# update '*/Dockerfile.yaml', 'cmd/gen-dockerfiles' and then
./scripts/docker/gen.sh
```

To build Docker container image:

```bash
./scripts/docker/build-cpu.sh
./scripts/docker/build-gpu.sh
```

To run tests:

```bash
./scripts/tests/frontend.sh
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

To run `dplearn` application (backend, web UI, worker) locally:

```bash
./scripts/docker/dplearn-cpu.sh
./scripts/docker/dplearn-gpu.sh
```

To deploy IPython Notebook and `dplearn` application on Google Cloud Platform:

```bash
GCP_KEY_PATH=/etc/gcp-key-dplearn.json ./scripts/gcp/ubuntu-gpu.gcp.sh

# create a Google Cloud Platform Compute Engine VM with a start-up script
# to provision GPU, init system, reverse proxy, and others
# (see ./scripts/gcp/ubuntu-gpu.ansible.sh for more detail)
```
