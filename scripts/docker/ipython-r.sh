#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/ipython-r.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker run \
  --privileged \
  --rm \
  -it \
  --volume=`pwd`/notebooks:/gopath/src/github.com/gyuho/dplearn/notebooks \
  -p 8888:8888 \
  gcr.io/gcp-dplearn/dplearn:latest-cpu \
  /bin/sh -c "pushd /gopath/src/github.com/gyuho/dplearn && source activate r && PASSWORD='' ./run_jupyter.sh -y --allow-root --notebook-dir=./notebooks"
