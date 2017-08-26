#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./tests-python3.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

if [[ "${BACKEND_WEB_SERVER_EXEC}" ]]; then
  echo BACKEND_WEB_SERVER_EXEC is defined: \""${BACKEND_WEB_SERVER_EXEC}"\"
else
  echo BACKEND_WEB_SERVER_EXEC is not defined!
  exit 255
fi

pushd ..
BACKEND_WEB_SERVER_EXEC=${BACKEND_WEB_SERVER_EXEC} python3 -m unittest worker.worker_test
python3 -m unittest worker.utils_test
popd


<<COMMENT
go install -v ./cmd/backend-web-server

pushd ..
BACKEND_WEB_SERVER_EXEC=${GOPATH}/bin/backend-web-server \
  ./tests-python3.sh
popd
COMMENT
