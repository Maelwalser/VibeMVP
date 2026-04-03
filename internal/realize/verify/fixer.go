package verify

import (
	"go/format"
	"regexp"
	"strings"

	"github.com/vibe-menu/internal/realize/dag"
)

// TryFix attempts deterministic, zero-LLM fixes on generated files based on
// the verifier output. Returns the (possibly modified) file list and true if
// any fix was applied. Fixes are applied only to Go files for now.
//
// Deterministic fixes implemented:
//   - gofmt: format Go source using go/format (stdlib)
//   - unused imports: remove import lines reported as "imported and not used"
//
// These classes of errors account for ~17% of LLM-fixable verification failures
// (LlmFix 2024 analysis across 12,837 code generation errors). Applying them
// before a retry saves the full cost of an additional LLM invocation.
func TryFix(files []dag.GeneratedFile, verifyOutput string) (bool, []dag.GeneratedFile) {
	result := make([]dag.GeneratedFile, len(files))
	copy(result, files)
	anyFixed := false

	for i, f := range result {
		if !strings.HasSuffix(strings.ToLower(f.Path), ".go") {
			continue
		}
		content := f.Content
		changed := false

		// Fix unused imports first (before gofmt, to avoid reformatting noise).
		if strings.Contains(verifyOutput, "imported and not used") {
			fixed := removeUnusedImports(content, verifyOutput)
			if fixed != content {
				content = fixed
				changed = true
			}
		}

		// Apply gofmt to all Go files regardless of whether gofmt was specifically
		// reported — it's free and often resolves secondary formatting complaints.
		if formatted, err := format.Source([]byte(content)); err == nil {
			formatted := string(formatted)
			if formatted != content {
				content = formatted
				changed = true
			}
		}

		if changed {
			result[i] = dag.GeneratedFile{Path: f.Path, Content: content}
			anyFixed = true
		}
	}

	return anyFixed, result
}

// unusedImportRe matches lines like:
//
//	"pkg/path" imported and not used
//	./local/path imported and not used
var unusedImportRe = regexp.MustCompile(`"([^"]+)" imported and not used`)

// removeUnusedImports removes import lines that the Go compiler flagged as unused.
// It parses the verifier output to find the specific package paths and removes
// matching lines from the import block.
func removeUnusedImports(content, verifyOutput string) string {
	// Collect all unused package paths from the error output.
	matches := unusedImportRe.FindAllStringSubmatch(verifyOutput, -1)
	if len(matches) == 0 {
		return content
	}
	unused := make(map[string]bool, len(matches))
	for _, m := range matches {
		unused[m[1]] = true
	}

	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		skip := false
		for pkg := range unused {
			// Match bare `"pkg"` or aliased `alias "pkg"` import lines.
			if strings.Contains(trimmed, `"`+pkg+`"`) {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
