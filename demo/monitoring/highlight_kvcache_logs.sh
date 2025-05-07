#!/bin/bash

# Deployment label selector
LABEL_SELECTOR="app=vllm-llama3-8b-instruct-epp"
NAMESPACE="default"

# Get the first matching pod name
POD_NAME=$(kubectl get pods -n "$NAMESPACE" -l "$LABEL_SELECTOR" \
  -o jsonpath='{.items[0].metadata.name}')

# Exit if no pod found
if [ -z "$POD_NAME" ]; then
  echo "❌ No pod found with label: $LABEL_SELECTOR in namespace $NAMESPACE"
  exit 1
fi

echo "📦 Fetching logs from pod: $POD_NAME"

# Fetch logs and highlight key terms
kubectl logs -n "$NAMESPACE" "$POD_NAME" |
  grep -E 'KVCacheAwareScorer.*podScores|Request handled' |
  sed -E \
    -e 's/(KVCacheAwareScorer)/\x1b[1;32m\1\x1b[0m/g' \
    -e 's/(podScores)/\x1b[1;34m\1\x1b[0m/g' \
    -e 's/(Request handled)/\x1b[1;33m\1\x1b[0m/g' \
    -e 's/(Address)/\x1b[1;35m\1\x1b[0m/g'
