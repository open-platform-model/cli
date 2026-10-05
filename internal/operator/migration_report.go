package operator

import "fmt"

// MigrationReport is what install prints about its migration: a heading,
// then each adopted object, the recreated Deployment, each deleted binding
// with the binding that replaces it, and each proven earlier-manifest object
// left in place. It is empty unless the plan adopts, recreates or deletes
// something, so an install after a completed migration prints nothing even
// when leftovers remain.
func MigrationReport(p *MigrationPlan) []string {
	if !p.Migrates() {
		return nil
	}
	lines := []string{"migrating the operator installed from an earlier release manifest"}
	for _, obj := range p.Adopt {
		lines = append(lines, "  adopted   "+objPath(obj.GetKind(), obj.GetNamespace(), obj.GetName()))
	}
	if d := p.RecreateDeployment; d != nil {
		lines = append(lines, fmt.Sprintf("  recreated %s: selector changed; patches made to the earlier Deployment are not carried over",
			objPath(d.GetKind(), d.GetNamespace(), d.GetName())))
	}
	for _, b := range p.DeleteBindings {
		lines = append(lines, fmt.Sprintf("  deleted   %s: superseded by %s",
			objPath(b.Live.GetKind(), b.Live.GetNamespace(), b.Live.GetName()), b.Replacement))
	}
	for _, o := range p.LeftInPlace {
		lines = append(lines, fmt.Sprintf("  left      %s: not part of the operator module", objPath(o.Kind, o.Namespace, o.Name)))
	}
	return lines
}
