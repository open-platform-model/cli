//go:build ignore

// Integration test for opm rel tree — tree building with real cluster resources.
//
// Tests covered:
//   - 14.1: Deploy test module with multiple components (Deployment, StatefulSet, Services, ConfigMaps)
//   - 14.2: Verify tree output at depth=0 (component summary)
//   - 14.3: Verify tree output at depth=1 (resources only, no children)
//   - 14.4: Verify tree output at depth=2 (Deployment→RS→Pod visible)
//   - 14.5: Verify JSON output structure matches schema
//   - 14.6: Verify no-release-found error (empty InventoryLive)
//   - 14.7: Verify component grouping with real resources
//   - 14.8: Verify StatefulSet→Pod chain at depth=2
//
// Requires a running kind cluster at context "kind-opm-dev".
// Run with: go run tests/integration/rel-tree/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

const (
	clusterContext = "kind-opm-dev"
	namespace      = "opm-tree-test"

	releaseName = "tree-test-rel"
	releaseID   = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeee0001"

	moduleName    = "tree-test-module"
	moduleVersion = "1.0.0"
	modulePath    = "tests/integration/rel-tree"
)

var (
	configMapGVR   = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	serviceGVR     = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	deploymentGVR  = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	statefulSetGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
)

func main() {
	ctx := context.Background()

	fmt.Println("=== OPM Release Tree Integration Test ===")
	fmt.Println()

	// ── Create Kubernetes client ─────────────────────────────────────────────
	client, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Context: clusterContext,
	})
	check("creating Kubernetes client", err)
	fmt.Println("OK: client created")
	fmt.Println()

	// ── Clean state ──────────────────────────────────────────────────────────
	cleanup(ctx, client)

	// ── Step 1: Deploy test module (14.1) ────────────────────────────────────
	step(1, "14.1: Deploy test module with multiple components")

	_, err = client.EnsureNamespace(ctx, namespace, false)
	check("ensuring test namespace", err)
	fmt.Println("   OK: namespace ensured")

	resources := buildAllResources()

	result, err := kubernetes.Apply(ctx, client, resources, releaseName, kubernetes.ApplyOptions{})
	check("applying resources", err)
	if len(result.Errors) > 0 {
		failf("apply errors: %v", result.Errors[0])
	}
	fmt.Printf("   OK: %d resources applied\n", len(resources))

	inv := buildInventory(resources)
	err = inventory.WriteInventory(ctx, client, inv, moduleName, "")
	check("writing inventory", err)
	fmt.Println("   OK: inventory written")

	// Wait for workloads to be ready (Pods scheduled and running).
	fmt.Println("   Waiting for workloads to become ready...")
	waitForDeploymentReady(ctx, client, namespace, "tree-server")
	waitForStatefulSetReady(ctx, client, namespace, "tree-db")
	fmt.Println("   OK: all workloads ready")

	// Read back inventory and discover live resources.
	readInv, err := inventory.GetInventory(ctx, client, releaseName, namespace, releaseID)
	check("reading back inventory", err)
	liveResources, missingEntries, err := inventory.DiscoverResourcesFromInventory(ctx, client, readInv)
	check("discovering resources from inventory", err)
	if len(missingEntries) != 0 {
		failf("expected 0 missing resources, got %d", len(missingEntries))
	}
	if len(liveResources) != len(resources) {
		failf("expected %d live resources, got %d", len(resources), len(liveResources))
	}
	fmt.Printf("   OK: %d live resources discovered from inventory\n", len(liveResources))

	// Build ComponentMap from inventory entries (mirrors command layer logic).
	entries := readInv.Changes[readInv.Index[0]].Inventory.Entries
	componentMap := make(map[string]string, len(entries))
	for _, e := range entries {
		key := e.Kind + "/" + e.Namespace + "/" + e.Name
		componentMap[key] = e.Component
	}

	releaseInfo := kubernetes.ReleaseInfo{
		Name:      releaseName,
		Namespace: namespace,
		Module:    moduleName,
		Version:   moduleVersion,
	}

	// ── Step 2: No-release-found error (14.6) ───────────────────────────────
	step(2, "14.6: Verify no-release-found error with empty InventoryLive")

	_, errEmpty := kubernetes.GetModuleTree(ctx, client, kubernetes.TreeOptions{
		ReleaseInfo:   releaseInfo,
		InventoryLive: nil,
		ComponentMap:  componentMap,
		Depth:         2,
		OutputFormat:  output.FormatTable,
	})
	if errEmpty == nil {
		failf("expected error from GetModuleTree with empty InventoryLive, got nil")
	}
	if !kubernetes.IsNoResourcesFound(errEmpty) {
		failf("expected IsNoResourcesFound=true, got error: %v", errEmpty)
	}
	fmt.Println("   OK: GetModuleTree returns noResourcesFoundError for empty InventoryLive")

	// ── Step 3: Depth=0 component summary (14.2) ────────────────────────────
	step(3, "14.2: Verify tree output at depth=0 (component summary)")

	tree0, err := kubernetes.GetModuleTree(ctx, client, kubernetes.TreeOptions{
		ReleaseInfo:   releaseInfo,
		InventoryLive: liveResources,
		ComponentMap:  componentMap,
		Depth:         0,
		OutputFormat:  output.FormatTable,
	})
	check("GetModuleTree depth=0", err)

	if len(tree0.Components) != 3 {
		failf("depth=0: expected 3 components, got %d", len(tree0.Components))
	}

	// Verify component ordering: database, server, (no component)
	if tree0.Components[0].Name != "database" {
		failf("depth=0: expected first component 'database', got %q", tree0.Components[0].Name)
	}
	if tree0.Components[1].Name != "server" {
		failf("depth=0: expected second component 'server', got %q", tree0.Components[1].Name)
	}
	if tree0.Components[2].Name != "(no component)" {
		failf("depth=0: expected third component '(no component)', got %q", tree0.Components[2].Name)
	}
	fmt.Println("   OK: 3 components in correct order: database, server, (no component)")

	// Verify resource counts.
	if tree0.Components[0].ResourceCount != 2 {
		failf("depth=0: expected 'database' to have 2 resources, got %d", tree0.Components[0].ResourceCount)
	}
	if tree0.Components[1].ResourceCount != 3 {
		failf("depth=0: expected 'server' to have 3 resources, got %d", tree0.Components[1].ResourceCount)
	}
	if tree0.Components[2].ResourceCount != 1 {
		failf("depth=0: expected '(no component)' to have 1 resource, got %d", tree0.Components[2].ResourceCount)
	}
	fmt.Println("   OK: resource counts correct: database=2, server=3, (no component)=1")

	// Verify no resources loaded (depth=0 suppresses them).
	for _, comp := range tree0.Components {
		if len(comp.Resources) != 0 {
			failf("depth=0: expected 0 resources on component %q, got %d", comp.Name, len(comp.Resources))
		}
	}
	fmt.Println("   OK: no resource details at depth=0 (summary only)")

	// Verify aggregate status is Unknown at depth=0 (resources weren't evaluated).
	for _, comp := range tree0.Components {
		if comp.Status != kubernetes.HealthUnknown {
			failf("depth=0: expected Unknown status for component %q, got %q", comp.Name, comp.Status)
		}
	}
	fmt.Println("   OK: all component statuses are Unknown at depth=0")

	// ── Step 4: Depth=1 resources only (14.3) ───────────────────────────────
	step(4, "14.3: Verify tree output at depth=1 (resources only, no children)")

	tree1, err := kubernetes.GetModuleTree(ctx, client, kubernetes.TreeOptions{
		ReleaseInfo:   releaseInfo,
		InventoryLive: liveResources,
		ComponentMap:  componentMap,
		Depth:         1,
		OutputFormat:  output.FormatTable,
	})
	check("GetModuleTree depth=1", err)

	// Verify resources are present in each component.
	for _, comp := range tree1.Components {
		if len(comp.Resources) == 0 && comp.ResourceCount > 0 {
			failf("depth=1: component %q has ResourceCount=%d but 0 resources loaded",
				comp.Name, comp.ResourceCount)
		}
	}
	fmt.Println("   OK: all components have resources loaded at depth=1")

	// Verify NO resource has children at depth=1 (ownership walking is skipped).
	for _, comp := range tree1.Components {
		for _, res := range comp.Resources {
			if len(res.Children) != 0 {
				failf("depth=1: resource %s/%s has %d children (expected 0)",
					res.Kind, res.Name, len(res.Children))
			}
		}
	}
	fmt.Println("   OK: no children on any resource at depth=1 (ownership walking skipped)")

	// Verify server component has correct resource kinds.
	serverComp := findComponent(tree1, "server")
	if serverComp == nil {
		failf("depth=1: server component not found")
	}
	serverKinds := resourceKinds(serverComp.Resources)
	assertContains("depth=1 server", serverKinds, "Deployment")
	assertContains("depth=1 server", serverKinds, "Service")
	assertContains("depth=1 server", serverKinds, "ConfigMap")
	fmt.Println("   OK: server component has Deployment, Service, ConfigMap")

	// Verify database component has correct resource kinds.
	dbComp := findComponent(tree1, "database")
	if dbComp == nil {
		failf("depth=1: database component not found")
	}
	dbKinds := resourceKinds(dbComp.Resources)
	assertContains("depth=1 database", dbKinds, "StatefulSet")
	assertContains("depth=1 database", dbKinds, "Service")
	fmt.Println("   OK: database component has StatefulSet, Service")

	// Verify (no component) has the orphan ConfigMap.
	noComp := findComponent(tree1, "(no component)")
	if noComp == nil {
		failf("depth=1: (no component) not found")
	}
	if len(noComp.Resources) != 1 || noComp.Resources[0].Kind != "ConfigMap" {
		failf("depth=1: expected (no component) to have 1 ConfigMap, got %d resources",
			len(noComp.Resources))
	}
	fmt.Println("   OK: (no component) has orphan ConfigMap")

	// ── Step 5: Depth=2 full tree + component grouping (14.4 + 14.7) ────────
	step(5, "14.4 + 14.7: Verify tree output at depth=2 (full tree) and component grouping")

	tree2, err := kubernetes.GetModuleTree(ctx, client, kubernetes.TreeOptions{
		ReleaseInfo:   releaseInfo,
		InventoryLive: liveResources,
		ComponentMap:  componentMap,
		Depth:         2,
		OutputFormat:  output.FormatTable,
	})
	check("GetModuleTree depth=2", err)

	// 14.7: Verify component grouping — alphabetical with (no component) last.
	if len(tree2.Components) != 3 {
		failf("depth=2: expected 3 components, got %d", len(tree2.Components))
	}
	if tree2.Components[0].Name != "database" {
		failf("depth=2: expected first component 'database', got %q", tree2.Components[0].Name)
	}
	if tree2.Components[1].Name != "server" {
		failf("depth=2: expected second component 'server', got %q", tree2.Components[1].Name)
	}
	if tree2.Components[2].Name != "(no component)" {
		failf("depth=2: expected third component '(no component)', got %q", tree2.Components[2].Name)
	}
	fmt.Println("   OK: 14.7 — components correctly ordered: database, server, (no component)")

	// 14.4: Find Deployment/tree-server and verify it has RS→Pod children.
	serverComp2 := findComponent(tree2, "server")
	if serverComp2 == nil {
		failf("depth=2: server component not found")
	}
	deploy := findResource(serverComp2.Resources, "Deployment", "tree-server")
	if deploy == nil {
		failf("depth=2: Deployment/tree-server not found in server component")
	}
	if len(deploy.Children) == 0 {
		failf("depth=2: Deployment/tree-server has 0 children (expected ReplicaSet(s))")
	}
	fmt.Printf("   OK: Deployment/tree-server has %d ReplicaSet children\n", len(deploy.Children))

	// Find the active ReplicaSet (the one with pods).
	var activeRS *kubernetes.ResourceNode
	for i := range deploy.Children {
		rs := &deploy.Children[i]
		if rs.Kind != "ReplicaSet" {
			failf("depth=2: expected child of Deployment to be ReplicaSet, got %q", rs.Kind)
		}
		if len(rs.Children) > 0 {
			activeRS = rs
		}
	}
	if activeRS == nil {
		failf("depth=2: no active ReplicaSet found with Pod children")
	}
	fmt.Printf("   OK: ReplicaSet/%s has %d Pod children\n", activeRS.Name, len(activeRS.Children))

	// Verify pods are actual Pod kind.
	for _, pod := range activeRS.Children {
		if pod.Kind != "Pod" {
			failf("depth=2: expected Pod child, got %q", pod.Kind)
		}
	}
	fmt.Println("   OK: 14.4 — Deployment→ReplicaSet→Pod chain verified")

	// ── Step 6: StatefulSet→Pod chain (14.8) ─────────────────────────────────
	step(6, "14.8: Verify StatefulSet→Pod chain at depth=2 (no RS layer)")

	dbComp2 := findComponent(tree2, "database")
	if dbComp2 == nil {
		failf("depth=2: database component not found")
	}
	sts := findResource(dbComp2.Resources, "StatefulSet", "tree-db")
	if sts == nil {
		failf("depth=2: StatefulSet/tree-db not found in database component")
	}
	if len(sts.Children) == 0 {
		failf("depth=2: StatefulSet/tree-db has 0 children (expected Pod(s))")
	}
	fmt.Printf("   OK: StatefulSet/tree-db has %d Pod children\n", len(sts.Children))

	// Verify children are Pods directly (no RS layer for StatefulSet).
	for _, child := range sts.Children {
		if child.Kind != "Pod" {
			failf("depth=2: expected StatefulSet child to be Pod, got %q", child.Kind)
		}
	}
	fmt.Println("   OK: 14.8 — StatefulSet→Pod chain verified (no ReplicaSet layer)")

	// ── Step 7: JSON output validation (14.5) ────────────────────────────────
	step(7, "14.5: Verify JSON output structure matches schema")

	jsonStr, err := kubernetes.FormatTreeJSON(tree2)
	check("FormatTreeJSON", err)

	var jsonData map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &jsonData)
	check("parsing JSON output", err)

	// Verify top-level keys.
	if _, ok := jsonData["release"]; !ok {
		failf("JSON: missing 'release' key")
	}
	if _, ok := jsonData["components"]; !ok {
		failf("JSON: missing 'components' key")
	}
	fmt.Println("   OK: JSON has 'release' and 'components' keys")

	// Verify release metadata.
	release, ok := jsonData["release"].(map[string]interface{})
	if !ok {
		failf("JSON: 'release' is not an object")
	}
	if release["name"] != releaseName {
		failf("JSON: expected release.name=%q, got %q", releaseName, release["name"])
	}
	if release["namespace"] != namespace {
		failf("JSON: expected release.namespace=%q, got %q", namespace, release["namespace"])
	}
	fmt.Println("   OK: release metadata correct in JSON")

	// Verify components array.
	components, ok := jsonData["components"].([]interface{})
	if !ok {
		failf("JSON: 'components' is not an array")
	}
	if len(components) != 3 {
		failf("JSON: expected 3 components, got %d", len(components))
	}
	fmt.Println("   OK: JSON has 3 components")

	// Verify nested children on the Deployment.
	found := false
	for _, c := range components {
		comp, _ := c.(map[string]interface{})
		if comp["name"] != "server" {
			continue
		}
		resources, _ := comp["resources"].([]interface{})
		for _, r := range resources {
			res, _ := r.(map[string]interface{})
			if res["kind"] == "Deployment" {
				children, hasChildren := res["children"].([]interface{})
				if !hasChildren || len(children) == 0 {
					failf("JSON: Deployment resource missing 'children' array")
				}
				found = true
				fmt.Printf("   OK: Deployment has %d children in JSON\n", len(children))

				// Check RS children have nested Pod children.
				for _, ch := range children {
					rs, _ := ch.(map[string]interface{})
					if pods, hasPods := rs["children"].([]interface{}); hasPods && len(pods) > 0 {
						fmt.Printf("   OK: ReplicaSet has %d Pod children in JSON\n", len(pods))
					}
				}
			}
		}
	}
	if !found {
		failf("JSON: Deployment not found in server component")
	}
	fmt.Println("   OK: 14.5 — JSON structure validated with nested children")

	// ── Cleanup ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("Cleaning up...")
	cleanup(ctx, client)
	fmt.Println("OK: cleanup complete")

	fmt.Println()
	fmt.Println("=== ALL SCENARIOS PASSED ===")
}

