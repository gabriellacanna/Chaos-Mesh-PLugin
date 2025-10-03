#!/bin/bash

set -e

echo "🧪 Testing Chaos Mesh Plugin..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in a Kubernetes cluster
if ! kubectl cluster-info &> /dev/null; then
    print_error "No Kubernetes cluster found. Please run setup-test-environment.sh first."
    exit 1
fi

# Check if Chaos Mesh is installed
if ! kubectl get crd podchaos.chaos-mesh.org &> /dev/null; then
    print_error "Chaos Mesh not found. Please run setup-test-environment.sh first."
    exit 1
fi

# Check if Argo Rollouts is installed
if ! kubectl get crd rollouts.argoproj.io &> /dev/null; then
    print_error "Argo Rollouts not found. Please run setup-test-environment.sh first."
    exit 1
fi

# Apply the AnalysisTemplate
print_status "Applying AnalysisTemplate..."
kubectl apply -f examples/analysis-template.yaml
print_success "AnalysisTemplate applied"

# Apply the Rollout
print_status "Applying Rollout..."
kubectl apply -f examples/rollout-with-chaos.yaml
print_success "Rollout applied"

# Wait for initial rollout to be ready
print_status "Waiting for initial rollout to be ready..."
kubectl wait --for=condition=Available rollout/demo-app --timeout=300s

# Show current status
print_status "Current rollout status:"
kubectl argo rollouts get rollout demo-app

# Trigger a new rollout
print_status "Triggering new rollout with nginx:1.21..."
kubectl argo rollouts set image demo-app demo-app=nginx:1.21

# Watch the rollout progress
print_status "Watching rollout progress..."
echo "Press Ctrl+C to stop watching"
kubectl argo rollouts get rollout demo-app --watch

print_success "Test completed!"