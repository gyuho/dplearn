#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/gen.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

go install -v ./cmd/gen-dockerfiles

gen-dockerfiles -config=./Dockerfile.yaml -logtostderr=true
gen-dockerfiles -config=./dockerfiles/gpu/Dockerfile.yaml -logtostderr=true
gen-dockerfiles -config=./dockerfiles/cpu/Dockerfile.yaml -logtostderr=true
