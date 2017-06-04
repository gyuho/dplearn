#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/deephardway-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

# -P
# -p hostPort:containerPort
# -p 4200:4200
# -p 80:80
docker run \
  --rm \
  -it \
  -p 4200:4200 \
  gcr.io/deephardway/github-gyuho-deephardway:latest-gpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && ./scripts/deephardway-gpu.sh"
