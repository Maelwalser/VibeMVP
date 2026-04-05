package deps

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vibe-menu/internal/realize/agent"
	"github.com/vibe-menu/internal/realize/dag"
)

// Agent resolves project dependencies by running the language-appropriate package
// manager in the shared service temp directory. It does not invoke an LLM.
//
// It is designed to run immediately after the plan task, which generates
// go.mod / package.json / requirements.in with direct dependencies only.
// The Agent runs the package manager to lock transitive dependency versions,
// then returns the resolved lock files so the runner can commit them to output
// and share them with downstream implementation agents via shared memory.
type Agent struct {
	tempDir string // shared service temp dir (populated by the plan task)
	verbose bool
}

// New returns a deps.Agent that resolves dependencies inside tempDir.
func New(tempDir string, verbose bool) *Agent {
	return &Agent{tempDir: tempDir, verbose: verbose}
}

// Run detects the service language and resolves its dependency tree.
// Resolved files (go.mod+go.sum, package-lock.json, requirements.txt) are
// returned so the TaskRunner can commit them to output and shared memory.
// This method never calls an LLM.
func (a *Agent) Run(ctx context.Context, ac *agent.Context) (*agent.Result, error) {
	lang := serviceLanguage(ac.Task.Payload)

	var (
		files []dag.GeneratedFile
		err   error
	)

	switch strings.ToLower(lang) {
	case "go":
		files, err = a.resolveGo(ctx)
	case "typescript", "javascript", "node.js", "typescript/node":
		files, err = a.resolveNode(ctx)
	case "python":
		files, err = a.resolvePython(ctx)
	default:
		// Language has no resolver yet — skip gracefully.
		if a.verbose {
			fmt.Printf("[deps] no resolver for language %q — skipping\n", lang)
		}
		return &agent.Result{Files: nil}, nil
	}

	if err != nil {
		return nil, err
	}
	return &agent.Result{Files: files}, nil
}

// resolveGo runs `go mod tidy` in every directory containing a go.mod file,
// then reads back the updated go.mod and the newly-created go.sum.
//
// `go mod tidy` queries the Go module proxy for the latest compatible versions
// of all transitive dependencies, so the returned files contain a complete,
// correctly-versioned module graph with no hallucinated pseudo-versions.
func (a *Agent) resolveGo(ctx context.Context) ([]dag.GeneratedFile, error) {
	modPaths, err := findFiles(a.tempDir, "go.mod")
	if err != nil {
		return nil, fmt.Errorf("find go.mod: %w", err)
	}
	if len(modPaths) == 0 {
		return nil, fmt.Errorf("no go.mod found in %s — the plan task must generate it first", a.tempDir)
	}

	var result []dag.GeneratedFile

	for _, modPath := range modPaths {
		dir := filepath.Dir(modPath)

		if a.verbose {
			fmt.Printf("[deps] go mod tidy in %s\n", dir)
		}

		out, err := runCmd(ctx, dir, "go", "mod", "tidy")
		if err != nil {
			return nil, fmt.Errorf("go mod tidy in %s failed: %w\n%s", dir, err, out)
		}

		// Read back the tidy'd go.mod (may have had versions added/updated).
		modContent, err := os.ReadFile(modPath)
		if err != nil {
			return nil, fmt.Errorf("read go.mod after tidy: %w", err)
		}
		relMod, _ := filepath.Rel(a.tempDir, modPath)
		result = append(result, dag.GeneratedFile{
			Path:    filepath.ToSlash(relMod),
			Content: string(modContent),
		})

		// Read go.sum (created or updated by tidy).
		sumPath := filepath.Join(dir, "go.sum")
		if sumContent, readErr := os.ReadFile(sumPath); readErr == nil {
			relSum, _ := filepath.Rel(a.tempDir, sumPath)
			result = append(result, dag.GeneratedFile{
				Path:    filepath.ToSlash(relSum),
				Content: string(sumContent),
			})
		}
	}

	return result, nil
}