// ─────────────────────────────────────────────────────────────────────────────
// Resource builders
// ─────────────────────────────────────────────────────────────────────────────

// opmLabels returns the standard OPM labels for test resources.
func opmLabels(component string) map[string]interface{} {
	labels := map[string]interface{}{
		"app.kubernetes.io/managed-by":    "open-platform-model",
		"module-release.opmodel.dev/name": releaseName,
		"module-release.opmodel.dev/uuid": releaseID,
		"module.opmodel.dev/name":         moduleName,
		"module.opmodel.dev/version":      moduleVersion,
	}
	if component != "" {
		labels["component.opmodel.dev/name"] = component
	}
	return labels
}

// buildAllResources constructs the full set of test resources.
// Order: database component, server component, then orphan (no component).
func buildAllResources() []*unstructured.Unstructured {
	return []*unstructured.Unstructured{
		// database component
		buildStatefulSet("tree-db", "database"),
		buildHeadlessService("tree-db-headless", "database", "tree-db"),
		// server component
		buildDeployment("tree-server", "server"),
		buildService("tree-svc", "server"),
		buildConfigMap("tree-config", "server"),
		// no component (orphan)
		buildConfigMap("tree-orphan", ""),
	}
}

func buildConfigMap(name, component string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels":    opmLabels(component),
		},
		"data": map[string]interface{}{
			"key": fmt.Sprintf("value-%s", name),
		},
	}}
}

