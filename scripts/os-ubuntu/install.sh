#!/usr/bin/env bash
set -e

apt-get -y update
apt-get -y upgrade
apt-get -f install

apt-get -y remove docker docker-engine && curl -sSL https://get.docker.com/ | sh

apt-get -f install

# Check for CUDA and try to install.
# https://cloud.google.com/compute/docs/gpus/add-gpus
if ! dpkg-query -W cuda; then
  wget -P /tmp https://github.com/NVIDIA/nvidia-docker/releases/download/v1.0.1/nvidia-docker_1.0.1-1_amd64.deb
  dpkg -i /tmp/nvidia-docker*.deb
  curl -O http://developer.download.nvidia.com/compute/cuda/repos/ubuntu1604/x86_64/cuda-repo-ubuntu1604_8.0.61-1_amd64.deb
  dpkg -i ./cuda-repo-ubuntu1604_8.0.61-1_amd64.deb
  apt-get -y update
  apt-get -y install cuda
  modprobe nvidia
  nvidia-smi
fi

apt-get -y update
apt-get -y install \
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
  nginx \
  libcupti-dev \
  rsync \
  python \
  python-pip \
  python-dev \
  python3-pip

apt-get -y update
apt-get -y upgrade
apt-get -y dist-upgrade
apt-get -y update
apt-get -y upgrade
apt-get -y autoremove
apt-get -y autoclean

echo "root ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers
