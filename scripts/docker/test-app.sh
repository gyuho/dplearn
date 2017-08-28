#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/test-app.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker run \
  --rm \
  -it \
  --net=host \
  gcr.io/gcp-dplearn/dplearn:latest-app \
  /bin/sh -c "./scripts/tests/frontend.sh && ./scripts/tests/go.sh"
