#!/usr/bin/env bash

if [ -z $1 ]; then
  echo "Usage $0 <tag>"
  exit 1
fi

cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" || exit 1

TAG=$1

DOCKER_BUILDKIT=1 docker build -f Dockerfile -t "allezon/api:$TAG" ../../../src || exit 1

# Only on local and if using kind
kind load docker-image "allezon/api:$TAG"