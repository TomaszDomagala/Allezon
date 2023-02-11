#!/bin/bash

PODS=$(kubectl get pods | grep api | awk '{print $1}')

mkdir -p logs

for POD in $PODS; do
  echo "Logs for $POD"
  kubectl logs $POD > logs/$POD.log
done
