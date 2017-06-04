
GCP_KEY_PATH=/etc/gcp-key-deephardway.json ./scripts/gcp/create-instance.sh

gcloud compute ssh --zone=us-west1-b deep

curl -L http://metadata.google.internal/computeMetadata/v1/instance/attributes/gcp-key -H 'Metadata-Flavor:Google'

tail -f /etc/ansible-install.log

cat /etc/gcp-key-deephardway.json


sudo systemctl cat nvidia-docker.service
sudo systemctl cat ipython-gpu.service
sudo systemctl cat deep-gpu.service


sudo journalctl -u nvidia-docker.service -l --no-pager|less
sudo journalctl -u ipython-gpu.service -l --no-pager|less
sudo journalctl -u deep-gpu.service -l --no-pager|less


sudo systemctl stop nvidia-docker.service
sudo systemctl disable nvidia-docker.service


sudo systemctl stop ipython-gpu.service
sudo systemctl disable ipython-gpu.service


sudo systemctl stop deep-gpu.service
sudo systemctl disable deep-gpu.service








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
sudo systemctl enable deep-gpu.service
sudo systemctl start deep-gpu.service
sudo systemctl cat deep-gpu.service
sudo systemd-analyze verify /etc/systemd/system/deep-gpu.service

sudo systemctl status deep-gpu.service -l --no-pager
sudo journalctl -u deep-gpu.service -l --no-pager|less
sudo journalctl -f -u deep-gpu.service

sudo systemctl stop deep-gpu.service
sudo systemctl disable deep-gpu.service



sudo /usr/bin/docker login -u _json_key -p "$(cat /etc/gcp-key-deephardway.json)" https://gcr.io
sudo /usr/bin/docker login -u oauth2accesstoken -p "$(/usr/bin/gcloud auth application-default print-access-token)" https://gcr.io


gcloud auth login

gcloud version
gcloud components update
gcloud components install beta

gcloud config set project deephardway
gcloud beta compute regions describe us-west1-b

gcloud compute instances list

gcloud compute instances list
gcloud compute ssh --zone=us-west1-b deep

gcloud compute instances reset --zone us-west1-b deep
gcloud compute instances start --zone us-west1-b deep
gcloud compute instances stop --zone us-west1-b deep
gcloud compute instances delete --zone us-west1-b deep
gcloud compute ssh --zone=us-west1-b deep

gcloud compute project-info add-metadata --metadata-from-file gcp-key=${GCP_KEY_PATH}
curl -L http://metadata.google.internal/computeMetadata/v1/project/attributes/gcp-key -H 'Metadata-Flavor:Google'
