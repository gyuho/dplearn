#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/tests/frontend.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

yarn lint
