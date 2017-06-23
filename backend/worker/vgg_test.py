# -*- coding: utf-8 -*-
"""This script tests VGG modules.
"""

from __future__ import print_function

import os
import unittest

import glog as log
from keras.preprocessing import image
from keras.optimizers import Adam
from .vgg import DOGS_AND_CATS_DATASETS_DIR
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

        # generates batches of augmented/normalized data
        # batches with an infinite loop.
        train_gen = image.ImageDataGenerator()
        train_batches = train_gen.flow_from_directory(os.path.join(DOGS_AND_CATS_DATASETS_DIR, 'train'),
                                                      target_size=(224, 224),
                                                      class_mode='categorical',
                                                      shuffle=True,
                                                      batch_size=64)

        log.info('train_batches.batch_size: {0}'.format(train_batches.batch_size))

        # list of all the class labels
        train_classes = list(iter(train_batches.class_indices))
        # sort the class labels by index according to batches.class_indices and update model.classes
        for idx in train_batches.class_indices:
            train_classes[train_batches.class_indices[idx]] = idx

        valid_gen = image.ImageDataGenerator()
        valid_batches = valid_gen.flow_from_directory(os.path.join(DOGS_AND_CATS_DATASETS_DIR, 'valid'),
                                                      target_size=(224, 224),
                                                      class_mode='categorical',
                                                      shuffle=True,
                                                      batch_size=64*2)

        log.info('valid_batches.batch_size: {0}'.format(valid_batches.batch_size))

        # create VGG model
        model = VGG()

        # compile with preloaded weights
        model.compile(optimizer=Adam(lr=0.001), loss='categorical_crossentropy', metrics=['accuracy'])

        model.fit_generator(train_batches,
                            train_batches.batch_size,
                            epochs=1,
                            validation_data=valid_batches,
                            validation_steps=valid_batches.batch_size)
        # ValueError: Error when checking target: expected predictions to have shape (None, 1000) but got array with shape (64, 2)


if __name__ == '__main__':
    unittest.main()
