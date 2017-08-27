# -*- coding: utf-8 -*-

from __future__ import division, print_function

import os.path
import sys

import h5py
import numpy as np
import glog as log


def load(dpath):
    """
    Loads data from directory.

    Arguments:
    dpath -- directory

    Returns:
    train_set_x_orig -- train set features
    train_set_y_orig -- train set labels
    test_set_x_orig -- train set features
    test_set_y_orig -- test set labels
    classes -- list of classes
    """

    if not os.path.exists(dpath):
        log.fatal('{0} does not eixst'.format(dpath))
        sys.exit(0)

    train_dataset = h5py.File(os.path.join(dpath, 'train_catvnoncat.h5'), "r")
    train_set_x_orig = np.array(train_dataset["train_set_x"][:])
    train_set_y_orig = np.array(train_dataset["train_set_y"][:])

    test_dataset = h5py.File(os.path.join(dpath, 'test_catvnoncat.h5'), "r")
    test_set_x_orig = np.array(test_dataset["test_set_x"][:])
    test_set_y_orig = np.array(test_dataset["test_set_y"][:])

    classes = np.array(test_dataset["list_classes"][:])

    train_set_y_orig = train_set_y_orig.reshape((1, train_set_y_orig.shape[0]))
    test_set_y_orig = test_set_y_orig.reshape((1, test_set_y_orig.shape[0]))

    return train_set_x_orig, train_set_y_orig, test_set_x_orig, test_set_y_orig, classes
