package platform

import (
	"fmt"

	"github.com/open-platform-model/library/opm/schema"
)

// CoreRepinHint is the remediation for a platform module pinning a core
// older than a field the kernel reads: the exact command re-pinning core in
// dir to the release the kernel was verified against. That release, not the
// floor that fired first, is named because it satisfies every floor at once.
func CoreRepinHint(dir string) string {
	return fmt.Sprintf("the platform module at %s pins a core release older than this opm reads: run 'cue mod get opmodel.dev/core@%s' in %s",
		dir, schema.DefaultSchemaVersion(), dir)
}
