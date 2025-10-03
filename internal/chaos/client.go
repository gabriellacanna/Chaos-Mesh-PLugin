package chaos

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
	log "github.com/sirupsen/logrus"
)

// Client represents a Chaos Mesh client
type Client struct {
	dynamicClient dynamic.Interface
	logger        log.Entry
}

// NewClient creates a new Chaos Mesh client
func NewClient(logger log.Entry) (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig if not running in cluster
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
		}
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		dynamicClient: dynamicClient,
		logger:        logger,
	}, nil
}

// CreateExperiment creates a Chaos Mesh experiment from YAML
func (c *Client) CreateExperiment(ctx context.Context, experimentYAML string, targetSelector map[string]string) (*unstructured.Unstructured, error) {
	// Parse the YAML
	var obj unstructured.Unstructured
	if err := yaml.Unmarshal([]byte(experimentYAML), &obj); err != nil {
		return nil, fmt.Errorf("failed to parse experiment YAML: %w", err)
	}

	// Inject the target selector
	if err := c.injectSelector(&obj, targetSelector); err != nil {
		return nil, fmt.Errorf("failed to inject selector: %w", err)
	}

	// Get the GVR for the resource
	gvr, err := c.getGVR(obj.GetKind())
	if err != nil {
		return nil, fmt.Errorf("failed to get GVR for kind %s: %w", obj.GetKind(), err)
	}

	// Create the resource
	namespace := obj.GetNamespace()
	if namespace == "" {
		namespace = "default"
	}

	c.logger.Infof("Creating Chaos Mesh experiment: %s/%s", namespace, obj.GetName())
	
	result, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, &obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create experiment: %w", err)
	}

	return result, nil
}

// WatchExperiment watches a Chaos Mesh experiment until completion or timeout
func (c *Client) WatchExperiment(ctx context.Context, namespace, name, kind string, timeout time.Duration) (bool, error) {
	gvr, err := c.getGVR(kind)
	if err != nil {
		return false, fmt.Errorf("failed to get GVR for kind %s: %w", kind, err)
	}

	c.logger.Infof("Watching Chaos Mesh experiment: %s/%s", namespace, name)

	// Create a context with timeout
	watchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Start watching
	watcher, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Watch(watchCtx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return false, fmt.Errorf("failed to start watching experiment: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return false, fmt.Errorf("watch channel closed unexpectedly")
			}

			if event.Type == watch.Error {
				return false, fmt.Errorf("watch error: %v", event.Object)
			}

			obj, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				continue
			}

			success, finished, err := c.checkExperimentStatus(obj)
			if err != nil {
				c.logger.Warnf("Error checking experiment status: %v", err)
				continue
			}

			if finished {
				c.logger.Infof("Experiment %s/%s finished with success=%t", namespace, name, success)
				return success, nil
			}

		case <-watchCtx.Done():
			return false, fmt.Errorf("timeout waiting for experiment to complete")
		}
	}
}

// DeleteExperiment deletes a Chaos Mesh experiment
func (c *Client) DeleteExperiment(ctx context.Context, namespace, name, kind string) error {
	gvr, err := c.getGVR(kind)
	if err != nil {
		return fmt.Errorf("failed to get GVR for kind %s: %w", kind, err)
	}

	c.logger.Infof("Deleting Chaos Mesh experiment: %s/%s", namespace, name)

	err = c.dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete experiment: %w", err)
	}

	return nil
}

// injectSelector injects the target selector into the experiment spec
func (c *Client) injectSelector(obj *unstructured.Unstructured, targetSelector map[string]string) error {
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return fmt.Errorf("failed to get spec: %w", err)
	}
	if !found {
		return fmt.Errorf("spec not found in experiment")
	}

	// Get existing selector or create new one
	selector, found, err := unstructured.NestedMap(spec, "selector")
	if err != nil {
		return fmt.Errorf("failed to get selector: %w", err)
	}
	if !found {
		selector = make(map[string]interface{})
	}

	// Inject labelSelectors
	if len(targetSelector) > 0 {
		labelSelectors := make(map[string]interface{})
		for k, v := range targetSelector {
			labelSelectors[k] = v
		}
		selector["labelSelectors"] = labelSelectors
	}

	// Set the updated selector back
	spec["selector"] = selector
	if err := unstructured.SetNestedMap(obj.Object, spec, "spec"); err != nil {
		return fmt.Errorf("failed to set updated spec: %w", err)
	}

	return nil
}

