#!/bin/bash

DEPLOYMENT_PREFIX="vllm-p2p-vllm-mistral7b"

# Get the list of pods in the deployment
pods=$(kubectl get pods -o name | grep "$DEPLOYMENT_PREFIX")

for pod in $pods; do
    pod_name=$(basename $pod)
    echo "🔍 Checking GPU on pod: $pod_name"

    kubectl exec "$pod_name" -- nvidia-smi --query-gpu=name --format=csv,noheader,nounits 2>/dev/null

    echo "-------------------------------"
done
