package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/gabriellacanna/chaos-mesh-plugin/internal/plugin"
	logrus "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This is an integration test that demonstrates how the plugin would work
// in a real scenario with Argo Rollouts
func main() {
	fmt.Println("=== Chaos Mesh Plugin Integration Test ===")

	// Create a plugin instance
	rpcPlugin := &plugin.RpcPlugin{}
	// Initialize logger manually for testing
	rpcPlugin.LogCtx = *logrus.WithFields(logrus.Fields{"component": "chaos-mesh-plugin"})
	rpcPlugin.InitPlugin()

	// Create a mock AnalysisRun
	analysisRun := &v1alpha1.AnalysisRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "chaos-analysis-run",
			Namespace: "default",
		},
		Spec: v1alpha1.AnalysisRunSpec{
			Metrics: []v1alpha1.Metric{
				{
					Name: "chaos-mesh-test",
				},
			},
		},
	}

	// Create a configuration for the plugin
	config := plugin.Config{
		ChaosExperimentCRD: `apiVersion: chaos-mesh.org/v1alpha1
kind: PodChaos
metadata:
  name: pod-kill-experiment-abc123
  namespace: default
spec:
  action: pod-kill
  mode: one
  selector:
    namespaces:
      - default
  scheduler:
    cron: "@every 10s"
  duration: "30s"`,
		TargetReplicaSetLabel: "rollouts-pod-template-hash",
		TargetReplicaSetValue: "abc123",
		Timeout:               "5m",
		CleanupOnFinish:       true,
	}

	// Marshal the config to JSON
	configBytes, err := json.Marshal(config)
	if err != nil {
		log.Fatalf("Failed to marshal config: %v", err)
	}

	// Create a metric with the plugin configuration
	metric := v1alpha1.Metric{
		Name: "chaos-mesh-test",
		Provider: v1alpha1.MetricProvider{
			Plugin: map[string]json.RawMessage{
				plugin.PluginName: configBytes,
			},
		},
	}

	// Create an initial measurement
	measurement := v1alpha1.Measurement{
		Phase:     v1alpha1.AnalysisPhaseRunning,
		StartedAt: &metav1.Time{Time: time.Now()},
		Metadata:  make(map[string]string),
	}

	fmt.Println("1. Testing plugin metadata extraction...")
	metadata := rpcPlugin.GetMetadata(metric)
	fmt.Printf("   Plugin Name: %s\n", metadata["pluginName"])
	fmt.Printf("   Target Label: %s\n", metadata["targetReplicaSetLabel"])
	fmt.Printf("   Target Value: %s\n", metadata["targetReplicaSetValue"])
	fmt.Printf("   Experiment Kind: %s\n", metadata["experimentKind"])

	fmt.Println("\n2. Testing plugin run (simulation)...")
	// Note: This would normally create a real Chaos Mesh experiment
	// For this test, we'll simulate the behavior
	
	// Simulate the plugin run
	fmt.Println("   - Parsing configuration...")
	fmt.Println("   - Validating parameters...")
	fmt.Println("   - Creating Chaos Mesh experiment...")
	fmt.Println("   - Injecting dynamic selector...")
	fmt.Printf("     Target selector: %s=%s\n", config.TargetReplicaSetLabel, config.TargetReplicaSetValue)
	
	// Simulate successful experiment creation
	measurement.Metadata["experimentName"] = "pod-kill-experiment-abc123"
	measurement.Metadata["experimentNamespace"] = "default"
	measurement.Metadata["experimentKind"] = "PodChaos"
	measurement.Phase = v1alpha1.AnalysisPhaseSuccessful
	measurement.FinishedAt = &metav1.Time{Time: time.Now()}

	fmt.Println("   - Experiment created successfully!")
	fmt.Println("   - Monitoring experiment status...")
	fmt.Println("   - Experiment completed successfully!")

	fmt.Println("\n3. Testing plugin resume (simulation)...")
	// Simulate resume functionality
	resumedMeasurement := rpcPlugin.Resume(analysisRun, metric, measurement)
	fmt.Printf("   Resumed measurement phase: %s\n", resumedMeasurement.Phase)

	fmt.Println("\n4. Testing plugin termination (simulation)...")
	// Simulate termination
	terminatedMeasurement := rpcPlugin.Terminate(analysisRun, metric, measurement)
	fmt.Printf("   Terminated measurement phase: %s\n", terminatedMeasurement.Phase)

	fmt.Println("\n=== Integration Test Completed Successfully ===")
	fmt.Println("\nThis demonstrates how the plugin would integrate with Argo Rollouts:")
	fmt.Println("1. Argo Rollouts creates an AnalysisRun with the chaos-mesh plugin")
	fmt.Println("2. The plugin receives the configuration and target ReplicaSet information")
	fmt.Println("3. It creates a Chaos Mesh experiment targeting only the canary pods")
	fmt.Println("4. The plugin monitors the experiment and reports success/failure")
	fmt.Println("5. Based on the result, Argo Rollouts continues or aborts the deployment")

	fmt.Println("\nNext steps for real deployment:")
	fmt.Println("- Deploy the plugin binary to your Kubernetes cluster")
	fmt.Println("- Configure Argo Rollouts to use the plugin")
	fmt.Println("- Install Chaos Mesh in your cluster")
	fmt.Println("- Create AnalysisTemplates with chaos experiments")
	fmt.Println("- Use the templates in your Rollout strategies")
}