#!/usr/bin/env bash
set -e

echo "root ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

#######################
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
EOF

ansible-playbook /etc/ansible-install.yml > /etc/ansible-install.log 2>&1
#######################

#######################
systemctl daemon-reload
#######################

#######################
cat > /tmp/app.service <<EOF
[Unit]
Description=dplearn CPU development service
Documentation=https://github.com/gyuho/dplearn

[Service]
Restart=always
RestartSec=5s
TimeoutStartSec=0
LimitNOFILE=40000

ExecStartPre=/usr/bin/docker login -u oauth2accesstoken -p "$(/usr/bin/gcloud auth application-default print-access-token)" https://gcr.io
ExecStartPre=/usr/bin/docker pull gcr.io/gcp-dplearn/dplearn:latest-app

ExecStart=/usr/bin/docker run \
  --rm \
  --name app \
  --volume=/tmp:/tmp \
  --volume=/var/lib/etcd:/var/lib/etcd \
  -p 4200:4200 \
  --ulimit nofile=262144:262144 \
  gcr.io/gcp-dplearn/dplearn:latest-app \
  /bin/sh -c "./scripts/docker/run/app.sh"

ExecStop=/usr/bin/docker rm --force app

[Install]
WantedBy=multi-user.target
EOF
cat /tmp/app.service
mv -f /tmp/app.service /etc/systemd/system/app.service
#######################

#######################
cat > /tmp/worker.service <<EOF
[Unit]
Description=dplearn CPU development service
Documentation=https://github.com/gyuho/dplearn

After=app.service

[Service]
Restart=always
RestartSec=5s
TimeoutStartSec=0
LimitNOFILE=40000

ExecStartPre=/usr/bin/docker login -u oauth2accesstoken -p "$(/usr/bin/gcloud auth application-default print-access-token)" https://gcr.io
ExecStartPre=/usr/bin/docker pull gcr.io/gcp-dplearn/dplearn:latest-python3-cpu

ExecStart=/usr/bin/docker run \
  --rm \
  --name worker \
  --env CATS_PARAM_PATH=/root/datasets/parameters-cats.npy \
  --volume=/tmp:/tmp \
  --volume=/var/lib/etcd:/var/lib/etcd \
  --volume=/var/lib/keras/datasets:/root/.keras/datasets \
  --volume=/var/lib/keras/models:/root/.keras/models \
  -p 4200:4200 \
  --ulimit nofile=262144:262144 \
  gcr.io/gcp-dplearn/dplearn:latest-python3-cpu \
  /bin/sh -c "./scripts/docker/run/worker-python3.sh"

ExecStop=/usr/bin/docker rm --force worker

[Install]
WantedBy=multi-user.target
EOF
cat /tmp/worker.service
mv -f /tmp/worker.service /etc/systemd/system/worker.service
#######################

#######################
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
  /bin/sh -c "./scripts/docker/run/reverse-proxy.sh"

ExecStop=/usr/bin/docker rm --force reverse-proxy

[Install]
WantedBy=multi-user.target
EOF
cat /tmp/reverse-proxy.service
mv -f /tmp/reverse-proxy.service /etc/systemd/system/reverse-proxy.service
#######################

#######################
systemctl daemon-reload

systemctl enable app.service
systemctl start app.service

systemctl enable worker.service
systemctl start worker.service

systemctl enable reverse-proxy.service
systemctl start reverse-proxy.service
#######################

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
