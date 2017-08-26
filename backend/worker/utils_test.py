# -*- coding: utf-8 -*-
"""This script tests VGG modules.
"""

from __future__ import print_function

import os
import unittest

import numpy as np
import glog as log

from .utils import activation


class TestUtils(unittest.TestCase):
    """Utils test function.

    Examples:
        pushd ..
        python3 -m unittest worker.utils_test
        popd
    """
    def test_activation(self):
        log.info('running tests...')
        Z = np.array([-1, 2, 3])
        log.info('Z: {0}'.format(Z))

        A1, cache1 = activation(Z, "sigmoid")
        log.info('sigmoid: {0}'.format(A1))

        A2, cache2 = activation(Z, "relu")
        log.info('relu: {0}'.format(A2))

        A_sig = np.array([0.26894142, 0.88079708, 0.95257413])
        self.assertTrue(np.allclose(A1, A_sig))
        A_rel = np.array([0, 2, 3])
        self.assertTrue(np.allclose(A2, A_rel))

        self.assertTrue(np.array_equal(cache1, Z))
        self.assertTrue(np.array_equal(cache2, Z))
        self.assertTrue(np.array_equal(cache1, cache2))



if __name__ == '__main__':
    unittest.main()
