#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/prod/deephardway-cpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

# rewrite package.json
./package-gen -config package.json -logtostderr=true

./backend-web-server -logtostderr=true &
yarn start-prod &
wait
