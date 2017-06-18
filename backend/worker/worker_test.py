"""
This script tests worker and backend-web-server clients.
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

import worker


class BACKEND(threading.Thread):
    """
    wraps backend-web-server subprocess
    """
    def __init__(self, BACKEND_WEB_SERVER_EXEC):
        self.stdout = None
        self.stderr = None
        self.process = None
        self.exec_path = BACKEND_WEB_SERVER_EXEC
        self.data_dir = os.path.join(tempfile.gettempdir(), 'etcd')
        if os.path.exists(self.data_dir):
            log.info('deleting {0}'.format(self.data_dir))
            shutil.rmtree(self.data_dir)
            log.info('deleted {0}'.format(self.data_dir))
        threading.Thread.__init__(self)

    def run(self):
        self.process = subprocess.Popen([
            self.exec_path,
            '-web-port', '2200',
            '-queue-port-client', '27000',
            '-queue-port-peer', '27001',
            '-data-dir', self.data_dir,
            '-logtostderr',
        ], shell=False, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

        self.stdout, self.stderr = self.process.communicate()

    def kill(self):
        """
        Kills the running backend-web-server process
        """
        log.info('killing process')
        self.process.kill()
        log.info('killed process')
        if os.path.exists(self.data_dir):
            log.info('deleting {0}'.format(self.data_dir))
            shutil.rmtree(self.data_dir)
            log.info('deleted {0}'.format(self.data_dir))


class TestBackend(unittest.TestCase):
    """
    backend-web-server testing methods
    """
    def test_backend(self):
        """
        backend-web-server test function
        """
        exec_path = os.environ['BACKEND_WEB_SERVER_EXEC']
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

        endpoint = 'http://localhost:2200/word-predict-request/queue'

        log.info('Posting client requests...')
        # invalid item
        item = {
            'bucket': '/word-predict-request',
            'key': '/word-predict-request',
            'value': '',
        }
        itemresp1 = worker.post_item(endpoint, item)
        self.assertIsNot(itemresp1['error'], '')

        # valid item
        item['value'] = 'foo'
        itemresp2 = worker.post_item(endpoint, item)
        self.assertEqual(itemresp2['error'], '')

        time.sleep(3)

        log.info('Fetching items...')
        itemresp3 = worker.fetch_item(endpoint)
        self.assertEqual(item['bucket'], itemresp3['bucket'])
        self.assertEqual(item['key'], itemresp3['key'])
        self.assertEqual(item['value'], itemresp3['value'])
        self.assertEqual(itemresp3['error'], '')

        print('Killing backend-web-server...')
        backend_proc.kill()

        backend_proc.join()
        print('backend-web-server output: {0}'.format(backend_proc.stderr))

        print('Done!')


if __name__ == '__main__':
    unittest.main()
