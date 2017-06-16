"""
This script interacts with etcd server via grpc-gateway.
See https://github.com/coreos/etcd/blob/master/Documentation/dev-guide/api_grpc_gateway.md for more.
Requires Python 3.
"""

from __future__ import print_function

import base64
import json
import sys
import time
import requests

def put(endpoint, key, val):
    """
    put sends write request to etcd.

    curl -L http://localhost:2379/v3alpha/kv/put \
      -X POST -d '{"key": "Zm9v", "value": "YmFy"}'
    """
    # Python 3 base64 requires utf-08 encoded bytes
    # Python 3 JSON encoder requires string
    key_str = base64.b64encode(bytes(key, "utf-8")).decode()
    val_str = base64.b64encode(bytes(val, "utf-8")).decode()
    req = {"key": key_str, "value": val_str}
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
    get sends read request to etcd.

    curl -L http://localhost:2379/v3alpha/kv/range \
      -X POST -d '{"key": "Zm9v"}'
    """
    # Python 3 base64 requires utf-08 encoded bytes
    # Python 3 JSON encoder requires string
    key_str = base64.b64encode(bytes(key, "utf-8")).decode()
    req = {"key": key_str}
    while True:
        try:
            rresp = requests.post(endpoint + '/v3alpha/kv/range', data=json.dumps(req))
            resp = json.loads(rresp.text)
            if 'kvs' not in resp:
                print('{0} does not exist'.format(key))
                return ''
            if len(resp['kvs']) != 1:
                print('{0} does not exist'.format(key))
                return ''
            return base64.b64decode(resp['kvs'][0]['value'])

        except requests.exceptions.ConnectionError as err:
            print('Connection error: {0}'.format(err))
            time.sleep(5)

        except:
            print('Unexpected error:', sys.exc_info()[0])
            raise

def watch(endpoint, key):
    """
    watch sends watch request to etcd.

    curl -L http://localhost:2379/v3alpha/watch \
      -X POST -d ''{"create_request": {"key":"Zm9v"} }'
    """
    # Python 3 base64 requires utf-08 encoded bytes
    # Python 3 JSON encoder requires string
    key_str = base64.b64encode(bytes(key, "utf-8")).decode()
    req = {'create_request': {"key": key_str}}
    while True:
        try:
            rresp = requests.post(endpoint + '/v3alpha/watch', data=json.dumps(req), stream=True)
            for line in rresp.iter_lines():
                # filter out keep-alive new lines
                if line:
                    decoded_line = line.decode('utf-8')
                    resp = json.loads(decoded_line)
                    if 'result' not in resp:
                        print('{0} does not have result'.format(resp))
                        return ''
                    if 'created' in resp['result']:
                        if resp['result']['created'] == True:
                            print('watching {0}'.format(key))
                            continue
                    if 'events' not in resp['result']:
                        print('{0} returned no events: {1}'.format(key, resp))
                        return None
                    if len(resp['result']['events']) != 1:
                        print('{0} returned more than 1 event: {1}'.format(key, resp))
                        return None
                    if 'kv' in resp['result']['events'][0]:
                        if 'value' in resp['result']['events'][0]['kv']:
                            return base64.b64decode(resp['result']['events'][0]['kv']['value'])
                        else:
                            print('no value in ', resp)
                            return None
                    else:
                        print('no kv in ', resp)
                        return None

        except requests.exceptions.ConnectionError as err:
            print('Connection error: {0}'.format(err))
            time.sleep(5)

        except:
            print('Unexpected error:', sys.exc_info()[0])
            raise
