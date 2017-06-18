#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/tests-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && ./scripts/tests/frontend.sh"

docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && ./scripts/tests/go.sh"

docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && BACKEND_WEB_SERVER_EXEC=/gopath/bin/backend-web-server ETCD_EXEC=/etcd ./scripts/tests/python.sh"
