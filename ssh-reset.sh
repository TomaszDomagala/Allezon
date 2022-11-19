#!/usr/bin/env bash

# This script will remove host from known_hosts file and then
# will copy the public key to the remote host. It is useful when
# the remote vm is recreated.

# usage: ssh-reset.sh user host password


if [ $# -ne 3 ]; then
    echo "usage: ssh-reset.sh user host password"
    exit 1
fi

# reset known_hosts file

host="$2"
ip=$(dig +short "$host")

if [ -z "$ip" ]; then
    echo "error: could not resolve $host"
    exit 1
fi

ssh-keygen -f "$HOME/.ssh/known_hosts" -R "$host"
ssh-keygen -f "$HOME/.ssh/known_hosts" -R "$ip"

# copy ssh key
sshpass -p "$3" ssh-copy-id -o StrictHostKeyChecking=no "$1@$2"
