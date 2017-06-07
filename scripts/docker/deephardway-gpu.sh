#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/deephardway-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

LOCAL_DIR=/var/lib/etcd
if [ $(uname) = "Darwin" ]; then
  echo "Running on MacOS"
  LOCAL_DIR=${HOME}/etcd
fi

# -P
# -p hostPort:containerPort
# -p 80:80
# -p 4200:4200
docker run \
  --rm \
  -it \
  -p 4200:4200 \
  --volume=${LOCAL_DIR}:/var/lib/etcd \
  gcr.io/deephardway/deephardway:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && ./scripts/run/deephardway-gpu.sh"
