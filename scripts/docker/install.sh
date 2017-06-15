#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/install.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

curl -sSL https://get.docker.com/ | sh
