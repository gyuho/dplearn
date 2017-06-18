#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/run/deephardway-cpu.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

gen-package-json -output package.json -logtostderr=true
cat package.json

gen-nginx-conf -output nginx.conf -target-port 4200 -logtostderr=true
cat nginx.conf

backend-web-server -web-port 2200 -queue-port-client 22000 -queue-port-peer 22001 -data-dir /var/lib/etcd -logtostderr=true &
yarn start-prod &
sleep 5s
python ./backend/worker/worker.py http://localhost:2200/cats-vs-dogs-request/queue &
python ./backend/worker/worker.py http://localhost:2200/word-predict-request/queue &
wait

<<COMMENT
rm -rf /tmp/etcd
go install -v ./cmd/backend-web-server
backend-web-server -web-port 2200 -queue-port-client 22000 -queue-port-peer 22001 -data-dir /tmp/etcd -logtostderr=true

ETCDCTL_API=3 /etcdctl --endpoints=localhost:22000 get "" --from-key

curl -L http://localhost:2200/cats-vs-dogs-request/queue

curl -L http://localhost:2200/cats-vs-dogs-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{}'

curl -L http://localhost:2200/cats-vs-dogs-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{"bucket" : "/cats-vs-dogs-request", "key" : "/cats-vs-dogs-request", "value" : ""}'

curl -L http://localhost:2200/cats-vs-dogs-request/queue \
  -H "Content-Type: application/json" \
  -X POST -d '{"bucket" : "/cats-vs-dogs-request", "key" : "/cats-vs-dogs-request", "value" : "bar"}'

sleep 5s

ETCDCTL_API=3 /etcdctl --endpoints=localhost:22000 get "" --from-key

python ./backend/worker/worker.py http://localhost:2200/cats-vs-dogs-request/queue
COMMENT
