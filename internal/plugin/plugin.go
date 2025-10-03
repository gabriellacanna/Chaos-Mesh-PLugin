package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gabriellacanna/chaos-mesh-plugin/internal/chaos"
	"github.com/argoproj/argo-rollouts/metricproviders/plugin"
	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/argoproj/argo-rollouts/utils/plugin/types"
	metricutil "github.com/argoproj/argo-rollouts/utils/metric"
	timeutil "github.com/argoproj/argo-rollouts/utils/time"
	log "github.com/sirupsen/logrus"
)

const (
	PluginName = "argo-rollouts-chaos-mesh-plugin"
	DefaultTimeout = 5 * time.Minute
)

// RpcPlugin implements the Argo Rollouts metric provider plugin interface
type RpcPlugin struct {
	LogCtx log.Entry
}

// Config represents the plugin configuration
type Config struct {
	// ChaosMeshEndpoint is the URL of the Chaos Mesh API (optional, uses in-cluster config by default)
	ChaosMeshEndpoint string `json:"chaosMeshEndpoint,omitempty"`
	
	// ChaosExperimentCRD is the YAML definition of the Chaos Mesh experiment
	ChaosExperimentCRD string `json:"chaosExperimentCRD"`
	
	// TargetReplicaSetLabel is the label key used to identify the target ReplicaSet
	TargetReplicaSetLabel string `json:"targetReplicaSetLabel"`
	
	// TargetReplicaSetValue is the label value for the target ReplicaSet
	TargetReplicaSetValue string `json:"targetReplicaSetValue"`
	
	// Timeout for the chaos experiment (default: 5 minutes)
	Timeout string `json:"timeout,omitempty"`
	
	// CleanupOnFinish determines if the experiment should be deleted after completion
	CleanupOnFinish bool `json:"cleanupOnFinish,omitempty"`
}

// InitPlugin initializes the plugin
func (r *RpcPlugin) InitPlugin() types.RpcError {
	r.LogCtx.Info("Initializing Chaos Mesh plugin")
	return types.RpcError{}
}