func buildService(name, component string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels":    opmLabels(component),
		},
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{
				"app": "tree-server",
			},
			"ports": []interface{}{
				map[string]interface{}{
					"port":       int64(80),
					"targetPort": int64(8080),
					"protocol":   "TCP",
				},
			},
		},
	}}
}

func buildHeadlessService(name, component, stsName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels":    opmLabels(component),
		},
		"spec": map[string]interface{}{
			"clusterIP": "None",
			"selector": map[string]interface{}{
				"app": stsName,
			},
			"ports": []interface{}{
				map[string]interface{}{
					"port":       int64(80),
					"targetPort": int64(80),
					"protocol":   "TCP",
				},
			},
		},
	}}
}

func buildDeployment(name, component string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels":    opmLabels(component),
		},
		"spec": map[string]interface{}{
			"replicas": int64(1),
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app": name,
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app": name,
					},
				},
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "pause",
							"image": "registry.k8s.io/pause:3.9",
						},
					},
				},
			},
		},
	}}
}

func buildStatefulSet(name, component string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "StatefulSet",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels":    opmLabels(component),
		},
		"spec": map[string]interface{}{
			"serviceName": "tree-db-headless",
			"replicas":    int64(1),
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app": name,
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app": name,
					},
				},
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "pause",
							"image": "registry.k8s.io/pause:3.9",
						},
					},
				},
			},
		},
	}}
}

// ─────────────────────────────────────────────────────────────────────────────
// Inventory
// ─────────────────────────────────────────────────────────────────────────────

func buildInventory(resources []*unstructured.Unstructured) *inventory.InventorySecret {
	entries := make([]inventory.InventoryEntry, len(resources))
	for i, r := range resources {
		entries[i] = inventory.NewEntryFromResource(r)
	}

	digest := inventory.ComputeManifestDigest(resources)
	source := inventory.ChangeSource{
		Path:        modulePath,
		Version:     moduleVersion,
		ReleaseName: releaseName,
	}

	inv := &inventory.InventorySecret{
		ReleaseMetadata: inventory.ReleaseMetadata{
			Kind:               "ModuleRelease",
			APIVersion:         "core.opmodel.dev/v1alpha1",
			ReleaseName:        releaseName,
			ReleaseNamespace:   namespace,
			ReleaseID:          releaseID,
			LastTransitionTime: time.Now().UTC().Format(time.RFC3339),
		},
		ModuleMetadata: inventory.ModuleMetadata{
			Kind:       "Module",
			APIVersion: "core.opmodel.dev/v1alpha1",
			Name:       moduleName,
		},
		Index:   []string{},
		Changes: map[string]*inventory.ChangeEntry{},
	}

	changeID, changeEntry := inventory.PrepareChange(source, "", digest, entries)
	inv.Changes[changeID] = changeEntry
	inv.Index = inventory.UpdateIndex(inv.Index, changeID)
	return inv
}

