# Last updated at 2017-06-18 02:59:47.580581499 -0700 PDT
# https://github.com/tensorflow/tensorflow/blob/master/tensorflow/tools/docker/Dockerfile
# https://github.com/tensorflow/tensorflow/blob/master/tensorflow/tools/docker/Dockerfile.gpu
# https://gcr.io/tensorflow/tensorflow

##########################
FROM gcr.io/tensorflow/tensorflow:1.2.0-gpu
##########################

##########################
# Set working directory
ENV ROOT_DIR /
WORKDIR ${ROOT_DIR}
##########################

##########################
# Update OS
# Configure 'bash' for 'source' commands
RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections \
  && rm /bin/sh \
  && ln -s /bin/bash /bin/sh \
  && ls -l $(which bash) \
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
  r-base \
  fonts-dejavu \
  gfortran \
  nginx \
  && echo "root ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers \
  && apt-get -y clean \
  && rm -rf /var/lib/apt/lists/* \
  && apt-get -y update \
  && apt-get -y upgrade \
  && apt-get -y dist-upgrade \
  && apt-get -y update \
  && apt-get -y upgrade \
  && apt-get -y autoremove \
  && apt-get -y autoclean \
  && wget http://repo.continuum.io/miniconda/Miniconda3-3.7.0-Linux-x86_64.sh -O /root/miniconda.sh \
  && bash /root/miniconda.sh -b -p /root/miniconda

# do not overwrite default '/usr/bin/python'
ENV PATH ${PATH}:/root/miniconda/bin

# Configure reverse proxy
RUN mkdir -p /etc/nginx/sites-available/
ADD nginx.conf /etc/nginx/sites-available/default
##########################

##########################
# Install additional Python libraries
ENV HOME /root

# install 'pip --no-cache-dir install' with default python
# use vanilla python from tensorflow base image, as much as possible
# install 'conda install --name r ...' just for R (source activate r)
# install 'conda install --name py36 ...' just for Python 3 (source activate py36)
# https://github.com/Anaconda-Platform/nb_conda
# https://github.com/jupyter/docker-stacks/blob/master/r-notebook/Dockerfile
RUN pip --no-cache-dir install \
  requests \
  glog \
  humanize \
  bcolz \
  theano \
  keras==1.2.2 \
  && echo $'[global]\n\
device = gpu\n\
floatX = float32\n\
[cuda]\n\
root = /usr/local/cuda\n'\
> ${HOME}/.theanorc \
  && cat ${HOME}/.theanorc \
  && mkdir -p ${HOME}/.keras \
  && echo $'{\n\
  "image_dim_ordering": "th",\n\
  "epsilon": 1e-07,\n\
  "floatx": "float32",\n\
  "backend": "theano"\n\
}\n'\
> ${HOME}/.keras/keras.json \
  && cat ${HOME}/.keras/keras.json \
  && conda update conda \
  && conda create --yes --name r \
  python=2.7 \
  ipykernel \
  r \
  r-essentials \
  'r-base=3.3.2' \
  'r-irkernel=0.7*' \
  'r-plyr=1.8*' \
  'r-devtools=1.12*' \
  'r-tidyverse=1.0*' \
  'r-shiny=0.14*' \
  'r-rmarkdown=1.2*' \
  'r-forecast=7.3*' \
  'r-rsqlite=1.1*' \
  'r-reshape2=1.4*' \
  'r-nycflights13=0.2*' \
  'r-caret=6.0*' \
  'r-rcurl=1.95*' \
  'r-crayon=1.3*' \
  'r-randomforest=4.6*' \
  && conda create --yes --name py36 \
  python=3.6 \
  ipykernel \
  requests \
  glog \
  humanize \
  && conda clean -tipsy \
  && conda list \
  && python -V \
  && pip freeze --all

# "image_dim_ordering": "tf"
# "backend": "tensorflow"

# Tensorflow GPU image already includes https://developer.nvidia.com/cudnn
# https://github.com/fastai/courses/blob/master/setup/install-gpu.sh
# RUN ls /usr/local/cuda/lib64/
# RUN ls /usr/local/cuda/include/

# Configure Jupyter
ADD ./jupyter_notebook_config.py /root/.jupyter/

