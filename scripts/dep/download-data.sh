#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/dep/download-data.sh" ]]; then
    echo "must be run from repository root"
    exit 255
fi

go install -v ./cmd/download-data

download-data -source-path http://files.fast.ai/data/dogscats.zip \
  -target-path $HOME/data/deephardway.data/dogscats.zip \
  -output-dir $HOME/data/deephardway.data/dogscats \
  -output-dir-overwrite \
  -logtostderr

download-data -source-path http://files.fast.ai/models/vgg16.h5 \
  -target-path $HOME/data/deephardway.data/vgg16.h5 \
  -logtostderr
