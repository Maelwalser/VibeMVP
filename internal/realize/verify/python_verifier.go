package verify

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

// PythonVerifier runs `ruff check`, `mypy` (if available), and `python -m py_compile`
// on generated Python code. All tools degrade gracefully if not installed.
type PythonVerifier struct{}

func NewPythonVerifier() *PythonVerifier { return &PythonVerifier{} }

func (p *PythonVerifier) Language() string { return "python" }

func (p *PythonVerifier) Verify(ctx context.Context, outputDir string, files []string) (*Result, error) {
	// Find directories with Python files.
	dirs := pythonProjectDirs(outputDir, files)
	if len(dirs) == 0 {
		return &Result{Passed: true, Output: "no Python files found"}, nil
	}

	var combined bytes.Buffer
	allPassed := true

	// ── ruff check ──────────────────────────────────────────────────────────
	ruffPath, ruffErr := exec.LookPath("ruff")
	if ruffErr != nil {
		combined.WriteString("ruff not found in PATH — skipping ruff check\n")
	}

	for _, dir := range dirs {
		absDir := filepath.Join(outputDir, dir)

		if ruffErr == nil {
			out, err := runCmd(ctx, absDir, ruffPath, "check", ".")
			combined.WriteString(fmt.Sprintf("=== ruff check in %s ===\n%s\n", dir, out))
			if err != nil {
				allPassed = false
			}
		}
	}

	// ── mypy ────────────────────────────────────────────────────────────────
	mypyPath, mypyErr := exec.LookPath("mypy")
	if mypyErr != nil {
		combined.WriteString("mypy not found in PATH — skipping mypy check\n")
	} else {
		for _, dir := range dirs {
			absDir := filepath.Join(outputDir, dir)
			out, err := runCmd(ctx, absDir, mypyPath, "--ignore-missing-imports", ".")
			combined.WriteString(fmt.Sprintf("=== mypy in %s ===\n%s\n", dir, out))
			if err != nil {
				allPassed = false
			}
		}
	}

	// ── py_compile syntax check ──────────────────────────────────────────────
	pythonPath, _ := exec.LookPath("python3")
	if pythonPath == "" {
		combined.WriteString("python3 not found in PATH — skipping py_compile check\n")
	} else {
		for _, f := range files {
			if filepath.Ext(f) != ".py" {
				continue
			}
			absFile := filepath.Join(outputDir, f)
			out, err := runCmd(ctx, filepath.Dir(absFile), pythonPath, "-m", "py_compile", absFile)
			if err != nil {
				combined.WriteString(fmt.Sprintf("=== py_compile %s ===\n%s\n", f, out))
				allPassed = false
			}
		}
	}

	return &Result{Passed: allPassed, Output: combined.String()}, nil
}

func pythonProjectDirs(outputDir string, files []string) []string {
	seen := make(map[string]bool)
	dirs := []string{}
	for _, f := range files {
		if filepath.Ext(f) == ".py" {
			dir := filepath.Dir(f)
			// Walk up to find pyproject.toml or the top-level service dir.
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
	}
	return dirs
}
