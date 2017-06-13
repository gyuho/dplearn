#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/push-cpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

go get -v github.com/GoogleCloudPlatform/docker-credential-gcr

gcloud docker -- login -u _json_key -p "$(cat ${GCP_KEY_PATH})" https://gcr.io

gcloud docker -- push gcr.io/deephardway/deephardway:latest-cpu
