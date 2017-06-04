"""
This script interacts with etcd server via gprespc-gateway
"""

from __future__ import print_function

import base64
import json
import sys
import time
import requests

def put(endpoint, key, val):
    """
    put sends write request to etcd
    e.g. curl -L http://localhost:2379/v3alpha/kv/put -X POST -d '{"key": "Zm9v", "value": "YmFy"}'
    """
    req = {"key": base64.b64encode(key), "value": base64.b64encode(val)}
    while True:
        try:
            return requests.post(endpoint + "/v3alpha/kv/put", data=json.dumps(req))
        except requests.exceptions.ConnectionError as err:
            print('Connection error: {0}'.format(err))
            time.sleep(5)
        except:
            print('Unexpected error:', sys.exc_info()[0])
            raise

def get(endpoint, key):
    """
    get sends read request to etcd
    e.g. curl -L http://localhost:2379/v3alpha/kv/range -X POST -d '{"key": "Zm9v"}'
    """
    req = {"key": base64.b64encode(key)}
    while True:
        try:
            presp = requests.post(endpoint + '/v3alpha/kv/range', data=json.dumps(req))
            resp = json.loads(presp.text)
            if 'kvs' not in resp:
                print('{0} does not exist', key)
                return ''
            if len(resp['kvs']) != 1:
                print('{0} does not exist', key)
                return ''
            return base64.b64decode(resp['kvs'][0]['value'])
        except requests.exceptions.ConnectionError as err:
            print('Connection error: {0}'.format(err))
            time.sleep(5)
        except:
            print('Unexpected error:', sys.exc_info()[0])
            raise
