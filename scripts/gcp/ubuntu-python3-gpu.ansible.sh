#!/usr/bin/env bash
set -e

echo "root ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

##########################################################
apt-get -y --allow-unauthenticated install ansible

cat > /etc/ansible-install.yml <<EOF
---
- name: a play that runs entirely on the ansible host
  hosts: localhost
  connection: local

  environment:
    PATH: /usr/local/go/bin:/opt/bin:/home/gyuho/go/bin:{{ ansible_env.PATH }}
    GOPATH: /home/gyuho/go

  tasks:
  - file:
      path: /opt/bin
      state: directory
      mode: 0777

  - file:
      path: /var/lib/etcd
      state: directory
      mode: 0777

  - file:
      path: /var/lib/keras/datasets
      state: directory
      mode: 0777

  - file:
      path: /var/lib/keras/models
      state: directory
      mode: 0777

  - file:
      path: /home/gyuho/go
      state: directory
      mode: 0777

  - name: Install Linux utils
    become: yes
    apt:
      name={{item}}
      state=latest
      update_cache=yes
      force=yes
    with_items:
    - build-essential
    - gcc
    - apt-utils
    - pkg-config
    - software-properties-common
    - apt-transport-https
    - libssl-dev
    - sudo
    - bash
    - bash-completion
    - tar
    - unzip
    - curl
    - wget
    - git
    - libcupti-dev
    - rsync
    - python
    - python-pip
    - python-dev
    - python3-pip

  - name: Download GCP key
    get_url:
      url=http://metadata.google.internal/computeMetadata/v1/instance/attributes/gcp-key
      dest=/etc/gcp-key-dplearn.json
      headers='Metadata-Flavor:Google'

  - name: Download Docker installer
    get_url:
      url=https://get.docker.com
      dest=/tmp/docker.sh

  - name: Execute the docker.sh
    script: /tmp/docker.sh

  - name: Download GPU driver
    get_url:
      url=https://github.com/NVIDIA/nvidia-docker/releases/download/v1.0.1/nvidia-docker_1.0.1-1_amd64.deb
      dest=/tmp/nvidia-docker_1.0.1-1_amd64.deb

  - name: Install GPU driver
    sudo: True
    command: dpkg -i /tmp/nvidia-docker_1.0.1-1_amd64.deb

  - name: Download CUDA driver
    get_url:
      url=http://developer.download.nvidia.com/compute/cuda/repos/ubuntu1604/x86_64/cuda-repo-ubuntu1604_8.0.61-1_amd64.deb
      dest=/tmp/cuda-repo-ubuntu1604_8.0.61-1_amd64.deb

  - name: Install CUDA driver
    sudo: True
    command: dpkg -i /tmp/cuda-repo-ubuntu1604_8.0.61-1_amd64.deb

  - name: Install CUDA in Linux
    become: yes
    apt:
      name={{item}}
      state=latest
      update_cache=yes
      force=yes
    with_items:
    - cuda

  - modprobe: name=nvidia state=absent

  - name: Check nvidia-smi
    command: nvidia-smi
    register: result
  - debug:
      var: result.stderr
EOF

ansible-playbook /etc/ansible-install.yml > /etc/ansible-install.log 2>&1
##########################################################

##########################################################
systemctl daemon-reload
systemctl stop nvidia-docker.service
systemctl disable nvidia-docker.service
systemctl enable nvidia-docker.service
systemctl start nvidia-docker.service
##########################################################

##########################################################
cat > /tmp/app.service <<EOF
[Unit]
Description=dplearn GPU development service
Documentation=https://github.com/gyuho/dplearn

[Service]
Restart=always
RestartSec=5s
TimeoutStartSec=0
LimitNOFILE=40000

ExecStartPre=/usr/bin/docker login -u oauth2accesstoken -p "$(/usr/bin/gcloud auth application-default print-access-token)" https://gcr.io
ExecStartPre=/usr/bin/docker pull gcr.io/gcp-dplearn/dplearn:latest-app

ExecStart=/usr/bin/nvidia-docker run \
  --rm \
  --name app \
  --volume=/var/lib/etcd:/var/lib/etcd \
  -p 4200:4200 \
  --ulimit nofile=262144:262144 \
  gcr.io/gcp-dplearn/dplearn:latest-app \
  /bin/sh -c "./scripts/run/app.sh"

ExecStop=/usr/bin/docker rm --force app

[Install]
WantedBy=multi-user.target
EOF
cat /tmp/app.service
mv -f /tmp/app.service /etc/systemd/system/app.service
##########################################################

##########################################################
cat > /tmp/worker.service <<EOF
[Unit]
Description=dplearn GPU development service
Documentation=https://github.com/gyuho/dplearn

After=app.service

