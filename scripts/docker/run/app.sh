#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/docker/run/app.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

backend-web-server \
  -web-host 0.0.0.0:2200 \
  -queue-port-client 22000 \
  -queue-port-peer 22001 \
  -data-dir /var/lib/etcd \
  -logtostderr=true  &

gen-package-json -output package.json -logtostderr=true \
  && cat package.json \
  && yarn start-prod &

wait

<<COMMENT
rm -rf /tmp/etcd
go install -v ./cmd/backend-web-server
backend-web-server \
  -web-host 0.0.0.0:2200 \
  -queue-port-client 22000 \
  -queue-port-peer 22001 \
  -data-dir /tmp/etcd \
  -logtostderr=true

ETCDCTL_API=3 /opt/bin/etcdctl --endpoints=localhost:22000 get "" --from-key

curl -L http://localhost:2200/cats-request/queue

curl -L http://localhost:2200/cats-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{}'

curl -L http://localhost:2200/cats-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{"bucket" : "/cats-request", "key" : "/cats-request", "value" : ""}'

curl -L http://localhost:2200/cats-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{"bucket" : "/cats-request", "key" : "/cats-request", "value" : "bar"}'

curl -L http://localhost:2200/cats-request/queue

sleep 5s

ETCDCTL_API=3 /opt/bin/etcdctl --endpoints=localhost:22000 get "" --from-key

python ./backend/worker/worker.py http://localhost:2200/cats-request/queue
COMMENT