// ─────────────────────────────────────────────────────────────────────────────
// Wait helpers
// ─────────────────────────────────────────────────────────────────────────────

func waitForDeploymentReady(ctx context.Context, client *kubernetes.Client, ns, name string) {
	deadline := time.After(90 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			failf("timed out waiting for Deployment/%s to become ready", name)
		case <-ticker.C:
			deploy, err := client.Clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				continue
			}
			if deploy.Status.ReadyReplicas >= 1 {
				return
			}
		}
	}
}

func waitForStatefulSetReady(ctx context.Context, client *kubernetes.Client, ns, name string) {
	deadline := time.After(90 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			failf("timed out waiting for StatefulSet/%s to become ready", name)
		case <-ticker.C:
			sts, err := client.Clientset.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				continue
			}
			if sts.Status.ReadyReplicas >= 1 {
				return
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tree result helpers
// ─────────────────────────────────────────────────────────────────────────────

func findComponent(tree *kubernetes.TreeResult, name string) *kubernetes.Component {
	for i := range tree.Components {
		if tree.Components[i].Name == name {
			return &tree.Components[i]
		}
	}
	return nil
}

func findResource(resources []kubernetes.ResourceNode, kind, name string) *kubernetes.ResourceNode {
	for i := range resources {
		if resources[i].Kind == kind && resources[i].Name == name {
			return &resources[i]
		}
	}
	return nil
}

func resourceKinds(resources []kubernetes.ResourceNode) []string {
	kinds := make([]string, len(resources))
	for i, r := range resources {
		kinds[i] = r.Kind
	}
	return kinds
}

func assertContains(label string, slice []string, value string) {
	for _, s := range slice {
		if s == value {
			return
		}
	}
	failf("%s: expected %q in %v", label, value, slice)
}

// ─────────────────────────────────────────────────────────────────────────────
// Cleanup
// ─────────────────────────────────────────────────────────────────────────────

func cleanup(ctx context.Context, client *kubernetes.Client) {
	// Delete workloads first (they own pods).
	_ = client.ResourceClient(deploymentGVR, namespace).Delete(ctx, "tree-server", metav1.DeleteOptions{})
	_ = client.ResourceClient(statefulSetGVR, namespace).Delete(ctx, "tree-db", metav1.DeleteOptions{})

	// Delete services.
	for _, name := range []string{"tree-svc", "tree-db-headless"} {
		_ = client.ResourceClient(serviceGVR, namespace).Delete(ctx, name, metav1.DeleteOptions{})
	}

	// Delete configmaps.
	for _, name := range []string{"tree-config", "tree-orphan"} {
		_ = client.ResourceClient(configMapGVR, namespace).Delete(ctx, name, metav1.DeleteOptions{})
	}

	// Delete inventory secret.
	secretName := inventory.SecretName(releaseName, releaseID)
	_ = client.Clientset.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})

	// Delete namespace.
	_ = client.Clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})

	// Wait briefly for namespace deletion to propagate.
	time.Sleep(2 * time.Second)
}

// ─────────────────────────────────────────────────────────────────────────────
// Standard test helpers
// ─────────────────────────────────────────────────────────────────────────────

func step(n int, desc string) {
	fmt.Printf("\n--- Step %d: %s\n", n, desc)
}

func check(label string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %s: %v\n", label, err)
		os.Exit(1)
	}
}

func failf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	os.Exit(1)
}
