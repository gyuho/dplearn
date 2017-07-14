#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/dplearn-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

LOCAL_DIR=/var/lib/etcd
if [[ $(uname) = "Darwin" ]]; then
  echo "Running locally with MacOS"
  LOCAL_DIR=/tmp/etcd
  rm -rf /tmp/etcd
fi

KERAS_DIR=/var/lib/keras
if [[ $(uname) = "Darwin" ]]; then
  echo "Running locally with MacOS"
  KERAS_DIR=${HOME}/.keras
fi
echo KERAS_DIR: ${KERAS_DIR}

# -P
# -p hostPort:containerPort
# -p 80:80
# -p 4200:4200
docker run \
  --rm \
  -it \
  --volume=${LOCAL_DIR}:/var/lib/etcd \
  --volume=${KERAS_DIR}/datasets:/root/.keras/datasets \
  --volume=${KERAS_DIR}/models:/root/.keras/models \
  -p 4200:4200 \
  --ulimit nofile=262144:262144 \
  gcr.io/gcp-dplearn/dplearn:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/dplearn && ./scripts/run/dplearn-gpu.sh"
