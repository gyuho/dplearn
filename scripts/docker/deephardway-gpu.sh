#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/deephardway-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

LOCAL_DIR=/var/lib/etcd
if [[ $(uname) = "Darwin" ]]; then
  echo "Running locally with MacOS"
  LOCAL_DIR=/tmp/etcd
  rm -rf /tmp/etcd
fi

# -P
# -p hostPort:containerPort
# -p 80:80
# -p 4200:4200
docker run \
  --rm \
  -it \
  --volume=${LOCAL_DIR}:/var/lib/etcd \
  --volume=${HOME}/.keras:/root/.keras \
  -p 4200:4200 \
  --ulimit nofile=262144:262144 \
  gcr.io/deephardway/deephardway:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && ./scripts/run/deephardway-gpu.sh"
