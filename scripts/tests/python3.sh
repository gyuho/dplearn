#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/tests/python3.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

if [[ "${DATASETS_DIR}" ]]; then
  echo DATASETS_DIR is defined: \""${DATASETS_DIR}"\"
else
  echo DATASETS_DIR is not defined!
  exit 255
fi

if [[ "${CATS_PARAM_PATH}" ]]; then
  echo CATS_PARAM_PATH is defined: \""${CATS_PARAM_PATH}"\"
else
  echo CATS_PARAM_PATH is not defined!
  exit 255
fi

echo "Running backend.worker.cats"
DATASETS_DIR=${DATASETS_DIR} python3 -m unittest backend.worker.cats.data_test
python3 -m unittest backend.worker.cats.initialize_test
python3 -m unittest backend.worker.cats.propagate_test
DATASETS_DIR=${DATASETS_DIR} CATS_PARAM_PATH=${CATS_PARAM_PATH} python3 -m unittest backend.worker.cats.model_test
DATASETS_DIR=${DATASETS_DIR} CATS_PARAM_PATH=${CATS_PARAM_PATH} python3 -m unittest backend.worker.cats_test

if [[ "${SERVER_EXEC}" ]]; then
  echo SERVER_EXEC is defined: \""${SERVER_EXEC}"\"
  echo "Running backend.worker.worker_test"
  go install -v ./cmd/backend-web-server
  SERVER_EXEC=${SERVER_EXEC} python3 -m unittest backend.worker.worker_test
else
  echo SERVER_EXEC is not defined!
fi

if [[ "${ETCD_EXEC}" ]]; then
  echo ETCD_EXEC is defined: \""${ETCD_EXEC}"\"
  echo "Running backend.etcd-python.etcd_test"
  ETCD_EXEC=${ETCD_EXEC} python3 -m unittest backend.etcd-python.etcd_test
else
  echo ETCD_EXEC is not defined!
fi

<<COMMENT
DATASETS_DIR=./datasets python3 -m unittest backend.worker.cats.data_test

DATASETS_DIR=./datasets \
  CATS_PARAM_PATH=./datasets/parameters-cats.npy \
  python3 -m unittest backend.worker.cats.model_test

DATASETS_DIR=./datasets \
  CATS_PARAM_PATH=./datasets/parameters-cats.npy \
  python3 -m unittest backend.worker.cats_test

DATASETS_DIR=./datasets \
  CATS_PARAM_PATH=./datasets/parameters-cats.npy \
  ./scripts/tests/python3.sh

DATASETS_DIR=./datasets \
  CATS_PARAM_PATH=./datasets/parameters-cats.npy \
  ETCD_EXEC=/opt/bin/etcd \
  SERVER_EXEC=${GOPATH}/bin/backend-web-server \
  ./scripts/tests/python3.sh
COMMENT
