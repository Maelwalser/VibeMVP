package verify

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GoVerifier runs `go build ./...` and `go vet ./...` on the generated Go code.
type GoVerifier struct{}

func NewGoVerifier() *GoVerifier { return &GoVerifier{} }

func (g *GoVerifier) Language() string { return "go" }

func (g *GoVerifier) Verify(ctx context.Context, outputDir string, files []string) (*Result, error) {
	// Find all directories that contain .go files among the generated files.
	dirs := goModDirs(outputDir, files)
	if len(dirs) == 0 {
		return &Result{Passed: true, Output: "no Go files found"}, nil
	}

	var combined bytes.Buffer
	allPassed := true

	for _, dir := range dirs {
		absDir := filepath.Join(outputDir, dir)
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			continue
		}

		// Resolve dependencies before building.
		tidyOut, tidyErr := runCmd(ctx, absDir, "go", "mod", "tidy")
		if tidyErr != nil {
			combined.WriteString(fmt.Sprintf("=== go mod tidy in %s ===\n%s\n", dir, tidyOut))
			if hint := modTidyHint(tidyOut); hint != "" {
				combined.WriteString(hint)
			}
			allPassed = false
			continue
		}

		// Run go build.
		buildOut, err := runCmd(ctx, absDir, "go", "build", "./...")
		combined.WriteString(fmt.Sprintf("=== go build in %s ===\n%s\n", dir, buildOut))
		if err != nil {
			allPassed = false
			continue // skip go vet if build failed
		}

		// Run go vet.
		vetOut, err := runCmd(ctx, absDir, "go", "vet", "./...")
		combined.WriteString(fmt.Sprintf("=== go vet in %s ===\n%s\n", dir, vetOut))
		if err != nil {
			allPassed = false
		}

		// Run go test with -short and timeout to catch test panics early
		// without running slow integration tests.
		testOut, testErr := runCmd(ctx, absDir, "go", "test", "-count=1", "-short", "-timeout", "30s", "./...")
		combined.WriteString(fmt.Sprintf("=== go test in %s ===\n%s\n", dir, testOut))
		if testErr != nil {
			allPassed = false
		}

		// Check gofmt formatting.
		fmtOut, fmtErr := runCmd(ctx, absDir, "gofmt", "-l", ".")
		combined.WriteString(fmt.Sprintf("=== gofmt -l in %s ===\n%s\n", dir, fmtOut))
		if fmtErr != nil || strings.TrimSpace(fmtOut) != "" {
			allPassed = false
			if strings.TrimSpace(fmtOut) != "" {
				combined.WriteString("files not gofmt-clean (run gofmt -w): " + fmtOut + "\n")
			}
		}
	}

	return &Result{Passed: allPassed, Output: combined.String()}, nil
}

// modTidyHint inspects go mod tidy output and returns an actionable fix suggestion
// when the failure is caused by an unresolvable transitive dependency.
// Returns "" when the failure has no specific actionable hint.
func modTidyHint(output string) string {
	// Pattern: "github.com/foo/bar@vX: invalid version: git ls-remote ... terminal prompts disabled"
	// or:      "github.com/foo/bar@vX: reading github.com/foo/bar/...: ... 404 Not Found"
	reInvalidVer := regexp.MustCompile(`([\w.-]+(?:/[\w.-]+)+)@(v[\w.+-]+):\s+invalid version`)
	reNotFound := regexp.MustCompile(`([\w.-]+(?:/[\w.-]+)+)@(v[\w.+-]+).*(?:404 Not Found|no matching versions)`)

	var broken []string
	seen := make(map[string]bool)

	for _, re := range []*regexp.Regexp{reInvalidVer, reNotFound} {
		for _, m := range re.FindAllStringSubmatch(output, -1) {
			key := m[1] + "@" + m[2]
			if !seen[key] {
				seen[key] = true
				broken = append(broken, key)
			}
		}
	}

	if len(broken) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\nFIX REQUIRED — unresolvable transitive dependencies detected:\n")
	for _, mod := range broken {
		b.WriteString(fmt.Sprintf("  • %s\n", mod))
	}
	b.WriteString(`
To fix this in go.mod, use one or more of these strategies:
  1. Add a 'replace' directive to redirect the broken module to a working fork or newer version:
       replace github.com/broken/pkg v0.0.0-old => github.com/working/pkg v1.0.0
  2. Upgrade the direct dependency that pulls in the broken module to a version that no longer needs it.
  3. Remove any direct dependency on packages that transitively require the broken module.
  4. Explicitly 'require' the broken module at a newer, resolvable version to override the transitive pin.

Regenerate go.mod with explicit 'require' pins for ALL direct dependencies at known-good recent versions.
`)
	return b.String()
}

// goModDirs finds unique directories containing a go.mod file among the generated files.
// Falls back to the root service directories if no go.mod is found.
func goModDirs(outputDir string, files []string) []string {
	seen := make(map[string]bool)
	dirs := []string{}

	for _, f := range files {
		if filepath.Base(f) == "go.mod" {
			dir := filepath.Dir(f)
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
	}

	// If no go.mod found, try top-level service dirs.
	if len(dirs) == 0 {
		for _, f := range files {
			if filepath.Ext(f) == ".go" {
				// Use the service root (first two path components).
				parts := strings.Split(filepath.ToSlash(f), "/")
				if len(parts) >= 2 {
					dir := filepath.Join(parts[0], parts[1])
					if !seen[dir] {
						seen[dir] = true
						dirs = append(dirs, dir)
					}
				}
			}
		}
	}
	return dirs
}

// runCmd executes a command in dir and returns combined stdout+stderr output.
func runCmd(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
