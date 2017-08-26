# -*- coding: utf-8 -*-
"""This script tests VGG modules.
"""

from __future__ import print_function

import os
import unittest

import glog as log

# from .utils import *


class TestUtils(unittest.TestCase):
    """Utils test function.

    Examples:
        pushd ..
        python -m unittest worker.utils_test
        popd
    """
    def test_utils(self):
        """Utils test function.
        """

        log.info('running tests...')
        self.assertEqual('', '')

        log.info('train_batches.batch_size: {0}'.format(1))
        log.info('valid_batches.batch_size: {0}'.format(2))


if __name__ == '__main__':
    unittest.main()
