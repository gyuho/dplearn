# -*- coding: utf-8 -*-
"""This script interacts with backend/web.
"""

from __future__ import print_function

import copy
import datetime
import json
import os.path
import sys
import time

import glog as log
import requests

ITEM_KEYS = ['bucket', 'key', 'value', 'progress', 'canceled', 'error',
             'request_id']


def fetch_item(endpoint, timeout=None):
    """fetch_item fetches a scheduled job from queue service.
    """
    while True:
        try:
            # blocks until first item is available
            log.info('fetching item from {0}'.format(endpoint))
            rresp = requests.get(endpoint, timeout=timeout)
            log.info('fetched item from {0}'.format(endpoint))

            # even empty, Go backend should encode every field
            item = json.loads(rresp.text)
            for key in ITEM_KEYS:
                if key not in item:
                    log.warning('{0} not in {1}'.format(key, rresp.text))
                    return None

            return item

        except requests.exceptions.ConnectionError as err:
            log.warning('Connection error: {0}'.format(err))
            time.sleep(5)

        except:
            log.warning('Unexpected error: {0}'.format(sys.exc_info()[0]))
            raise


def post_item(endpoint, item):
    """post posts the processed job to the queue service.
    """
    headers = {'Content-Type': 'application/json'}
    while True:
        try:
            req_id = item['request_id']
            log.info('posting item to {0} with request ID {1}'.format(endpoint, req_id))
            rresp = requests.post(endpoint, data=json.dumps(item),
                                  headers=headers)
            log.info('posted item to {0} with request ID {1}'.format(endpoint, req_id))

            item = json.loads(rresp.text)
            # even empty, Go backend should encode every field
            for key in ITEM_KEYS:
                if key not in item:
                    log.warning('{0} not in {1}'.format(key, rresp.text))
                    return None

            return item

        except requests.exceptions.ConnectionError as err:
            log.warning('Connection error: {0}'.format(err))
            time.sleep(5)

        except:
            log.warning('Unexpected error: {0}'.format(sys.exc_info()[0]))
            raise


if __name__ == "__main__":
    if len(sys.argv) == 1:
        log.fatal('Got empty endpoint: {0}'.format(sys.argv))
        sys.exit(1)

    EP = sys.argv[1]
    if EP == '':
        log.fatal('Got empty endpoint: {0}'.format(sys.argv))
        sys.exit(1)

    log.info("starting worker on {0}".format(EP))

    PREV_ITEM = None
    while True:
        ITEM = fetch_item(EP)
        if ITEM['error'] not in ['', u'']:
            log.warning(ITEM['error'])
            time.sleep(5)
            continue

        # in case previous post request is
        # not processed yet in backend
        if ITEM == PREV_ITEM:
            log.warning('duplicate: {0} == prev {1}?'.format(ITEM, PREV_ITEM))
            time.sleep(5)
            continue

        # for future comparison
        PREV_ITEM = copy.deepcopy(ITEM)

        if ITEM['bucket'] == '/cats-vs-dogs-request':
            IMAGE_PATH = ITEM['value']
            if not os.path.exists(IMAGE_PATH):
                log.warning('cannot find image {0}'.format(IMAGE_PATH))
                ITEM['progress'] = 100
                ITEM['error'] = 'cannot find image {0}'.format(IMAGE_PATH)
            else:
                """TODO: implement actual worker with Tensorflow
                """
                with open(IMAGE_PATH, "r") as f:
                    log.info('opened image {0}'.format(f.read(5)))
                    # process_cats_vs_dogs(ITEM['value'])
                    ITEM['progress'] = 100
                    NOW = datetime.datetime.now().isoformat()
                    ITEM['value'] = "[FAKE] it's a cat! " + NOW

            POST_RESPONSE = post_item(EP, ITEM)
            if POST_RESPONSE['error'] not in ['', u'']:
                log.warning(POST_RESPONSE['error'])

        elif ITEM['bucket'] == '/mnist-request':
            log.info('/mnist-request is not ready yet')

        elif ITEM['bucket'] == '/word-predict-request':
            TXT = ITEM['value']
            if len(TXT) < 5:
                log.warning('text is too short {0}'.format(TXT))
                ITEM['progress'] = 100
                ITEM['error'] = 'text is too short {0}'.format(TXT)
            else:
                """TODO: implement actual worker with Tensorflow
                """
                ITEM['progress'] = 100
                NOW = datetime.datetime.now().isoformat()
                ITEM['value'] = "[FAKE] it's ... " + NOW

            POST_RESPONSE = post_item(EP, ITEM)
            if POST_RESPONSE['error'] not in ['', u'']:
                log.warning(POST_RESPONSE['error'])
