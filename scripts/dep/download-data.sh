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

<<COMMENT
download-data -source-path https://github.com/fchollet/deep-learning-models/releases/download/v0.1/vgg16_weights_tf_dim_ordering_tf_kernels.h5 \
  -target-path ${HOME}/.keras/models/vgg16_weights_tf_dim_ordering_tf_kernels.h5 \
  -verbose \
  -logtostderr

download-data -source-path https://github.com/fchollet/deep-learning-models/releases/download/v0.1/vgg16_weights_tf_dim_ordering_tf_kernels_notop.h5 \
  -target-path ${HOME}/.keras/models/vgg16_weights_tf_dim_ordering_tf_kernels_notop.h5 \
  -verbose \
  -logtostderr
COMMENT
