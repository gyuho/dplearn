#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/gen.sh" ]]; then
    echo "must be run from repository root"
    exit 255
fi

go install -v ./cmd/dockerfile-gen
dockerfile-gen --config=./dockerfiles/dev-gpu/config.yaml --logtostderr=true
dockerfile-gen --config=./dockerfiles/dev-cpu/config.yaml --logtostderr=true
