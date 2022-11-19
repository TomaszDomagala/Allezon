#!/usr/bin/env bash

# Prints ips of all hosts from .vms file.
# Usage: hosts-ips.sh

# read envs from .vms file
if [ ! -f .vms ]; then
    echo "error: .vms file not found"
    echo "create .vms file according to the example in .vms_example"
    exit 1
fi
export $(cat .vms | xargs)

# HOSTS holds vms ids in the format of "id1;id2;id3"
HOSTS_IDS=(${HOSTS//;/ })
IPS=()

for id in "${HOSTS_IDS[@]}"; do
    HOSTNAME="${USERNAME}vm${id}.rtb-lab.pl"
    ip=$(dig +short "$HOSTNAME")
    if [ -z "$ip" ]; then
        echo "error: could not resolve $HOSTNAME"
    fi
    IPS+=("$ip")
done

echo "${IPS[@]}"
