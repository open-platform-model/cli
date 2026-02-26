package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/opmodel/cli/internal/output"
)

// noComponentLabel is the placeholder used for resources missing a component mapping.
const noComponentLabel = "(no component)"

// displayKind returns an abbreviated kind name for terminal display.
// PersistentVolumeClaim is shortened to PVC to keep tree lines compact.
// The full kind is preserved in ResourceNode.Kind for JSON/YAML output.
func displayKind(kind string) string {
	if kind == "PersistentVolumeClaim" {
		return "PVC"
	}
	return kind
}

// Tree chrome constants — box-drawing connectors used in terminal rendering.
const (
	treeConnMid  = "├── "
	treeConnLast = "└── "
)

// treeMinStatusGap is the minimum number of spaces between the end of a
// resource name and its status token. The longest name in the tree is the
// anchor; all other names get more padding to reach the same column.
const treeMinStatusGap = 6

// Kubernetes workload kind constants shared across the kubernetes package.
const (
	kindDeployment  = "Deployment"
	kindStatefulSet = "StatefulSet"
	kindDaemonSet   = "DaemonSet"
	kindReplicaSet  = "ReplicaSet"
	kindJob         = "Job"
)

// ReleaseInfo holds release metadata for tree display.
// Populated by the command layer from the inventory Secret.
type ReleaseInfo struct {
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`
	Module    string `json:"module,omitempty" yaml:"module,omitempty"`
	Version   string `json:"version,omitempty" yaml:"version,omitempty"`
}

// Component represents a group of resources sharing the same component name.
type Component struct {
	Name          string         `json:"name" yaml:"name"`
	ResourceCount int            `json:"resourceCount" yaml:"resourceCount"`
	Status        HealthStatus   `json:"status" yaml:"status"`
	Resources     []ResourceNode `json:"resources,omitempty" yaml:"resources,omitempty"`
}

// ResourceNode represents a single resource in the tree hierarchy.
// Used for both OPM-managed resources and their K8s-owned children.
//
// For Pod nodes Status holds the raw Kubernetes phase string (e.g. "Running",
// "CrashLoop", "Pending") and Ready reflects the Pod Ready condition. This
// matches the display convention used by `mod status` via output.FormatPodPhase.
// For all other nodes Status is a HealthStatus value ("Ready", "NotReady", etc.)
// and Ready is unused.
type ResourceNode struct {
	Kind      string         `json:"kind" yaml:"kind"`
	Name      string         `json:"name" yaml:"name"`
	Namespace string         `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Status    HealthStatus   `json:"status" yaml:"status"`
	Ready     bool           `json:"ready,omitempty" yaml:"ready,omitempty"`
	Replicas  string         `json:"replicas,omitempty" yaml:"replicas,omitempty"`
	Children  []ResourceNode `json:"children,omitempty" yaml:"children,omitempty"`
}

// TreeResult is the intermediate data structure produced by BuildTree.
// It can be rendered as a terminal tree or serialized to JSON/YAML.
type TreeResult struct {
	Release    ReleaseInfo `json:"release" yaml:"release"`
	Components []Component `json:"components" yaml:"components"`
}

// TreeOptions configures a tree operation.
// Resource discovery and component mapping are resolved by the command layer,
// mirroring the StatusOptions pattern.
type TreeOptions struct {
	// ReleaseInfo is populated from the inventory Secret by the command layer.
	ReleaseInfo ReleaseInfo

	// InventoryLive is the list of live resources fetched from the inventory Secret.
	InventoryLive []*unstructured.Unstructured

	// ComponentMap maps "Kind/Namespace/Name" to component name.
	// Built from inventory entries by the command layer (same key format as status.go).
	ComponentMap map[string]string

	// Depth controls tree expansion:
	//   0 = component summary only (no resource rows, no K8s queries)
	//   1 = OPM-managed resources (no K8s children)
	//   2 = full tree with K8s ownership walking
	Depth int

	// OutputFormat selects the output serialization.
	OutputFormat output.Format
}

// ─────────────────────────────────────────────────────────────────────────────
// Entry point
// ─────────────────────────────────────────────────────────────────────────────

