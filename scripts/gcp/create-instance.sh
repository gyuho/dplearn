#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/gcp/create-instance.sh" ]]; then
  echo "must be from repository root"
  exit 255
fi

if [ -z "$GCP_KEY_PATH" ]; then
  GCP_KEY_PATH="fmt bom dep compile build unit"
  echo GCP_KEY_PATH is not defined
  exit 255
else
  echo Reading "${GCP_KEY_PATH}"
fi

gcloud config set project deephardway

gcloud beta compute instances create deephardway \
  --custom-cpu=8 --custom-memory=30 --zone us-west1-b \
  --image-family=ubuntu-1604-lts --image-project=ubuntu-os-cloud \
  --boot-disk-size=150 --boot-disk-type="pd-ssd" \
  --network default \
  --tags=deephardway,http-server,https-server \
  --maintenance-policy=TERMINATE --restart-on-failure \
  --accelerator type=nvidia-tesla-k80,count=1 \
  --metadata-from-file gcp-key=${GCP_KEY_PATH},startup-script=./scripts/gcp/ubuntu-gpu.ansible.sh
