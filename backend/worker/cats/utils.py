# -*- coding: utf-8 -*-

from __future__ import division, print_function

import glog as log
import numpy as np


def activation(Z, opt):
    """
    Implements activation function.

    Arguments:
    Z -- numpy array of any shape
    opt -- 'sigmoid' or 'relu'

    Returns:
    cache -- returns Z as well, useful during backpropagation
    """

    cache = Z

    if opt == "sigmoid":
        A = 1/(1+np.exp(-Z))
    elif opt == "relu":
        A = np.maximum(0,Z)

    assert(A.shape == Z.shape)
    return A, cache


def backward_propagation(dA, cache, opt):
    """
    Implements backward propagation for a single activation unit.

    Arguments:
    dA -- post-activation gradient, of any shape
    cache -- 'Z' stored for computing backward propagation efficiently
    opt -- 'sigmoid' or 'relu'

    Returns:
    dZ -- gradient of the cost with respect to Z
    """

    Z = cache

    if opt == "sigmoid":
        # sigmoid of original Z
        s = 1/(1+np.exp(-Z))
        dZ = dA * s * (1-s)
    elif opt == "relu":
        dZ = np.array(dA, copy=True)
        dZ[Z <= 0] = 0

    assert(dZ.shape == Z.shape)
    return dZ
