#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/run/worker.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

python ./backend/worker/worker.py http://localhost:2200/cats-request/queue &

wait
