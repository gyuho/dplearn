# -*- coding: utf-8 -*-
"""This module implements a pretrained VGG model wrapper.

Reference:
- https://github.com/fchollet/keras/blob/master/keras/applications
"""

from __future__ import division, print_function

import os

import glog as log
import numpy as np
from keras import backend as K
from keras.layers.convolutional import (Convolution2D, MaxPooling2D,
                                        ZeroPadding2D)
from keras.layers.core import Dense, Dropout, Flatten, Lambda
from keras.models import Sequential
from keras.utils.data_utils import get_file

KERAS_DIR = os.path.join(os.path.expanduser('~'), '.keras')
DOGS_AND_CATS_DATASETS_DIR = os.path.join(KERAS_DIR, 'datasets', 'dogscats')
CLASS_INDEX_PATH = 'http://files.fast.ai/models/imagenet_class_index.json'
WEIGHTS_PATH = 'http://files.fast.ai/models/vgg16.h5'
WEIGHTS_PATH_TF = 'https://github.com/fchollet/deep-learning-models/releases/download/v0.1/vgg16_weights_tf_dim_ordering_tf_kernels.h5'


VGG_MEAN = np.array([123.68, 116.779, 103.939], dtype=np.float32).reshape((3, 1, 1))


def rgb_to_bgr(img):
    """Subtracts the mean RGB value, and transposes RGB to BGR.
    The mean RGB was computed on the image set used to train the VGG model.

    Args
        img: Image array (height x width x channels)

    Returns
        Image array (height x width x transposed_channels)
    """
    img = img - VGG_MEAN
    # reverse axis rgb->bgr
    return img[:, ::-1]


def VGG():
    """VGG implements VGG 16 Imagenet model.

    Reference
        keras/applications/vgg16.py

    Examples
        vgg = Vgg16()
    """
    if not os.path.exists(KERAS_DIR):
        raise ValueError('{0} not exist'.format(KERAS_DIR))

    log.info('running backend {0}'.format(K.backend()))

    # default input width/height for the model (min_size 48)
    # usef default shape when 'include_top' is true
    default_size = 224

    # optional shape tuple
    # '(3, 224, 224)' with 'channels_first' data format (Theano)
    # '(224, 224, 3)' with 'channels_last' data format (Tensorflow)
    if K.backend() == 'theano':
        input_shape = (3, default_size, default_size)
    elif K.backend() == 'tensorflow':
        input_shape = (default_size, default_size, 3)

    layers = Sequential()
    layers.add(Lambda(rgb_to_bgr, input_shape=input_shape, output_shape=input_shape))

    # Convolution layers are for finding patterns in images
    # Dense (fully connected) layers are for combining patterns across an image

    # Block 1
    # add ZeroPadding and Covolution layers to the model, MaxPooling layer at the very end
    for _ in range(2):
        layers.add(ZeroPadding2D((1, 1)))
        layers.add(Convolution2D(64, 3, 3, activation='relu'))
    layers.add(MaxPooling2D((2, 2), strides=(2, 2)))

    # Block 2
    for _ in range(2):
        layers.add(ZeroPadding2D((1, 1)))
        layers.add(Convolution2D(128, 3, 3, activation='relu'))
    layers.add(MaxPooling2D((2, 2), strides=(2, 2)))

    # Block 3
    for _ in range(3):
        layers.add(ZeroPadding2D((1, 1)))
        layers.add(Convolution2D(256, 3, 3, activation='relu'))
    layers.add(MaxPooling2D((2, 2), strides=(2, 2)))

    # Block 4
    for _ in range(3):
        layers.add(ZeroPadding2D((1, 1)))
        layers.add(Convolution2D(512, 3, 3, activation='relu'))
    layers.add(MaxPooling2D((2, 2), strides=(2, 2)))

    # Block 5
    for _ in range(3):
        layers.add(ZeroPadding2D((1, 1)))
        layers.add(Convolution2D(512, 3, 3, activation='relu'))
    layers.add(MaxPooling2D((2, 2), strides=(2, 2)))

    # Classification block
    layers.add(Flatten())

    for _ in range(2):
        layers.add(Dense(4096, activation='relu'))
        layers.add(Dropout(0.5))

    # optional number of classes to classify images into
    classes = 1000
    layers.add(Dense(classes, activation='softmax'))

    # weights that the VGG creators trained, part of the model, learnt from data
    wgt_path = get_file('vgg16.h5', WEIGHTS_PATH, cache_subdir='models')

    log.info("loading weights from {0}".format(wgt_path))
    layers.load_weights(wgt_path)
    log.info("loaded weights from {0}".format(wgt_path))

    return layers
