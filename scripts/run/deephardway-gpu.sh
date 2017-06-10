#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/run/deephardway-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

./gen-package-json -output package.json -logtostderr=true
cat package.json

./gen-nginx-conf -output nginx.conf -logtostderr=true
cat nginx.conf

./backend-web-server -web-port 2200 -queue-port-client 22000 -queue-port-peer 22001 -data-dir /var/lib/etcd -logtostderr=true &
yarn start-prod &
wait
