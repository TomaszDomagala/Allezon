#!/usr/bin/env bash

cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" || exit 1

protoc -I=./ --go_out=./ ./aggregates.proto