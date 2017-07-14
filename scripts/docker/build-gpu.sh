#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/build-gpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker build \
  --tag gcr.io/gcp-dplearn/dplearn:latest-gpu \
  --file ./dockerfiles/gpu/Dockerfile \
  .

<<COMMENT
sudo groupadd docker
sudo gpasswd -a $USER docker
sudo usermod -aG docker $USER
newgrp docker

sudo chmod +x /home/gyuho/.docker/config.json
COMMENT
