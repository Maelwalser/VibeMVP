package verify

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

		// Run go test.
		testOut, testErr := runCmd(ctx, absDir, "go", "test", "./...")
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
				parts := filepath.SplitList(filepath.ToSlash(f))
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
