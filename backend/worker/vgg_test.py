# -*- coding: utf-8 -*-
"""This script tests VGG modules.
"""

from __future__ import print_function

import unittest

from vgg import VGG16
import glog as log


class TestVGG(unittest.TestCase):
    """VGG test function.

    Examples:
        python -m unittest discover --pattern=vgg*.py -v
    """
    def test_vgg(self):
        """VGG test function.
        """

        log.info('running tests...')
        self.assertEqual('', '')

        vgg = VGG16()
        print(vgg)


if __name__ == '__main__':
    unittest.main()
