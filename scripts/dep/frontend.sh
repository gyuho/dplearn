#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/dep/frontend.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

source ${NVM_DIR}/nvm.sh
nvm install v7.10.0

go install -v ./cmd/gen-package-json
gen-package-json --output package.json --logtostderr

# npm install -g tslint
yarn install
npm rebuild node-sass
yarn install
npm install
