#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/run-r.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker run \
  --rm \
  -it \
  --publish 8888:8888 \
  --ulimit nofile=262144:262144 \
  --volume=`pwd`/notebooks:/notebooks \
  gcr.io/gcp-dplearn/dplearn:latest-r \
  /bin/sh -c "source activate r && PASSWORD='' ./run_jupyter.sh -y --allow-root --notebook-dir=/notebooks"
