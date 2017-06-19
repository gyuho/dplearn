# -*- coding: utf-8 -*-
"""This module implements a pretrained VGG model wrapper.

Reference:
- https://github.com/fastai/courses
"""

from __future__ import division, print_function

import json

import glog as log
import numpy as np
from keras import backend as K
from keras.layers.convolutional import (Convolution2D, MaxPooling2D,
                                        ZeroPadding2D)
from keras.layers.core import Dense, Dropout, Flatten, Lambda
from keras.models import Sequential
from keras.utils.data_utils import get_file

K.set_image_dim_ordering('th')


RGB_MEAN = np.array([123.68, 116.779, 103.939],
                    dtype=np.float32).reshape((3, 1, 1))


def rgb_to_bgr(img):
    """
        Subtracts the mean RGB value, and transposes RGB to BGR.
        The mean RGB was computed on the image set used to train the VGG model.

        Args:
            img: Image array (height x width x channels)

        Returns:
            Image array (height x width x transposed_channels)
    """
    img = img - RGB_MEAN

    # reverse axis rgb->bgr
    return img[:, ::-1]


DOGSCATS_ZIP_PATH = 'http://files.fast.ai/data/dogscats.zip'

VGG16_H5_PATH = 'http://files.fast.ai/models/vgg16.h5'
IMAGENET_INDEX_PATH = 'http://files.fast.ai/models/imagenet_class_index.json'


class VGG16(object):
    """The VGG 16 Imagenet model.
    """

    def __init__(self):
        self.create()
        self.get_classes()

    def create(self):
        """Creates the VGG16 network achitecture and loads pretrained weights.
        """
        # default cache_dir ${HOME}/.keras in keras/utils/data_utils.py
        # default cache_subdir ${HOME}/.keras/datasets
        log.info('downloading {0}'.format(VGG16_H5_PATH))
        vgg16_h5 = get_file('vgg16.h5', VGG16_H5_PATH,
                            cache_subdir='datasets/dogscats/models')
        log.info('downloaded {0}'.format(VGG16_H5_PATH))

        model = self.model = Sequential()
        model.add(Lambda(rgb_to_bgr, input_shape=(3, 224, 224),
                         output_shape=(3, 224, 224)))

        self.add_convolution_layers(2, 64)
        self.add_convolution_layers(2, 128)
        self.add_convolution_layers(3, 256)
        self.add_convolution_layers(3, 512)
        self.add_convolution_layers(3, 512)

        model.add(Flatten())
        self.add_fully_connected_layers()
        self.add_fully_connected_layers()
        model.add(Dense(1000, activation='softmax'))

        model.load_weights(vgg16_h5)

    def get_classes(self):
        """Downloads the Imagenet classes index file and loads it to self.classes.
        """
        log.info('downloading {0}'.format(IMAGENET_INDEX_PATH))
        class_index_file = get_file('imagenet_class_index.json',
                                    IMAGENET_INDEX_PATH,
                                    cache_subdir='datasets/dogscats/models')
        log.info('downloaded {0}'.format(IMAGENET_INDEX_PATH))

        with open(class_index_file, 'r') as idx_file:
            class_dict = json.load(idx_file)

        self.classes = [class_dict[str(i)][1] for i in range(len(class_dict))]

    def add_convolution_layers(self, layers, filters):
        """Adds a specified number of ZeroPadding and Convolution layers
        to the model, and a MaxPooling layer at the very end.
        Same as 'ConvBlock' in deeplearning1/nbs/vgg16.py.

        Arguments:
            layers (int):   The number of zero padded convolution layers
                            to be added to the model.
            filters (int):  The number of convolution filters to be
                            created for each layer.
        """
        model = self.model
        for _ in range(layers):
            model.add(ZeroPadding2D((1, 1)))
            model.add(Convolution2D(filters, 3, 3, activation='relu'))
        model.add(MaxPooling2D((2, 2), strides=(2, 2)))

    def add_fully_connected_layers(self):
        """Adds a fully connected layer of 4096 neurons to the model
        with a Dropout of 0.5. Same as 'FCBlock' in deeplearning1/nbs/vgg16.py.
        """
        model = self.model
        model.add(Dense(4096, activation='relu'))
        model.add(Dropout(0.5))
