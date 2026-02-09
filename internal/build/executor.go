package build

import (
	"context"
	"sort"
	"sync"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/output"
)

// Executor runs transformers in parallel.
type Executor struct {
	workers int
}

// NewExecutor creates a new Executor with the specified worker count.
func NewExecutor(workers int) *Executor {
	if workers < 1 {
		workers = 1
	}
	return &Executor{workers: workers}
}

// Job is a unit of work for a worker.
type Job struct {
	Transformer *LoadedTransformer
	Component   *LoadedComponent
	Release     *BuiltRelease
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
	Resources []*Resource
	Errors    []error
}

// ExecuteWithTransformers runs transformations with explicit transformer map.
func (e *Executor) ExecuteWithTransformers(
	ctx context.Context,
	match *MatchResult,
	release *BuiltRelease,
	transformers map[string]*LoadedTransformer,
) *ExecuteResult {
	result := &ExecuteResult{Resources: make([]*Resource, 0), Errors: make([]error, 0)}

	var jobs []Job
	for tfFQN, components := range match.ByTransformer {
		transformer, ok := transformers[tfFQN]
		if !ok {
			output.Debug("transformer not found for FQN", "fqn", tfFQN)
			continue
		}
		for _, comp := range components {
			jobs = append(jobs, Job{
				Transformer: transformer,
				Component:   comp,
				Release:     release,
			})
		}
	}

	if len(jobs) == 0 {
		return result
	}

	output.Debug("executing jobs", "count", len(jobs), "workers", e.workers)

	jobChan := make(chan Job, len(jobs))
	resultChan := make(chan JobResult, len(jobs))

	var wg sync.WaitGroup
	workerCount := e.workers
	if workerCount > len(jobs) {
		workerCount = len(jobs)
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.runWorker(ctx, jobChan, resultChan)
		}()
	}

	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for jobResult := range resultChan {
		if jobResult.Error != nil {
			result.Errors = append(result.Errors, jobResult.Error)
			continue
		}
		for _, obj := range jobResult.Resources {
			result.Resources = append(result.Resources, &Resource{
				Object:      obj,
				Component:   jobResult.Component,
				Transformer: jobResult.Transformer,
			})
		}
	}

	output.Debug("execution complete", "resources", len(result.Resources), "errors", len(result.Errors))
	return result
}

func (e *Executor) runWorker(ctx context.Context, jobs <-chan Job, results chan<- JobResult) {
	for job := range jobs {
		select {
		case <-ctx.Done():
			results <- JobResult{Component: job.Component.Name, Transformer: job.Transformer.FQN, Error: ctx.Err()}
			return
		default:
			results <- e.executeJob(job)
		}
	}
}

func (e *Executor) executeJob(job Job) JobResult {
	result := JobResult{
		Component:   job.Component.Name,
		Transformer: job.Transformer.FQN,
		Resources:   make([]*unstructured.Unstructured, 0),
	}

	cueCtx := job.Transformer.Value.Context()

	// Get the #transform from the transformer
	transformValue := job.Transformer.Value.LookupPath(cue.ParsePath("#transform"))
	if !transformValue.Exists() {
		result.Error = &TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          errMissingTransform,
		}
		return result
	}

	// Component.Value is already concrete from release building
	// Use FillPath to inject #component directly
	unified := transformValue.FillPath(cue.ParsePath("#component"), job.Component.Value)
	if unified.Err() != nil {
		result.Error = &TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          unified.Err(),
		}
		return result
	}

	// Build #context by filling individual fields
	// We need to use regular field names, not definitions, since we're using Encode
	tfCtx := NewTransformerContext(job.Release, job.Component)

	// Fill the regular fields first
	unified = unified.FillPath(cue.ParsePath("#context.name"), cueCtx.Encode(tfCtx.Name))
	unified = unified.FillPath(cue.ParsePath("#context.namespace"), cueCtx.Encode(tfCtx.Namespace))

	// Fill the definition fields using the proper definition selector
	moduleMetaMap := map[string]any{
		"name":    tfCtx.ModuleMetadata.Name,
		"version": tfCtx.ModuleMetadata.Version,
	}
	if len(tfCtx.ModuleMetadata.Labels) > 0 {
		moduleMetaMap["labels"] = tfCtx.ModuleMetadata.Labels
	}
	unified = unified.FillPath(cue.MakePath(cue.Def("context"), cue.Def("moduleMetadata")), cueCtx.Encode(moduleMetaMap))

	compMetaMap := map[string]any{
		"name": tfCtx.ComponentMetadata.Name,
	}
	if len(tfCtx.ComponentMetadata.Labels) > 0 {
		compMetaMap["labels"] = tfCtx.ComponentMetadata.Labels
	}
	if len(tfCtx.ComponentMetadata.Resources) > 0 {
		compMetaMap["resources"] = tfCtx.ComponentMetadata.Resources
	}
	if len(tfCtx.ComponentMetadata.Traits) > 0 {
		compMetaMap["traits"] = tfCtx.ComponentMetadata.Traits
	}
	unified = unified.FillPath(cue.MakePath(cue.Def("context"), cue.Def("componentMetadata")), cueCtx.Encode(compMetaMap))

	if unified.Err() != nil {
		result.Error = &TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          unified.Err(),
		}
		return result
	}

	// Extract output
	outputValue := unified.LookupPath(cue.ParsePath("output"))
	if !outputValue.Exists() {
		// No output is valid - transformer doesn't apply
		return result
	}

	// Check for errors in output (e.g., missing required fields)
	if outputValue.Err() != nil {
		result.Error = &TransformError{
			ComponentName:  job.Component.Name,
			TransformerFQN: job.Transformer.FQN,
			Cause:          outputValue.Err(),
		}
		return result
	}

	// Decode output - handles three cases:
	// 1. List: iterate elements, decode each as a resource
	// 2. Struct with apiVersion: single resource (e.g., Deployment)
	// 3. Struct without apiVersion: map of resources keyed by name (e.g., PVC per volume)
	if outputValue.Kind() == cue.ListKind {
		iter, err := outputValue.List()
		if err != nil {
			result.Error = &TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
			return result
		}
		for iter.Next() {
			obj, err := e.decodeResource(iter.Value())
			if err != nil {
				result.Error = &TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
				return result
			}
			result.Resources = append(result.Resources, obj)
		}
	} else if e.isSingleResource(outputValue) {
		obj, err := e.decodeResource(outputValue)
		if err != nil {
			result.Error = &TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
			return result
		}
		result.Resources = append(result.Resources, obj)
	} else {
		// Map of resources: iterate struct fields and decode each value
		iter, err := outputValue.Fields()
		if err != nil {
			result.Error = &TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
			return result
		}
		for iter.Next() {
			obj, err := e.decodeResource(iter.Value())
			if err != nil {
				result.Error = &TransformError{ComponentName: job.Component.Name, TransformerFQN: job.Transformer.FQN, Cause: err}
				return result
			}
			result.Resources = append(result.Resources, obj)
		}
	}

	return result
}

