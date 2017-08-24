#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/untagged.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker rmi --force $(docker images | grep "^<none>" | awk "{print $3}")