// GetModuleTree is the main entry point for the tree command.
// It returns noResourcesFoundError when InventoryLive is empty (mirrors GetReleaseStatus).
func GetModuleTree(ctx context.Context, client *Client, opts TreeOptions) (*TreeResult, error) {
	if len(opts.InventoryLive) == 0 {
		return nil, &noResourcesFoundError{
			ReleaseName: opts.ReleaseInfo.Name,
			Namespace:   opts.ReleaseInfo.Namespace,
		}
	}
	return BuildTree(ctx, client, opts), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Tree building
// ─────────────────────────────────────────────────────────────────────────────

// BuildTree constructs a TreeResult from opts, respecting the requested depth.
// At depth=0 resources are counted but not included. At depth>=2 K8s ownership
// is walked to produce child nodes (ReplicaSets, Pods, etc.).
func BuildTree(ctx context.Context, client *Client, opts TreeOptions) *TreeResult {
	result := &TreeResult{Release: opts.ReleaseInfo}

	groups := groupByComponent(opts.InventoryLive, opts.ComponentMap)
	for _, compName := range sortedComponentNames(groups) {
		resources := groups[compName]
		comp := Component{
			Name:          compName,
			ResourceCount: len(resources),
		}

		if opts.Depth >= 1 {
			for _, res := range resources {
				comp.Resources = append(comp.Resources, buildResourceNode(ctx, client, res, opts.Depth))
			}
		}

		comp.Status = aggregateStatus(comp.Resources, len(resources))
		result.Components = append(result.Components, comp)
	}

	return result
}

// buildResourceNode constructs a ResourceNode for a single live unstructured resource.
func buildResourceNode(ctx context.Context, client *Client, res *unstructured.Unstructured, depth int) ResourceNode {
	node := ResourceNode{
		Kind:      res.GetKind(),
		Name:      res.GetName(),
		Namespace: res.GetNamespace(),
		Status:    EvaluateHealth(res),
		Replicas:  getReplicaCount(res),
	}
	if depth >= 2 {
		node.Children = walkOwnership(ctx, client, res)
	}
	return node
}

// ─────────────────────────────────────────────────────────────────────────────
// Component grouping
// ─────────────────────────────────────────────────────────────────────────────

// groupByComponent groups resources by their entry in componentMap.
// Resources absent from the map (or with an empty value) land in noComponentLabel.
// The input slice order is preserved within each group.
//
// NOTE: The inventory Secret stores resources in the order they were written by
// `opm mod apply`, which is weight-ascending order (same as the transformer apply
// order). Preserving input slice order here therefore implicitly preserves weight
// order within each component — no explicit weight sort is needed.
func groupByComponent(resources []*unstructured.Unstructured, componentMap map[string]string) map[string][]*unstructured.Unstructured {
	groups := make(map[string][]*unstructured.Unstructured)
	for _, res := range resources {
		key := res.GetKind() + "/" + res.GetNamespace() + "/" + res.GetName()
		comp := componentMap[key]
		if comp == "" {
			comp = noComponentLabel
		}
		groups[comp] = append(groups[comp], res)
	}
	return groups
}

// sortedComponentNames returns component names alphabetically, with noComponentLabel last.
func sortedComponentNames(groups map[string][]*unstructured.Unstructured) []string {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i] == noComponentLabel {
			return false
		}
		if names[j] == noComponentLabel {
			return true
		}
		return names[i] < names[j]
	})
	return names
}

// aggregateStatus returns the rollup health of a component.
// When depth=0 no ResourceNodes are available, so resourceCount is used to
// signal "unknown" vs "empty". At depth>=1 all resources must be Ready/Complete.
func aggregateStatus(resources []ResourceNode, resourceCount int) HealthStatus {
	if len(resources) == 0 {
		if resourceCount > 0 {
			return HealthUnknown // depth=0 — we have resources but didn't evaluate them
		}
		return HealthUnknown
	}
	for _, r := range resources {
		if r.Status != HealthReady && r.Status != HealthComplete && r.Status != HealthBound {
			return HealthNotReady
		}
	}
	return HealthReady
}

// ─────────────────────────────────────────────────────────────────────────────
// Replica count extraction (for OPM-managed unstructured resources)
// ─────────────────────────────────────────────────────────────────────────────

