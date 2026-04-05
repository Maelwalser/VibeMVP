package verify

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ImportError describes a bad internal import in a Go source file.
type ImportError struct {
	File      string // absolute path of the file containing the bad import
	Import    string // the broken import path
	Suggested string // suggested replacement (empty when no deterministic fix found)
}

func (e ImportError) String() string {
	if e.Suggested != "" {
		return fmt.Sprintf("%s: imports %q → should be %q", e.File, e.Import, e.Suggested)
	}
	return fmt.Sprintf("%s: imports %q (package directory not found)", e.File, e.Import)
}

// ValidateImportPaths scans all .go files under outputDir and returns one
// ImportError for every internal import whose package directory does not exist.
//
// Two classes of broken import are detected:
//
//  1. Known-module prefix: the import starts with a module path declared in a
//     go.mod file found under outputDir, but the sub-package directory is missing.
//     This catches cross-task path drift, e.g. service A importing service B's
//     internal packages using B's module path when those packages were not yet
//     generated.
//
//  2. Wrong directory prefix: the import's first component has no dot and matches
//     the base name of a module directory, but is not the full module path. This
//     catches the common LLM failure of writing "backend/internal/domain" instead
//     of "github.com/acme/monolith/internal/domain".
//
// When a Suggested replacement is available the caller can auto-apply it; entries
// without a Suggested value require manual or LLM-driven repair.
func ValidateImportPaths(outputDir string) []ImportError {
	modules := findGoModules(outputDir)
	if len(modules) == 0 {
		return nil
	}

	var errs []ImportError
	_ = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return skipImportValidatorDir(info.Name())
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		imports, parseErr := extractGoImports(path)
		if parseErr != nil {
			return nil // skip unparseable files; let the compiler report syntax errors
		}
		for _, imp := range imports {
			if e, ok := checkImport(imp, path, modules); !ok {
				errs = append(errs, e)
			}
		}
		return nil
	})
	return errs
}

