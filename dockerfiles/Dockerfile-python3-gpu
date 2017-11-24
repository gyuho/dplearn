##########################
# https://github.com/tensorflow/tensorflow/blob/master/tensorflow/tools/docker/Dockerfile
# https://github.com/tensorflow/tensorflow/blob/master/tensorflow/tools/docker/Dockerfile.gpu
# https://gcr.io/tensorflow/tensorflow
FROM gcr.io/tensorflow/tensorflow:1.3.0-gpu-py3
##########################

##########################
# Set working directory
ENV ROOT_DIR /
WORKDIR ${ROOT_DIR}
ENV HOME /root
##########################

##########################
# Update OS
# Configure 'bash' for 'source' commands
RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections \
  && apt-get -y update \
  && apt-get -y install \
  build-essential \
  gcc \
  apt-utils \
  pkg-config \
  software-properties-common \
  apt-transport-https \
  libssl-dev \
  sudo \
  bash \
  bash-completion \
  tar \
  unzip \
  curl \
  wget \
  git \
  libcupti-dev \
  rsync \
  python \
  python-pip \
  python-dev \
  python3-pip \
  libhdf5-dev \
  python-tk \
  python3-tk \
  && rm /bin/sh \
  && ln -s /bin/bash /bin/sh \
  && ls -l $(which bash) \
  && echo "root ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers \
  && apt-get -y clean \
  && rm -rf /var/lib/apt/lists/* \
  && apt-get -y update \
  && apt-get -y upgrade \
  && apt-get -y dist-upgrade \
  && apt-get -y update \
  && apt-get -y upgrade \
  && apt-get -y autoremove \
  && apt-get -y autoclean
##########################

##########################
# Tensorflow GPU image already includes https://developer.nvidia.com/cudnn
# https://github.com/fastai/courses/blob/master/setup/install-gpu.sh
# RUN ls /usr/local/cuda/lib64/
# RUN ls /usr/local/cuda/include/
##########################

##########################
# install basic packages
RUN pip3 --no-cache-dir install \
  requests \
  glog \
  humanize \
  bcolz \
  h5py
##########################

##########################
# install Keras
RUN pip3 --no-cache-dir install \
  theano \
  keras==2.0.8 \
  && echo $'[global]\n\
device = gpu\n\
floatX = float32\n\
[cuda]\n\
root = /usr/local/cuda\n'\
> ${HOME}/.theanorc \
  && cat ${HOME}/.theanorc \
  && mkdir -p ${HOME}/.keras/datasets \
  && mkdir -p ${HOME}/.keras/models \
  && echo $'{\n\
  "image_data_format": "channels_last",\n\
  "epsilon": 1e-07,\n\
  "floatx": "float32",\n\
  "backend": "tensorflow"\n\
}\n'\
> ${HOME}/.keras/keras.json \
  && cat ${HOME}/.keras/keras.json
##########################

##########################
# Clone source code, static assets
ADD ./datasets/parameters-cats.npy /root/datasets/parameters-cats.npy
ADD ./backend ${ROOT_DIR}/backend
ADD ./scripts/docker/run ${ROOT_DIR}/scripts/docker/run
ADD ./scripts/tests ${ROOT_DIR}/scripts/tests
##########################

##########################
# Configure Jupyter
ADD ./jupyter_notebook_config.py /root/.jupyter/

# Jupyter has issues with being run directly: https://github.com/ipython/ipython/issues/7062
# We just add a little wrapper script.
ADD ./run_jupyter.sh /
##########################

