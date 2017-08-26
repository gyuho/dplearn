#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/dep/python3.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

pip3 --no-cache-dir install \
  requests \
  glog \
  humanize \
  bcolz \
  h5py \
  ipykernel \
  jupyter \
  matplotlib \
  numpy \
  pandas \
  scipy \
  sklearn \
  Pillow
