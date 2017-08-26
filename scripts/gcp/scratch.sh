#!/usr/bin/env bash
set -e

<<COMMENT
edit "Networking"->"VPC networks"->"default"->"Firewall rules"->"Add firewall rules"
Target tags: dplearn
Source IP ranges: 0.0.0.0/0
Protocols and ports: tcp:80; tcp:4200; tcp:8888;
COMMENT

GCP_KEY_PATH=/etc/gcp-key-dplearn.json ./scripts/gcp/ubuntu-python3-gpu.gcp.sh

gcloud compute ssh --zone=us-west1-b dplearn

curl -L http://metadata.google.internal/computeMetadata/v1/instance/attributes/gcp-key-dplearn -H 'Metadata-Flavor:Google'

tail -f /etc/ansible-install.log

cat /etc/gcp-key-dplearn.json


sudo systemctl cat nvidia-docker.service
sudo systemctl cat ipython-gpu.service
sudo systemctl cat dplearn-gpu.service


sudo journalctl -u nvidia-docker.service -l --no-pager|less
sudo journalctl -u ipython-gpu.service -l --no-pager|less
sudo journalctl -u dplearn-gpu.service -l --no-pager|less


sudo systemctl stop nvidia-docker.service
sudo systemctl disable nvidia-docker.service


sudo systemctl stop ipython-gpu.service
sudo systemctl disable ipython-gpu.service


sudo systemctl stop dplearn-gpu.service
sudo systemctl disable dplearn-gpu.service


systemctl enable reverse-proxy.service
systemctl start reverse-proxy.service

sudo journalctl -u reverse-proxy.service -l --no-pager|less



sudo systemctl daemon-reload
sudo systemctl enable ipython-gpu.service
sudo systemctl start ipython-gpu.service
sudo systemctl cat ipython-gpu.service
sudo systemd-analyze verify /etc/systemd/system/ipython-gpu.service

sudo systemctl status ipython-gpu.service -l --no-pager
sudo journalctl -u ipython-gpu.service -l --no-pager|less
sudo journalctl -f -u ipython-gpu.service

sudo systemctl stop ipython-gpu.service
sudo systemctl disable ipython-gpu.service



sudo systemctl daemon-reload
sudo systemctl enable dplearn-gpu.service
sudo systemctl start dplearn-gpu.service
sudo systemctl cat dplearn-gpu.service
sudo systemd-analyze verify /etc/systemd/system/dplearn-gpu.service

sudo systemctl status dplearn-gpu.service -l --no-pager
sudo journalctl -u dplearn-gpu.service -l --no-pager|less
sudo journalctl -f -u dplearn-gpu.service

sudo systemctl stop dplearn-gpu.service
sudo systemctl disable dplearn-gpu.service



sudo /usr/bin/docker login -u _json_key -p "$(cat /etc/gcp-key-dplearn.json)" https://gcr.io
sudo /usr/bin/docker login -u oauth2accesstoken -p "$(/usr/bin/gcloud auth application-default print-access-token)" https://gcr.io


gcloud auth login

gcloud version
gcloud components update
gcloud components install beta

gcloud config set project dplearn
gcloud beta compute regions describe us-west1-b

gcloud compute instances list

gcloud compute instances list
gcloud compute ssh --zone=us-west1-b deep

gcloud compute instances reset --zone us-west1-b deep
gcloud compute instances start --zone us-west1-b deep
gcloud compute instances stop --zone us-west1-b deep
gcloud compute instances delete --zone us-west1-b deep
gcloud compute ssh --zone=us-west1-b deep

gcloud compute project-info add-metadata --metadata-from-file gcp-key-dplearn=${GCP_KEY_PATH}
curl -L http://metadata.google.internal/computeMetadata/v1/project/attributes/gcp-key-dplearn -H 'Metadata-Flavor:Google'
