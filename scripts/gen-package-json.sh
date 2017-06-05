#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/gen-package-json.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

go install -v ./cmd/package-gen
package-gen --config package.json --logtostderr
