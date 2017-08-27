# -*- coding: utf-8 -*-

from __future__ import print_function

import unittest
import os.path

import numpy as np
import glog as log

from .cats.model import *


class TestCats(unittest.TestCase):
    def test_cats(self):
        dpath = os.environ['DATASETS_DIR']
        if dpath == '':
            log.fatal('Got empty DATASETS_DIR')
            sys.exit(0)

        param_path = os.environ['CATS_PARAM_PATH']
        if param_path == '':
            log.fatal('Got empty CATS_PARAM_PATH')
            sys.exit(0)

        log.info('running test_cats...')
        log.info('directory path: {0}'.format(dpath))
        log.info('parameters path: {0}'.format(param_path))

        img_path = os.path.join(dpath, 'gray-cat.jpeg')
        parameters = np.load(param_path).item()

        img_result = classify(img_path, parameters)

        log.info('img_result: {0}'.format(img_result))
        self.assertEqual(img_result, 'cat')


if __name__ == '__main__':
    unittest.main()
