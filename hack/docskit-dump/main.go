// Command docskit-dump prints the opm command tree, or with "pins" the
// versions this cli build documents against, as the JSON documents docs-kit
// reads when it builds the cli's docs bundle (docs-kit contracts C14, C15,
// C19). docs-kit.cue runs it in the source tree:
//
//	go run ./hack/docskit-dump        # the docs.opmodel.dev/cobradump/v1 dump
//	go run ./hack/docskit-dump pins   # the docs.opmodel.dev/pins/v1 document
//
// It is not linked into the opm binary.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/open-platform-model/docs-kit/cobradump"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/cmd"
	"github.com/open-platform-model/cli/internal/operator"
)

const (
	libraryModule = "github.com/open-platform-model/library"
	corePrefix    = "opmodel.dev/core@v"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "docskit-dump:", err)
		os.Exit(1)
	}
}

func run(args []string, w io.Writer) error {
	switch {
	case len(args) == 0:
		return cobradump.Write(cmd.NewRootCmd(), w, cobradump.Options{})
	case len(args) == 1 && args[0] == "pins":
		p, err := pins()
		if err != nil {
			return err
		}
		return cobradump.WritePins(w, p)
	default:
		return errors.New("usage: docskit-dump [pins]")
	}
}

// pins returns the versions this build compiles in, each bare SemVer without
// "v": the library module the program links, the core release that
// library's schema loader pins, and the operator release the cli installs.
// These are the values opmodel.dev's resolve-versions.sh read from the cli's
// go.mod, the library's opm/schema/loader.go and internal/operator/manifest.go;
// main_test.go keeps the two readings equal.
func pins() (map[string]string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, errors.New("no build info: run it with go run or go build")
	}
	lib, err := linkedVersion(info, libraryModule)
	if err != nil {
		return nil, err
	}
	core, err := coreRelease(schema.DefaultSchemaModule)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"library":      strings.TrimPrefix(lib, "v"),
		"core":         core,
		"opm-operator": strings.TrimPrefix(operator.PinnedOperatorVersion, "v"),
	}, nil
}

// linkedVersion returns the version of the dependency path in info,
// following a replacement. A replacement by a directory has no version and
// yields "(devel)", which WritePins refuses as not a release, so a dev tree
// never publishes pins.
func linkedVersion(info *debug.BuildInfo, path string) (string, error) {
	for _, dep := range info.Deps {
		if dep.Path != path {
			continue
		}
		if dep.Replace != nil {
			if dep.Replace.Version == "" {
				return "(devel)", nil
			}
			return dep.Replace.Version, nil
		}
		return dep.Version, nil
	}
	return "", fmt.Errorf("%s is not in the build", path)
}

// coreRelease returns the exact core release a DefaultSchemaModule names,
// without "v"; one that names only a major is refused.
func coreRelease(module string) (string, error) {
	v, ok := strings.CutPrefix(module, corePrefix)
	if !ok || !strings.Contains(v, ".") {
		return "", fmt.Errorf("library DefaultSchemaModule %q pins no exact core release", module)
	}
	return v, nil
}
