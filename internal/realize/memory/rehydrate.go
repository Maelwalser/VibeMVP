package memory

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vibe-menu/internal/realize/dag"
)

// RehydrateFromDisk walks the output directory for a completed task and reads
// all source files into GeneratedFile slices. This is used on resume to
// repopulate SharedMemory with outputs from tasks that completed in a prior run.
//
// baseDir is the writer's base directory (absolute path to output root).
// outputDir is the task's Payload.OutputDir (e.g. "backend", "frontend").
// Returns module-relative files (without the outputDir prefix) for type/constructor
// registration, and prefixed files for rawPaths storage — mirroring what the
// TaskRunner.commit() method produces.
func RehydrateFromDisk(baseDir, outputDir string) (moduleRelative, prefixed []dag.GeneratedFile, err error) {
	// Determine the directory to walk.
	walkDir := baseDir
	if outputDir != "" && outputDir != "." {
		walkDir = filepath.Join(baseDir, outputDir)
	}

	// If the directory doesn't exist, the task produced no files.
	if _, err := os.Stat(walkDir); os.IsNotExist(err) {
		return nil, nil, nil
	}

	err = filepath.Walk(walkDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".tmp" || name == "vendor" || name == ".realize" ||
				name == "node_modules" || name == ".next" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only read source files that the pipeline would have generated.
		if !isSourceFile(path) {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // skip unreadable files
		}

		// Module-relative path: relative to walkDir (the outputDir).
		relToWalk, relErr := filepath.Rel(walkDir, path)
		if relErr != nil {
			return nil
		}
		relToWalk = filepath.ToSlash(relToWalk)

		moduleRelative = append(moduleRelative, dag.GeneratedFile{
			Path:    relToWalk,
			Content: string(content),
		})

		// Prefixed path: relative to baseDir (includes outputDir prefix).
		relToBase, relErr := filepath.Rel(baseDir, path)
		if relErr != nil {
			return nil
		}
		relToBase = filepath.ToSlash(relToBase)

		prefixed = append(prefixed, dag.GeneratedFile{
			Path:    relToBase,
			Content: string(content),
		})

		return nil
	})

	return moduleRelative, prefixed, err
}

// isSourceFile reports whether a file path looks like a source file the pipeline
// would have generated (Go, TypeScript, Python, SQL, config files, etc.).
func isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx",
		".py", ".sql", ".proto", ".yaml", ".yml",
		".json", ".toml", ".mod", ".sum", ".lock",
		".tf", ".hcl", ".md", ".html", ".css",
		".dockerfile", ".sh":
		return true
	}
	// Dockerfiles without extension.
	base := strings.ToLower(filepath.Base(path))
	return base == "dockerfile" || base == "makefile" || base == ".env.example"
}
