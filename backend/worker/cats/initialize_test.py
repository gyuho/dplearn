# -*- coding: utf-8 -*-

from __future__ import print_function

import unittest

import glog as log

from .initialize import *


class TestInitializeParameters(unittest.TestCase):
    def test_initialize(self):
        log.info('running test_initialize...')
        #  5-layer model
        layers_dims = [12288, 20, 7, 5, 1]
        log.info('layers_dims: {0}'.format(layers_dims))
        parameters = deep_parameters(layers_dims)
        log.info('parameters: {0}'.format(parameters))
        self.assertEqual(parameters['W1'].shape, (20, 12288))
        self.assertEqual(parameters['b1'].shape, (20, 1))
        self.assertEqual(parameters['W3'].shape, (5, 7))
        self.assertEqual(parameters['b3'].shape, (5, 1))


if __name__ == '__main__':
    unittest.main()
