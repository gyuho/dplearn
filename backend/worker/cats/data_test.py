# -*- coding: utf-8 -*-

from __future__ import print_function

import sys
import unittest

import glog as log

from .data import *


class TestData(unittest.TestCase):
    def test_load(self):
        dpath = os.environ['DATASETS_DIR']
        if dpath == '':
            log.fatal('Got empty DATASETS_DIR')
            sys.exit(0)

        log.info('running test_load...')
        log.info('directory: {0}'.format(dpath))

        train_x_orig, train_y, test_x_orig, test_y, classes = load(dpath)
        log.info('train_x_orig.shape: {0}'.format(train_x_orig.shape))
        log.info('train_y.shape: {0}'.format(train_y.shape))
        log.info('test_x_orig.shape: {0}'.format(test_x_orig.shape))
        log.info('test_y.shape: {0}'.format(test_y.shape))
        log.info('classes.shape: {0}'.format(classes.shape))

        self.assertEqual(train_x_orig.shape, (209, 64, 64, 3))
        self.assertEqual(train_y.shape, (1, 209))
        self.assertEqual(test_x_orig.shape, (50, 64, 64, 3))
        self.assertEqual(test_y.shape, (1, 50))
        self.assertEqual(classes.shape, (2,))


if __name__ == '__main__':
    unittest.main()
