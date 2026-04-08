package verify

import (
	"fmt"
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
		lower := strings.ToLower(f.Path)
		isGo := strings.HasSuffix(lower, ".go")
		isTS := strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx")
		isPy := strings.HasSuffix(lower, ".py")

		if !isGo && !isTS && !isPy {
			continue
		}

		content := f.Content
		changed := false

		if isGo {
			// Fix unused imports first (before gofmt, to avoid reformatting noise).
			if strings.Contains(verifyOutput, "imported and not used") {
				fixed := removeUnusedImports(content, verifyOutput)
				if fixed != content {
					content = fixed
					changed = true
				}
			}

			// Fix := where all vars are already declared.
			if strings.Contains(verifyOutput, "no new variables on left side of :=") {
				fixed := fixShortDeclInMemory(content, f.Path, verifyOutput)
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
		}

		if isTS {
			// Fix let/const re-declarations in the same scope.
			if strings.Contains(verifyOutput, "Cannot redeclare block-scoped variable") {
				fixed := fixTSRedeclInMemory(content)
				if fixed != content {
					content = fixed
					changed = true
				}
			}
		}

		if isPy {
			// Fix duplicate class definitions — LLMs sometimes define the same
			// class twice in a single file, causing redefinition warnings.
			fixed := fixPyDuplicateClassesInMemory(content)
			if fixed != content {
				content = fixed
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

// fixShortDeclInMemory replaces := with = on lines reported by the compiler
// for a specific file path. Works on in-memory content (no disk I/O).
func fixShortDeclInMemory(content, filePath, verifyOutput string) string {
	lineNums := make(map[int]bool)
	for _, errLine := range strings.Split(verifyOutput, "\n") {
		m := regexp.MustCompile(`^(.+\.go):(\d+):\d+: no new variables on left side of :=`).FindStringSubmatch(strings.TrimSpace(errLine))
		if m == nil {
			continue
		}
		// Match if the error file ends with our file path (handles relative path differences).
		if !strings.HasSuffix(m[1], filePath) && m[1] != filePath {
			continue
		}
		var n int
		fmt.Sscanf(m[2], "%d", &n)
		lineNums[n] = true
	}
	if len(lineNums) == 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		if !lineNums[i+1] {
			continue
		}
		idx := strings.Index(l, ":=")
		if idx >= 0 {
			lines[i] = l[:idx] + "=" + l[idx+2:]
		}
	}
	return strings.Join(lines, "\n")
}

// fixTSRedeclInMemory converts duplicate let/const declarations to assignments
// in-memory. Uses the same logic as fixTSRedeclaration but works on strings.
func fixTSRedeclInMemory(content string) string {
	lines := strings.Split(content, "\n")
	declared := make(map[string]bool)
	depth := 0
	changed := false

	for i, line := range lines {
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth <= 0 {
			declared = make(map[string]bool)
			depth = 0
		}
		m := regexp.MustCompile(`^(\s*)(let|const)\s+(\w+)\s*=`).FindStringSubmatch(line)
		if m == nil {
			continue
		}
		indent, keyword, varName := m[1], m[2], m[3]
		if declared[varName] {
			lines[i] = strings.Replace(line, keyword+" "+varName, varName, 1)
			lines[i] = indent + strings.TrimLeft(lines[i], " \t")
			changed = true
		} else {
			declared[varName] = true
		}
	}
	if !changed {
		return content
	}
	return strings.Join(lines, "\n")
}

// fixPyDuplicateClassesInMemory removes duplicate class definitions from Python
// source. LLMs sometimes generate the same class twice in a single file. This
// keeps the first definition and removes subsequent ones with the same name.
func fixPyDuplicateClassesInMemory(content string) string {
	lines := strings.Split(content, "\n")
	seen := make(map[string]bool)
	var out []string
	skipUntilNextTopLevel := false
	skippedClassName := ""

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Detect top-level class definitions (no leading whitespace).
		if strings.HasPrefix(trimmed, "class ") && (len(line) == len(trimmed) || line[0] == 'c') {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				name := strings.Split(parts[1], "(")[0]
				name = strings.TrimRight(name, ":")
				if seen[name] {
					skipUntilNextTopLevel = true
					skippedClassName = name
					continue
				}
				seen[name] = true
			}
		}

		if skipUntilNextTopLevel {
			// Skip until we hit a non-indented, non-empty line (next top-level definition).
			if trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				skipUntilNextTopLevel = false
				skippedClassName = ""
				// Don't skip this line — it's the next definition.
			} else {
				continue
			}
		}

		_ = skippedClassName
		out = append(out, line)
	}

	result := strings.Join(out, "\n")
	if result == content {
		return content
	}
	return result
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
