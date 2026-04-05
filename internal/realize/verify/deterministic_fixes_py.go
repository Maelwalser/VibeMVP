package verify

import (
	"fmt"
	"os/exec"
	"strings"
)

// fixPython applies deterministic, zero-LLM fixes to Python files.
//
// Applied in order:
//  1. ruff --fix       — auto-fixes lint violations: unused imports (F401),
//     star imports (F403), bare excepts, PEP 8 whitespace, and more.
//  2. autoflake        — targeted unused-import removal when ruff is absent,
//     covering F401 and F811 (redefinition of unused import).
//  3. black / autopep8 — code formatting for consistent style.
//
// isort (import ordering) already runs via fixLanguageImports in the main
// ApplyDeterministicFixes dispatcher before this function is called.
func fixPython(dir string, files []string) string {
	var msgs []string
	if m := fixPythonRuff(dir, files); m != "" {
		msgs = append(msgs, m)
	}
	if m := fixPythonAutoflake(dir, files); m != "" {
		msgs = append(msgs, m)
	}
	if m := fixPythonFormat(dir, files); m != "" {
		msgs = append(msgs, m)
	}
	if len(msgs) == 0 {
		return ""
	}
	return strings.Join(msgs, "; ")
}

// fixPythonRuff runs `ruff check --fix` on Python files.
// ruff is the preferred Python linter/fixer — it is fast and covers the most
// auto-fixable rules including F401 (unused imports), E/W whitespace, and more.
func fixPythonRuff(dir string, files []string) string {
	ruffPath, err := exec.LookPath("ruff")
	if err != nil {
		return ""
	}

	pyFiles := filterByExt(dir, files, ".py")
	if len(pyFiles) == 0 {
		return ""
	}

	before := snapshotFiles(pyFiles)
	args := append([]string{"check", "--fix", "--quiet"}, pyFiles...)
	cmd := exec.Command(ruffPath, args...)
	cmd.Dir = dir
	_ = cmd.Run()
	n := countChanged(pyFiles, before)
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("ruff --fix applied to %d Python file(s)", n)
}

// fixPythonAutoflake runs autoflake --in-place to remove unused imports when
// ruff is not available. autoflake targets F401 (unused imports) and F811
// (redefinition of unused name from import).
func fixPythonAutoflake(dir string, files []string) string {
	// Skip when ruff already ran — it covers the same violation classes.
	if _, err := exec.LookPath("ruff"); err == nil {
		return ""
	}

	autoflakePath, err := exec.LookPath("autoflake")
	if err != nil {
		return ""
	}

	pyFiles := filterByExt(dir, files, ".py")
	if len(pyFiles) == 0 {
		return ""
	}

	before := snapshotFiles(pyFiles)
	args := append([]string{
		"--in-place",
		"--remove-all-unused-imports",
		"--remove-unused-variables",
	}, pyFiles...)
	cmd := exec.Command(autoflakePath, args...)
	cmd.Dir = dir
	_ = cmd.Run()
	n := countChanged(pyFiles, before)
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("autoflake removed unused imports in %d Python file(s)", n)
}

// fixPythonFormat runs black (preferred) or autopep8 to format Python files.
// Both are idempotent; black produces the most consistent output and is the
// de-facto standard formatter for modern Python projects.
func fixPythonFormat(dir string, files []string) string {
	pyFiles := filterByExt(dir, files, ".py")
	if len(pyFiles) == 0 {
		return ""
	}

	if blackPath, err := exec.LookPath("black"); err == nil {
		before := snapshotFiles(pyFiles)
		args := append([]string{"--quiet"}, pyFiles...)
		cmd := exec.Command(blackPath, args...)
		cmd.Dir = dir
		_ = cmd.Run()
		n := countChanged(pyFiles, before)
		if n > 0 {
			return fmt.Sprintf("black formatted %d Python file(s)", n)
		}
		return ""
	}

	if autopep8Path, err := exec.LookPath("autopep8"); err == nil {
		before := snapshotFiles(pyFiles)
		args := append([]string{"--in-place", "--aggressive"}, pyFiles...)
		cmd := exec.Command(autopep8Path, args...)
		cmd.Dir = dir
		_ = cmd.Run()
		n := countChanged(pyFiles, before)
		if n > 0 {
			return fmt.Sprintf("autopep8 formatted %d Python file(s)", n)
		}
	}

	return ""
}
