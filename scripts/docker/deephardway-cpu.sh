#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/deephardway-cpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

# -P
# -p hostPort:containerPort
# -p 80:80
# -p 4200:4200
docker run \
  --rm \
  -it \
  -p 80:80 \
  -p 4200:4200 \
  gcr.io/deephardway/deephardway:latest-cpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/deephardway && ./scripts/run/deephardway-cpu.sh"
