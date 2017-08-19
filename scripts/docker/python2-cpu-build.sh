#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/python2-cpu-build.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker build \
  --tag gcr.io/gcp-dplearn/dplearn:latest-python2-cpu \
  --file ./dockerfiles/Dockerfile-python2-cpu \
  .
