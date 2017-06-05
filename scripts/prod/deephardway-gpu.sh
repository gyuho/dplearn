#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/prod/deephardway-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

# rewrite package.json
./gen-package-json -config package.json -logtostderr=true

./backend-web-server -logtostderr=true &
yarn start-prod &
wait
