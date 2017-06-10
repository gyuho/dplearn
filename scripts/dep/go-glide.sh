#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ "./scripts/dep/go-glide.sh" ]]; then
  echo "must be run from repository root"
  exit 255
fi

echo "Updating Go dependencies"
DEP_ROOT="$GOPATH/src/github.com/Masterminds/glide"
go get -d -u github.com/Masterminds/glide
pushd "${DEP_ROOT}"
  git reset --hard HEAD
  go install -v
popd

if [ ! $(command -v glide) ]; then
  echo "glide: command not found"
  exit 1
fi

DEP_ROOT="$GOPATH/src/github.com/sgotti/glide-vc"
go get -d -u github.com/sgotti/glide-vc
pushd "${DEP_ROOT}"
  git reset --hard HEAD
  go install -v
popd
if [ ! $(command -v glide-vc) ]; then
  echo "glide-vc: command not found"
  exit 1
fi

glide update --strip-vendor
glide-vc --only-code --no-tests
