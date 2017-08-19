#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/push.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

if [[ "${GCP_KEY_PATH}" ]]; then
  echo GCP_KEY_PATH is defined: \""${GCP_KEY_PATH}"\"
else
  echo GCP_KEY_PATH is not defined!
  exit 255
fi

# go get -v github.com/GoogleCloudPlatform/docker-credential-gcr
gcloud docker -- login -u _json_key -p "$(cat ${GCP_KEY_PATH})" https://gcr.io

gcloud docker -- push gcr.io/gcp-dplearn/dplearn:latest-app
gcloud docker -- push gcr.io/gcp-dplearn/dplearn:latest-reverse-proxy
gcloud docker -- push gcr.io/gcp-dplearn/dplearn:latest-python2-cpu
gcloud docker -- push gcr.io/gcp-dplearn/dplearn:latest-python2-gpu
gcloud docker -- push gcr.io/gcp-dplearn/dplearn:latest-python3-cpu
gcloud docker -- push gcr.io/gcp-dplearn/dplearn:latest-python3-gpu
gcloud docker -- push gcr.io/gcp-dplearn/dplearn:latest-r
