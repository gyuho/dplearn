#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/push-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

gcloud docker -- push gcr.io/deephardway/deephardway:latest-gpu

<<COMMENT
gcloud docker -- login -u _json_key -p "$(cat ${HOME}/gcp-key-deephardway.json)" https://gcr.io
COMMENT
