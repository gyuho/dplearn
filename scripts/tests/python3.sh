#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/tests/python3.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

if [[ -z "${ETCD_EXEC}" ]]; then
  echo ETCD_EXEC is not defined!
  exit 255
fi

if [[ "${DATASETS_DIR}" ]]; then
  echo DATASETS_DIR is defined: \""${DATASETS_DIR}"\"
else
  echo DATASETS_DIR is not defined!
  exit 255
fi

if [[ "${SERVER_EXEC}" ]]; then
  echo SERVER_EXEC is defined: \""${SERVER_EXEC}"\"
else
  echo SERVER_EXEC is not defined!
  exit 255
fi

go install -v ./cmd/backend-web-server

echo "Running backend.worker.cats tests..."
DATASETS_DIR=${DATASETS_DIR} python3 -m unittest backend.worker.cats.data_test
python3 -m unittest backend.worker.cats.initialize_parameters_test
python3 -m unittest backend.worker.cats.utils_test

ETCD_EXEC=${ETCD_EXEC} python3 -m unittest backend.etcd-python.etcd_test
SERVER_EXEC=${SERVER_EXEC} python3 -m unittest backend.worker.worker_test

<<COMMENT
DATASETS_DIR=./datasets \
  ETCD_EXEC=/opt/bin/etcd \
  SERVER_EXEC=${GOPATH}/bin/backend-web-server \
  ./scripts/tests/python3.sh
COMMENT
