# -*- coding: utf-8 -*-
"""This module implements various utilities.
"""

from __future__ import division, print_function

import os

import glog as log
import numpy as np


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
