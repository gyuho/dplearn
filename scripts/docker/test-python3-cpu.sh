#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/test-python3-cpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker run \
  --rm \
  -it \
  --env DATASETS_DIR=/root/datasets \
  --env CATS_PARAM_PATH=/root/datasets/parameters-cats.npy \
  --net=host \
  --volume=`pwd`/datasets:/root/datasets \
  gcr.io/gcp-dplearn/dplearn:latest-python3-cpu \
  /bin/sh -c "./scripts/tests/python3.sh"
