# -*- coding: utf-8 -*-

from __future__ import division, print_function

import glog as log
import numpy as np


def parameters_2layer(n_x, n_h, n_y):
    """
    Argument:
    n_x -- size of the input layer
    n_h -- size of the hidden layer
    n_y -- size of the output layer

    Returns:
    parameters -- dictionary containing your parameters:
                  W1 -- weight matrix of shape (n_h, n_x)
                  b1 -- bias vector of shape (n_h, 1)
                  W2 -- weight matrix of shape (n_y, n_h)
                  b2 -- bias vector of shape (n_y, 1)
    """

    np.random.seed(1)

    W1 = np.random.randn(n_h, n_x)*0.01
    b1 = np.zeros((n_h, 1))
    W2 = np.random.randn(n_y, n_h)*0.01
    b2 = np.zeros((n_y, 1))

    assert(W1.shape == (n_h, n_x))
    assert(b1.shape == (n_h, 1))
    assert(W2.shape == (n_y, n_h))
    assert(b2.shape == (n_y, 1))

    parameters = {"W1": W1,
                  "b1": b1,
                  "W2": W2,
                  "b2": b2}

    return parameters


def deep(layer_dims):
    """
    Arguments:
    layer_dims -- list of dimensions in each layer

    Returns:
    parameters -- dictionary with parameters "W1", "b1", ..., "WL", "bL":
                  Wl -- weight matrix of shape (layer_dims[l], layer_dims[l-1])
                  bl -- bias vector of shape (layer_dims[l], 1)
    """

    np.random.seed(1)

    # number of layers in the network
    L = len(layer_dims)

    parameters = {}

    for i in range(1, L):
        w = 'W' + str(i)
        b = 'b' + str(i)
        prevd = layer_dims[i-1]
        curd = layer_dims[i]

        parameters[w] = np.random.randn(curd, prevd) / np.sqrt(prevd)
        parameters[b] = np.zeros((curd, 1))

        assert(parameters[w].shape == (curd, prevd))
        assert(parameters[b].shape == (curd, 1))

    return parameters

