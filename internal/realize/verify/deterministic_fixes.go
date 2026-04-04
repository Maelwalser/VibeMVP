package verify

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ApplyDeterministicFixes applies mechanical, always-correct transformations to
// generated code before running the language verifier. Returns a description
// of fixes applied, or "" if none were needed.
//
// Run this BEFORE every verification attempt — not just on retries — so that
// first-attempt code gets the same cleanup benefit without consuming a retry slot.
func ApplyDeterministicFixes(dir string, files []string) string {
	var fixes []string

	// Fix placeholder import paths first — rewriting imports may introduce
	// temporarily un-gofmt'd lines, so run gofmt after.
	if f := fixPlaceholderImportPaths(dir, files); f != "" {
		fixes = append(fixes, f)
	}
	if f := fixGoEscapeSequences(dir, files); f != "" {
		fixes = append(fixes, f)
	}
	if f := fixDuplicateTypes(dir, files); f != "" {
		fixes = append(fixes, f)
	}
	if f := fixGofmt(dir, files); f != "" {
		fixes = append(fixes, f)
	}

	if len(fixes) == 0 {
		return ""
	}
	return strings.Join(fixes, "; ")
}

// ── Placeholder import path fix ──────────────────────────────────────────────
//
// LLMs sometimes use placeholder organisation names in import paths
// (e.g. "github.com/your-org/app-name/internal/domain") instead of the actual
// Go module path declared in go.mod. This function reads go.mod from the temp
// directory, extracts the module path, and rewrites any placeholder-org imports
// to use the correct path — without requiring an LLM retry.

// placeholderOrgRe matches quoted import paths whose second path component is a
// known placeholder organisation name, e.g. "github.com/your-org/repo/internal/...".
var placeholderOrgRe = regexp.MustCompile(
	`"github\.com/(?:your-org|your-company|yourcompany|your_company|` +
		`mycompany|my-company|example-org|my-org|acme|acme-corp|` +
		`your-app|myapp|company|org-name|team-name)/[^"]+"`,
)

