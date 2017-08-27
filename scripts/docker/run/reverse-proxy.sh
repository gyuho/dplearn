#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/run/reverse-proxy.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

gen-nginx-conf -output nginx.conf -target-port 4200 -logtostderr=true \
  && cat nginx.conf \
  && cp nginx.conf /etc/nginx/sites-available/default \
  && /usr/sbin/nginx -g 'daemon off;'
