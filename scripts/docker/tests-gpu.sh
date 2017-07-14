#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/tests-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

KERAS_DIR=/var/lib/keras
if [[ $(uname) = "Darwin" ]]; then
  echo "Running locally with MacOS"
  KERAS_DIR=${HOME}/.keras
fi
echo KERAS_DIR: ${KERAS_DIR}

nvidia-docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/dplearn \
  gcr.io/gcp-dplearn/dplearn:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/dplearn && ./scripts/tests/frontend.sh"

nvidia-docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/dplearn \
  gcr.io/gcp-dplearn/dplearn:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/dplearn && ./scripts/tests/go.sh"

nvidia-docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/dplearn \
  --volume=${KERAS_DIR}/datasets:/root/.keras/datasets \
  --volume=${KERAS_DIR}/models:/root/.keras/models \
  gcr.io/gcp-dplearn/dplearn:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/dplearn && ETCD_EXEC=/etcd BACKEND_WEB_SERVER_EXEC=/gopath/bin/backend-web-server ./scripts/tests/python.sh"
