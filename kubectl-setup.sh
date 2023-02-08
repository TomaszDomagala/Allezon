#!/bin/bash

# This script will set kubectl credentials for a given host.

if [ $# -ne 2 ]; then
  echo "Usage: $0 <user> <host>"
  exit 1
fi
USER=$1
HOST=$2

SSH_HOST="$USER@$HOST"

TEMP_DIR=$(mktemp -d)

ssh "$SSH_HOST" sudo cat /etc/kubernetes/ssl/apiserver-kubelet-client.key >"$TEMP_DIR"/client.key
ssh "$SSH_HOST" sudo cat /etc/kubernetes/ssl/apiserver-kubelet-client.crt >"$TEMP_DIR"/client.crt
ssh "$SSH_HOST" sudo cat /etc/kubernetes/ssl/ca.crt >"$TEMP_DIR"/ca.crt

# kubectl config does not support hostnames, only IPs.
IP=$(dig +short "$HOST")
if [ -z "$IP" ]; then
  IP=$HOST
fi


# Set cluster
kubectl config set-cluster default-cluster \
  --server="https://$IP:6443" \
  --certificate-authority="$TEMP_DIR"/ca.crt \
  --embed-certs=true

# Set credentials
kubectl config set-credentials default-admin \
  --certificate-authority="$TEMP_DIR"/ca.crt \
  --client-key="$TEMP_DIR"/client.key \
  --client-certificate="$TEMP_DIR"/client.crt \
  --embed-certs=true

# Create context
kubectl config set-context default-context \
 --cluster=default-cluster \
 --user=default-admin

# Set active context
kubectl config use-context default-context

rm -rf "$TEMP_DIR"
