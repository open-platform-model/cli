package operator

// LegacyObject is one object an earlier opm-operator release's install
// manifest created, as the migration proves it: its identity, the labels
// every manifest that shipped it set (none for some), and, for the
// controller Deployment, its pod selector. It carries no spec.
type LegacyObject struct {
	Group, Kind, Namespace, Name string
	// Labels are the labels every manifest that shipped the object set; a
	// live object must carry each of them to be proven. May be empty.
	Labels map[string]string
	// Selector is the Deployment's spec.selector.matchLabels; nil otherwise.
	Selector map[string]string
	// Releases is the first and last operator release that shipped it.
	Releases string
}

// FirstLegacyRelease is the first operator release whose manifest uses the
// names the operator module keeps (the opm-operator prefix, in
// OperatorNamespace). The releases before it that attach a manifest
// (v0.4.2 to v0.4.4) install a poc-controller in poc-controller-system, a
// different operator the migration does not take over.
const FirstLegacyRelease = "v0.5.0"

var (
	kustomizeLabels = map[string]string{
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/name":       "opm-operator",
	}
	kustomizeControlPlaneLabels = map[string]string{
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/name":       "opm-operator",
		"control-plane":                "controller-manager",
	}
)

// LegacyObjects is the proof list: the union of the objects of every
// opm-operator release (tag v<semver>) from FirstLegacyRelease on that
// published an install manifest. An operator module release
// (opm_operator-vX.Y.Z) is never a source: its manifest is a render of the
// module, whose objects carry the operator instance's identity. The test
// checks it against testdata/legacy-manifests.json, which
// hack/operator-legacy writes; an operator release that still attaches a
// manifest is added by re-running that program.
var LegacyObjects = []LegacyObject{
	{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "bundlereleases.releases.opmodel.dev", Releases: "v0.5.0..v0.6.4"},
	{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "moduleinstances.opmodel.dev", Releases: "v1.0.0-alpha..v1.0.0-beta.8"},
	{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "modulepackages.opmodel.dev", Releases: "v1.0.0-alpha..v1.0.0-beta.8"},
	{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "modulereleases.releases.opmodel.dev", Releases: "v0.5.0..v0.7.5"},
	{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "platforms.opmodel.dev", Releases: "v1.0.0-alpha..v1.0.0-beta.8"},
	{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "platforms.releases.opmodel.dev", Releases: "v0.7.0..v0.7.5"},
	{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "releases.releases.opmodel.dev", Releases: "v0.5.0..v0.7.5"},
	{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "transformerregistrations.opmodel.dev", Releases: "v1.0.0-alpha.18..v1.0.0-beta.8"},
	{Group: "", Kind: "Namespace", Name: OperatorNamespace, Labels: kustomizeControlPlaneLabels, Releases: "v0.5.0..v1.0.0-beta.8"},
	{Group: "", Kind: "ServiceAccount", Namespace: OperatorNamespace, Name: "opm-operator-controller-manager", Labels: kustomizeLabels, Releases: "v0.5.0..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "Role", Namespace: OperatorNamespace, Name: "opm-operator-leader-election-role", Labels: kustomizeLabels, Releases: "v0.5.0..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-bundlerelease-admin-role", Labels: kustomizeLabels, Releases: "v0.5.0..v0.6.4"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-bundlerelease-editor-role", Labels: kustomizeLabels, Releases: "v0.5.0..v0.6.4"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-bundlerelease-viewer-role", Labels: kustomizeLabels, Releases: "v0.5.0..v0.6.4"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-manager-role", Releases: "v0.5.0..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-metrics-auth-role", Releases: "v0.5.0..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-metrics-reader", Releases: "v0.5.0..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-moduleinstance-admin-role", Labels: kustomizeLabels, Releases: "v1.0.0-alpha..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-moduleinstance-editor-role", Labels: kustomizeLabels, Releases: "v1.0.0-alpha..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-moduleinstance-viewer-role", Labels: kustomizeLabels, Releases: "v1.0.0-alpha..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-modulepackage-viewer-role", Labels: kustomizeLabels, Releases: "v1.0.0-beta.8..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-modulerelease-admin-role", Labels: kustomizeLabels, Releases: "v0.5.0..v0.7.5"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-modulerelease-editor-role", Labels: kustomizeLabels, Releases: "v0.5.0..v0.7.5"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-modulerelease-viewer-role", Labels: kustomizeLabels, Releases: "v0.5.0..v0.7.5"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-platform-viewer-role", Labels: kustomizeLabels, Releases: "v1.0.0-beta.8..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-transformerregistration-admin-role", Labels: kustomizeLabels, Releases: "v1.0.0-alpha.18..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-transformerregistration-viewer-role", Labels: kustomizeLabels, Releases: "v1.0.0-beta.8..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding", Namespace: OperatorNamespace, Name: "opm-operator-leader-election-rolebinding", Labels: kustomizeLabels, Releases: "v0.5.0..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRoleBinding", Name: "opm-operator-manager-rolebinding", Labels: kustomizeLabels, Releases: "v0.5.0..v1.0.0-beta.8"},
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRoleBinding", Name: "opm-operator-metrics-auth-rolebinding", Releases: "v0.5.0..v1.0.0-beta.8"},
	{Group: "", Kind: "Service", Namespace: OperatorNamespace, Name: "opm-operator-controller-manager-metrics-service", Labels: kustomizeControlPlaneLabels, Releases: "v0.5.0..v1.0.0-beta.8"},
	{Group: "apps", Kind: "Deployment", Namespace: OperatorNamespace, Name: ControllerDeploymentName, Labels: kustomizeControlPlaneLabels, Selector: map[string]string{"app.kubernetes.io/name": "opm-operator", "control-plane": "controller-manager"}, Releases: "v0.5.0..v1.0.0-beta.8"},
}

// SupersededBindings are the earlier manifests' role bindings the operator
// module replaces under other names. The replacement is found in the
// render by roleRef, never by a name kept here.
var SupersededBindings = []LegacyObject{
	legacyObject("ClusterRoleBinding", "", "opm-operator-manager-rolebinding"),
	legacyObject("ClusterRoleBinding", "", "opm-operator-metrics-auth-rolebinding"),
	legacyObject("RoleBinding", OperatorNamespace, "opm-operator-leader-election-rolebinding"),
}

// legacyObject returns the LegacyObjects entry of a kind, namespace and
// name; it panics on a name the list does not hold, at package init.
func legacyObject(kind, namespace, name string) LegacyObject {
	for _, o := range LegacyObjects {
		if o.Kind == kind && o.Namespace == namespace && o.Name == name {
			return o
		}
	}
	panic("operator: no legacy object " + kind + " " + namespace + "/" + name)
}

// legacyDeployment is the earlier manifests' controller Deployment.
var legacyDeployment = legacyObject(kindDeployment, OperatorNamespace, ControllerDeploymentName)
