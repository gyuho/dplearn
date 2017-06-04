"""
This script interacts with etcd server via gprespc-gateway
"""

# base64 encoding/decoding for gprespc-gateway
import base64

# json encoding/decoding for etcd requests
import json

# requests for etcd requests
import requests

# TODO(gyuho): error handling
# TODO(gyuho): watch

def put(endpoint, key, val):
    """
    put sends write request to etcd
    e.g. curl -L http://localhost:2379/v3alpha/kv/put -X POST -d '{"key": "Zm9v", "value": "YmFy"}'
    e.g. put("http://localhost:2379", "foo", "bar")
    """
    req = {"key": base64.b64encode(key), "value": base64.b64encode(val)}
    presp = requests.post(endpoint + "/v3alpha/kv/put", data=json.dumps(req))
    print presp.text

def get(endpoint, key):
    """
    get sends read request to etcd
    e.g. curl -L http://localhost:2379/v3alpha/kv/range -X POST -d '{"key": "Zm9v"}'
    e.g. get("http://localhost:2379", "foo")
    """
    req = {"key": base64.b64encode(key)}
    presp = requests.post(endpoint + "/v3alpha/kv/range", data=json.dumps(req))
    resp = json.loads(presp.text)
    return base64.b64decode(resp["kvs"][0]["value"])
