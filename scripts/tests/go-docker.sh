#!/usr/bin/env bash
set -e

<<COMMENT
https://www.tensorflow.org/install/install_mac
COMMENT

if ! [[ "$0" =~ "./scripts/tests/go-docker.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker run --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/github-deep:latest \
  ./scripts/tests/go.sh

<<COMMENT
docker run --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/github-deep:latest \
  pwd

docker run --rm \
  --volume=`pwd`:/gopath/src/github.com/gyuho/deephardway \
  gcr.io/deephardway/github-deep:latest \
  ls
COMMENT
