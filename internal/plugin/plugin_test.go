package plugin

import (
	"encoding/json"
	"testing"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	log "github.com/sirupsen/logrus"
)

func TestParseConfig(t *testing.T) {
	plugin := &RpcPlugin{
		LogCtx: *log.WithFields(log.Fields{"test": "plugin"}),
	}

	// Test valid configuration
	config := Config{
		ChaosExperimentCRD:    "apiVersion: chaos-mesh.org/v1alpha1\nkind: PodChaos",
		TargetReplicaSetLabel: "rollouts-pod-template-hash",
		TargetReplicaSetValue: "abc123",
		Timeout:               "5m",
		CleanupOnFinish:       true,
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	metric := v1alpha1.Metric{
		Provider: v1alpha1.MetricProvider{
			Plugin: map[string]json.RawMessage{
				PluginName: configBytes,
			},
		},
	}

	parsedConfig, err := plugin.parseConfig(metric)
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if parsedConfig.TargetReplicaSetLabel != "rollouts-pod-template-hash" {
		t.Errorf("Expected TargetReplicaSetLabel to be 'rollouts-pod-template-hash', got '%s'", parsedConfig.TargetReplicaSetLabel)
	}

	if parsedConfig.TargetReplicaSetValue != "abc123" {
		t.Errorf("Expected TargetReplicaSetValue to be 'abc123', got '%s'", parsedConfig.TargetReplicaSetValue)
	}

	if parsedConfig.Timeout != "5m" {
		t.Errorf("Expected Timeout to be '5m', got '%s'", parsedConfig.Timeout)
	}

	if !parsedConfig.CleanupOnFinish {
		t.Errorf("Expected CleanupOnFinish to be true, got false")
	}
}

func TestValidateConfig(t *testing.T) {
	plugin := &RpcPlugin{
		LogCtx: *log.WithFields(log.Fields{"test": "plugin"}),
	}

	// Test valid configuration
	validConfig := &Config{
		ChaosExperimentCRD:    "apiVersion: chaos-mesh.org/v1alpha1\nkind: PodChaos",
		TargetReplicaSetLabel: "rollouts-pod-template-hash",
		TargetReplicaSetValue: "abc123",
		Timeout:               "5m",
		CleanupOnFinish:       true,
	}

	err := plugin.validateConfig(validConfig)
	if err != nil {
		t.Errorf("Expected valid config to pass validation, got error: %v", err)
	}

	// Test missing ChaosExperimentCRD
	invalidConfig1 := &Config{
		TargetReplicaSetLabel: "rollouts-pod-template-hash",
		TargetReplicaSetValue: "abc123",
	}

	err = plugin.validateConfig(invalidConfig1)
	if err == nil {
		t.Errorf("Expected validation to fail for missing ChaosExperimentCRD")
	}

	// Test missing TargetReplicaSetLabel
	invalidConfig2 := &Config{
		ChaosExperimentCRD:    "apiVersion: chaos-mesh.org/v1alpha1\nkind: PodChaos",
		TargetReplicaSetValue: "abc123",
	}

	err = plugin.validateConfig(invalidConfig2)
	if err == nil {
		t.Errorf("Expected validation to fail for missing TargetReplicaSetLabel")
	}

	// Test missing TargetReplicaSetValue
	invalidConfig3 := &Config{
		ChaosExperimentCRD:    "apiVersion: chaos-mesh.org/v1alpha1\nkind: PodChaos",
		TargetReplicaSetLabel: "rollouts-pod-template-hash",
	}

	err = plugin.validateConfig(invalidConfig3)
	if err == nil {
		t.Errorf("Expected validation to fail for missing TargetReplicaSetValue")
	}

	// Test invalid timeout format
	invalidConfig4 := &Config{
		ChaosExperimentCRD:    "apiVersion: chaos-mesh.org/v1alpha1\nkind: PodChaos",
		TargetReplicaSetLabel: "rollouts-pod-template-hash",
		TargetReplicaSetValue: "abc123",
		Timeout:               "invalid-timeout",
	}

	err = plugin.validateConfig(invalidConfig4)
	if err == nil {
		t.Errorf("Expected validation to fail for invalid timeout format")
	}
}

func TestGetMetadata(t *testing.T) {
	plugin := &RpcPlugin{
		LogCtx: *log.WithFields(log.Fields{"test": "plugin"}),
	}

	config := Config{
		ChaosExperimentCRD: `apiVersion: chaos-mesh.org/v1alpha1
kind: PodChaos
metadata:
  name: test-chaos
spec:
  action: pod-kill`,
		TargetReplicaSetLabel: "rollouts-pod-template-hash",
		TargetReplicaSetValue: "abc123",
		Timeout:               "5m",
		CleanupOnFinish:       true,
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	metric := v1alpha1.Metric{
		Provider: v1alpha1.MetricProvider{
			Plugin: map[string]json.RawMessage{
				PluginName: configBytes,
			},
		},
	}

	metadata := plugin.GetMetadata(metric)

	if metadata["pluginName"] != PluginName {
		t.Errorf("Expected pluginName to be '%s', got '%s'", PluginName, metadata["pluginName"])
	}

	if metadata["targetReplicaSetLabel"] != "rollouts-pod-template-hash" {
		t.Errorf("Expected targetReplicaSetLabel to be 'rollouts-pod-template-hash', got '%s'", metadata["targetReplicaSetLabel"])
	}

	if metadata["targetReplicaSetValue"] != "abc123" {
		t.Errorf("Expected targetReplicaSetValue to be 'abc123', got '%s'", metadata["targetReplicaSetValue"])
	}

	if metadata["experimentKind"] != "PodChaos" {
		t.Errorf("Expected experimentKind to be 'PodChaos', got '%s'", metadata["experimentKind"])
	}
}