// isSingleResource checks whether a CUE struct value represents a single Kubernetes
// resource (has apiVersion at top level) vs a map of multiple resources keyed by name.
func (e *Executor) isSingleResource(value cue.Value) bool {
	apiVersion := value.LookupPath(cue.ParsePath("apiVersion"))
	return apiVersion.Exists()
}

func (e *Executor) decodeResource(value cue.Value) (*unstructured.Unstructured, error) {
	var obj map[string]any
	if err := value.Decode(&obj); err != nil {
		return nil, err
	}
	// Post-process to convert OPM maps to Kubernetes arrays
	normalizeK8sResource(obj)
	return &unstructured.Unstructured{Object: obj}, nil
}

// normalizeK8sResource converts OPM-style maps to Kubernetes-style arrays
// for container ports and env variables throughout the resource tree.
func normalizeK8sResource(obj map[string]any) {
	// Process spec.template.spec.containers (Deployment, StatefulSet, DaemonSet, Job)
	if spec, ok := obj["spec"].(map[string]any); ok {
		// Direct containers (for Pod-like resources)
		if containers, ok := spec["containers"].([]any); ok {
			normalizeContainers(containers)
		}
		// template.spec.containers (Deployment, StatefulSet, DaemonSet)
		if template, ok := spec["template"].(map[string]any); ok {
			if templateSpec, ok := template["spec"].(map[string]any); ok {
				if containers, ok := templateSpec["containers"].([]any); ok {
					normalizeContainers(containers)
				}
				if initContainers, ok := templateSpec["initContainers"].([]any); ok {
					normalizeContainers(initContainers)
				}
			}
		}
		// jobTemplate.spec.template.spec.containers (CronJob)
		if jobTemplate, ok := spec["jobTemplate"].(map[string]any); ok {
			if jobSpec, ok := jobTemplate["spec"].(map[string]any); ok {
				if template, ok := jobSpec["template"].(map[string]any); ok {
					if templateSpec, ok := template["spec"].(map[string]any); ok {
						if containers, ok := templateSpec["containers"].([]any); ok {
							normalizeContainers(containers)
						}
						if initContainers, ok := templateSpec["initContainers"].([]any); ok {
							normalizeContainers(initContainers)
						}
					}
				}
			}
		}
	}
}

// normalizeContainers processes a list of containers, converting maps to arrays.
func normalizeContainers(containers []any) {
	for _, c := range containers {
		container, ok := c.(map[string]any)
		if !ok {
			continue
		}
		// Convert ports map to array
		if ports, ok := container["ports"].(map[string]any); ok {
			container["ports"] = mapToPortsArray(ports)
		}
		// Convert env map to array
		if env, ok := container["env"].(map[string]any); ok {
			container["env"] = mapToEnvArray(env)
		}
	}
}

// mapToPortsArray converts a map of port definitions to a Kubernetes ports array.
// Input: {"http": {"name": "http", "targetPort": 80, "protocol": "TCP"}}
// Output: [{"name": "http", "containerPort": 80, "protocol": "TCP"}]
func mapToPortsArray(ports map[string]any) []any {
	result := make([]any, 0, len(ports))
	// Collect keys and sort for deterministic output
	keys := make([]string, 0, len(ports))
	for k := range ports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, portName := range keys {
		port, ok := ports[portName].(map[string]any)
		if !ok {
			continue
		}
		k8sPort := map[string]any{
			"name": portName,
		}
		// Map targetPort to containerPort
		if targetPort, ok := port["targetPort"]; ok {
			k8sPort["containerPort"] = targetPort
		}
		if protocol, ok := port["protocol"]; ok {
			k8sPort["protocol"] = protocol
		}
		result = append(result, k8sPort)
	}
	return result
}

// mapToEnvArray converts a map of env definitions to a Kubernetes env array.
// Input: {"apiUrl": {"name": "API_URL", "value": "http://api:3000"}}
// Output: [{"name": "API_URL", "value": "http://api:3000"}]
func mapToEnvArray(env map[string]any) []any {
	result := make([]any, 0, len(env))
	// Collect keys and sort for deterministic output
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		envVar, ok := env[k].(map[string]any)
		if !ok {
			continue
		}
		// Copy the env var definition directly - it already has name/value
		result = append(result, envVar)
	}
	return result
}

var errMissingTransform = &transformMissingError{}

type transformMissingError struct{}

func (e *transformMissingError) Error() string {
	return "transformer missing #transform function"
}
