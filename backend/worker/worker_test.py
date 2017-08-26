# -*- coding: utf-8 -*-
"""This script tests worker and backend-web-server clients.
"""

from __future__ import print_function

import os
import shutil
import subprocess
import sys
import tempfile
import threading
import time
import unittest

import glog as log
import requests

from .worker import fetch_item, post_item


class BACKEND(threading.Thread):
    def __init__(self, SERVER_EXEC):
        self.stdout = None
        self.stderr = None
        self.process = None
        self.exec_path = SERVER_EXEC
        self.data_dir = os.path.join(tempfile.gettempdir(), 'etcd')
        if os.path.exists(self.data_dir):
            log.info('deleting {0}'.format(self.data_dir))
            shutil.rmtree(self.data_dir)
            log.info('deleted {0}'.format(self.data_dir))
        threading.Thread.__init__(self)

    def run(self):
        self.process = subprocess.Popen([
            self.exec_path,
            '-web-host', 'localhost:2200',
            '-queue-port-client', '27000',
            '-queue-port-peer', '27001',
            '-data-dir', self.data_dir,
            '-logtostderr',
        ], shell=False, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

        self.stdout, self.stderr = self.process.communicate()

    def kill(self):
        log.info('killing process')
        self.process.kill()
        log.info('killed process')
        if os.path.exists(self.data_dir):
            log.info('deleting {0}'.format(self.data_dir))
            shutil.rmtree(self.data_dir)
            log.info('deleted {0}'.format(self.data_dir))


class TestBackend(unittest.TestCase):
    def test_backend(self):
        exec_path = os.environ['SERVER_EXEC']
        if exec_path == '':
            log.fatal('Got empty backend-web-server path')
            sys.exit(0)
        if not os.path.exists(exec_path):
            log.fatal('{0} does not eixst'.format(exec_path))
            sys.exit(0)

        log.info('Running {0}'.format(exec_path))
        backend_proc = BACKEND(exec_path)
        backend_proc.setDaemon(True)
        backend_proc.start()

        log.info('Sleeping...')
        time.sleep(5)

        endpoint = 'http://localhost:2200/cats-request/queue'

        log.info('Posting client requests...')
        # invalid item
        item = {
            'bucket': '/cats-request',
            'key': '/cats-request',
            'value': '',
            'request_id': '',
        }
        itemresp1 = post_item(endpoint, item)
        self.assertIsNot(itemresp1['error'], '')

        # valid item
        item['value'] = 'foo'
        item['request_id'] = 'id'
        itemresp2 = post_item(endpoint, item)
        self.assertEqual(itemresp2['error'], u'unknown request ID \"id\"')

        def cleanup():
            log.info('Killing backend-web-server...')
            backend_proc.kill()

            backend_proc.join()
            log.info('backend-web-server output: {0}'.format(backend_proc.stderr))

        self.addCleanup(cleanup)

        time.sleep(3)

        log.info('Fetching items...')
        try:
            fetch_item(endpoint, timeout=5)

        except requests.exceptions.ReadTimeout:
            log.info('Got expected timeout!')

        log.info('Done!')


if __name__ == '__main__':
    unittest.main()
