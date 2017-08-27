#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/run/worker-python3.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

python3 ./backend/worker/worker.py http://localhost:2200/cats-request/queue &

wait
