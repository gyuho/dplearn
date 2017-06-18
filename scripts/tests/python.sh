#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/tests/python.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

if [[ -z "${ETCD_EXEC}" ]]; then
  echo ETCD_EXEC is not defined!
  exit 255
fi

pushd ./backend/etcd-python >/dev/null
./tests.sh
popd >/dev/null

if [[ -z "${BACKEND_WEB_SERVER_EXEC}" ]]; then
  echo BACKEND_WEB_SERVER_EXEC is not defined!
  exit 255
fi

pushd ./backend/worker >/dev/null
./tests.sh
popd >/dev/null