// getReplicaCount returns a "ready/desired" replica string for workload resources,
// or an empty string for non-workload resources.
func getReplicaCount(res *unstructured.Unstructured) string {
	switch res.GetKind() {
	case kindDeployment, kindStatefulSet:
		desired, _, _ := unstructured.NestedInt64(res.Object, "spec", "replicas") //nolint:errcheck // best-effort; missing field treated as 0
		if desired == 0 {
			desired = 1 // K8s default when spec.replicas is omitted
		}
		ready, _, _ := unstructured.NestedInt64(res.Object, "status", "readyReplicas") //nolint:errcheck // best-effort replica display
		return fmt.Sprintf("%d/%d", ready, desired)
	case kindDaemonSet:
		desired, _, _ := unstructured.NestedInt64(res.Object, "status", "desiredNumberScheduled") //nolint:errcheck // best-effort replica display
		ready, _, _ := unstructured.NestedInt64(res.Object, "status", "numberReady")              //nolint:errcheck // best-effort replica display
		return fmt.Sprintf("%d/%d", ready, desired)
	case kindJob:
		completions, _, _ := unstructured.NestedInt64(res.Object, "spec", "completions") //nolint:errcheck // best-effort replica display
		succeeded, _, _ := unstructured.NestedInt64(res.Object, "status", "succeeded")   //nolint:errcheck // best-effort replica display
		return fmt.Sprintf("%d/%d", succeeded, completions)
	case "PersistentVolumeClaim":
		// Prefer the actual provisioned capacity; fall back to the requested size.
		if cap, found, _ := unstructured.NestedString(res.Object, "status", "capacity", "storage"); found && cap != "" { //nolint:errcheck // best-effort capacity display
			return cap
		}
		cap, _, _ := unstructured.NestedString(res.Object, "spec", "resources", "requests", "storage") //nolint:errcheck // best-effort capacity display
		return cap
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// Kubernetes ownership walking
// ─────────────────────────────────────────────────────────────────────────────

// walkOwnership returns K8s-owned child nodes for a resource.
// Only Deployment, StatefulSet, DaemonSet, and Job produce children.
// On any API error the resource is returned without children and the error is
// logged at DEBUG level (graceful degradation).
func walkOwnership(ctx context.Context, client *Client, res *unstructured.Unstructured) []ResourceNode {
	switch res.GetKind() {
	case kindDeployment:
		return walkDeployment(ctx, client, res)
	case kindStatefulSet:
		return walkStatefulSet(ctx, client, res)
	case kindDaemonSet:
		return walkDaemonSet(ctx, client, res)
	case kindJob:
		return walkJob(ctx, client, res)
	}
	return nil
}

// walkDeployment returns ReplicaSet nodes, each with their Pod children.
func walkDeployment(ctx context.Context, client *Client, res *unstructured.Unstructured) []ResourceNode {
	ns := res.GetNamespace()
	uid := res.GetUID()

	rsList, err := client.Clientset.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		output.Debug("failed to list replicasets",
			"deployment", res.GetName(), "namespace", ns, "error", err)
		return nil
	}

	var nodes []ResourceNode
	for i := range rsList.Items {
		rs := &rsList.Items[i]
		if !hasOwnerWithUID(rs.OwnerReferences, uid) {
			continue
		}
		node := ResourceNode{
			Kind:      kindReplicaSet,
			Name:      rs.Name,
			Namespace: rs.Namespace,
			Status:    replicaSetHealth(rs),
			Replicas:  fmt.Sprintf("%d pods", rs.Status.Replicas),
			Children:  walkReplicaSet(ctx, client, rs),
		}
		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
	return nodes
}

// walkReplicaSet returns Pod nodes owned by a ReplicaSet.
func walkReplicaSet(ctx context.Context, client *Client, rs *appsv1.ReplicaSet) []ResourceNode {
	podList, err := client.Clientset.CoreV1().Pods(rs.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		output.Debug("failed to list pods",
			"replicaset", rs.Name, "namespace", rs.Namespace, "error", err)
		return nil
	}

	uid := rs.UID
	var nodes []ResourceNode
	for i := range podList.Items {
		pod := &podList.Items[i]
		if !hasOwnerWithUID(pod.OwnerReferences, uid) {
			continue
		}
		nodes = append(nodes, podToNode(pod))
	}

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
	return nodes
}

// walkStatefulSet returns Pod nodes directly owned by a StatefulSet (no RS layer).
func walkStatefulSet(ctx context.Context, client *Client, res *unstructured.Unstructured) []ResourceNode {
	return walkPodsOwnedBy(ctx, client, res.GetNamespace(), res.GetUID(),
		"statefulset", res.GetName())
}

// walkDaemonSet returns Pod nodes owned by a DaemonSet.
func walkDaemonSet(ctx context.Context, client *Client, res *unstructured.Unstructured) []ResourceNode {
	return walkPodsOwnedBy(ctx, client, res.GetNamespace(), res.GetUID(),
		"daemonset", res.GetName())
}

// walkJob returns Pod nodes owned by a Job.
func walkJob(ctx context.Context, client *Client, res *unstructured.Unstructured) []ResourceNode {
	return walkPodsOwnedBy(ctx, client, res.GetNamespace(), res.GetUID(),
		"job", res.GetName())
}

// walkPodsOwnedBy lists all Pods in ns and returns those owned by the given uid.
func walkPodsOwnedBy(ctx context.Context, client *Client, ns string, uid types.UID, kind, name string) []ResourceNode {
	podList, err := client.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		output.Debug("failed to list pods", kind, name, "namespace", ns, "error", err)
		return nil
	}

	var nodes []ResourceNode
	for i := range podList.Items {
		pod := &podList.Items[i]
		if !hasOwnerWithUID(pod.OwnerReferences, uid) {
			continue
		}
		nodes = append(nodes, podToNode(pod))
	}

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
	return nodes
}

// hasOwnerWithUID reports whether any OwnerReference in refs has the given UID.
func hasOwnerWithUID(refs []metav1.OwnerReference, uid types.UID) bool {
	for _, ref := range refs {
		if ref.UID == uid {
			return true
		}
	}
	return false
}

// replicaSetHealth returns HealthReady when all replicas are ready (or the RS is scaled to 0).
func replicaSetHealth(rs *appsv1.ReplicaSet) HealthStatus {
	if rs.Status.Replicas == 0 {
		return HealthReady
	}
	if rs.Status.ReadyReplicas >= rs.Status.Replicas {
		return HealthReady
	}
	return HealthNotReady
}

// podToNode converts a corev1.Pod to a ResourceNode for tree display.
// It delegates to extractPodInfoFromPod (pods.go) so that waiting-reason
// overrides (e.g. CrashLoopBackOff → "CrashLoop") and the raw K8s phase are
// both preserved — matching the display convention of `mod status`.
func podToNode(pod *corev1.Pod) ResourceNode {
	info := extractPodInfoFromPod(pod)
	return ResourceNode{
		Kind:      "Pod",
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Status:    HealthStatus(info.Phase),
		Ready:     info.Ready,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Rendering — table (terminal tree)
// ─────────────────────────────────────────────────────────────────────────────

// treeMinColGap is the minimum number of spaces between adjacent columns.
const treeMinColGap = 2

// treeRow is a single rendered line in the terminal tree.
//
// Rows come in two flavours:
//   - isResource=false: literal lines (header, │ separators, component names).
//     These are emitted verbatim; they do not participate in column measurement.
//   - isResource=true: columnar resource lines. col1/col2/col3 are kept separate
//     so that a second pass can measure the global maximum widths and align all
//     three columns uniformly across every depth level.
//
// All width fields store the visual (ANSI-free) character count so that color
// escape codes never disturb alignment arithmetic.
type treeRow struct {
	// Literal line — used when isResource=false.
	literal string

	// Column 1: tree chrome (dim connectors) + "Kind/Name".
	col1      string
	col1Width int

	// Column 2: status token ("Ready", "ContainerCreating", "2 pods", …).
	col2      string
	col2Width int

	// Column 3: extra annotation ("3/3", "15Gi"). Empty for most rows.
	col3 string

	// isResource marks columnar rows that participate in global alignment.
	isResource bool
}

// FormatTree formats a TreeResult according to the requested output format.
// Color stripping for non-TTY environments is handled automatically: lipgloss
// uses termenv.ColorProfile() to detect the terminal capability and falls back
// to termenv.Ascii (no color) when stdout is a pipe or CI environment. No
// explicit TTY check is required here.
func FormatTree(result *TreeResult, format output.Format) (string, error) {
	switch format {
	case output.FormatJSON:
		return FormatTreeJSON(result)
	case output.FormatYAML:
		return FormatTreeYAML(result)
	case output.FormatTable, output.FormatWide, output.FormatDir:
		// FormatWide and FormatDir are rejected by the command layer; FormatTable is the default.
		return formatTreeTable(result, true), nil
	default:
		return formatTreeTable(result, true), nil
	}
}

// formatTreeTable renders the tree as a terminal-friendly string.
// colored=true applies ANSI color via the output package helpers.
func formatTreeTable(result *TreeResult, colored bool) string {
	var sb strings.Builder

	// ── Release header ────────────────────────────────────────────────────────
	header := result.Release.Name
	if result.Release.Module != "" {
		mod := result.Release.Module
		if result.Release.Version != "" {
			mod += "@" + result.Release.Version
		}
		header += " (" + mod + ")"
	}
	sb.WriteString(header + "\n")

	if len(result.Components) == 0 {
		return sb.String()
	}

	// ── Detect depth=0 (no resources loaded into components) ─────────────────
	depth0 := true
	for _, c := range result.Components {
		if len(c.Resources) > 0 {
			depth0 = false
			break
		}
	}

	if depth0 {
		renderDepth0Components(&sb, result.Components, colored)
		return sb.String()
	}

	// ── Depth>=1: collect rows, then format with globally aligned columns ─────
	rows := collectTreeRows(result, colored)
	sb.WriteString(formatTreeRows(rows))
	return sb.String()
}

// collectTreeRows walks the TreeResult and produces a flat slice of treeRows.
// Literal rows (separator pipes, component headings) are interspersed with
// columnar resource rows. The caller passes the result to formatTreeRows which
// measures global column widths and emits the final string.
func collectTreeRows(result *TreeResult, colored bool) []treeRow {
	pipe := "│"
	if colored {
		pipe = output.Dim(pipe)
	}

	var rows []treeRow
	for i, comp := range result.Components {
		isLast := i == len(result.Components)-1

		// ── │ separator between components ────────────────────────────────────
		rows = append(rows, treeRow{literal: pipe + "\n"})

		// ── Component heading ─────────────────────────────────────────────────
		conn := treeConnMid
		if isLast {
			conn = treeConnLast
		}
		chrome := conn
		name := comp.Name
		if colored {
			chrome = output.Dim(chrome)
			name = output.StyleNoun(name)
		}
		rows = append(rows, treeRow{literal: chrome + name + "\n"})

		// ── Resource rows ─────────────────────────────────────────────────────
		childPrefix := "│   "
		if isLast {
			childPrefix = "    "
		}
		for j, res := range comp.Resources {
			resIsLast := j == len(comp.Resources)-1
			rows = append(rows, collectResourceRows(res, childPrefix, resIsLast, false, colored)...)
		}
	}
	return rows
}

// collectResourceRows produces treeRows for a single ResourceNode and its
// children. isChild=true dims the kind/name and replicas (K8s-owned nodes).
func collectResourceRows(node ResourceNode, prefix string, isLast, isChild, colored bool) []treeRow {
	conn := treeConnMid
	if isLast {
		conn = treeConnLast
	}

	// Raw (uncolored) values for width measurement — must be captured before
	// any ANSI escape codes are applied.
	kindName := displayKind(node.Kind) + "/" + node.Name
	rawStatus := string(node.Status)
	rawReplicas := node.Replicas
	rawChrome := prefix + conn

	// col1 raw width: chrome + kindName (connector is always 4 chars).
	col1Width := len(rawChrome) + len(kindName)

	// Apply ANSI coloring after all widths are locked in.
	chrome := rawChrome
	kindNameColored := kindName
	statusColored := rawStatus
	replicasColored := rawReplicas
	if colored {
		chrome = output.Dim(chrome)
		if isChild {
			kindNameColored = output.Dim(kindName)
			if rawReplicas != "" {
				replicasColored = output.Dim(rawReplicas)
			}
		}
		if node.Kind == "Pod" {
			statusColored = output.FormatPodPhase(rawStatus, node.Ready)
		} else {
			statusColored = output.FormatHealthStatus(rawStatus)
		}
	}

	col1 := chrome + kindNameColored

	// Assign col2 and col3 based on resource kind.
	//   ReplicaSet → col2=replicas ("2 pods"),  col3=""
	//   Pod        → col2=phase,                col3=""
	//   Everything → col2=status,               col3=replicas (if any)
	var col2, col3 string
	var col2Width int
	switch {
	case node.Kind == kindReplicaSet:
		col2 = replicasColored
		col2Width = len(rawReplicas)
	case node.Kind == "Pod":
		col2 = statusColored
		col2Width = len(rawStatus)
	default:
		col2 = statusColored
		col2Width = len(rawStatus)
		col3 = replicasColored
	}

	rows := []treeRow{{
		col1:       col1,
		col1Width:  col1Width,
		col2:       col2,
		col2Width:  col2Width,
		col3:       col3,
		isResource: true,
	}}

	// Recurse into K8s-owned children (isChild=true from here down).
	childPrefix := prefix + "│   "
	if isLast {
		childPrefix = prefix + "    "
	}
	for i, child := range node.Children {
		childIsLast := i == len(node.Children)-1
		rows = append(rows, collectResourceRows(child, childPrefix, childIsLast, true, colored)...)
	}
	return rows
}

// formatTreeRows takes the flat row slice from collectTreeRows, measures global
// maximum column widths, and emits the final aligned string.
//
// Column layout (for isResource=true rows):
//
//	col1 <gap> col2 <gap> col3
//
// col1 is right-padded to the global maximum so col2 starts at a fixed position.
// col2 is right-padded to the global maximum (only when col3 is non-empty on any
// row) so col3 starts at a fixed position — left-aligned like col2.
// Rows with no col3 are not padded after col2 (avoids trailing whitespace).
func formatTreeRows(rows []treeRow) string {
	// ── Pass 1: measure global maxima ────────────────────────────────────────
	var maxCol1, maxCol2 int
	hasCol3 := false
	for _, r := range rows {
		if !r.isResource {
			continue
		}
		if r.col1Width > maxCol1 {
			maxCol1 = r.col1Width
		}
		if r.col2Width > maxCol2 {
			maxCol2 = r.col2Width
		}
		if r.col3 != "" {
			hasCol3 = true
		}
	}

	// ── Pass 2: emit ─────────────────────────────────────────────────────────
	var sb strings.Builder
	for _, r := range rows {
		if !r.isResource {
			sb.WriteString(r.literal)
			continue
		}

		col1Pad := maxCol1 - r.col1Width + treeMinColGap
		line := r.col1 + strings.Repeat(" ", col1Pad) + r.col2
		if r.col3 != "" {
			col2Pad := maxCol2 - r.col2Width + treeMinColGap
			line += strings.Repeat(" ", col2Pad) + r.col3
		} else if hasCol3 {
			// col2 is the last token on this line — no trailing spaces needed.
			_ = hasCol3
		}
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

// renderDepth0Components renders a flat summary list (depth=0 view).
func renderDepth0Components(sb *strings.Builder, components []Component, colored bool) {
	for i, comp := range components {
		isLast := i == len(components)-1
		conn := treeConnMid
		if isLast {
			conn = treeConnLast
		}

		chrome := conn
		name := comp.Name
		status := string(comp.Status)
		resourceWord := "resources"
		if comp.ResourceCount == 1 {
			resourceWord = "resource"
		}

		if colored {
			chrome = output.Dim(chrome)
			name = output.StyleNoun(name)
			status = output.FormatHealthStatus(status)
		}

		fmt.Fprintf(sb, "%s%s   %d %s   %s\n",
			chrome, name, comp.ResourceCount, resourceWord, status)
	}
}

// formatPlainTree renders the tree without any ANSI color codes.
// Primarily used in unit tests to assert tree structure without dealing with
// escape sequences. In production, color stripping is handled automatically
// by termenv (see FormatTree); this function is not called on the hot path.
func formatPlainTree(result *TreeResult) string {
	return formatTreeTable(result, false)
}

// ─────────────────────────────────────────────────────────────────────────────
// Structured output (JSON / YAML)
// ─────────────────────────────────────────────────────────────────────────────

// FormatTreeJSON serializes a TreeResult to indented JSON.
func FormatTreeJSON(result *TreeResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling tree to JSON: %w", err)
	}
	return string(data), nil
}

// FormatTreeYAML serializes a TreeResult to YAML.
func FormatTreeYAML(result *TreeResult) (string, error) {
	data, err := yaml.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshaling tree to YAML: %w", err)
	}
	return string(data), nil
}
