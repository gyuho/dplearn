#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./tests.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

if [[ "${BACKEND_WEB_SERVER_EXEC}" ]]; then
  echo BACKEND_WEB_SERVER_EXEC is defined: \""${BACKEND_WEB_SERVER_EXEC}"\"
else
  echo BACKEND_WEB_SERVER_EXEC is not defined!
  exit 255
fi

if [[ "${INDEX_FILE}" ]]; then
  echo INDEX_FILE is defined: \""${INDEX_FILE}"\"
else
  echo INDEX_FILE is not defined!
  exit 255
fi

if [[ "${VGG_FILE}" ]]; then
  echo VGG_FILE is defined: \""${VGG_FILE}"\"
else
  echo VGG_FILE is not defined!
  exit 255
fi

BACKEND_WEB_SERVER_EXEC=${BACKEND_WEB_SERVER_EXEC} \
  INDEX_FILE=${INDEX_FILE} \
  python -m unittest discover --pattern=*.py -v
