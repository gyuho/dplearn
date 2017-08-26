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

pushd ./backend/etcd-python >/dev/null
./tests-python3.sh
popd >/dev/null

sleep 3s

if [[ "${BACKEND_WEB_SERVER_EXEC}" ]]; then
  echo BACKEND_WEB_SERVER_EXEC is defined: \""${BACKEND_WEB_SERVER_EXEC}"\"
else
  echo BACKEND_WEB_SERVER_EXEC is not defined!
  exit 255
fi

pushd ./backend/worker >/dev/null
./tests-python3.sh
popd >/dev/null

<<COMMENT
ETCD_EXEC=/opt/bin/etcd \
  BACKEND_WEB_SERVER_EXEC=${HOME}/go/bin/backend-web-server \
  ./scripts/tests/python3.sh
COMMENT
