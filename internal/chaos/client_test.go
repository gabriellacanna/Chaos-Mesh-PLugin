package chaos

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	log "github.com/sirupsen/logrus"
)

func TestGetGVR(t *testing.T) {
	client := &Client{
		logger: *log.WithFields(log.Fields{"test": "chaos"}),
	}

	tests := []struct {
		kind     string
		expected schema.GroupVersionResource
		hasError bool
	}{
		{
			kind: "PodChaos",
			expected: schema.GroupVersionResource{
				Group:    "chaos-mesh.org",
				Version:  "v1alpha1",
				Resource: "podchaos",
			},
			hasError: false,
		},
		{
			kind: "NetworkChaos",
			expected: schema.GroupVersionResource{
				Group:    "chaos-mesh.org",
				Version:  "v1alpha1",
				Resource: "networkchaos",
			},
			hasError: false,
		},
		{
			kind: "StressChaos",
			expected: schema.GroupVersionResource{
				Group:    "chaos-mesh.org",
				Version:  "v1alpha1",
				Resource: "stresschaos",
			},
			hasError: false,
		},
		{
			kind:     "UnsupportedChaos",
			expected: schema.GroupVersionResource{},
			hasError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.kind, func(t *testing.T) {
			gvr, err := client.getGVR(test.kind)
			
			if test.hasError {
				if err == nil {
					t.Errorf("Expected error for kind %s, but got none", test.kind)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for kind %s: %v", test.kind, err)
				return
			}

			if gvr != test.expected {
				t.Errorf("Expected GVR %+v for kind %s, got %+v", test.expected, test.kind, gvr)
			}
		})
	}
}

func TestInjectSelector(t *testing.T) {
	client := &Client{
		logger: *log.WithFields(log.Fields{"test": "chaos"}),
	}

	// Create a test unstructured object
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "chaos-mesh.org/v1alpha1",
			"kind":       "PodChaos",
			"metadata": map[string]interface{}{
				"name":      "test-chaos",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"action": "pod-kill",
				"mode":   "one",
				"selector": map[string]interface{}{
					"namespaces": []interface{}{"default"},
				},
			},
		},
	}

	targetSelector := map[string]string{
		"rollouts-pod-template-hash": "abc123",
		"app":                        "test-app",
	}

	err := client.injectSelector(obj, targetSelector)
	if err != nil {
		t.Fatalf("Failed to inject selector: %v", err)
	}

	// Verify the selector was injected correctly
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		t.Fatalf("Failed to get spec from object: %v", err)
	}

	selector, found, err := unstructured.NestedMap(spec, "selector")
	if err != nil || !found {
		t.Fatalf("Failed to get selector from spec: %v", err)
	}

	labelSelectors, found, err := unstructured.NestedMap(selector, "labelSelectors")
	if err != nil || !found {
		t.Fatalf("Failed to get labelSelectors from selector: %v", err)
	}

	// Check if the target selector was injected
	if labelSelectors["rollouts-pod-template-hash"] != "abc123" {
		t.Errorf("Expected rollouts-pod-template-hash to be 'abc123', got '%v'", labelSelectors["rollouts-pod-template-hash"])
	}

	if labelSelectors["app"] != "test-app" {
		t.Errorf("Expected app to be 'test-app', got '%v'", labelSelectors["app"])
	}
}

func TestCheckExperimentStatus(t *testing.T) {
	client := &Client{
		logger: *log.WithFields(log.Fields{"test": "chaos"}),
	}

	tests := []struct {
		name             string
		obj              *unstructured.Unstructured
		expectedSuccess  bool
		expectedFinished bool
		expectedError    bool
	}{
		{
			name: "Running experiment",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"experiment": map[string]interface{}{
							"phase": "Running",
						},
					},
				},
			},
			expectedSuccess:  false,
			expectedFinished: false,
			expectedError:    false,
		},
		{
			name: "Finished successful experiment",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"experiment": map[string]interface{}{
							"phase": "Finished",
						},
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "AllInjected",
								"status": "True",
							},
							map[string]interface{}{
								"type":   "AllRecovered",
								"status": "True",
							},
						},
					},
				},
			},
			expectedSuccess:  true,
			expectedFinished: true,
			expectedError:    false,
		},
		{
			name: "Failed experiment",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"experiment": map[string]interface{}{
							"phase": "Failed",
						},
					},
				},
			},
			expectedSuccess:  false,
			expectedFinished: true,
			expectedError:    false,
		},
		{
			name: "No status",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			expectedSuccess:  false,
			expectedFinished: false,
			expectedError:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			success, finished, err := client.checkExperimentStatus(test.obj)

			if test.expectedError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !test.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if success != test.expectedSuccess {
				t.Errorf("Expected success=%t, got success=%t", test.expectedSuccess, success)
			}

			if finished != test.expectedFinished {
				t.Errorf("Expected finished=%t, got finished=%t", test.expectedFinished, finished)
			}
		})
	}
}