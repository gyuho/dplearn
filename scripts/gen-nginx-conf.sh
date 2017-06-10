#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/gen-nginx-conf.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

go install -v ./cmd/gen-nginx-conf
gen-nginx-conf --output nginx.conf --target-port 4200 --logtostderr
