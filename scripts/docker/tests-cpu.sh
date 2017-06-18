#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/tests-cpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

nvidia-docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest-cpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && ./scripts/tests/frontend.sh"

nvidia-docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest-cpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && ./scripts/tests/go.sh"

nvidia-docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest-cpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && BACKEND_WEB_SERVER_EXEC=/gopath/bin/backend-web-server ETCD_EXEC=/etcd ./scripts/tests/python.sh"
