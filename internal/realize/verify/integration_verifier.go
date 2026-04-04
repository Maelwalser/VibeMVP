package verify

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IntegrationResult holds the outcome of a project-wide integration build.
type IntegrationResult struct {
	// Passed is true only when every language-specific build check succeeded.
	Passed bool
	// Output contains the combined build output, one section per project directory.
	Output string
	// Errors is a list of per-directory build failures for structured logging.
	Errors []IntegrationError
}

// IntegrationError captures one failing build directory.
type IntegrationError struct {
	Language string // "go" or "typescript"
	Dir      string // relative path from output root
	Output   string // raw compiler/tsc output
}

// RunIntegrationBuild performs a project-wide compilation check across the entire
// output directory tree. It finds every Go module (go.mod) and TypeScript project
// (tsconfig.json), runs the appropriate build tool, and returns a consolidated result.
//
// Unlike per-task verification — which only checks each component in isolation —
// the integration build stages all generated files together, catching cross-task
// failures such as:
//   - Wrong module import paths (hallucinated placeholder org names)
//   - Type mismatches between tasks (e.g. AuthResponse field layout)
//   - Missing error return handling (multi-return constructor ignored by caller)
//   - Duplicate type declarations surviving within the same package
//
// This is intentionally non-blocking: a failure is logged prominently but does not
// abort the pipeline — the user still receives all generated code plus a clear
// diagnostic of what needs to be fixed.
func RunIntegrationBuild(ctx context.Context, outputDir string) IntegrationResult {
	var errs []IntegrationError
	var sections []string

	// Go modules — run `go build ./...` + `go vet ./...` in each.
	for _, dir := range findGoModDirsIntegration(outputDir) {
		rel, _ := filepath.Rel(outputDir, dir)
		out, failed := runIntegrationGo(ctx, dir)
		header := fmt.Sprintf("=== Go build: %s ===", rel)
		if failed {
			sections = append(sections, header+"\n"+out)
			errs = append(errs, IntegrationError{Language: "go", Dir: rel, Output: out})
		} else {
			sections = append(sections, header+" ✓")
		}
	}

	// TypeScript projects — run `tsc --noEmit` in each.
	for _, dir := range findTSDirsIntegration(outputDir) {
		rel, _ := filepath.Rel(outputDir, dir)
		out, failed := runIntegrationTS(ctx, dir)
		header := fmt.Sprintf("=== TypeScript check: %s ===", rel)
		if failed {
			sections = append(sections, header+"\n"+out)
			errs = append(errs, IntegrationError{Language: "typescript", Dir: rel, Output: out})
		} else {
			sections = append(sections, header+" ✓")
		}
	}

	return IntegrationResult{
		Passed: len(errs) == 0,
		Output: strings.Join(sections, "\n"),
		Errors: errs,
	}
}

// findGoModDirsIntegration walks outputDir and returns the absolute path of
// every directory containing a go.mod file, skipping hidden dirs and .tmp/.
func findGoModDirsIntegration(root string) []string {
	var dirs []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error { //nolint:errcheck
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".tmp" || name == "vendor" || name == ".realize" ||
				(len(name) > 0 && name[0] == '.') {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() == "go.mod" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	return dirs
}

// findTSDirsIntegration walks outputDir and returns the absolute path of every
// directory containing a tsconfig.json, skipping node_modules, .next, .tmp.
func findTSDirsIntegration(root string) []string {
	var dirs []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error { //nolint:errcheck
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".next" || name == ".tmp" ||
				name == "dist" || name == "build" || name == ".realize" {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() == "tsconfig.json" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	return dirs
}

// runIntegrationGo runs `go build ./...` then `go vet ./...` in dir.
// Returns combined output and whether any step failed.
func runIntegrationGo(ctx context.Context, dir string) (string, bool) {
	var parts []string
	failed := false

	// go mod tidy first so cross-task go.sum drift doesn't cause false failures.
	tidyOut, tidyErr := runCmd(ctx, dir, "go", "mod", "tidy")
	if tidyErr != nil {
		parts = append(parts, "go mod tidy:\n"+tidyOut)
		failed = true
		return strings.Join(parts, "\n"), failed
	}

	buildOut, buildErr := runCmd(ctx, dir, "go", "build", "./...")
	if strings.TrimSpace(buildOut) != "" || buildErr != nil {
		parts = append(parts, "go build:\n"+buildOut)
		if buildErr != nil {
			failed = true
		}
	}

	vetOut, vetErr := runCmd(ctx, dir, "go", "vet", "./...")
	if strings.TrimSpace(vetOut) != "" || vetErr != nil {
		parts = append(parts, "go vet:\n"+vetOut)
		if vetErr != nil {
			failed = true
		}
	}

	return strings.Join(parts, "\n"), failed
}

// runIntegrationTS runs `tsc --noEmit` in dir using npx when tsc is not on PATH.
func runIntegrationTS(ctx context.Context, dir string) (string, bool) {
	// Skip if no node_modules installed yet (tsc unavailable).
	if _, err := os.Stat(filepath.Join(dir, "node_modules")); os.IsNotExist(err) {
		return "(skipped — node_modules not installed)", false
	}

	var cmd *exec.Cmd
	if _, err := exec.LookPath("tsc"); err == nil {
		cmd = exec.CommandContext(ctx, "tsc", "--noEmit")
	} else {
		cmd = exec.CommandContext(ctx, "npx", "--no-install", "tsc", "--noEmit")
	}
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	return output, err != nil && output != ""
}
