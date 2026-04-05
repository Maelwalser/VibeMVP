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
	// Remove invalid pgxpool v5 fields before gofmt so the result is clean.
	if f := fixInvalidPgxpoolConfig(dir, files); f != "" {
		fixes = append(fixes, f)
	}
	// Remove import statements that appear inside function bodies — always a bug.
	if f := fixMisplacedImports(dir, files); f != "" {
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
			_ = os.WriteFile(path, []byte(newContent), 0644)
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
			_ = os.WriteFile(path, []byte(result), 0644)
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
		_ = os.WriteFile(filepath.Join(baseDir, f), []byte(content), 0644)
	}
	return true
}

// ── Misplaced import fix ──────────────────────────────────────────────────────
//
// Go does not allow import statements inside function bodies. LLMs sometimes
// generate `import (...)` blocks inside functions (often as commentary stubs).
// These cause "syntax error: unexpected keyword import" compile errors.
// This fix detects indented import blocks inside function bodies and removes them.

var indentedImportRe = regexp.MustCompile(`(?m)^[ \t]+import\s*(?:\([\s\S]*?\)|\".+?\")[ \t]*\n?`)

func fixMisplacedImports(dir string, files []string) string {
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
		newContent := indentedImportRe.ReplaceAll(data, nil)
		if len(newContent) != len(data) {
			_ = os.WriteFile(path, newContent, 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("removed misplaced import statement(s) in %d file(s)", fixed)
}

// ── pgxpool v5 invalid field fix ─────────────────────────────────────────────
//
// pgxpool.Config in github.com/jackc/pgx/v5 removed the ConnectTimeout field.
// LLMs trained on older documentation frequently generate config.ConnectTimeout = ...
// which causes a compile error. This fix removes those lines deterministically.

var pgxpoolInvalidFieldRe = regexp.MustCompile(`(?m)^\s*config\.ConnectTimeout\s*=.*\n?`)

func fixInvalidPgxpoolConfig(dir string, files []string) string {
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
		newContent := pgxpoolInvalidFieldRe.ReplaceAll(data, nil)
		if len(newContent) != len(data) {
			_ = os.WriteFile(path, newContent, 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("removed invalid pgxpool v5 fields in %d file(s)", fixed)
}

// ── Type-as-string conversion fix ────────────────────────────────────────────
//
// LLMs frequently define response struct fields as string when the domain type
// is bool, time.Time, or uuid.UUID, then assign the typed value without conversion.
// This causes "cannot use X (variable of type T) as string value" compile errors.
//
// This fix detects those patterns and rewrites the offending expression with the
// idiomatic Go conversion: .String() for UUID, strconv.FormatBool for bool,
// .Format(time.RFC3339) for time.Time.
//
// The same fix also handles struct literal context ("as string value in struct literal")
// and function argument context ("as string value in argument to").

var typeAsStringErrRe = regexp.MustCompile(
	`^(.+\.go):(\d+):\d+: cannot use (\S+) \(variable of (?:array |struct )?type ([^)]+)\) as string`)

// ApplyUUIDToStringFixes reads go compiler output, finds type-as-string errors for
// well-known types (uuid.UUID, bool, time.Time), and patches source files in dir.
// The name is kept for backward-compatibility with callers.
func ApplyUUIDToStringFixes(dir string, verifyOutput string) string {
	type fix struct {
		file    string
		line    int
		varName string
		srcType string
	}
	var fixes []fix
	seen := make(map[string]bool)
	for _, line := range strings.Split(verifyOutput, "\n") {
		m := typeAsStringErrRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		relFile, lineStr, varName, srcType := m[1], m[2], m[3], m[4]
		key := relFile + ":" + lineStr
		if seen[key] {
			continue
		}
		seen[key] = true
		lineNum := 0
		fmt.Sscanf(lineStr, "%d", &lineNum)
		fixes = append(fixes, fix{file: relFile, line: lineNum, varName: varName, srcType: srcType})
	}
	if len(fixes) == 0 {
		return ""
	}
	// Group fixes by file so we apply all line patches and then a single import pass.
	type fileFix struct {
		lineIdx     int
		varName     string
		replacement string
		needImport  string // package name to ensure is imported, or ""
	}
	byFile := make(map[string][]fileFix)
	for _, fx := range fixes {
		replacement := typeToStringExpr(fx.varName, fx.srcType)
		if replacement == "" {
			continue // unsupported type — let LLM handle on retry
		}
		needImport := requiredImport(fx.srcType)
		path := filepath.Join(dir, fx.file)
		if _, err := os.Stat(path); err != nil {
			path = fx.file
		}
		byFile[path] = append(byFile[path], fileFix{
			lineIdx:    fx.line - 1,
			varName:    fx.varName,
			replacement: replacement,
			needImport: needImport,
		})
	}
	applied := 0
	for path, fileFixes := range byFile {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		fileLines := strings.Split(string(data), "\n")
		changed := false
		for _, ff := range fileFixes {
			if ff.lineIdx < 0 || ff.lineIdx >= len(fileLines) {
				continue
			}
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(ff.varName) + `\b`)
			patched := re.ReplaceAllString(fileLines[ff.lineIdx], ff.replacement)
			if patched != fileLines[ff.lineIdx] {
				fileLines[ff.lineIdx] = patched
				changed = true
			}
			// Ensure the required package is imported.
			if ff.needImport != "" {
				fileLines = ensureGoImport(fileLines, ff.needImport)
			}
		}
		if !changed {
			continue
		}
		_ = os.WriteFile(path, []byte(strings.Join(fileLines, "\n")), 0644)
		applied++
	}
	if applied == 0 {
		return ""
	}
	return fmt.Sprintf("applied type→string conversions to %d file(s)", applied)
}

// typeToStringExpr returns the idiomatic Go expression that converts varName of
// srcType to string, or "" when the conversion is not known / safe to automate.
func typeToStringExpr(varName, srcType string) string {
	switch {
	case srcType == "uuid.UUID":
		return varName + ".String()"
	case srcType == "bool":
		return "strconv.FormatBool(" + varName + ")"
	case srcType == `"time".Time`, srcType == "time.Time":
		return varName + ".Format(time.RFC3339)"
	case srcType == "int", srcType == "int64", srcType == "int32":
		return "strconv.Itoa(int(" + varName + "))"
	default:
		return ""
	}
}

// requiredImport returns the Go standard-library package that must be imported
// when the given type conversion is applied, or "" when no extra import is needed.
func requiredImport(srcType string) string {
	switch srcType {
	case "bool", "int", "int64", "int32":
		return "strconv"
	default:
		return ""
	}
}

// ensureGoImport ensures that importPkg is present in the import block of the
// given file lines. If it is not present, it is added. Returns the (possibly
// modified) line slice.
func ensureGoImport(lines []string, importPkg string) []string {
	quoted := `"` + importPkg + `"`
	for _, l := range lines {
		if strings.Contains(l, quoted) {
			return lines // already imported
		}
	}
	// Find the import block or single-line import and add the package.
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "import (" {
			// Insert before the closing paren.
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == ")" {
					newLines := make([]string, 0, len(lines)+1)
					newLines = append(newLines, lines[:j]...)
					newLines = append(newLines, "\t"+quoted)
					newLines = append(newLines, lines[j:]...)
					return newLines
				}
			}
		}
		// Single-line import: insert a new import block after it.
		if strings.HasPrefix(trimmed, `import "`) {
			newLines := make([]string, 0, len(lines)+3)
			newLines = append(newLines, lines[:i+1]...)
			newLines = append(newLines, "import "+quoted)
			newLines = append(newLines, lines[i+1:]...)
			return newLines
		}
	}
	return lines
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
		_ = exec.Command("gofmt", "-w", path).Run()
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
