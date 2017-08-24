#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/build-app.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker build \
  --tag gcr.io/gcp-dplearn/dplearn:latest-app \
  --file ./dockerfiles/Dockerfile-app \
  .
