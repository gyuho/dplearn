#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/ipython-cpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

if [[ "${ACTIVATE_COMMAND}" ]]; then
  echo ACTIVATE_COMMAND is defined: \""${ACTIVATE_COMMAND}"\"
else
  echo ACTIVATE_COMMAND is not defined
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
# -p 2200:2200
# -p 4200:4200
docker run \
  --privileged \
  --rm \
  -it \
  --volume=`pwd`/notebooks:/gopath/src/github.com/gyuho/dplearn/notebooks \
  --volume=${KERAS_DIR}/datasets:/root/.keras/datasets \
  --volume=${KERAS_DIR}/models:/root/.keras/models \
  -p 8888:8888 \
  gcr.io/gcp-dplearn/dplearn:latest-cpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/dplearn && ${ACTIVATE_COMMAND} PASSWORD='' ./run_jupyter.sh -y --allow-root --notebook-dir=./notebooks"

<<COMMENT
./scripts/docker/ipython-cpu.sh
ACTIVATE_COMMAND="source activate r &&" ./scripts/docker/ipython-cpu.sh

source activate r && PASSWORD='' ./run_jupyter.sh -y --allow-root --notebook-dir=./notebooks
source activate py36 && PASSWORD='' ./run_jupyter.sh -y --allow-root --notebook-dir=./notebooks
COMMENT