// FixImportPaths auto-corrects broken internal imports identified by
// ValidateImportPaths. Source files are rewritten in place. Returns a
// human-readable summary of changes, or "" when nothing was changed.
//
// Only ImportErrors with a non-empty Suggested field are applied automatically;
// unfixable errors are left for LLM repair.
func FixImportPaths(outputDir string) string {
	errs := ValidateImportPaths(outputDir)
	if len(errs) == 0 {
		return ""
	}

	// Group fixable errors by file so we make one write per file.
	byFile := make(map[string][]ImportError)
	for _, e := range errs {
		if e.Suggested != "" {
			byFile[e.File] = append(byFile[e.File], e)
		}
	}
	if len(byFile) == 0 {
		return ""
	}

	fixed := 0
	for filePath, fileErrs := range byFile {
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		content := string(data)
		changed := false
		for _, e := range fileErrs {
			old := `"` + e.Import + `"`
			replacement := `"` + e.Suggested + `"`
			if strings.Contains(content, old) {
				content = strings.ReplaceAll(content, old, replacement)
				changed = true
			}
		}
		if !changed {
			continue
		}
		if writeErr := os.WriteFile(filePath, []byte(content), 0o644); writeErr == nil {
			fixed++
		}
	}

	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("fixed cross-module import paths in %d file(s)", fixed)
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// checkImport validates a single import path against the known module map.
// Returns (ImportError, false) when the import is broken; (zero, true) otherwise.
func checkImport(imp, filePath string, modules map[string]string) (ImportError, bool) {
	// Class 1: import starts with a known module path — verify the sub-package exists.
	for modDir, modPath := range modules {
		prefix := modPath + "/"
		if !strings.HasPrefix(imp, prefix) {
			continue
		}
		subPath := strings.TrimPrefix(imp, prefix)
		targetDir := filepath.Join(modDir, filepath.FromSlash(subPath))
		if _, err := os.Stat(targetDir); err == nil {
			return ImportError{}, true // package directory exists — import is valid
		}
		// Package directory is missing. Try to find it in another module.
		suggested := ""
		if alt := findAlternativeModulePath(subPath, modules, modPath); alt != "" {
			suggested = alt + "/" + subPath
		}
		return ImportError{File: filePath, Import: imp, Suggested: suggested}, false
	}

	// Class 2: first path component has no dot — likely a bare directory prefix
	// instead of a full module path (e.g. "backend/internal/domain").
	if looksLikeBareDirectoryPrefix(imp) {
		if suggested := resolveFromDirectoryPrefix(imp, modules); suggested != "" {
			return ImportError{File: filePath, Import: imp, Suggested: suggested}, false
		}
	}

	return ImportError{}, true // not an internal import we can evaluate
}

// looksLikeBareDirectoryPrefix returns true when the import's first component
// contains no dot. All valid Go module paths start with a host name (e.g.
// "github.com", "golang.org"), which always contains a dot. A dotless first
// component (e.g. "backend") is a strong signal that the LLM used the service
// directory name as a module prefix.
func looksLikeBareDirectoryPrefix(imp string) bool {
	slash := strings.Index(imp, "/")
	if slash < 0 {
		return false // stdlib single-word imports (fmt, os) are fine
	}
	first := imp[:slash]
	return !strings.Contains(first, ".")
}

// resolveFromDirectoryPrefix fixes "svcdir/internal/pkg" → "github.com/org/svc/internal/pkg"
// by matching the first component against the base name of known module directories.
func resolveFromDirectoryPrefix(imp string, modules map[string]string) string {
	slash := strings.Index(imp, "/")
	if slash < 0 {
		return ""
	}
	dirPrefix := imp[:slash]
	subPath := imp[slash+1:]

	for modDir, modPath := range modules {
		if filepath.Base(modDir) != dirPrefix {
			continue
		}
		target := filepath.Join(modDir, filepath.FromSlash(subPath))
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			return modPath + "/" + subPath
		}
	}
	return ""
}

// findAlternativeModulePath searches modules (excluding skipMod) for one whose
// directory contains a sub-directory matching subPath. Returns the module path
// of the first match, or "".
func findAlternativeModulePath(subPath string, modules map[string]string, skipMod string) string {
	target := filepath.FromSlash(subPath)
	for modDir, modPath := range modules {
		if modPath == skipMod {
			continue
		}
		if info, err := os.Stat(filepath.Join(modDir, target)); err == nil && info.IsDir() {
			return modPath
		}
	}
	return ""
}

// findGoModules walks root and returns a map of {absolute module directory → module path}
// for every go.mod file found, skipping vendor, hidden dirs, and build caches.
func findGoModules(root string) map[string]string {
	modules := make(map[string]string)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return skipImportValidatorDir(info.Name())
		}
		if info.Name() != "go.mod" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				modPath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
				if modPath != "" {
					modules[filepath.Dir(path)] = modPath
				}
				break
			}
		}
		return nil
	})
	return modules
}

// extractGoImports parses a .go file with the Go parser (imports-only mode) and
// returns the list of import paths. This is more reliable than regex: it handles
// multi-line import blocks, aliased imports, and blank imports correctly.
func extractGoImports(filePath string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	imports := make([]string, 0, len(f.Imports))
	for _, spec := range f.Imports {
		if spec.Path != nil {
			imports = append(imports, strings.Trim(spec.Path.Value, `"`))
		}
	}
	return imports, nil
}

// skipImportValidatorDir returns filepath.SkipDir for directories that should
// be excluded from import validation walks.
func skipImportValidatorDir(name string) error {
	switch name {
	case ".tmp", "vendor", ".realize", "node_modules", ".next", "dist", "build":
		return filepath.SkipDir
	}
	if len(name) > 0 && name[0] == '.' {
		return filepath.SkipDir
	}
	return nil
}
