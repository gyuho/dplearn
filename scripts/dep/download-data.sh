#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/dep/download-data.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

KERAS_DIR=/var/lib/keras
if [[ $(uname) = "Darwin" ]]; then
  echo "Running locally with MacOS"
  KERAS_DIR=${HOME}/.keras
fi

echo KERAS_DIR: ${KERAS_DIR}

go install -v ./cmd/download-data

# '-output-dir-overwrite' to overwrite the whole directory
download-data -source-path http://files.fast.ai/data/dogscats.zip \
  -target-path ${KERAS_DIR}/datasets/dogscats.zip \
  -output-dir ${KERAS_DIR}/datasets/dogscats \
  -verbose \
  -smart-rename \
  -logtostderr

download-data -source-path http://files.fast.ai/models/vgg16.h5 \
  -target-path ${KERAS_DIR}/models/vgg16.h5 \
  -verbose \
  -logtostderr

download-data -source-path http://files.fast.ai/models/imagenet_class_index.json \
  -target-path ${KERAS_DIR}/models/imagenet_class_index.json \
  -verbose \
  -logtostderr

download-data -source-path https://github.com/fchollet/deep-learning-models/releases/download/v0.1/vgg16_weights_tf_dim_ordering_tf_kernels.h5 \
  -target-path ${KERAS_DIR}/models/vgg16_weights_tf_dim_ordering_tf_kernels.h5 \
  -verbose \
  -logtostderr

download-data -source-path https://github.com/fchollet/deep-learning-models/releases/download/v0.1/vgg16_weights_tf_dim_ordering_tf_kernels_notop.h5 \
  -target-path ${KERAS_DIR}/models/vgg16_weights_tf_dim_ordering_tf_kernels_notop.h5 \
  -verbose \
  -logtostderr
