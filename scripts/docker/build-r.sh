#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/build-r.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker build \
  --tag gcr.io/gcp-dplearn/dplearn:latest-r \
  --file ./dockerfiles/Dockerfile-r \
  .
