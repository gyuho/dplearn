# -*- coding: utf-8 -*-
"""This script tests VGG modules.
"""

from __future__ import print_function

import unittest

import glog as log

from .vgg import VGG


class TestVGG(unittest.TestCase):
    """VGG test function.

    Examples:
        pushd ..
        python -m unittest worker.vgg_test
        popd
    """
    def test_vgg(self):
        """VGG test function.
        """

        log.info('running tests...')
        self.assertEqual('', '')

        vgg = VGG()
        print(vgg)


if __name__ == '__main__':
    unittest.main()
