#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/run-reverse-proxy.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker run \
  --rm \
  -it \
  --net=host \
  --ulimit nofile=262144:262144 \
  gcr.io/gcp-dplearn/dplearn:latest-reverse-proxy \
  /bin/sh -c "./scripts/run/reverse-proxy.sh"

<<COMMENT
http://localhost:80
COMMENT
