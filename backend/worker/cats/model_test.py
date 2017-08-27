# -*- coding: utf-8 -*-

from __future__ import print_function

import unittest
import os.path

import numpy as np
import glog as log
import scipy
from scipy import ndimage

from .data import *
from .initialize import *
from .model import *
from .propagate import *


class TestModel(unittest.TestCase):
    def test_model(self):
        dpath = os.environ['DATASETS_DIR']
        if dpath == '':
            log.fatal('Got empty DATASETS_DIR')
            sys.exit(0)

        param_path = os.environ['CATS_PARAM_PATH']
        if param_path == '':
            log.fatal('Got empty CATS_PARAM_PATH')
            sys.exit(0)

        log.info('running test_model...')
        log.info('directory path: {0}'.format(dpath))
        log.info('parameters path: {0}'.format(param_path))

        train_x_orig, train_y, test_x_orig, test_y, classes = load(dpath)
        log.info('classes: {0} {1}'.format(classes, type(classes)))

        # Reshape the training and test examples
        # The "-1" makes reshape flatten the remaining dimensions
        train_x_flatten = train_x_orig.reshape(train_x_orig.shape[0], -1).T
        test_x_flatten = test_x_orig.reshape(test_x_orig.shape[0], -1).T

        # Standardize data to have feature values between 0 and 1.
        train_x = train_x_flatten/255.
        test_x = test_x_flatten/255.

        log.info("train_x's shape: {0}".format(train_x.shape))
        log.info("test_x's shape: {0}".format(test_x.shape))
        self.assertEqual(train_x.shape, (12288, 209))
        self.assertEqual(test_x.shape, (12288, 50))

        # 1. Initialize parameters / Define hyperparameters
        # 2. Loop for num_iterations:
        #     a. Forward propagation
        #     b. Compute cost function
        #     c. Backward propagation
        #     d. Update parameters (using parameters, and grads from backprop)
        # 4. Use trained parameters to predict labels

        # 5-layer model
        layers_dims = [12288, 20, 7, 5, 1]
        parameters = L_layer(
            train_x,
            train_y,
            layers_dims,
            num_iterations=2500)

        log.info("parameters: {0}".format(parameters))
        log.info("parameters type: {0}".format(type(parameters)))

        pred_train = predict(train_x, train_y, parameters)
        pred_train_accuracy = np.sum((pred_train == train_y) / train_x.shape[1])
        pred_test = predict(test_x, test_y, parameters)
        pred_test_accuracy = np.sum((pred_test == test_y) / test_x.shape[1])

        log.info("pred_train: {0}".format(pred_train_accuracy))
        log.info("pred_test: {0}".format(pred_test_accuracy))
        self.assertGreater(pred_train_accuracy, 0.98)
        self.assertGreaterEqual(pred_test_accuracy, 0.8)

        log.info('saving: {0}'.format(param_path))
        np.save(param_path, parameters)
        log.info('saved: {0}'.format(param_path))

        num_px = train_x_orig.shape[1]
        my_label_y = [1]  # true class (1->cat, 0->non-cat)
        img_path = os.path.join(dpath, 'gray-cat.jpeg')
        img = np.array(ndimage.imread(img_path, flatten=False))
        img_resized = scipy.misc.imresize(img, size=(num_px,num_px)).reshape((num_px*num_px*3,1))
        img_pred = predict(img_resized, my_label_y, parameters)

        img_accuracy = np.squeeze(img_pred)
        img_class = classes[int(img_accuracy),].decode("utf-8")
        log.info('img_accuracy: {0}'.format(img_accuracy))
        log.info('img_class: {0}'.format(img_class))
        self.assertEqual(img_accuracy, 1.0)
        self.assertEqual(img_class, 'cat')


    def test_persistent_model(self):
        dpath = os.environ['DATASETS_DIR']
        if dpath == '':
            log.fatal('Got empty DATASETS_DIR')
            sys.exit(0)

        param_path = os.environ['CATS_PARAM_PATH']
        if param_path == '':
            log.fatal('Got empty CATS_PARAM_PATH')
            sys.exit(0)

        log.info('running test_test_persistent_model...')
        log.info('directory path: {0}'.format(dpath))
        log.info('parameters path: {0}'.format(param_path))

        parameters = np.load(param_path).item()

        classes = np.array([b'non-cat', b'cat'])
        num_px = 64
        my_label_y = [1]  # true class (1->cat, 0->non-cat)
        img_path = os.path.join(dpath, 'gray-cat.jpeg')
        img = np.array(ndimage.imread(img_path, flatten=False))
        img_resized = scipy.misc.imresize(img, size=(num_px,num_px)).reshape((num_px*num_px*3,1))
        img_pred = predict(img_resized, my_label_y, parameters)

        img_accuracy = np.squeeze(img_pred)
        img_class = classes[int(img_accuracy),].decode("utf-8")
        log.info('img_accuracy: {0}'.format(img_accuracy))
        log.info('img_class: {0}'.format(img_class))
        self.assertEqual(img_accuracy, 1.0)
        self.assertEqual(img_class, 'cat')


if __name__ == '__main__':
    unittest.main()
