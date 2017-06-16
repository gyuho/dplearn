#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/dep/package-json.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

go install -v ./cmd/gen-package-json

gen-package-json --output package.json --logtostderr
