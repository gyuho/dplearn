#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./tests.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

if [[ -z "${BACKEND_WEB_SERVER_EXEC}" ]]; then
  echo BACKEND_WEB_SERVER_EXEC is not defined!
  exit 255
fi

BACKEND_WEB_SERVER_EXEC=${BACKEND_WEB_SERVER_EXEC} python -m unittest discover --pattern=*.py -v
