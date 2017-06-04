#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/dep/tensorflow.sh" ]]; then
    echo "must be run from repository root"
    exit 255
fi

rm -rf ./tensorflow

echo "Downloading 'tensorflow'"
git clone https://github.com/tensorflow/tensorflow.git --branch master
curl -o ./git-tensorflow.json https://api.github.com/repos/tensorflow/tensorflow/git/refs/heads/master

rm -rf ./notebooks/tensorflow-sample-notebooks
cp -rf ./tensorflow/tensorflow/tools/docker/notebooks ./notebooks/tensorflow-sample-notebooks
cp ./tensorflow/tensorflow/tools/docker/jupyter_notebook_config.py .
cp ./tensorflow/tensorflow/tools/docker/run_jupyter.sh .

rm -rf ./tensorflow

# echo "
# c.ContentsManager.root_dir = '/notebooks'" >> ./jupyter_notebook_config.py
