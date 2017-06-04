#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/push-cpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

gcloud docker -- push gcr.io/deephardway/github-deep:latest-cpu

<<COMMENT
gcloud docker -- login -u _json_key -p "$(cat ${HOME}/gcp-key-deephardway.json)" https://gcr.io
COMMENT
