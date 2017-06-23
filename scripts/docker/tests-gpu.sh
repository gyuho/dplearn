#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/tests-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

nvidia-docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && ./scripts/tests/frontend.sh"

nvidia-docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && ./scripts/tests/go.sh"

nvidia-docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  --volume=${HOME}/.keras/datasets:/root/.keras/datasets \
  --volume=${HOME}/.keras/models:/root/.keras/models \
  gcr.io/deephardway/deephardway:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && ETCD_EXEC=/etcd BACKEND_WEB_SERVER_EXEC=/gopath/bin/backend-web-server ./scripts/tests/python.sh"
