#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/dep/download-data.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

go install -v ./cmd/download-data

<<COMMENT
download-data -source-path http://files.fast.ai/data/dogscats.zip \
  -target-path ${HOME}/.keras/datasets/dogscats.zip \
  -output-dir ${HOME}/.keras/datasets/dogscats \
  -output-dir-overwrite \
  -verbose \
  -smart-rename \
  -logtostderr
COMMENT

download-data -source-path http://files.fast.ai/data/dogscats.zip \
  -target-path ${HOME}/.keras/datasets/dogscats.zip \
  -output-dir ${HOME}/.keras/datasets/dogscats \
  -verbose \
  -smart-rename \
  -logtostderr

download-data -source-path http://files.fast.ai/models/vgg16.h5 \
  -target-path ${HOME}/.keras/datasets/dogscats/models/vgg16.h5 \
  -verbose \
  -logtostderr

download-data -source-path http://files.fast.ai/models/imagenet_class_index.json \
  -target-path ${HOME}/.keras/datasets/dogscats/models/imagenet_class_index.json \
  -verbose \
  -logtostderr
