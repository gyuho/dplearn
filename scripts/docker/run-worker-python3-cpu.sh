#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/run-worker-python3-cpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

KERAS_DIR=/var/lib/keras
if [[ $(uname) = "Darwin" ]]; then
  echo "Running locally with MacOS"
  KERAS_DIR=${HOME}/.keras
fi
echo KERAS_DIR: ${KERAS_DIR}

IMAGE_DIR=/tmp
if [[ -z "${IMAGE_DIR}" ]]; then
  IMAGE_DIR=${IMAGE_DIR}
fi
echo IMAGE_DIR: ${IMAGE_DIR}

docker run \
  --rm \
  -it \
  --env CATS_PARAM_PATH=/root/datasets/parameters-cats.npy \
  --net=host \
  --volume=${IMAGE_DIR}:/tmp \
  --volume=`pwd`/datasets:/root/datasets \
  --volume=`pwd`/datasets:/root/datasets \
  --volume=${KERAS_DIR}/datasets:/root/.keras/datasets \
  --volume=${KERAS_DIR}/models:/root/.keras/models \
  gcr.io/gcp-dplearn/dplearn:latest-python3-cpu \
  /bin/sh -c "./scripts/docker/run/worker-python3.sh"

<<COMMENT
docker run \
  --rm \
  -it \
  --net=host \
  gcr.io/gcp-dplearn/dplearn:latest-python3-cpu \
  /bin/sh -c "
curl -L http://localhost:2200/health
"

rm -rf /tmp/etcd
go install -v ./cmd/backend-web-server
backend-web-server -web-port 2200 -queue-port-client 22000 -queue-port-peer 22001 -data-dir /tmp/etcd -logtostderr=true

ETCDCTL_API=3 /etcdctl --endpoints=localhost:22000 get "" --from-key

curl -L http://localhost:2200/cats-request/queue

curl -L http://localhost:2200/cats-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{}'

curl -L http://localhost:2200/cats-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{"bucket" : "/cats-request", "key" : "/cats-request", "value" : ""}'

curl -L http://localhost:2200/cats-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{"bucket" : "/cats-request", "key" : "/cats-request", "value" : "bar"}'

sleep 5s

ETCDCTL_API=3 /etcdctl --endpoints=localhost:22000 get "" --from-key

python ./backend/worker/worker.py http://localhost:2200/cats-request/queue
COMMENT
