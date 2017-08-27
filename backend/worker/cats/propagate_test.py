# -*- coding: utf-8 -*-

from __future__ import print_function

import unittest

import numpy as np
import glog as log

from .propagate import *


class TestPropagate(unittest.TestCase):
    def test_activate(self):
        log.info('running test_activate...')
        Z = np.array([-1, 2, 3])
        log.info('Z: {0}'.format(Z))

        A1, cache1 = activate(Z, "sigmoid")
        log.info('sigmoid: {0}'.format(A1))

        A2, cache2 = activate(Z, "relu")
        log.info('relu: {0}'.format(A2))

        Z_sig = np.array([0.26894142, 0.88079708, 0.95257413])
        self.assertTrue(np.allclose(A1, Z_sig))
        Z_rel = np.array([0, 2, 3])
        self.assertTrue(np.allclose(A2, Z_rel))

        self.assertTrue(np.array_equal(cache1, Z))
        self.assertTrue(np.array_equal(cache2, Z))
        self.assertTrue(np.array_equal(cache1, cache2))

        dZ1 = backward_single(A1, cache1, "sigmoid")
        dZ1_expected = np.array([0.05287709, 0.09247804, 0.04303412])
        log.info('sigmoid dZ: {0}'.format(dZ1))
        self.assertTrue(np.allclose(dZ1, dZ1_expected))

        dZ2 = backward_single(A2, cache2, "relu")
        dZ2_expected = np.array([0, 2, 3])
        log.info('relu dZ: {0}'.format(dZ2))
        self.assertTrue(np.allclose(dZ2, dZ2_expected))


if __name__ == '__main__':
    unittest.main()
