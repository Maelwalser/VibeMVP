package verify

import (
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/vibe-menu/internal/realize/dag"
)

// ValidateStructure performs fast structural checks on generated files before
// they are written to disk for verification. Returns a list of issues found.
// These checks catch common LLM output problems (wrong package declarations,
// empty files, duplicate paths) that would otherwise cause cascading failures.
func ValidateStructure(files []dag.GeneratedFile, language string) []string {
	var issues []string

	// Check for duplicate file paths.
	seen := make(map[string]bool, len(files))
	for _, f := range files {
		normalized := filepath.ToSlash(f.Path)
		if seen[normalized] {
			issues = append(issues, fmt.Sprintf("duplicate file path: %s", normalized))
		}
		seen[normalized] = true
	}

	for _, f := range files {
		content := strings.TrimSpace(f.Content)

		// Empty file check.
		if len(content) < 10 {
			issues = append(issues, fmt.Sprintf("file %s is empty or near-empty (%d bytes)", f.Path, len(content)))
			continue
		}

		ext := strings.ToLower(filepath.Ext(f.Path))

		switch {
		case ext == ".go" || (language == "go" && ext == ""):
			issues = append(issues, validateGoStructure(f)...)
		case ext == ".ts" || ext == ".tsx":
			issues = append(issues, validateTSStructure(f)...)
		case ext == ".py":
			// Python: check for __init__.py is handled by the caller if needed.
			// No per-file structural check beyond emptiness.
		}
	}

	return issues
}

// validateGoStructure checks Go-specific structural invariants.
func validateGoStructure(f dag.GeneratedFile) []string {
	var issues []string

	// Parse the file to check package declaration.
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, f.Path, f.Content, parser.PackageClauseOnly)
	if err != nil {
		issues = append(issues, fmt.Sprintf("%s: cannot parse package clause: %v", f.Path, err))
		return issues
	}

	// Verify package name matches directory.
	expectedPkg := filepath.Base(filepath.Dir(f.Path))
	if expectedPkg == "." || expectedPkg == "" {
		expectedPkg = "main"
	}
	// Special case: _test files can have pkg_test package names.
	actualPkg := parsed.Name.Name
	if actualPkg != expectedPkg && actualPkg != expectedPkg+"_test" {
		// Some legitimate exceptions: "main" package can be in cmd/* directories,
		// and internal packages can use a different name convention.
		if actualPkg != "main" {
			issues = append(issues, fmt.Sprintf("%s: package %q does not match directory %q",
				f.Path, actualPkg, expectedPkg))
		}
	}

	return issues
}

// validateTSStructure checks TypeScript-specific structural invariants.
func validateTSStructure(f dag.GeneratedFile) []string {
	var issues []string

	// Check that the file has at least one export statement.
	if !strings.Contains(f.Content, "export ") {
		issues = append(issues, fmt.Sprintf("%s: no export statement found", f.Path))
	}

	return issues
}
