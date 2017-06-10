#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./tests.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

if [ -z "$ETCD_TEST_EXEC" ]; then
  echo ETCD_TEST_EXEC is not defined!
  exit 255
fi

ETCD_TEST_EXEC=${ETCD_TEST_EXEC} python -m unittest discover --pattern=*.py -v
