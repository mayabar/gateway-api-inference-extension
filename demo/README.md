# KVCache-Aware Routing Demo

This guide walks you through deploying KVCache-Aware Routing using `kgateway`.

---

## 🚀 Deployment Steps

### 1. Deploy vLLM in Peer-to-Peer Mode

```bash
export HF_TOKEN=<HF_TOKEN>
export MODEL_NAME="mistralai/Mistral-7B-Instruct-v0.2"
export MODEL_LABEL="mistral7b"
```

#### Deploying

From the repo root:

```bash
helm upgrade --install vllm-p2p ./vllm-setup-helm \
  --set secret.create=true \
  --set secret.hfTokenValue="$HF_TOKEN" \
  --set vllm.model.name="$MODEL_NAME" \
  --set vllm.model.label="$MODEL_LABEL" \
  --set vllm.poolLabelValue="vllm-llama3-8b-instruct" \
  --set vllm.replicaCount=4
```

### 2. Deploy Inference API Components

Install GIE CRDs if not already installed:

```bash
kubectl apply -f "https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v0.3.0/manifests.yaml"
```

Deploy InferenceModel, InferencePool, and the endpoint picker (EPP) with the KVCache-Aware Routing scorer:

```bash
kubectl apply -f ./demo/config/manifests/inferencemodel.yaml
kubectl apply -f ./demo/config/manifests/inferencepool-resources.yaml
```

### 3. Install KGateway

Install KGateway controller and CRDs if not already installed:

```bash
KGTW_VERSION="v2.0.0"
KGATEWAY_IMAGE="cr.kgateway.dev/kgateway-dev/envoy-wrapper:v2.0.0"

kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml
helm upgrade -i --create-namespace --namespace kgateway-system --version v2.0.0 kgateway-crds oci://cr.kgateway.dev/kgateway-dev/charts/kgateway-crds
helm upgrade -i --namespace kgateway-system --version "$KGTW_VERSION" kgateway oci://cr.kgateway.dev/kgateway-dev/charts/kgateway \
  --set inferenceExtension.enabled=true
```

### 4. Apply Gateway and Routes

Deploy the KGateway instance in the namespace. This creates the inference router pod (Envoy + HTTPRoute to InferencePool):

```bash
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/raw/main/config/manifests/gateway/kgateway/gateway.yaml
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/raw/main/config/manifests/gateway/kgateway/httproute.yaml
```

### 5. Test Inference Routing

Use `curl` to test inference routing:

```bash
IP=$(kubectl get gateway/inference-gateway -o jsonpath='{.status.addresses[0].value}')
PORT=80

echo "📨  Sending test inference request to $IP:$PORT..."
curl -i "${IP}:${PORT}/v1/completions" \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "mistralai/Mistral-7B-Instruct-v0.2",
    "prompt": "Write as if you were a critic: San Francisco",
    "max_tokens": 100,
    "temperature": 0
  }'
```

### 6. Deploy Chatbots

To deploy 3 chatbots that connect to the inference router:

```bash
make chatbot
```

### 7. Deploy Grafana Metrics

To deploy the Grafana dashboard:

```bash
kubectl apply -f ./demo/monitoring/
```

Do port forwarding to access Prometheus and Grafana:

```bash
kubectl port-forward svc/prometheus  9090:9090 &
kubectl port-forward svc/grafana     3000:3000 &
```

Set up the dashboard at http://localhost:3000 using `./demo/monitoring/monitoring/grafana.json`

---

