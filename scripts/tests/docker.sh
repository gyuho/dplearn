#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/tests/docker.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest \
  ./scripts/tests/go.sh

docker run \
  --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest \
  /bin/sh -c "ETCD_TEST_PATH='/etcd' ./scripts/tests/python.sh"

<<COMMENT
docker run --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest \
  pwd

docker run --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/deephardway:latest \
  ls
COMMENT
