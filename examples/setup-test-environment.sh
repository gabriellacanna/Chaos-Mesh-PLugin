#!/bin/bash

set -e

echo "🚀 Setting up test environment for Chaos Mesh Plugin..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker first."
    exit 1
fi

# Check if KIND is installed
if ! command -v kind &> /dev/null; then
    print_status "Installing KIND..."
    curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.22.0/kind-linux-amd64
    chmod +x ./kind
    sudo mv ./kind /usr/local/bin/
    print_success "KIND installed successfully"
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    print_status "Installing kubectl..."
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x kubectl
    sudo mv ./kubectl /usr/local/bin/
    print_success "kubectl installed successfully"
fi

# Check if Helm is installed
if ! command -v helm &> /dev/null; then
    print_status "Installing Helm..."
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    print_success "Helm installed successfully"
fi

# Create KIND cluster
CLUSTER_NAME="chaos-mesh-test"
print_status "Creating KIND cluster: $CLUSTER_NAME"

if kind get clusters | grep -q "$CLUSTER_NAME"; then
    print_warning "Cluster $CLUSTER_NAME already exists. Deleting it first..."
    kind delete cluster --name "$CLUSTER_NAME"
fi

# Create cluster with specific configuration
cat <<EOF | kind create cluster --name "$CLUSTER_NAME" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: kindest/node:v1.28.0
  extraPortMappings:
  - containerPort: 30080
    hostPort: 8080
    protocol: TCP
- role: worker
  image: kindest/node:v1.28.0
- role: worker
  image: kindest/node:v1.28.0
EOF

print_success "KIND cluster created successfully"

# Wait for cluster to be ready
print_status "Waiting for cluster to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=300s

# Install Chaos Mesh
print_status "Installing Chaos Mesh..."
helm repo add chaos-mesh https://charts.chaos-mesh.org
helm repo update

kubectl create namespace chaos-mesh || true

helm install chaos-mesh chaos-mesh/chaos-mesh \
    --namespace=chaos-mesh \
    --set chaosDaemon.runtime=containerd \
    --set chaosDaemon.socketPath=/run/containerd/containerd.sock \
    --set dashboard.create=true \
    --version 2.6.2

print_status "Waiting for Chaos Mesh to be ready..."
kubectl wait --for=condition=Ready pods --all -n chaos-mesh --timeout=300s

# Install Argo Rollouts
print_status "Installing Argo Rollouts..."
kubectl create namespace argo-rollouts || true
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml

print_status "Waiting for Argo Rollouts to be ready..."
kubectl wait --for=condition=Ready pods --all -n argo-rollouts --timeout=300s

# Create RBAC for Argo Rollouts to access Chaos Mesh
print_status "Creating RBAC permissions..."
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: argo-rollouts-chaos-mesh
rules:
- apiGroups: ["chaos-mesh.org"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: [""]
  resources: ["pods", "services", "endpoints"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["replicasets", "deployments"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: argo-rollouts-chaos-mesh
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: argo-rollouts-chaos-mesh
subjects:
- kind: ServiceAccount
  name: argo-rollouts
  namespace: argo-rollouts
EOF

print_success "RBAC permissions created"

# Build the plugin
print_status "Building Chaos Mesh plugin..."
cd "$(dirname "$0")/.."
make build

print_success "Plugin built successfully"

# Copy plugin to a location accessible by Argo Rollouts
print_status "Copying plugin to cluster..."
PLUGIN_PATH="/tmp/chaos-mesh-plugin"
cp dist/chaos-mesh-plugin-linux-amd64 "$PLUGIN_PATH"

# Create ConfigMap with plugin configuration
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: argo-rollouts-config
  namespace: argo-rollouts
data:
  metricProviderPlugins: |-
    - name: "argo-rollouts-chaos-mesh-plugin"
      location: "file://$PLUGIN_PATH"
EOF

print_success "Plugin configuration applied"

# Restart Argo Rollouts to pick up the plugin
print_status "Restarting Argo Rollouts to load plugin..."
kubectl rollout restart deployment/argo-rollouts -n argo-rollouts
kubectl wait --for=condition=Ready pods --all -n argo-rollouts --timeout=300s

print_success "Environment setup completed!"

echo ""
echo "🎉 Test environment is ready!"
echo ""
echo "Cluster info:"
echo "  - Cluster name: $CLUSTER_NAME"
echo "  - Chaos Mesh dashboard: kubectl port-forward -n chaos-mesh svc/chaos-dashboard 2333:2333"
echo "  - Access dashboard at: http://localhost:2333"
echo ""
echo "Next steps:"
echo "  1. Apply the AnalysisTemplate: kubectl apply -f examples/analysis-template.yaml"
echo "  2. Apply the Rollout: kubectl apply -f examples/rollout-with-chaos.yaml"
echo "  3. Trigger a rollout: kubectl argo rollouts set image demo-app demo-app=nginx:1.21"
echo "  4. Watch the rollout: kubectl argo rollouts get rollout demo-app --watch"
echo ""
echo "To cleanup:"
echo "  kind delete cluster --name $CLUSTER_NAME"
EOF