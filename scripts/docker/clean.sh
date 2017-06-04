#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/clean.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker stop $(docker ps -q)
docker kill $(docker ps -q)
docker rm $(docker ps -a -q)
docker rmi -f $(docker images -q -f dangling=true)
docker rmi -f $(docker images -q)
