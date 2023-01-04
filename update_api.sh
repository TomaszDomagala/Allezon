#!/usr/bin/env bash

if [ -z $1 ]; then
  echo "Usage $0 <tag>"
  exit 1
fi

cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" || exit 1

TAG=$1

./cluster/services/api/new_version.sh "$TAG"

kubectl set image deployment/api api=allezon/api:"$TAG"