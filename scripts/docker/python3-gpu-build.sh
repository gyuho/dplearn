#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/python3-gpu-build.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker build \
  --tag gcr.io/gcp-dplearn/dplearn:latest-python3-gpu \
  --file ./dockerfiles/Dockerfile-python3-gpu \
  .
