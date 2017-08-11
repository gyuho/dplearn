#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/clean.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

docker rmi --force $(docker images --quiet)

<<COMMENT
pgrep "docker rm" && exit 0
docker rm $(docker ps -a | grep "Dead\|Exited" | awk '{print $1}'); true

docker rmi -f $(docker images -qf)
docker rmi -f $(docker images -qf dangling=true); true

docker volume rm $(docker volume ls -qf dangling=true); true

docker stop $(docker ps -q)
docker kill $(docker ps -q)
COMMENT
