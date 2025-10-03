package chaos

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ChaosMeshExperiment represents a generic Chaos Mesh experiment
type ChaosMeshExperiment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              runtime.RawExtension `json:"spec,omitempty"`
	Status            ExperimentStatus     `json:"status,omitempty"`
}

// ExperimentStatus represents the status of a Chaos Mesh experiment
type ExperimentStatus struct {
	Experiment ExperimentPhase `json:"experiment,omitempty"`
	Conditions []Condition     `json:"conditions,omitempty"`
}

// ExperimentPhase represents the phase of an experiment
type ExperimentPhase struct {
	Phase        string `json:"phase,omitempty"`
	DesiredPhase string `json:"desiredPhase,omitempty"`
	Message      string `json:"message,omitempty"`
}

// Condition represents a condition of the experiment
type Condition struct {
	Type    string `json:"type,omitempty"`
	Status  string `json:"status,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

// Selector represents the selector for targeting resources
type Selector struct {
	Namespaces     []string          `json:"namespaces,omitempty"`
	LabelSelectors map[string]string `json:"labelSelectors,omitempty"`
	FieldSelectors map[string]string `json:"fieldSelectors,omitempty"`
	AnnotationSelectors map[string]string `json:"annotationSelectors,omitempty"`
	ExpressionSelectors []ExpressionSelector `json:"expressionSelectors,omitempty"`
}

// ExpressionSelector represents a label selector requirement
type ExpressionSelector struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// PodChaosSpec represents the spec for PodChaos
type PodChaosSpec struct {
	Selector  Selector `json:"selector"`
	Action    string   `json:"action"`
	Mode      string   `json:"mode"`
	Value     string   `json:"value,omitempty"`
	Duration  string   `json:"duration,omitempty"`
	Scheduler *Scheduler `json:"scheduler,omitempty"`
}

// NetworkChaosSpec represents the spec for NetworkChaos
type NetworkChaosSpec struct {
	Selector  Selector `json:"selector"`
	Action    string   `json:"action"`
	Mode      string   `json:"mode"`
	Value     string   `json:"value,omitempty"`
	Duration  string   `json:"duration,omitempty"`
	Direction string   `json:"direction,omitempty"`
	Target    *Selector `json:"target,omitempty"`
	Delay     *DelaySpec `json:"delay,omitempty"`
	Loss      *LossSpec  `json:"loss,omitempty"`
	Duplicate *DuplicateSpec `json:"duplicate,omitempty"`
	Corrupt   *CorruptSpec   `json:"corrupt,omitempty"`
	Bandwidth *BandwidthSpec `json:"bandwidth,omitempty"`
}

// Scheduler represents the scheduler configuration
type Scheduler struct {
	Cron string `json:"cron,omitempty"`
}

// DelaySpec represents network delay configuration
type DelaySpec struct {
	Latency     string `json:"latency,omitempty"`
	Correlation string `json:"correlation,omitempty"`
	Jitter      string `json:"jitter,omitempty"`
}

// LossSpec represents packet loss configuration
type LossSpec struct {
	Loss        string `json:"loss,omitempty"`
	Correlation string `json:"correlation,omitempty"`
}

// DuplicateSpec represents packet duplication configuration
type DuplicateSpec struct {
	Duplicate   string `json:"duplicate,omitempty"`
	Correlation string `json:"correlation,omitempty"`
}

// CorruptSpec represents packet corruption configuration
type CorruptSpec struct {
	Corrupt     string `json:"corrupt,omitempty"`
	Correlation string `json:"correlation,omitempty"`
}

// BandwidthSpec represents bandwidth limitation configuration
type BandwidthSpec struct {
	Rate     string `json:"rate,omitempty"`
	Limit    uint32 `json:"limit,omitempty"`
	Buffer   uint32 `json:"buffer,omitempty"`
	Peakrate string `json:"peakrate,omitempty"`
	Minburst uint32 `json:"minburst,omitempty"`
}