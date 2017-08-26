# -*- coding: utf-8 -*-
"""This module implements various utilities.
"""

from __future__ import division, print_function

import os

import glog as log
import numpy as np


def activation(Z, opt):
    """
    Implements activation functions.

    Args
        Z: numpy array of any shape
        opt: 'sigmoid' or 'relu'

    Returns
        A: Post-activation parameter, of same shape as Z
        cache: returns Z as well, useful during backpropagation
    """

    cache = Z

    if opt == "sigmoid":
        A = 1/(1+np.exp(-Z))
    elif opt == "relu":
        A = np.maximum(0,Z)

    assert(A.shape == Z.shape)

    return A, cache
