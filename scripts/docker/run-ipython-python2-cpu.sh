#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/run-ipython-python2-cpu.sh" ]]; then
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
  --publish 8888:8888 \
  --volume=`pwd`/notebooks:/notebooks \
  --volume=`pwd`/datasets:/root/datasets \
  --volume=${KERAS_DIR}/datasets:/root/.keras/datasets \
  --volume=${KERAS_DIR}/models:/root/.keras/models \
  gcr.io/gcp-dplearn/dplearn:latest-python2-cpu \
  /bin/sh -c "PASSWORD='' ./run_jupyter.sh -y --allow-root --notebook-dir=/notebooks"

<<COMMENT
http://localhost:8888
COMMENT