// Run executes the chaos experiment and returns the measurement
func (r *RpcPlugin) Run(analysisRun *v1alpha1.AnalysisRun, metric v1alpha1.Metric) v1alpha1.Measurement {
	startTime := timeutil.MetaNow()
	newMeasurement := v1alpha1.Measurement{
		StartedAt: &startTime,
	}

	// Parse configuration
	config, err := r.parseConfig(metric)
	if err != nil {
		r.LogCtx.Errorf("Failed to parse config: %v", err)
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	// Validate configuration
	if err := r.validateConfig(config); err != nil {
		r.LogCtx.Errorf("Invalid configuration: %v", err)
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	// Create Chaos Mesh client
	chaosClient, err := chaos.NewClient(r.LogCtx)
	if err != nil {
		r.LogCtx.Errorf("Failed to create Chaos Mesh client: %v", err)
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	// Build target selector
	targetSelector := map[string]string{
		config.TargetReplicaSetLabel: config.TargetReplicaSetValue,
	}

	r.LogCtx.Infof("Creating chaos experiment with target selector: %v", targetSelector)

	// Create the chaos experiment
	ctx := context.Background()
	experiment, err := chaosClient.CreateExperiment(ctx, config.ChaosExperimentCRD, targetSelector)
	if err != nil {
		r.LogCtx.Errorf("Failed to create chaos experiment: %v", err)
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	experimentName := experiment.GetName()
	experimentNamespace := experiment.GetNamespace()
	experimentKind := experiment.GetKind()

	r.LogCtx.Infof("Created chaos experiment: %s/%s (kind: %s)", experimentNamespace, experimentName, experimentKind)

	// Parse timeout
	timeout := DefaultTimeout
	if config.Timeout != "" {
		if parsedTimeout, err := time.ParseDuration(config.Timeout); err == nil {
			timeout = parsedTimeout
		} else {
			r.LogCtx.Warnf("Invalid timeout format '%s', using default %v", config.Timeout, DefaultTimeout)
		}
	}

	// Watch the experiment until completion
	success, err := chaosClient.WatchExperiment(ctx, experimentNamespace, experimentName, experimentKind, timeout)
	if err != nil {
		r.LogCtx.Errorf("Failed to watch chaos experiment: %v", err)
		// Try to cleanup the experiment
		if config.CleanupOnFinish {
			if cleanupErr := chaosClient.DeleteExperiment(ctx, experimentNamespace, experimentName, experimentKind); cleanupErr != nil {
				r.LogCtx.Warnf("Failed to cleanup experiment after error: %v", cleanupErr)
			}
		}
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	// Cleanup experiment if requested
	if config.CleanupOnFinish {
		if err := chaosClient.DeleteExperiment(ctx, experimentNamespace, experimentName, experimentKind); err != nil {
			r.LogCtx.Warnf("Failed to cleanup experiment: %v", err)
		} else {
			r.LogCtx.Infof("Cleaned up chaos experiment: %s/%s", experimentNamespace, experimentName)
		}
	}

	// Set measurement result
	finishedTime := timeutil.MetaNow()
	newMeasurement.FinishedAt = &finishedTime
	newMeasurement.Metadata = map[string]string{
		"experimentName":      experimentName,
		"experimentNamespace": experimentNamespace,
		"experimentKind":      experimentKind,
		"targetSelector":      fmt.Sprintf("%s=%s", config.TargetReplicaSetLabel, config.TargetReplicaSetValue),
	}

	if success {
		r.LogCtx.Infof("Chaos experiment completed successfully")
		newMeasurement.Phase = v1alpha1.AnalysisPhaseSuccessful
		newMeasurement.Value = "1"
	} else {
		r.LogCtx.Errorf("Chaos experiment failed")
		newMeasurement.Phase = v1alpha1.AnalysisPhaseFailed
		newMeasurement.Value = "0"
	}

	return newMeasurement
}

// Resume resumes a paused measurement (not implemented for chaos experiments)
func (r *RpcPlugin) Resume(analysisRun *v1alpha1.AnalysisRun, metric v1alpha1.Metric, measurement v1alpha1.Measurement) v1alpha1.Measurement {
	r.LogCtx.Debug("Resume called - not implemented for chaos experiments")
	return measurement
}

// Terminate terminates a running measurement
func (r *RpcPlugin) Terminate(analysisRun *v1alpha1.AnalysisRun, metric v1alpha1.Metric, measurement v1alpha1.Measurement) v1alpha1.Measurement {
	r.LogCtx.Info("Terminating chaos experiment measurement")
	
	// Try to cleanup the experiment if it exists
	config, err := r.parseConfig(metric)
	if err != nil {
		r.LogCtx.Errorf("Failed to parse config during termination: %v", err)
		return measurement
	}

	if experimentName, exists := measurement.Metadata["experimentName"]; exists {
		experimentNamespace := measurement.Metadata["experimentNamespace"]
		experimentKind := measurement.Metadata["experimentKind"]
		
		chaosClient, err := chaos.NewClient(r.LogCtx)
		if err != nil {
			r.LogCtx.Errorf("Failed to create Chaos Mesh client during termination: %v", err)
			return measurement
		}

		ctx := context.Background()
		if err := chaosClient.DeleteExperiment(ctx, experimentNamespace, experimentName, experimentKind); err != nil {
			r.LogCtx.Warnf("Failed to cleanup experiment during termination: %v", err)
		} else {
			r.LogCtx.Infof("Cleaned up chaos experiment during termination: %s/%s", experimentNamespace, experimentName)
		}
	}

	return measurement
}

// GarbageCollect cleans up old measurements (not implemented)
func (r *RpcPlugin) GarbageCollect(*v1alpha1.AnalysisRun, v1alpha1.Metric, int) types.RpcError {
	r.LogCtx.Debug("GarbageCollect called - not implemented")
	return types.RpcError{}
}

// Type returns the plugin type
func (r *RpcPlugin) Type() string {
	return plugin.ProviderType
}

// GetMetadata returns metadata about the metric
func (r *RpcPlugin) GetMetadata(metric v1alpha1.Metric) map[string]string {
	metadata := make(map[string]string)
	
	config, err := r.parseConfig(metric)
	if err != nil {
		r.LogCtx.Errorf("Failed to parse config for metadata: %v", err)
		return metadata
	}

	metadata["pluginName"] = PluginName
	metadata["targetReplicaSetLabel"] = config.TargetReplicaSetLabel
	metadata["targetReplicaSetValue"] = config.TargetReplicaSetValue
	metadata["timeout"] = config.Timeout
	metadata["cleanupOnFinish"] = fmt.Sprintf("%t", config.CleanupOnFinish)
	
	// Extract experiment kind from CRD
	if strings.Contains(config.ChaosExperimentCRD, "kind:") {
		lines := strings.Split(config.ChaosExperimentCRD, "\n")
		for _, line := range lines {
			if strings.Contains(line, "kind:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					metadata["experimentKind"] = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	return metadata
}

// parseConfig parses the plugin configuration from the metric
func (r *RpcPlugin) parseConfig(metric v1alpha1.Metric) (*Config, error) {
	config := &Config{
		CleanupOnFinish: true, // Default to cleanup
		Timeout:         DefaultTimeout.String(),
	}

	// The plugin configuration should be under the plugin name key
	configData, exists := metric.Provider.Plugin[PluginName]
	if !exists {
		return nil, fmt.Errorf("plugin configuration not found under key '%s'", PluginName)
	}

	if err := json.Unmarshal(configData, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin configuration: %w", err)
	}

	return config, nil
}

// validateConfig validates the plugin configuration
func (r *RpcPlugin) validateConfig(config *Config) error {
	if config.ChaosExperimentCRD == "" {
		return fmt.Errorf("chaosExperimentCRD is required")
	}

	if config.TargetReplicaSetLabel == "" {
		return fmt.Errorf("targetReplicaSetLabel is required")
	}

	if config.TargetReplicaSetValue == "" {
		return fmt.Errorf("targetReplicaSetValue is required")
	}

	// Validate timeout format if provided
	if config.Timeout != "" {
		if _, err := time.ParseDuration(config.Timeout); err != nil {
			return fmt.Errorf("invalid timeout format: %w", err)
		}
	}

	return nil
}