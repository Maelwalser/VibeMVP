package verify

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/vibe-menu/internal/realize/dag"
	"github.com/vibe-menu/internal/realize/memory"
)

// CheckInterfaceCompliance verifies that generated Go files implement all methods
// required by the interface contracts from shared memory. Returns warnings for
// missing methods (advisory — not blocking, but recorded in CrossTaskIssues).
//
// This catches the most common failure mode: truncated interface implementations
// where a struct is missing one or more methods required by its interface contract.
func CheckInterfaceCompliance(files []dag.GeneratedFile, contracts []memory.InterfaceContract) []string {
	if len(contracts) == 0 {
		return nil
	}

	// Build a map of all method signatures found in the generated files.
	// Key: method name, Values: files that declare it.
	methodsByName := make(map[string][]string) // method name → declaring files

	methodRe := regexp.MustCompile(`(?m)^func\s+\([^)]+\)\s+(\w+)\s*\(`)

	for _, f := range files {
		if !strings.HasSuffix(f.Path, ".go") || strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		for _, m := range methodRe.FindAllStringSubmatch(f.Content, -1) {
			methodsByName[m[1]] = append(methodsByName[m[1]], f.Path)
		}
	}

	var warnings []string
	for _, contract := range contracts {
		for _, methodSig := range contract.Methods {
			// Extract method name from signature like "FindByEmail(ctx context.Context, ...) ..."
			methodName := extractMethodName(methodSig)
			if methodName == "" {
				continue
			}
			if _, found := methodsByName[methodName]; !found {
				warnings = append(warnings, fmt.Sprintf(
					"interface %s requires method %s but no implementation found (defined in %s)",
					contract.InterfaceName, methodName, contract.File))
			}
		}
	}

	return warnings
}

// extractMethodName pulls the method name from an interface method signature.
// E.g. "FindByEmail(ctx context.Context, email string) (*User, error)" → "FindByEmail"
func extractMethodName(sig string) string {
	sig = strings.TrimSpace(sig)
	paren := strings.Index(sig, "(")
	if paren < 0 {
		return ""
	}
	return strings.TrimSpace(sig[:paren])
}
