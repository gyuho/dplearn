#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/run/reverse-proxy.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

./gen-package-json -output package.json -logtostderr=true
cat package.json

./gen-nginx-conf -output nginx.conf -logtostderr=true
cat nginx.conf

/usr/sbin/nginx -g 'daemon off;'
