#!/usr/bin/env bash
set -e

brew install python
brew install python3

sudo easy_install pip
sudo easy_install --upgrade pip
pip -V

sudo easy_install six
sudo easy_install --upgrade six
pip3 -V

# CPU support
pip install tensorflow
pip3 install tensorflow
pip install matplotlib
pip3 install matplotlib
