#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/gen.sh" ]]; then
    echo "must be run from repository root"
    exit 255
fi

go install -v ./cmd/gen-dockerfiles
gen-dockerfiles -config=./dockerfile.yaml -logtostderr=true
gen-dockerfiles -config=./dockerfiles/gpu/dockerfile.yaml -logtostderr=true
gen-dockerfiles -config=./dockerfiles/cpu/dockerfile.yaml -logtostderr=true
