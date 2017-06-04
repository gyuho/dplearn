#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/deephardway-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

./backend-web-server -logtostderr=true &
yarn start-prod &
wait
