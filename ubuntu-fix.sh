#!/usr/bin/env bash

# This script will run ansible playbook to fix cluster.

# read envs from .vms file
if [ ! -f .vms ]; then
    echo "error: .vms file not found"
    echo "create .vms file according to the example in .vms_example"
    exit 1
fi
export $(cat .vms | xargs)

docker run --rm -it \
    --mount type=bind,source="$(pwd)"/cluster,dst=/inventory \
    --mount type=bind,source="${HOME}"/.ssh/id_rsa,dst=/root/.ssh/id_rsa \
    --mount type=bind,source="$(pwd)"/ubuntu-fix.yaml,dst=/ubuntu-fix.yaml \
    kubesplay ansible-playbook -u "$USERNAME" -i /inventory/hosts.yaml --private-key /root/.ssh/id_rsa --become /ubuntu-fix.yaml
