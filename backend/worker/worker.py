"""
This script interacts with backend/web/queue-service.
"""

from __future__ import print_function

import copy
import json
import os
import sys
import time

import glog as log
import pendulum
import requests


def fetch_item(endpoint):
    """
    fetch_item fetches a scheduled job from queue service.
    """
    keys = ['bucket', 'key', 'value', 'progress', 'canceled', 'error']
    while True:
        try:
            rresp = requests.get(endpoint)
            item = json.loads(rresp.text)
            for key in range(keys):
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
    """
    post posts the processed job to the queue service.
    """
    while True:
        try:
            return requests.post(endpoint, data=json.dumps(item))

        except requests.exceptions.ConnectionError as err:
            log.warning('Connection error: {0}'.format(err))
            time.sleep(5)

        except:
            log.warning('Unexpected error: {0}'.format(sys.exc_info()[0]))
            raise


if __name__ == "__main__":
    log.info("starting worker")

    EP = os.environ['QUEUE_ENDPOINT']
    if EP == '':
        log.fatal('Got empty QUEUE_ENDPOINT')
        sys.exit(1)

    PREV = None
    while True:
        ITEM = fetch_item(EP)
        if ITEM['key'] == '' and ITEM['value'] == '':
            log.info('No job to process in {0}'.format(EP))
            time.sleep(5)
            continue

        log.info("fetched item: {0}".format(ITEM))

        # in case previous post request is
        # not processed yet in backend
        if ITEM == PREV:
            log.warning('{0} == prev {1}?'.format(ITEM, PREV))
            time.sleep(5)
            continue

        # for future comparison
        PREV = copy.deepcopy(ITEM)

        if ITEM['bucket'] == '/cats-vs-dogs-request':
            log.info('/cats-vs-dogs-request is not ready yet; testing')

            # simulate worker
            time.sleep(5)
            ITEM['progress'] = 100
            ITEM['value'] = 'cats-vs-dogs at ' + pendulum.now().isoformat()
            post_item(EP, ITEM)

        elif ITEM['bucket'] == '/mnist-request':
            log.info('/mnist-request is not ready yet')

        elif ITEM['bucket'] == '/word-predict-request':
            log.info('/word-predict-request is not ready yet; testing')

            # simulate worker
            time.sleep(5)
            ITEM['progress'] = 100
            ITEM['value'] = 'word-predict at ' + pendulum.now().isoformat()
            post_item(EP, ITEM)
