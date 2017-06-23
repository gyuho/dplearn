# -*- coding: utf-8 -*-
"""This module implements a pretrained VGG model wrapper.

Reference:
- https://github.com/fchollet/keras/blob/master/keras/applications
"""

from __future__ import division, print_function

import os

import glog as log
from keras import backend as K
from keras.layers import Conv2D, Dense, Flatten, Input, MaxPooling2D
from keras.models import Model
from keras.utils.data_utils import get_file

KERAS_DIR = os.path.join(os.path.expanduser('~'), '.keras')
DOGS_AND_CATS_DATASETS_DIR = os.path.join(KERAS_DIR, 'datasets', 'dogscats')
WEIGHTS_PATH = 'https://github.com/fchollet/deep-learning-models/releases/download/v0.1/vgg16_weights_tf_dim_ordering_tf_kernels.h5'


def VGG():
    """VGG implements VGG 16 Imagenet model.

    Reference
        keras/applications/vgg16.py

    Examples
        vgg = Vgg16()
    """
    if not os.path.exists(KERAS_DIR):
        raise ValueError('{0} not exist'.format(KERAS_DIR))

    if K.backend() == 'theano' or K.backend() != 'tensorflow':
        raise ValueError('only support TensorFlow for now')

    if K.image_data_format() != 'channels_last':
        raise ValueError('TensorFlow backend image data format'
                         'expects channels_last')

    # include the 3 fully-connected layers at the top of the network.
    include_top = True

    # one of `None` (random initialization)
    # or "imagenet" (pre-training on ImageNet)
    weights = 'imagenet'

    # optional number of classes to classify images into
    classes = 1000

    # default input width/height for the model (min_size 48)
    default_size = 224

    # optional shape tuple
    # '(3, 224, 224)' with 'channels_first' data format
    # '(224, 224, 3)' with 'channels_last' data format
    # usef default shape when 'include_top' is true
    input_shape = (default_size, default_size, 3)

    # instantiate a Keras tensor, `shape=(32,)` indicates that
    # expected input will be batches of 32-dimensional vectors.
    img_input = Input(shape=input_shape)

    # Convolution layers are for finding patterns in images
    # Dense (fully connected) layers are for combining patterns across an image

    # Block 1
    layer = Conv2D(64, (3, 3), activation='relu', padding='same', name='block1_conv1')(img_input)
    layer = Conv2D(64, (3, 3), activation='relu', padding='same', name='block1_conv2')(layer)
    layer = MaxPooling2D((2, 2), strides=(2, 2), name='block1_pool')(layer)

    # Block 2
    layer = Conv2D(128, (3, 3), activation='relu', padding='same', name='block2_conv1')(layer)
    layer = Conv2D(128, (3, 3), activation='relu', padding='same', name='block2_conv2')(layer)
    layer = MaxPooling2D((2, 2), strides=(2, 2), name='block2_pool')(layer)

    # Block 3
    layer = Conv2D(256, (3, 3), activation='relu', padding='same', name='block3_conv1')(layer)
    layer = Conv2D(256, (3, 3), activation='relu', padding='same', name='block3_conv2')(layer)
    layer = Conv2D(256, (3, 3), activation='relu', padding='same', name='block3_conv3')(layer)
    layer = MaxPooling2D((2, 2), strides=(2, 2), name='block3_pool')(layer)

    # Block 4
    layer = Conv2D(512, (3, 3), activation='relu', padding='same', name='block4_conv1')(layer)
    layer = Conv2D(512, (3, 3), activation='relu', padding='same', name='block4_conv2')(layer)
    layer = Conv2D(512, (3, 3), activation='relu', padding='same', name='block4_conv3')(layer)
    layer = MaxPooling2D((2, 2), strides=(2, 2), name='block4_pool')(layer)

    # Block 5
    layer = Conv2D(512, (3, 3), activation='relu', padding='same', name='block5_conv1')(layer)
    layer = Conv2D(512, (3, 3), activation='relu', padding='same', name='block5_conv2')(layer)
    layer = Conv2D(512, (3, 3), activation='relu', padding='same', name='block5_conv3')(layer)
    layer = MaxPooling2D((2, 2), strides=(2, 2), name='block5_pool')(layer)

    # Classification block
    layer = Flatten(name='flatten')(layer)
    layer = Dense(4096, activation='relu', name='fc1')(layer)
    layer = Dense(4096, activation='relu', name='fc2')(layer)
    layer = Dense(classes, activation='softmax', name='predictions')(layer)

    # (assume no 'input_tensor')
    log.info("creating model with include_top {0}, weights {1}".format(include_top, weights))
    model = Model(img_input, layer, name='vgg16')
    log.info("created model with include_top {0}, weights {1}".format(include_top, weights))

    # weights that the VGG creators trained, part of the model, learnt from data
    wgt_path = get_file('vgg16_weights_tf_dim_ordering_tf_kernels.h5',
                        WEIGHTS_PATH,
                        cache_subdir='models')

    log.info("loading weights from {0}".format(wgt_path))
    model.load_weights(wgt_path)
    log.info("loaded weights from {0}".format(wgt_path))

    return model
