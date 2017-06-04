#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/build-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker build \
  --tag gcr.io/deephardway/github-deep:latest-gpu \
  --file ./dockerfiles/dev-gpu/Dockerfile \
  .
