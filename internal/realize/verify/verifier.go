package verify

import (
	"context"
	"fmt"

	"github.com/vibe-menu/internal/realize/dag"
)

// Result describes the outcome of running verification checks on generated files.
type Result struct {
	Passed bool
	Output string // combined stdout+stderr from linters/build tools
}

// Verifier runs language-specific checks on generated files.
type Verifier interface {
	// Verify runs checks against files written to outputDir.
	// files is the subset of relative paths written by the task being verified.
	Verify(ctx context.Context, outputDir string, files []string) (*Result, error)
	// Language returns the language/technology string this verifier handles.
	Language() string
}

// Registry maps task kinds and languages to their Verifier.
type Registry struct {
	byLanguage map[string]Verifier
	null       Verifier
}

// NewRegistry returns a Registry pre-populated with all built-in verifiers.
func NewRegistry() *Registry {
	r := &Registry{
		byLanguage: make(map[string]Verifier),
		null:       &NullVerifier{},
	}
	r.Register(NewGoVerifier())
	r.Register(NewTsVerifier())
	r.Register(NewPythonVerifier())
	r.Register(NewTfVerifier())
	return r
}

// Register adds a verifier to the registry, keyed by its Language() string.
func (r *Registry) Register(v Verifier) {
	r.byLanguage[v.Language()] = v
}

// ForTask returns the best Verifier for a task based on the service language.
// Falls back to NullVerifier for unknown languages or non-service tasks.
func (r *Registry) ForTask(task *dag.Task) Verifier {
	lang := taskLanguage(task)
	if v, ok := r.byLanguage[lang]; ok {
		return v
	}
	// Infra tasks use terraform verifier when the IaC tool is Terraform.
	if task.Kind == dag.TaskKindInfraTerraform {
		if v, ok := r.byLanguage["terraform"]; ok {
			return v
		}
	}
	return r.null
}

// taskLanguage extracts the primary language from a task payload.
func taskLanguage(task *dag.Task) string {
	switch task.Kind {
	case dag.TaskKindService:
		if task.Payload.Service != nil {
			return normalizeLanguage(task.Payload.Service.Language)
		}
		// Monolith: check all services, use first one.
		if len(task.Payload.AllServices) > 0 {
			return normalizeLanguage(task.Payload.AllServices[0].Language)
		}
	case dag.TaskKindDataSchemas, dag.TaskKindDataMigrations:
		// Data tasks: language determined by primary service stack; use null.
		return ""
	case dag.TaskKindFrontend:
		if task.Payload.Frontend != nil {
			return normalizeFrontendLanguage(task.Payload.Frontend.Tech.Language)
		}
	case dag.TaskKindInfraTerraform:
		return "terraform"
	}
	return ""
}

func normalizeLanguage(lang string) string {
	switch lang {
	case "Go":
		return "go"
	case "TypeScript", "JavaScript", "Node.js", "TypeScript/Node":
		return "typescript"
	case "Python":
		return "python"
	case "Rust":
		return "rust"
	}
	return ""
}

func normalizeFrontendLanguage(lang string) string {
	switch lang {
	case "TypeScript", "JavaScript":
		return "typescript"
	case "Dart":
		return "" // Flutter — no simple tsc-equivalent
	}
	return ""
}

// FilePaths returns the relative paths from a slice of GeneratedFiles.
func FilePaths(files []dag.GeneratedFile) []string {
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	return paths
}

// ErrVerificationFailed is a typed error for verification failures.
type ErrVerificationFailed struct {
	TaskID string
	Output string
}

func (e *ErrVerificationFailed) Error() string {
	return fmt.Sprintf("verification failed for task %s:\n%s", e.TaskID, e.Output)
}
