#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/app-run.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

LOCAL_DIR=/var/lib/etcd
if [[ $(uname) = "Darwin" ]]; then
  echo "Running locally with MacOS"
  LOCAL_DIR=/tmp/etcd
  rm -rf /tmp/etcd
fi

# TODO: shared volume with worker for downloaded cat images
docker run \
  --rm \
  -it \
  --publish 2200:2200 \
  --publish 4200:4200 \
  --ulimit nofile=262144:262144 \
  --volume=${LOCAL_DIR}:/var/lib/etcd \
  gcr.io/gcp-dplearn/dplearn:latest-app \
  /bin/sh -c "./scripts/run/app.sh"

<<COMMENT
http://localhost:4200


curl -L http://localhost:2200/health

docker run \
  --rm \
  -it \
  --net=host \
  gcr.io/gcp-dplearn/dplearn:latest-app \
  /bin/sh -c "
curl -L http://localhost:2200/health
"
COMMENT