// checkExperimentStatus checks if the experiment is finished and successful
func (c *Client) checkExperimentStatus(obj *unstructured.Unstructured) (success bool, finished bool, err error) {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil {
		return false, false, fmt.Errorf("failed to get status: %w", err)
	}
	if !found {
		// No status yet, experiment is still starting
		return false, false, nil
	}

	// Check experiment phase
	experiment, found, err := unstructured.NestedMap(status, "experiment")
	if err != nil {
		return false, false, fmt.Errorf("failed to get experiment status: %w", err)
	}
	if !found {
		return false, false, nil
	}

	phase, found, err := unstructured.NestedString(experiment, "phase")
	if err != nil {
		return false, false, fmt.Errorf("failed to get phase: %w", err)
	}
	if !found {
		return false, false, nil
	}

	c.logger.Debugf("Experiment phase: %s", phase)

	switch phase {
	case "Running":
		return false, false, nil
	case "Finished":
		// Check if there are any error conditions
		conditions, found, err := unstructured.NestedSlice(status, "conditions")
		if err != nil {
			return false, true, fmt.Errorf("failed to get conditions: %w", err)
		}
		if found {
			for _, conditionInterface := range conditions {
				condition, ok := conditionInterface.(map[string]interface{})
				if !ok {
					continue
				}
				
				condType, found, err := unstructured.NestedString(condition, "type")
				if err != nil || !found {
					continue
				}
				
				condStatus, found, err := unstructured.NestedString(condition, "status")
				if err != nil || !found {
					continue
				}
				
				// If there's an error condition, the experiment failed
				if condType == "AllInjected" && condStatus != "True" {
					return false, true, nil
				}
				if condType == "AllRecovered" && condStatus != "True" {
					return false, true, nil
				}
			}
		}
		return true, true, nil
	case "Failed", "Error":
		return false, true, nil
	default:
		return false, false, nil
	}
}

// getGVR returns the GroupVersionResource for a given kind
func (c *Client) getGVR(kind string) (schema.GroupVersionResource, error) {
	switch kind {
	case "PodChaos":
		return schema.GroupVersionResource{
			Group:    "chaos-mesh.org",
			Version:  "v1alpha1",
			Resource: "podchaos",
		}, nil
	case "NetworkChaos":
		return schema.GroupVersionResource{
			Group:    "chaos-mesh.org",
			Version:  "v1alpha1",
			Resource: "networkchaos",
		}, nil
	case "StressChaos":
		return schema.GroupVersionResource{
			Group:    "chaos-mesh.org",
			Version:  "v1alpha1",
			Resource: "stresschaos",
		}, nil
	case "IOChaos":
		return schema.GroupVersionResource{
			Group:    "chaos-mesh.org",
			Version:  "v1alpha1",
			Resource: "iochaos",
		}, nil
	case "TimeChaos":
		return schema.GroupVersionResource{
			Group:    "chaos-mesh.org",
			Version:  "v1alpha1",
			Resource: "timechaos",
		}, nil
	case "KernelChaos":
		return schema.GroupVersionResource{
			Group:    "chaos-mesh.org",
			Version:  "v1alpha1",
			Resource: "kernelchaos",
		}, nil
	case "DNSChaos":
		return schema.GroupVersionResource{
			Group:    "chaos-mesh.org",
			Version:  "v1alpha1",
			Resource: "dnschaos",
		}, nil
	case "HTTPChaos":
		return schema.GroupVersionResource{
			Group:    "chaos-mesh.org",
			Version:  "v1alpha1",
			Resource: "httpchaos",
		}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unsupported chaos kind: %s", kind)
	}
}