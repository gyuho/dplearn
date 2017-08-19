#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/python2-cpu-worker-run.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

KERAS_DIR=/var/lib/keras
if [[ $(uname) = "Darwin" ]]; then
  echo "Running locally with MacOS"
  KERAS_DIR=${HOME}/.keras
fi
echo KERAS_DIR: ${KERAS_DIR}

docker run \
  --rm \
  -it \
  --net=host \
  --volume=${KERAS_DIR}/datasets:/root/.keras/datasets \
  --volume=${KERAS_DIR}/models:/root/.keras/models \
  gcr.io/gcp-dplearn/dplearn:latest-python2-cpu \
  /bin/sh -c "./scripts/run/worker.sh"

<<COMMENT
docker run \
  --rm \
  -it \
  --net=host \
  gcr.io/gcp-dplearn/dplearn:latest-python2-cpu \
  /bin/sh -c "
curl -L http://localhost:2200/health
"

rm -rf /tmp/etcd
go install -v ./cmd/backend-web-server
backend-web-server -web-port 2200 -queue-port-client 22000 -queue-port-peer 22001 -data-dir /tmp/etcd -logtostderr=true

ETCDCTL_API=3 /etcdctl --endpoints=localhost:22000 get "" --from-key

curl -L http://localhost:2200/cats-vs-dogs-request/queue

curl -L http://localhost:2200/cats-vs-dogs-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{}'

curl -L http://localhost:2200/cats-vs-dogs-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{"bucket" : "/cats-vs-dogs-request", "key" : "/cats-vs-dogs-request", "value" : ""}'

curl -L http://localhost:2200/cats-vs-dogs-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{"bucket" : "/cats-vs-dogs-request", "key" : "/cats-vs-dogs-request", "value" : "bar"}'

sleep 5s

ETCDCTL_API=3 /etcdctl --endpoints=localhost:22000 get "" --from-key

python ./backend/worker/worker.py http://localhost:2200/word-predict-request/queue
python ./backend/worker/worker.py http://localhost:2200/cats-vs-dogs-request/queue
COMMENT