[Service]
Restart=always
RestartSec=5s
TimeoutStartSec=0
LimitNOFILE=40000

ExecStartPre=/usr/bin/docker login -u oauth2accesstoken -p "$(/usr/bin/gcloud auth application-default print-access-token)" https://gcr.io
ExecStartPre=/usr/bin/docker pull gcr.io/gcp-dplearn/dplearn:latest-python3-gpu

ExecStart=/usr/bin/nvidia-docker run \
  --rm \
  --name worker \
  --volume=/var/lib/etcd:/var/lib/etcd \
  --volume=/var/lib/keras/datasets:/root/.keras/datasets \
  --volume=/var/lib/keras/models:/root/.keras/models \
  -p 4200:4200 \
  --ulimit nofile=262144:262144 \
  gcr.io/gcp-dplearn/dplearn:latest-python3-gpu \
  /bin/sh -c "./scripts/run/worker-python3.sh"

ExecStop=/usr/bin/docker rm --force worker

[Install]
WantedBy=multi-user.target
EOF
cat /tmp/worker.service
mv -f /tmp/worker.service /etc/systemd/system/worker.service
##########################################################

##########################################################
cat > /tmp/reverse-proxy.service <<EOF
[Unit]
Description=dplearn reverse proxy
Documentation=https://github.com/gyuho/dplearn

After=app.service

[Service]
Restart=always
RestartSec=5s
TimeoutStartSec=0
LimitNOFILE=40000

ExecStartPre=/usr/bin/docker login -u oauth2accesstoken -p "$(/usr/bin/gcloud auth application-default print-access-token)" https://gcr.io
ExecStartPre=/usr/bin/docker pull gcr.io/gcp-dplearn/dplearn:latest-reverse-proxy

ExecStart=/usr/bin/docker \
  run \
  --rm \
  --name reverse-proxy \
  --net=host \
  --ulimit nofile=262144:262144 \
  gcr.io/gcp-dplearn/dplearn:latest-reverse-proxy \
  /bin/sh -c "./scripts/run/reverse-proxy.sh"

ExecStop=/usr/bin/docker rm --force reverse-proxy

[Install]
WantedBy=multi-user.target
EOF
cat /tmp/reverse-proxy.service
mv -f /tmp/reverse-proxy.service /etc/systemd/system/reverse-proxy.service
##########################################################

##########################################################
cat > /tmp/python3-ipython-gpu.service <<EOF
[Unit]
Description=dplearn GPU development service
Documentation=https://github.com/gyuho/dplearn

[Service]
Restart=always
RestartSec=5s
TimeoutStartSec=0
LimitNOFILE=40000

ExecStartPre=/usr/bin/docker login -u oauth2accesstoken -p "$(/usr/bin/gcloud auth application-default print-access-token)" https://gcr.io
ExecStartPre=/usr/bin/docker pull gcr.io/gcp-dplearn/dplearn:latest-python3-gpu

ExecStart=/usr/bin/nvidia-docker run \
  --rm \
  --name python3-ipython-gpu \
  --publish 8888:8888 \
  --volume=`pwd`/notebooks:/notebooks \
  --volume=/var/lib/keras/datasets:/root/.keras/datasets \
  --volume=/var/lib/keras/models:/root/.keras/models \
  --ulimit nofile=262144:262144 \
  gcr.io/gcp-dplearn/dplearn:latest-python3-gpu \
  /bin/sh -c "PASSWORD='' ./run_jupyter.sh -y --allow-root --notebook-dir=./notebooks"

ExecStop=/usr/bin/docker rm --force python3-ipython-gpu

[Install]
WantedBy=multi-user.target
EOF
cat /tmp/python3-ipython-gpu.service
mv -f /tmp/python3-ipython-gpu.service /etc/systemd/system/python3-ipython-gpu.service
##########################################################

##########################################################
systemctl daemon-reload

<<COMMENT
systemctl enable python3-ipython-gpu.service
systemctl start python3-ipython-gpu.service
COMMENT

systemctl enable app.service
systemctl start app.service

systemctl enable worker.service
systemctl start worker.service

systemctl enable reverse-proxy.service
systemctl start reverse-proxy.service
##########################################################

<<COMMENT
if grep -q GOPATH "$(echo $HOME)/.bashrc"; then
  echo "bashrc already has GOPATH";
else
  echo "adding GOPATH to bashrc";
  echo "export GOPATH=$(echo $HOME)/go" >> $HOME/.bashrc;
  PATH_VAR=$PATH":/opt/bin:/usr/local/go/bin:$(echo $HOME)/go/bin";
  echo "export PATH=$(echo $PATH_VAR)" >> $HOME/.bashrc;
  source $HOME/.bashrc;
fi

mkdir -p $GOPATH/bin/
source $HOME/.bashrc
COMMENT
