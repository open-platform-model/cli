// Package transform provides provider loading, component matching, and transformer execution.
package transform

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/core"
)

// Job is a unit of work: one transformer applied to one component.
type Job struct {
	Transformer *LoadedTransformer
	Component   *core.Component
	Release     *core.ModuleRelease
}

// JobResult is the result of executing a job.
type JobResult struct {
	Component   string
	Transformer string
	Resources   []*unstructured.Unstructured
	Error       error
}

// ExecuteResult is the combined result of all jobs.
type ExecuteResult struct {
	Resources []*core.Resource
	Errors    []error
}
