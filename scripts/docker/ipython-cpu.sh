#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/ipython-cpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

# -P
# -p hostPort:containerPort
# -p 2200:2200
# -p 4200:4200
# -p 80:80
docker run \
  --rm \
  -it \
  -p 8888:8888 \
  --volume=`pwd`/notebooks:/gopath/src/github.com/gyuho/deephardway/notebooks \
  gcr.io/deephardway/deephardway:latest-cpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && PASSWORD='' ./run_jupyter.sh -y --allow-root --notebook-dir=./notebooks"

<<COMMENT
https://hub.docker.com/r/tensorflow/tensorflow/
https://console.cloud.google.com/gcr/images/tensorflow/GLOBAL/tensorflow?pli=1
docker run -it -p 8888:8888 gcr.io/tensorflow/tensorflow:latest
COMMENT
