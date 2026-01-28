package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/cli/experiments/004-oci-module-loading/loader"
)

// Command-line flags
var (
	home       = flag.String("home", "testdata/.opm", "Config home directory")
	registry   = flag.String("registry", "", "OCI registry URL override (highest priority)")
	pathSlice  arrayFlags
	moduleDir  = flag.String("module", "testdata/simple-app", "Module directory")
	verbose    = flag.Bool("v", false, "Verbose output")
	dumpModule = flag.Bool("dump", false, "Dump loaded module structure")
)

// arrayFlags allows multiple --path flags
type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ",")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func init() {
	flag.Var(&pathSlice, "path", "Local path overlay (repeatable)")
}

func main() {
	flag.Parse()

	fmt.Println("=== Experiment 004: OCI Module Loading ===")
	fmt.Println()

	// Display configuration
	fmt.Printf("Config Home:    %s\n", *home)
	fmt.Printf("Module Dir:     %s\n", *moduleDir)
	if len(pathSlice) > 0 {
		fmt.Printf("Path Overlays:  %s\n", strings.Join(pathSlice, ", "))
	} else {
		fmt.Printf("Path Overlays:  (none)\n")
	}
	fmt.Println()

	// Create loader with config support
	l, err := loader.NewLoader(&loader.LoaderConfig{
		HomeDir:      *home,
		Registry:     *registry,
		PathOverlays: pathSlice,
		ModuleDir:    *moduleDir,
		Verbose:      *verbose,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create loader: %v\n", err)
		os.Exit(1)
	}

	// Load module
	fmt.Println("[1/3] Loading module...")
	result, err := l.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load module: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Module loaded successfully")
	fmt.Println()

	// Extract and display metadata
	fmt.Println("[2/3] Extracting module metadata...")
	if err := displayMetadata(result.Value); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to extract metadata: %v\n", err)
	}
	fmt.Println()

	// Verify provider access
	fmt.Println("[3/3] Verifying provider access...")
	if err := verifyProvider(result); err != nil {
		fmt.Printf("  ℹ No providers configured: %v\n", err)
	}
	fmt.Println()

	// Dump module structure if requested
	if *dumpModule {
		fmt.Println("=== Module Structure ===")
		fmt.Println()
		dumpValue(result.Value, "")
		fmt.Println()
	}

	fmt.Println("=== Success ===")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Verify module structure with: go run . --dump")
	fmt.Println("  2. Try path overlay: go run . --path ../../../catalog/v0")
	fmt.Println("  3. Override registry: go run . --registry localhost:5001")
	fmt.Println("  4. Try env override: OPM_REGISTRY=localhost:9999 go run .")
}

// displayMetadata extracts and displays module metadata
func displayMetadata(v cue.Value) error {
	// Try to extract module metadata
	moduleVal := v.LookupPath(cue.ParsePath("module"))
	if !moduleVal.Exists() {
		return fmt.Errorf("module field not found")
	}

	metadataVal := moduleVal.LookupPath(cue.ParsePath("metadata"))
	if !metadataVal.Exists() {
		return fmt.Errorf("module.metadata field not found")
	}

	name, _ := metadataVal.LookupPath(cue.ParsePath("name")).String()
	version, _ := metadataVal.LookupPath(cue.ParsePath("version")).String()
	description, _ := metadataVal.LookupPath(cue.ParsePath("description")).String()

	fmt.Printf("  Name:        %s\n", name)
	fmt.Printf("  Version:     %s\n", version)
	if description != "" {
		fmt.Printf("  Description: %s\n", description)
	}

	// Count components
	componentsVal := moduleVal.LookupPath(cue.ParsePath("components"))
	if componentsVal.Exists() {
		iter, _ := componentsVal.Fields()
		count := 0
		for iter.Next() {
			count++
		}
		fmt.Printf("  Components:  %d\n", count)
	}

	return nil
}

// verifyProvider checks that the provider is accessible
func verifyProvider(result *loader.LoadResult) error {
	// First check if providers are configured in config
	if result.Config != nil && len(result.Config.Providers) > 0 {
		providerNames := result.Config.ProviderNames()
		fmt.Printf("  Config Providers: %s\n", strings.Join(providerNames, ", "))

		// Verify each provider is valid
		for _, name := range providerNames {
			providerVal, ok := result.Config.GetProvider(name)
			if !ok {
				return fmt.Errorf("provider %s not found in config", name)
			}

			if !providerVal.Exists() {
				return fmt.Errorf("provider %s exists but is not accessible", name)
			}

			// Try to get provider metadata
			metadataVal := providerVal.LookupPath(cue.ParsePath("metadata"))
			if metadataVal.Exists() {
				if provName, err := metadataVal.LookupPath(cue.ParsePath("name")).String(); err == nil {
					fmt.Printf("  Provider:    %s (from config)\n", provName)
				}
			}
		}
		fmt.Println("  ✓ Providers accessible from config")
		return nil
	}

	// Fall back to checking module for provider (legacy)
	providerVal := result.Value.LookupPath(cue.ParsePath("provider"))
	if !providerVal.Exists() {
		return fmt.Errorf("no providers configured (add to config.cue or module)")
	}

	metadataVal := providerVal.LookupPath(cue.ParsePath("metadata"))
	if !metadataVal.Exists() {
		return fmt.Errorf("provider.metadata field not found")
	}

	name, _ := metadataVal.LookupPath(cue.ParsePath("name")).String()
	fmt.Printf("  Provider:    %s (from module)\n", name)
	fmt.Println("  ✓ Provider accessible via CUE import")

	return nil
}

// dumpValue recursively dumps the CUE value structure
func dumpValue(v cue.Value, indent string) {
	// This is a simplified dump - in a real implementation you'd want
	// to handle more cases and format more nicely
	iter, _ := v.Fields()
	for iter.Next() {
		label := iter.Selector().String()
		value := iter.Value()

		// Try to get concrete value
		if s, err := value.String(); err == nil {
			fmt.Printf("%s%s: %q\n", indent, label, s)
		} else if value.Kind().String() == "struct" {
			fmt.Printf("%s%s: {\n", indent, label)
			dumpValue(value, indent+"  ")
			fmt.Printf("%s}\n", indent)
		} else {
			fmt.Printf("%s%s: %s\n", indent, label, value.Kind())
		}
	}
}
