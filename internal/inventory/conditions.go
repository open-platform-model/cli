package inventory

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ConditionTypeReady is the ModuleInstance status condition the operator sets
// to report reconcile outcome. The CLI only ever reads it (enhancement 0006
// D2/D25: conditions are operator-owned status).
const ConditionTypeReady = "Ready"

// Condition status values, matching metav1.ConditionStatus.
const (
	ConditionTrue    = "True"
	ConditionFalse   = "False"
	ConditionUnknown = "Unknown"
)

// Condition is the CLI's read-only view of one ModuleInstance status condition.
type Condition struct {
	Type    string
	Status  string
	Reason  string
	Message string
	// ObservedGeneration is the condition's own observedGeneration. Operators
	// may omit it; 0 means "not stated" and the CR-level
	// status.observedGeneration is the authority (see Record.ReadyFor).
	ObservedGeneration int64
}

// IsTrue reports whether the condition holds.
func (c *Condition) IsTrue() bool {
	return c != nil && c.Status == ConditionTrue
}

// Describe renders a condition for user-facing output: "False (ReconcileFailed):
// <message>". An absent condition describes as "no Ready condition reported".
func (c *Condition) Describe() string {
	if c == nil {
		return "no " + ConditionTypeReady + " condition reported"
	}
	out := c.Status
	if c.Reason != "" {
		out += " (" + c.Reason + ")"
	}
	if c.Message != "" {
		out += ": " + c.Message
	}
	return out
}

// FindCondition returns the condition of the given type, or nil when absent.
func (r *Record) FindCondition(condType string) *Condition {
	if r == nil {
		return nil
	}
	for i := range r.Conditions {
		if r.Conditions[i].Type == condType {
			return &r.Conditions[i]
		}
	}
	return nil
}

// ReadyCondition returns the Ready condition regardless of which generation it
// reports on, or nil when the operator has not written one.
func (r *Record) ReadyCondition() *Condition {
	return r.FindCondition(ConditionTypeReady)
}

// ReadyFor returns the Ready condition only when it reports on generation or
// newer — the signal that the operator has finished reconciling a specific
// write rather than reporting a stale verdict from before it. A condition that
// states no observedGeneration of its own is attributed to the CR-level
// status.observedGeneration.
func (r *Record) ReadyFor(generation int64) *Condition {
	cond := r.ReadyCondition()
	if cond == nil {
		return nil
	}
	if cond.ObservedGeneration > 0 {
		if cond.ObservedGeneration < generation {
			return nil
		}
		return cond
	}
	if r.ObservedGeneration < generation {
		return nil
	}
	return cond
}

// conditionsFromUnstructured reads status.conditions best-effort: a missing or
// wrong-typed conditions array yields no conditions rather than an error, since
// every caller treats "not reported yet" and "unreadable" identically.
func conditionsFromUnstructured(obj *unstructured.Unstructured) []Condition {
	raw, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !found {
		return nil
	}

	conditions := make([]Condition, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		c := Condition{
			Type:    nestedString(m, "type"),
			Status:  nestedString(m, "status"),
			Reason:  nestedString(m, "reason"),
			Message: nestedString(m, "message"),
		}
		if c.Type == "" {
			continue
		}
		//nolint:errcheck // best-effort read; a wrong-typed observedGeneration is treated as unstated
		if gen, ok, _ := unstructured.NestedInt64(m, "observedGeneration"); ok {
			c.ObservedGeneration = gen
		}
		conditions = append(conditions, c)
	}
	return conditions
}
