# -*- coding: utf-8 -*-
"""This script tests VGG modules.
"""

from __future__ import print_function

import os
import unittest

import glog as log
from keras.layers.core import Dense
from keras.optimizers import Adam
from keras.preprocessing import image

from .vgg import DOGS_AND_CATS_DATASETS_DIR, VGG


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

        # create VGG model
        model = VGG()

        # get_batches: generates batches of augmented/normalized data
        # batches with an infinite loop.
        train_gen = image.ImageDataGenerator()
        train_batches = train_gen.flow_from_directory(os.path.join(DOGS_AND_CATS_DATASETS_DIR, 'train'),
                                                      target_size=(224, 224),
                                                      class_mode='categorical',
                                                      shuffle=True,
                                                      batch_size=64)

        valid_gen = image.ImageDataGenerator()
        valid_batches = valid_gen.flow_from_directory(os.path.join(DOGS_AND_CATS_DATASETS_DIR, 'valid'),
                                                      target_size=(224, 224),
                                                      class_mode='categorical',
                                                      shuffle=True,
                                                      batch_size=64*2)

        log.info('train_batches.batch_size: {0}'.format(train_batches.batch_size))
        log.info('valid_batches.batch_size: {0}'.format(valid_batches.batch_size))

        # ft: list of all the class labels
        model.pop()
        for layer in model.layers:
            layer.trainable = False
        model.add(Dense(train_batches.nb_class, activation='softmax'))
        # compile with preloaded weights
        model.compile(optimizer=Adam(lr=0.001), loss='categorical_crossentropy', metrics=['accuracy'])

        # vgg.finetune(batches)
        # finetune: sort the class labels by index according
        # to batches.class_indices and update model.classes
        train_classes = list(iter(train_batches.class_indices))
        for idx in train_batches.class_indices:
            train_classes[train_batches.class_indices[idx]] = idx

        # vgg.fit(batches, val_batches, nb_epoch=1)
        model.fit_generator(train_batches,
                            samples_per_epoch=train_batches.nb_sample,
                            nb_epoch=1,
                            validation_data=valid_batches,
                            nb_val_samples=valid_batches.nb_sample)


if __name__ == '__main__':
    unittest.main()
