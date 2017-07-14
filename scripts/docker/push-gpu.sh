#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/push-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

if [[ "${GCP_KEY_PATH}" ]]; then
  echo GCP_KEY_PATH is defined: \""${GCP_KEY_PATH}"\"
else
  echo GCP_KEY_PATH is not defined!
  exit 255
fi

# gcloud auth login

# go get -v github.com/GoogleCloudPlatform/docker-credential-gcr
# gcloud docker -- login -u _json_key -p "$(cat ${GCP_KEY_PATH})" https://gcr.io
gcloud docker -- login -u oauth2accesstoken -p "$(gcloud auth application-default print-access-token)" https://gcr.io

gcloud docker -- push gcr.io/gcp-dplearn/dplearn:latest-gpu