// resolveNode runs `npm install --package-lock-only` to produce a locked
// package-lock.json without downloading node_modules.
func (a *Agent) resolveNode(ctx context.Context) ([]dag.GeneratedFile, error) {
	pkgPaths, err := findFiles(a.tempDir, "package.json")
	if err != nil {
		return nil, fmt.Errorf("find package.json: %w", err)
	}
	if len(pkgPaths) == 0 {
		return nil, nil // nothing to resolve
	}

	var result []dag.GeneratedFile

	for _, pkgPath := range pkgPaths {
		dir := filepath.Dir(pkgPath)

		if a.verbose {
			fmt.Printf("[deps] npm install --package-lock-only in %s\n", dir)
		}

		out, err := runCmd(ctx, dir, "npm", "install", "--package-lock-only", "--ignore-scripts", "--legacy-peer-deps")
		if err != nil {
			return nil, fmt.Errorf("npm install in %s failed: %w\n%s", dir, err, out)
		}

		// Run full npm install so node_modules exists for tsc verification.
		// Non-fatal: lock file is the critical output; missing node_modules just
		// degrades TypeScript verification to a skip rather than a hard failure.
		if out2, err2 := runCmd(ctx, dir, "npm", "install", "--ignore-scripts", "--legacy-peer-deps"); err2 != nil {
			if a.verbose {
				fmt.Printf("[deps] npm install (full) warning in %s: %s\n", dir, out2)
			}
		}

		// Read the generated package-lock.json.
		lockPath := filepath.Join(dir, "package-lock.json")
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			return nil, fmt.Errorf("read package-lock.json: %w", err)
		}
		relLock, _ := filepath.Rel(a.tempDir, lockPath)
		result = append(result, dag.GeneratedFile{
			Path:    filepath.ToSlash(relLock),
			Content: string(lockContent),
		})

		// Also return package.json in case npm normalised it.
		pkgContent, err := os.ReadFile(pkgPath)
		if err != nil {
			return nil, fmt.Errorf("read package.json: %w", err)
		}
		relPkg, _ := filepath.Rel(a.tempDir, pkgPath)
		result = append(result, dag.GeneratedFile{
			Path:    filepath.ToSlash(relPkg),
			Content: string(pkgContent),
		})
	}

	return result, nil
}

// resolvePython runs pip-compile to lock a requirements.in into requirements.txt.
// If no requirements.in is found, it returns nil (nothing to resolve).
func (a *Agent) resolvePython(ctx context.Context) ([]dag.GeneratedFile, error) {
	reqPaths, err := findFiles(a.tempDir, "requirements.in")
	if err != nil {
		return nil, fmt.Errorf("find requirements.in: %w", err)
	}
	if len(reqPaths) == 0 {
		return nil, nil // nothing to compile
	}

	var result []dag.GeneratedFile

	for _, reqIn := range reqPaths {
		dir := filepath.Dir(reqIn)
		outPath := filepath.Join(dir, "requirements.txt")

		if a.verbose {
			fmt.Printf("[deps] pip-compile in %s\n", dir)
		}

		out, err := runCmd(ctx, dir, "pip-compile",
			"--output-file=requirements.txt", filepath.Base(reqIn))
		if err != nil {
			return nil, fmt.Errorf("pip-compile in %s failed: %w\n%s", dir, err, out)
		}

		content, err := os.ReadFile(outPath)
		if err != nil {
			return nil, fmt.Errorf("read requirements.txt: %w", err)
		}
		rel, _ := filepath.Rel(a.tempDir, outPath)
		result = append(result, dag.GeneratedFile{
			Path:    filepath.ToSlash(rel),
			Content: string(content),
		})
	}

	return result, nil
}

// serviceLanguage extracts the primary backend language from the task payload.
func serviceLanguage(p dag.TaskPayload) string {
	if p.Service != nil {
		return p.Service.Language
	}
	if len(p.AllServices) > 0 {
		return p.AllServices[0].Language
	}
	return ""
}

// findFiles walks dir recursively and returns all absolute paths of files
// whose base name matches name.
func findFiles(dir, name string) ([]string, error) {
	var paths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() && filepath.Base(path) == name {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// runCmd executes a command in dir and returns combined stdout+stderr.
func runCmd(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