func fixPlaceholderImportPaths(dir string, files []string) string {
	modulePath := extractModulePathFromDir(dir)
	if modulePath == "" {
		return ""
	}

	fixed := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		path := filepath.Join(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)
		newContent := placeholderOrgRe.ReplaceAllStringFunc(content, func(match string) string {
			// match is e.g. `"github.com/your-org/auth-api/internal/domain"`
			inner := match[1 : len(match)-1] // strip surrounding quotes
			// Split: ["github.com", "your-org", "auth-api", "internal", "domain", ...]
			parts := strings.SplitN(inner, "/", 4)
			if len(parts) < 3 {
				return match // unexpected shape — leave unchanged
			}
			if len(parts) == 4 {
				// Has sub-path: replace org+repo with modulePath, keep sub-path.
				return `"` + modulePath + "/" + parts[3] + `"`
			}
			// Import is the module root itself (e.g. "github.com/your-org/auth-api").
			return `"` + modulePath + `"`
		})
		if newContent != content {
			os.WriteFile(path, []byte(newContent), 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("rewrote placeholder import paths in %d file(s) (module: %s)", fixed, modulePath)
}

// extractModulePathFromDir reads the module path from the go.mod file located
// in dir or any of its parent directories, returning "" when none is found.
func extractModulePathFromDir(dir string) string {
	current := dir
	for {
		data, err := os.ReadFile(filepath.Join(current, "go.mod"))
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "module "))
				}
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

// ── Escape sequence fix ──────────────────────────────────────────────────────

func fixGoEscapeSequences(dir string, files []string) string {
	fixed := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		path := filepath.Join(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)
		result := rewriteInvalidEscapes(content)
		if result != content {
			os.WriteFile(path, []byte(result), 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("fixed escape sequences in %d file(s)", fixed)
}

func rewriteInvalidEscapes(src string) string {
	var out strings.Builder
	i := 0
	for i < len(src) {
		// Skip raw strings.
		if src[i] == '`' {
			end := strings.IndexByte(src[i+1:], '`')
			if end >= 0 {
				out.WriteString(src[i : i+end+2])
				i += end + 2
			} else {
				out.WriteByte(src[i])
				i++
			}
			continue
		}
		// Skip // comments.
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '/' {
			end := strings.IndexByte(src[i:], '\n')
			if end >= 0 {
				out.WriteString(src[i : i+end])
				i += end
			} else {
				out.WriteString(src[i:])
				i = len(src)
			}
			continue
		}
		// Skip /* */ comments.
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '*' {
			end := strings.Index(src[i+2:], "*/")
			if end >= 0 {
				out.WriteString(src[i : i+end+4])
				i += end + 4
			} else {
				out.WriteString(src[i:])
				i = len(src)
			}
			continue
		}
		// Process double-quoted strings.
		if src[i] == '"' {
			strEnd := findQuoteEnd(src, i+1)
			if strEnd < 0 {
				out.WriteByte(src[i])
				i++
				continue
			}
			inner := src[i+1 : strEnd]
			if hasInvalidGoEscape(inner) && !strings.Contains(inner, "`") && !strings.Contains(inner, "\n") {
				rawInner := interpretedToRaw(inner)
				out.WriteByte('`')
				out.WriteString(rawInner)
				out.WriteByte('`')
			} else {
				out.WriteString(src[i : strEnd+1])
			}
			i = strEnd + 1
			continue
		}
		out.WriteByte(src[i])
		i++
	}
	return out.String()
}

func findQuoteEnd(s string, start int) int {
	escaped := false
	for i := start; i < len(s); i++ {
		if escaped {
			escaped = false
			continue
		}
		if s[i] == '\\' {
			escaped = true
			continue
		}
		if s[i] == '"' {
			return i
		}
		if s[i] == '\n' {
			return -1
		}
	}
	return -1
}

func hasInvalidGoEscape(inner string) bool {
	for i := 0; i < len(inner)-1; i++ {
		if inner[i] == '\\' {
			next := inner[i+1]
			switch next {
			case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '"', '\'',
				'0', '1', '2', '3', '4', '5', '6', '7',
				'x', 'u', 'U':
				// valid escape sequence
			default:
				return true
			}
			i++
		}
	}
	return false
}

func interpretedToRaw(inner string) string {
	var out strings.Builder
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' && i+1 < len(inner) {
			next := inner[i+1]
			switch next {
			case 'n':
				out.WriteByte('\n')
			case 't':
				out.WriteByte('\t')
			case '\\':
				out.WriteByte('\\')
			case '"':
				out.WriteByte('"')
			default:
				out.WriteByte('\\')
				out.WriteByte(next)
			}
			i++
		} else {
			out.WriteByte(inner[i])
		}
	}
	return out.String()
}

// ── Duplicate type fix ───────────────────────────────────────────────────────

func fixDuplicateTypes(dir string, files []string) string {
	byDir := make(map[string][]string)
	for _, f := range files {
		if filepath.Ext(f) != ".go" || strings.HasSuffix(f, "_test.go") {
			continue
		}
		byDir[filepath.Dir(f)] = append(byDir[filepath.Dir(f)], f)
	}
	fixed := 0
	for _, goFiles := range byDir {
		if len(goFiles) >= 2 && removeDuplicateDecls(dir, goFiles) {
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("fixed duplicate types in %d package(s)", fixed)
}

func removeDuplicateDecls(baseDir string, files []string) bool {
	re := regexp.MustCompile(`(?m)^type\s+(\w+)\s+`)

	typesByFile := make(map[string][]string)
	allTypes := make(map[string][]string)

	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(baseDir, f))
		if err != nil {
			continue
		}
		var types []string
		for _, m := range re.FindAllStringSubmatch(string(data), -1) {
			types = append(types, m[1])
			allTypes[m[1]] = append(allTypes[m[1]], f)
		}
		typesByFile[f] = types
	}

	// Find duplicated type names.
	duplicates := make(map[string][]string)
	for name, declFiles := range allTypes {
		if len(declFiles) > 1 {
			duplicates[name] = declFiles
		}
	}
	if len(duplicates) == 0 {
		return false
	}

	// Keep each type in the file with the most declarations; remove from others.
	filesToFix := make(map[string]map[string]bool)
	for typeName, declFiles := range duplicates {
		bestFile, bestCount := declFiles[0], len(typesByFile[declFiles[0]])
		for _, f := range declFiles[1:] {
			if len(typesByFile[f]) > bestCount {
				bestFile, bestCount = f, len(typesByFile[f])
			}
		}
		for _, f := range declFiles {
			if f != bestFile {
				if filesToFix[f] == nil {
					filesToFix[f] = make(map[string]bool)
				}
				filesToFix[f][typeName] = true
			}
		}
	}

	for f, typesToRemove := range filesToFix {
		data, err := os.ReadFile(filepath.Join(baseDir, f))
		if err != nil {
			continue
		}
		content := string(data)
		for typeName := range typesToRemove {
			typeRe := regexp.MustCompile(
				fmt.Sprintf(`(?ms)^type %s\s+(?:struct|interface)\s*\{[^}]*\}\s*\n?`,
					regexp.QuoteMeta(typeName)))
			content = typeRe.ReplaceAllString(content, "")
		}
		os.WriteFile(filepath.Join(baseDir, f), []byte(content), 0644)
	}
	return true
}

// ── gofmt fix ────────────────────────────────────────────────────────────────

func fixGofmt(dir string, files []string) string {
	fixed := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		path := filepath.Join(dir, f)
		before, _ := os.ReadFile(path)
		exec.Command("gofmt", "-w", path).Run()
		after, _ := os.ReadFile(path)
		if !bytes.Equal(before, after) {
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("gofmt fixed %d file(s)", fixed)
}
