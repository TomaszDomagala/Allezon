#!/usr/bin/env bash

# This script will read config from .vms file and will reset ssh
# for each host found in the file.

# read envs from .vms file
if [ ! -f .vms ]; then
    echo "error: .vms file not found"
    echo "create .vms file according to the example in .vms_example"
    exit 1
fi
export $(cat .vms | xargs)

# HOSTS holds vms ids in the format of "id1;id2;id3"
HOSTS_IDS=(${HOSTS//;/ })

for id in "${HOSTS_IDS[@]}"; do
    HOSTNAME="${USERNAME}vm${id}.rtb-lab.pl"
    ./ssh-reset.sh "$USERNAME" "$HOSTNAME" "$PASSWORD"
done
