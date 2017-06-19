"""
This script tests etcd clients.
"""

from __future__ import print_function

import os.path
import shutil
import subprocess
import sys
import tempfile
import threading
import time
import unittest

import glog as log

from etcd import get, put, watch


class ETCD(threading.Thread):
    """
        wraps etcd subprocess
    """
    def __init__(self, ETCD_PATH):
        self.stdout = None
        self.stderr = None
        self.process = None
        self.exec_path = ETCD_PATH
        self.data_dir = os.path.join(tempfile.gettempdir(), 'etcd')
        if os.path.exists(self.data_dir):
            log.info('deleting {0}'.format(self.data_dir))
            shutil.rmtree(self.data_dir)
            log.info('deleted {0}'.format(self.data_dir))
        threading.Thread.__init__(self)

    def run(self):
        self.process = subprocess.Popen([
            self.exec_path,
            '--name', 's1',
            '--data-dir', self.data_dir,
            '--listen-client-urls', 'http://localhost:2379',
            '--advertise-client-urls', 'http://localhost:2379',
            '--listen-peer-urls', 'http://localhost:2380',
            '--initial-advertise-peer-urls', 'http://localhost:2380',
            '--initial-cluster', 's1=http://localhost:2380',
            '--initial-cluster-token', 'mytoken',
            '--initial-cluster-state', 'new',
            '--auto-compaction-retention', '1',
        ], shell=False, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

        self.stdout, self.stderr = self.process.communicate()

    def kill(self):
        """
            Kills the running etcd process
        """
        log.info('killing process')
        self.process.kill()
        log.info('killed process')
        if os.path.exists(self.data_dir):
            log.info('deleting {0}'.format(self.data_dir))
            shutil.rmtree(self.data_dir)
            log.info('deleted {0}'.format(self.data_dir))


class TestETCDMethods(unittest.TestCase):
    """
        etcd testing methods
    """
    def watch_routine(self):
        """
            test watch API
        """
        self.assertEqual(watch('http://localhost:2379', 'foo'), 'bar')
        # Python 3
        # self.assertEqual(watch('http://localhost:2379', 'foo'), b'bar')

    def test_etcd(self):
        """
            etcd test function
        """
        exec_path = os.environ['ETCD_EXEC']
        if exec_path == '':
            log.fatal('Got empty etcd path')
            sys.exit(0)
        if not os.path.exists(exec_path):
            log.fatal('{0} does not eixst'.format(exec_path))
            sys.exit(0)

        log.info('Running {0}'.format(exec_path))
        etcd_proc = ETCD(exec_path)
        etcd_proc.setDaemon(True)
        etcd_proc.start()

        log.info('Sleeping...')
        time.sleep(5)

        log.info('Launching watch requests...')
        watch_thread = threading.Thread(target=self.watch_routine)
        watch_thread.setDaemon(True)
        watch_thread.start()

        time.sleep(3)

        log.info('Launching client requests...')
        log.info(put('http://localhost:2379', 'foo', 'bar'))

        self.assertEqual(get('http://localhost:2379', 'foo'), 'bar')
        # Python 3
        # self.assertEqual(get('http://localhost:2379', 'foo'), b'bar')

        log.info('Waing for watch...')
        watch_thread.join()

        log.info('Killing etcd...')
        etcd_proc.kill()

        etcd_proc.join()
        log.info('etcd output: {0}'.format(etcd_proc.stderr))

        log.info('Done!')


if __name__ == '__main__':
    unittest.main()