# Jupyter has issues with being run directly: https://github.com/ipython/ipython/issues/7062
# We just add a little wrapper script.
ADD ./run_jupyter.sh /
##########################

##########################
# Install Go for backend
ENV GOROOT /usr/local/go
ENV GOPATH /gopath
ENV PATH ${GOPATH}/bin:${GOROOT}/bin:${PATH}
ENV GO_VERSION 1.9beta1
ENV GO_DOWNLOAD_URL https://storage.googleapis.com/golang
RUN rm -rf ${GOROOT} \
  && curl -s ${GO_DOWNLOAD_URL}/go${GO_VERSION}.linux-amd64.tar.gz | tar -v -C /usr/local/ -xz \
  && mkdir -p ${GOPATH}/src ${GOPATH}/bin \
  && go version
##########################

##########################
# Install etcd
ENV ETCD_GIT_PATH github.com/coreos/etcd

RUN mkdir -p ${GOPATH}/src/github.com/coreos \
  && git clone https://github.com/coreos/etcd --branch master ${GOPATH}/src/${ETCD_GIT_PATH} \
  && pushd ${GOPATH}/src/${ETCD_GIT_PATH} \
  && git reset --hard HEAD \
  && ./build \
  && cp ./bin/* / \
  && popd \
  && rm -rf ${GOPATH}/src/${ETCD_GIT_PATH}
##########################

##########################
# Clone source code, dependencies
RUN mkdir -p ${GOPATH}/src/github.com/gyuho/deephardway
ADD . ${GOPATH}/src/github.com/gyuho/deephardway

# Symlinks to notebooks notebooks
RUN ln -s /gopath/src/github.com/gyuho/deephardway /git-deep \
  && pushd ${GOPATH}/src/github.com/gyuho/deephardway \
  && go install -v ./cmd/backend-web-server \
  && go install -v ./cmd/gen-nginx-conf \
  && go install -v ./cmd/gen-package-json \
  && go install -v ./cmd/download-data \
  && popd
##########################

##########################
# Install Angular, NodeJS for frontend
# 'node' needs to be in $PATH for 'yarn start' command
ENV NVM_DIR /usr/local/nvm
RUN pushd ${GOPATH}/src/github.com/gyuho/deephardway \
  && curl https://raw.githubusercontent.com/creationix/nvm/v0.33.2/install.sh | /bin/bash \
  && echo "Running nvm scripts..." \
  && source $NVM_DIR/nvm.sh \
  && nvm ls-remote \
  && nvm install 7.10.0 \
  && curl https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - \
  && echo "deb http://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list \
  && apt-get -y update && apt-get -y install yarn \
  && rm -rf ./node_modules \
  && yarn install \
  && npm rebuild node-sass \
  && npm install \
  && cp /usr/local/nvm/versions/node/v7.10.0/bin/node /usr/bin/node \
  && popd
##########################

##########################
# Backend, do not expose to host
# Just run with frontend, in one container
# EXPOSE 2200

# Frontend
EXPOSE 4200

# TensorBoard
# EXPOSE 6006

# IPython
EXPOSE 8888

# Web server
EXPOSE 80
##########################

##########################
# Log installed components
RUN cat /etc/lsb-release >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && uname -a >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo Python2: $(python -V 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo Python3: $(python3 -V 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo IPython: $(ipython -V 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo Jupyter: $(jupyter --version 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo pip-freeze: $(pip freeze --all 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo R: $(R --version 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo Anaconda version: $(conda --version 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo Anaconda list: $(conda list 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo Anaconda info --envs: $(conda info --envs 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo Go: $(go version 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo yarn: $(yarn --version 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo node: $(node --version 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo NPM: $(/usr/local/nvm/versions/node/v7.10.0/bin/npm --version 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo Angular-CLI: $(${GOPATH}/src/github.com/gyuho/deephardway/node_modules/.bin/ng --version 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo etcd: $(/etcd --version 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo etcdctl: $(ETCDCTL_API=3 /etcdctl version 2>&1) >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && cat ${GOPATH}/src/github.com/gyuho/deephardway/git-tensorflow.json >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && cat ${GOPATH}/src/github.com/gyuho/deephardway/git-fastai-courses.json >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && cat /container-version.txt
##########################
