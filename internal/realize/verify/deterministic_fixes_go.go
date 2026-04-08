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

// ── Go import fixer ───────────────────────────────────────────────────────────
//
// fixGoImports runs goimports on all .go files in the directory.
// goimports adds missing stdlib/module imports and removes unused ones.
// Falls back to removeUnusedGoImports when goimports is not installed.
func fixGoImports(dir string, files []string) string {
	// Check if goimports is available.
	goimportsPath, err := exec.LookPath("goimports")
	if err != nil {
		// goimports not installed — fall back to the simpler unused-import remover.
		return removeUnusedGoImports(dir, files)
	}

	fixed := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		path := filepath.Join(dir, f)
		before, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Run goimports in the file's own directory so it can resolve module imports.
		cmd := exec.Command(goimportsPath, "-w", path)
		cmd.Dir = dir
		_ = cmd.Run()
		after, _ := os.ReadFile(path)
		if !bytes.Equal(before, after) {
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("goimports fixed imports in %d file(s)", fixed)
}

// removeUnusedGoImports is a fallback for when goimports is not installed.
// It parses "go build" errors for "imported and not used" lines and removes
// the offending import from the named file. This handles the most common
// case where the LLM imports a package it doesn't actually use.
func removeUnusedGoImports(dir string, files []string) string {
	if len(files) == 0 {
		return ""
	}

	// Find the go.mod root so we can run go build in the right module directory.
	// We need the nearest go.mod directory, not dir itself.
	goModDir := findGoModDir(dir, files)
	if goModDir == "" {
		return ""
	}

	// Run go build to find unused import errors.
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = goModDir
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		return ""
	}

	// Parse "imported and not used" errors.
	// Format: <file>:<line>:<col>: "<pkg>" imported and not used
	unusedImportRe := regexp.MustCompile(`^(.+\.go):(\d+):\d+: "([^"]+)" imported and not used`)
	type fix struct {
		file string
		line int
		pkg  string
	}
	var fixes []fix
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		m := unusedImportRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		relFile, lineStr, pkg := m[1], m[2], m[3]
		key := relFile + ":" + lineStr
		if seen[key] {
			continue
		}
		seen[key] = true
		lineNum := 0
		fmt.Sscanf(lineStr, "%d", &lineNum)
		fixes = append(fixes, fix{file: relFile, line: lineNum, pkg: pkg})
	}
	if len(fixes) == 0 {
		return ""
	}

	// Group by file and remove the import lines.
	byFile := make(map[string][]fix)
	for _, fx := range fixes {
		byFile[fx.file] = append(byFile[fx.file], fx)
	}

	applied := 0
	for relFile, fileFixes := range byFile {
		// Try both the path as reported by go build and relative to goModDir.
		path := relFile
		if !filepath.IsAbs(path) {
			path = filepath.Join(goModDir, relFile)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		changed := false
		for _, fx := range fileFixes {
			if fx.line < 1 || fx.line > len(lines) {
				continue
			}
			lineIdx := fx.line - 1
			trimmed := strings.TrimSpace(lines[lineIdx])
			// Match the import line: \t"pkg" or \t_ "pkg" or \talias "pkg"
			if strings.Contains(trimmed, `"`+fx.pkg+`"`) {
				lines[lineIdx] = ""
				changed = true
			}
		}
		if !changed {
			continue
		}
		_ = os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
		applied++
	}
	if applied == 0 {
		return ""
	}
	return fmt.Sprintf("removed unused imports in %d file(s) (goimports fallback)", applied)
}

// findGoModDir returns the directory containing go.mod that is a parent of
// the given dir, walking up the tree. Returns "" when no go.mod is found.
func findGoModDir(dir string, files []string) string {
	// First check if any of the listed files are under a go.mod directory.
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		abs := filepath.Join(dir, f)
		current := filepath.Dir(abs)
		for {
			if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
				return current
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}
	// Fall back to walking up from dir itself.
	current := dir
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
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

// ── Duplicate var fix ───────────────────────────────────────────────────────
//
// LLMs (and cross-task staging) can produce duplicate top-level var declarations
// in the same Go package — e.g. both data.schemas' user.go and the plan task's
// errors.go declare `var ErrTokenExpired = errors.New(...)`. This causes
// "redeclared in this block" compile errors.
//
// Strategy: group files by package directory, collect all top-level var names,
// and when a var is declared in multiple files, remove it from the file that has
// fewer total declarations (keeping the "home" file that defines most vars).

// topLevelVarRe matches standalone "var X = ..." declarations.
var topLevelVarRe = regexp.MustCompile(`(?m)^var\s+(\w+)\s*=`)

func fixDuplicateVars(dir string, files []string) string {
	byDir := make(map[string][]string)
	for _, f := range files {
		if filepath.Ext(f) != ".go" || strings.HasSuffix(f, "_test.go") {
			continue
		}
		byDir[filepath.Dir(f)] = append(byDir[filepath.Dir(f)], f)
	}
	fixed := 0
	for _, goFiles := range byDir {
		if len(goFiles) < 2 {
			continue
		}
		if removeDuplicateVarDecls(dir, goFiles) {
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("fixed duplicate vars in %d package(s)", fixed)
}

// varBlockEntryRe matches entries inside a "var ( ... )" block.
var varBlockEntryRe = regexp.MustCompile(`(?m)^\t(\w+)\s*=`)

// varBlockRe matches the full "var ( ... )" block.
var varBlockRe = regexp.MustCompile(`(?ms)^var\s*\(\s*\n(.*?)\n\)`)

// extractVarNames returns all package-level var names from Go source.
func extractVarNames(src string) []string {
	var names []string
	// Standalone: var X = ...
	for _, m := range topLevelVarRe.FindAllStringSubmatch(src, -1) {
		names = append(names, m[1])
	}
	// Grouped: var ( X = ... )
	for _, block := range varBlockRe.FindAllStringSubmatch(src, -1) {
		for _, m := range varBlockEntryRe.FindAllStringSubmatch(block[1], -1) {
			names = append(names, m[1])
		}
	}
	return names
}

func removeDuplicateVarDecls(baseDir string, files []string) bool {
	varsByFile := make(map[string][]string)
	allVars := make(map[string][]string)

	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(baseDir, f))
		if err != nil {
			continue
		}
		vars := extractVarNames(string(data))
		for _, v := range vars {
			allVars[v] = append(allVars[v], f)
		}
		varsByFile[f] = vars
	}

	// Find duplicated var names.
	duplicates := make(map[string][]string)
	for name, declFiles := range allVars {
		if len(declFiles) > 1 {
			duplicates[name] = declFiles
		}
	}
	if len(duplicates) == 0 {
		return false
	}

	// Keep each var in the file with the most declarations; remove from others.
	filesToFix := make(map[string]map[string]bool)
	for varName, declFiles := range duplicates {
		bestFile, bestCount := declFiles[0], len(varsByFile[declFiles[0]])
		for _, f := range declFiles[1:] {
			if len(varsByFile[f]) > bestCount {
				bestFile, bestCount = f, len(varsByFile[f])
			}
		}
		for _, f := range declFiles {
			if f != bestFile {
				if filesToFix[f] == nil {
					filesToFix[f] = make(map[string]bool)
				}
				filesToFix[f][varName] = true
			}
		}
	}

	changed := false
	for f, varsToRemove := range filesToFix {
		data, err := os.ReadFile(filepath.Join(baseDir, f))
		if err != nil {
			continue
		}
		content := string(data)
		for varName := range varsToRemove {
			// Remove standalone: var X = ...
			standaloneRe := regexp.MustCompile(
				fmt.Sprintf(`(?m)^var %s\s*=\s*[^\n]+\n?`, regexp.QuoteMeta(varName)))
			content = standaloneRe.ReplaceAllString(content, "")
			// Remove from var ( ... ) blocks: \tX = ...\n
			blockEntryRe := regexp.MustCompile(
				fmt.Sprintf(`(?m)^\t%s\s*=\s*[^\n]+\n?`, regexp.QuoteMeta(varName)))
			content = blockEntryRe.ReplaceAllString(content, "")
		}
		// Clean up empty var blocks: var (\n)
		content = regexp.MustCompile(`(?m)^var\s*\(\s*\n\s*\)\s*\n?`).ReplaceAllString(content, "")
		_ = os.WriteFile(filepath.Join(baseDir, f), []byte(content), 0644)
		changed = true
	}
	return changed
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

// ── Orphaned interface fragment fix ──────────────────────────────────────────
//
// When an LLM response is truncated mid-way through a type declaration, the file
// can contain a top-level line that starts with ", " — the tail of a return-type
// expression whose opening was cut off. For example:
//
//   // PgxPool is the interface for pgxpool.Pool operations.
//   , error)                             ← truncated: "type PgxPool interface {\n\tExec(..., error)" was lost
//       Query(ctx context.Context, ...) (pgx.Rows, error)
//       Close()
//   }
//
// Go rejects `, error)` at package scope with "non-declaration statement outside
// function body". This fix detects the pattern and reconstructs the missing
// `type X interface {` declaration so the file at least parses. The resulting
// interface will be incomplete (missing the first method), but syntactically valid
// — gofmt accepts it and the LLM retry can fill in the rest.

// fixOrphanedInterfaceFragments scans Go files for top-level orphaned return
// fragments and reconstructs the missing interface declaration header.
func fixOrphanedInterfaceFragments(dir string, files []string) string {
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
		patched, changed := repairOrphanedFragments(string(data))
		if changed {
			_ = os.WriteFile(path, []byte(patched), 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("repaired truncated interface declaration(s) in %d file(s)", fixed)
}

// repairOrphanedFragments is the pure-function core of fixOrphanedInterfaceFragments.
// It scans src for lines at brace depth 0 that start with ", " (orphaned return
// fragments), extracts the interface name from the preceding doc comment, removes
// the fragment, and inserts "type <Name> interface {" in its place.
func repairOrphanedFragments(src string) (string, bool) {
	lines := strings.Split(src, "\n")
	depth := 0
	changed := false

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])

		// Track brace depth using naive counting.
		// This is safe here: we only act on depth==0 lines with the ", " prefix,
		// which cannot appear in valid Go at package scope.
		depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")

		// Only act on lines at package scope (depth 0) that look like an orphaned
		// return-type tail: starts with ", " followed by a non-space character.
		if depth != 0 {
			continue
		}
		if len(trimmed) < 2 || trimmed[0] != ',' || trimmed[1] != ' ' {
			continue
		}

		// Found an orphan. Determine the interface name from the preceding comment.
		name := extractInterfaceNameFromComment(lines, i)

		// Remove the orphaned line.
		lines = append(lines[:i], lines[i+1:]...)
		changed = true

		// Insert "type <Name> interface {" at position i (before the remaining body).
		header := "type " + name + " interface {"
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:i]...)
		newLines = append(newLines, header)
		newLines = append(newLines, lines[i:]...)
		lines = newLines

		// The inserted "{" increases depth to 1. Skip past the inserted header.
		depth = 1
		i++
	}

	if !changed {
		return src, false
	}
	return strings.Join(lines, "\n"), true
}

// extractInterfaceNameFromComment scans backwards from lineIdx looking for the
// nearest doc comment whose first word is an exported identifier (uppercase).
// Returns "UnknownInterface" when no qualifying comment is found.
func extractInterfaceNameFromComment(lines []string, lineIdx int) string {
	for j := lineIdx - 1; j >= 0; j-- {
		trimmed := strings.TrimSpace(lines[j])
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "//") {
			break // hit a non-comment line — stop
		}
		comment := strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
		if comment == "" {
			continue
		}
		word := strings.Fields(comment)[0]
		// Must be an exported identifier: starts with an uppercase ASCII letter.
		if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
			return word
		}
	}
	return "UnknownInterface"
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

// ── pgconn v4-vs-v5 import fix ──────────────────────────────────────────────
//
// LLMs frequently import "github.com/jackc/pgconn" (the standalone v4-era package)
// instead of "github.com/jackc/pgx/v5/pgconn" when building interfaces for pgx v5.
// The types are DIFFERENT — pgconn.CommandTag from the standalone package is not the
// same type as pgx/v5/pgconn.CommandTag — so *pgxpool.Pool does not satisfy interfaces
// that use the wrong pgconn import. This causes "wrong type for method Exec" errors
// that the LLM cannot easily self-diagnose because the type names look identical.
//
// This fix detects when go.mod contains pgx/v5 and rewrites any standalone pgconn
// imports to the v5 sub-package. It also removes "github.com/jackc/pgconn" from
// go.mod's require block to prevent go mod tidy from pulling in the wrong module.

func fixPgconnImportPath(dir string, files []string) string {
	// Only apply when pgx/v5 is in use.
	modulePath := ""
	hasPgxV5 := false
	current := dir
	for {
		data, err := os.ReadFile(filepath.Join(current, "go.mod"))
		if err == nil {
			content := string(data)
			if strings.Contains(content, "github.com/jackc/pgx/v5") {
				hasPgxV5 = true
				modulePath = current
			}
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	if !hasPgxV5 {
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
		// Replace standalone pgconn import with v5 sub-package.
		// Match both bare and aliased imports.
		newContent := strings.ReplaceAll(content,
			`"github.com/jackc/pgconn"`,
			`"github.com/jackc/pgx/v5/pgconn"`)
		if newContent != content {
			_ = os.WriteFile(path, []byte(newContent), 0644)
			fixed++
		}
	}

	// Also clean go.mod: remove standalone pgconn from require block.
	if modulePath != "" {
		goModPath := filepath.Join(modulePath, "go.mod")
		data, err := os.ReadFile(goModPath)
		if err == nil {
			content := string(data)
			// Remove "github.com/jackc/pgconn vX.Y.Z" lines from require blocks.
			pgconnReqRe := regexp.MustCompile(`(?m)^\s*github\.com/jackc/pgconn\s+v[^\n]+\n?`)
			newContent := pgconnReqRe.ReplaceAllString(content, "")
			if newContent != content {
				_ = os.WriteFile(goModPath, []byte(newContent), 0644)
				fixed++
			}
		}
	}

	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("fixed pgconn v4→v5 import path in %d file(s)", fixed)
}

// ── pgxmock hallucination fix ────────────────────────────────────────────────
//
// LLMs commonly hallucinate pgxmock API methods that don't exist:
//   - ExpectQueryRow → should be ExpectQuery (handles both Query and QueryRow)
//   - pgx.PgError   → should be pgconn.PgError (moved to pgconn sub-package in v5)
//
// These cause test compilation failures that trigger retries where the LLM
// often breaks the working implementation code trying to "fix" non-existent
// test issues. Fixing them deterministically prevents this cascade.

// pgxPoolWrongRe matches "pgxmock.PgxPool" only when NOT followed by "Iface"
// (the correct type). This avoids double-replacing already-correct names.
// Go regex lacks negative lookahead, so we match the trailing char and use
// ReplaceAllStringFunc to check context.
var pgxPoolWrongRe = regexp.MustCompile(`pgxmock\.PgxPool\b`)

// pgxPoolBareRe matches "pgxmock.Pool" (not preceded by "Pgx" which would be
// caught by pgxPoolWrongRe).
var pgxPoolBareRe = regexp.MustCompile(`pgxmock\.Pool\b`)

func fixPgxmockHallucinations(dir string, files []string) string {
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
		newContent := content

		// ExpectQueryRow → ExpectQuery (pgxmock matches both Query and QueryRow
		// via ExpectQuery; ExpectQueryRow does not exist).
		newContent = strings.ReplaceAll(newContent, ".ExpectQueryRow(", ".ExpectQuery(")
		newContent = strings.ReplaceAll(newContent, ".ExpectQueryRow\n", ".ExpectQuery\n")

		// pgx.PgError → pgconn.PgError (PgError lives in the pgconn sub-package).
		// Also handles pointer form: *pgx.PgError → *pgconn.PgError.
		if strings.Contains(newContent, "pgx.PgError") {
			newContent = strings.ReplaceAll(newContent, "pgx.PgError", "pgconn.PgError")
			// Ensure pgconn import is present (may need adding if only pgx was imported).
			if !strings.Contains(newContent, `"github.com/jackc/pgx/v5/pgconn"`) {
				// Insert pgconn import after pgx import line.
				newContent = strings.Replace(newContent,
					`"github.com/jackc/pgx/v5"`,
					"\"github.com/jackc/pgx/v5\"\n\t\"github.com/jackc/pgx/v5/pgconn\"",
					1)
			}
		}

		// pgxmock.PgxPool, pgxmock.PgxPoolMock, pgxmock.PgxMock, pgxmock.MockPool,
		// pgxmock.Pool → pgxmock.PgxPoolIface (the only exported pool interface in v4).
		// LLMs frequently hallucinate these non-existent type names.
		// Process longer strings first to avoid substring collisions
		// (e.g. "PgxPool" matching inside "PgxPoolIface").
		for _, wrongType := range []string{
			"pgxmock.PgxPoolMock",
			"pgxmock.PgxMock",
			"pgxmock.MockPool",
		} {
			if strings.Contains(newContent, wrongType) {
				newContent = strings.ReplaceAll(newContent, wrongType, "pgxmock.PgxPoolIface")
			}
		}
		// Handle "pgxmock.PgxPool" last — must not match "pgxmock.PgxPoolIface".
		// Use a regex with a word boundary to avoid substring collision.
		if pgxPoolWrongRe.MatchString(newContent) {
			newContent = pgxPoolWrongRe.ReplaceAllString(newContent, "pgxmock.PgxPoolIface")
		}
		// Handle bare "pgxmock.Pool" — also needs word boundary to avoid matching "pgxmock.PgxPoolIface".
		if pgxPoolBareRe.MatchString(newContent) {
			newContent = pgxPoolBareRe.ReplaceAllString(newContent, "pgxmock.PgxPoolIface")
		}

		// Fix hallucinated pgxmock error sentinels. The LLM sometimes writes
		// pgxmock.ErrNoRows, pgxmock.ErrConnectionDead, etc. — these don't exist.
		// ErrNoRows is pgx.ErrNoRows; there's no ErrConnectionDead at all.
		if strings.Contains(newContent, "pgxmock.Err") {
			newContent = strings.ReplaceAll(newContent, "pgxmock.ErrNoRows", "pgx.ErrNoRows")
			newContent = strings.ReplaceAll(newContent, "pgxmock.ErrClosed", "pgx.ErrNoRows")
			// Remove references to completely invented error sentinels by
			// replacing them with a generic errors.New() that compiles.
			pgxmockErrRe := regexp.MustCompile(`pgxmock\.(Err\w+)`)
			newContent = pgxmockErrRe.ReplaceAllStringFunc(newContent, func(match string) string {
				// Already handled above
				if match == "pgxmock.ErrNoRows" {
					return "pgx.ErrNoRows"
				}
				// Replace unknown pgxmock errors with errors.New("...")
				name := strings.TrimPrefix(match, "pgxmock.")
				return `errors.New("` + camelToMessage(name) + `")`
			})
		}

		// Fix SQL regex patterns in ExpectQuery/ExpectExec that use \s+ mixed
		// with literal spaces, causing "could not match actual sql" failures.
		// Pattern " \s+" requires 2+ whitespace chars but the implementation uses 1 space.
		// Simplest correct fix: replace all `\s+` with a single space since the
		// implementation SQL is always single-line with single spaces.
		if strings.HasSuffix(f, "_test.go") && strings.Contains(newContent, `\s+`) {
			newContent = strings.ReplaceAll(newContent, ` \s+`, " ")
			newContent = strings.ReplaceAll(newContent, `\s+ `, " ")
			newContent = strings.ReplaceAll(newContent, `\s+`, " ")
		}

		if newContent != content {
			_ = os.WriteFile(path, []byte(newContent), 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("fixed pgxmock hallucinations in %d file(s)", fixed)
}

// camelToMessage converts a CamelCase error name like "ErrConnectionDead" into
// a human-readable message like "connection dead" for use in errors.New().
func camelToMessage(name string) string {
	name = strings.TrimPrefix(name, "Err")
	var words []string
	start := 0
	for i := 1; i < len(name); i++ {
		if name[i] >= 'A' && name[i] <= 'Z' {
			words = append(words, strings.ToLower(name[start:i]))
			start = i
		}
	}
	words = append(words, strings.ToLower(name[start:]))
	return strings.Join(words, " ")
}

// ── shadowed testing.T fix ───────────────────────────────────────────────────
//
// LLMs frequently write table-driven tests with `for _, t := range tests` which
// shadows the *testing.T parameter `t`. This causes "t.Run undefined" errors
// because `t` is now the test-case struct, not the testing.T value.
//
// Fix: rename the loop variable from `t` to `tc` and update all references
// within the loop body (tc.name, tc.field, etc.).

// shadowedTRangeRe matches "for _, t := range" or "for i, t := range" in test files.
var shadowedTRangeRe = regexp.MustCompile(`(?m)^(\s*for\s+\S+,\s+)t(\s+:=\s+range\s+)`)

// shadowedTFieldRe matches "t.fieldName" inside a loop body where t was the loop var.
// We only replace accesses that are NOT method calls on *testing.T.
// testing.T methods: Run, Fatal, Fatalf, Error, Errorf, Log, Logf, Skip, Skipf, Helper, Cleanup, Parallel, etc.
var testingTMethods = map[string]bool{
	"Run": true, "Fatal": true, "Fatalf": true, "Error": true, "Errorf": true,
	"Log": true, "Logf": true, "Skip": true, "Skipf": true, "SkipNow": true,
	"Helper": true, "Cleanup": true, "Parallel": true, "Name": true,
	"Failed": true, "Skipped": true, "TempDir": true, "Deadline": true,
	"Setenv": true,
}

func fixShadowedTestingT(dir string, files []string) string {
	fixed := 0
	for _, f := range files {
		if !strings.HasSuffix(f, "_test.go") {
			continue
		}
		path := filepath.Join(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)

		// Only fix if there's actually a "for _, t := range" pattern.
		if !shadowedTRangeRe.MatchString(content) {
			continue
		}

		// Rename loop variable: "for _, t := range" → "for _, tc := range"
		newContent := shadowedTRangeRe.ReplaceAllString(content, "${1}tc${2}")

		if newContent == content {
			continue
		}

		// Now rename field accesses from t.xxx to tc.xxx in the region BETWEEN
		// the "for _, tc := range" line and the subtest function literal
		// "func(t *testing.T) {". Inside the subtest body, t is rebound to
		// the inner *testing.T and must NOT be renamed.
		//
		// Regions where t.field should become tc.field:
		//   for _, tc := range tests {     ← inRenameZone starts
		//     t.Run(tc.name, func(t *testing.T) {  ← inRenameZone ends at "func(t"
		//       t.Fatal(...)              ← t is *testing.T here, leave alone
		//     })
		//   }                             ← loop ends
		//
		// This handles the common pattern: the for-range header, the t.Run call
		// with tc.name, and any pre-subtest assertions that reference tc.field.
		lines := strings.Split(newContent, "\n")
		inLoop := false
		inRenameZone := false
		loopBraceDepth := 0

		for i, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Detect start of the renamed for loop.
			if strings.Contains(line, "tc := range") {
				inLoop = true
				inRenameZone = true
				loopBraceDepth = 0
			}

			if inLoop {
				loopBraceDepth += strings.Count(line, "{") - strings.Count(line, "}")

				if inRenameZone && strings.Contains(line, "t.") {
					result := replaceShadowedTDot(line)
					if result != line {
						lines[i] = result
					}
				}

				// Once we enter a "func(t *testing.T) {" subtest body,
				// stop renaming — t is now the inner *testing.T.
				if inRenameZone && subtestFuncRe.MatchString(trimmed) {
					inRenameZone = false
				}

				if loopBraceDepth <= 0 && trimmed == "}" {
					inLoop = false
					inRenameZone = false
				}
			}
		}

		newContent = strings.Join(lines, "\n")
		if newContent != content {
			_ = os.WriteFile(path, []byte(newContent), 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("fixed shadowed testing.T in %d test file(s)", fixed)
}

// subtestFuncRe matches "func(t *testing.T) {" — the subtest function literal
// where t is rebound to the inner *testing.T.
var subtestFuncRe = regexp.MustCompile(`func\s*\(\s*t\s+\*testing\.T\s*\)\s*\{`)

// ── undefined sentinel error fix ────────────────────────────────────────────
//
// LLMs frequently invent sentinel error names that don't exist (e.g. ErrConflict
// when errors.go only defines ErrAlreadyExists). This fix scans test files for
// "undefined: <pkg>.Err*" errors from the verifier output, then checks the
// package for similar sentinels and rewrites the reference.
//
// Mapping of commonly hallucinated sentinels to their standard counterparts:
var sentinelAliases = map[string][]string{
	// "not found" family
	"ErrNotExist":  {"ErrNotFound"},
	"ErrMissing":   {"ErrNotFound"},
	"ErrNoRows":    {"ErrNotFound"},
	"ErrNoRecord":  {"ErrNotFound"},
	"ErrNotExists": {"ErrNotFound"},

	// "already exists / duplicate / conflict" family — bidirectional
	"ErrConflict":                  {"ErrAlreadyExists", "ErrUniqueConstraintViolation", "ErrDuplicate"},
	"ErrDuplicate":                 {"ErrAlreadyExists", "ErrUniqueConstraintViolation", "ErrConflict"},
	"ErrAlreadyExist":              {"ErrAlreadyExists", "ErrUniqueConstraintViolation"},
	"ErrExists":                    {"ErrAlreadyExists", "ErrUniqueConstraintViolation"},
	"ErrAlreadyExists":             {"ErrUniqueConstraintViolation", "ErrDuplicate", "ErrConflict"},
	"ErrUniqueConstraintViolation": {"ErrAlreadyExists", "ErrDuplicate", "ErrConflict"},
	"ErrDuplicateKey":              {"ErrAlreadyExists", "ErrUniqueConstraintViolation", "ErrDuplicate"},
	"ErrUniqueViolation":           {"ErrUniqueConstraintViolation", "ErrAlreadyExists"},

	// "unauthorized / forbidden" family
	"ErrForbidden":       {"ErrUnauthorized", "ErrAccessDenied"},
	"ErrAccessDenied":    {"ErrUnauthorized", "ErrForbidden"},
	"ErrNotAuthorized":   {"ErrUnauthorized", "ErrForbidden"},
	"ErrPermissionDenied": {"ErrUnauthorized", "ErrForbidden"},

	// "invalid" family
	"ErrBadRequest":    {"ErrInvalidInput", "ErrValidation"},
	"ErrInvalid":       {"ErrInvalidInput", "ErrValidation"},
	"ErrValidation":    {"ErrInvalidInput", "ErrBadRequest"},
	"ErrInvalidInput":  {"ErrValidation", "ErrBadRequest"},
}

// pkgNameRe extracts the Go package name from a "package foo" declaration.
var pkgNameRe = regexp.MustCompile(`(?m)^package\s+(\w+)`)

// sentinelRefRe matches qualified sentinel references like "repository.ErrNotFound"
// or "domain.ErrAlreadyExists" — captures (pkgAlias, sentinelName).
var sentinelRefRe = regexp.MustCompile(`(\w+)\.(Err\w+)`)

// fixUndefinedSentinels rewrites references to undefined Err* sentinels in Go
// source files. It handles two cases:
//  1. Alias rewriting: when a known alias mapping exists (e.g. ErrAlreadyExists → ErrUniqueConstraintViolation)
//  2. Cross-package rewriting: when a sentinel is referenced via the wrong package
//     (e.g. domain.ErrNotFound when ErrNotFound is actually defined in repository)
func fixUndefinedSentinels(dir string, files []string) string {
	// Collect all defined Err* sentinels per package directory and per package name.
	definedByPkg := make(map[string]map[string]bool)    // pkgDir → set of sentinel names
	definedByName := make(map[string]map[string]bool)    // pkgName → set of sentinel names
	pkgDirToName := make(map[string]string)              // pkgDir → pkgName
	pkgNameToDirs := make(map[string][]string)            // pkgName → []pkgDir (for imports)
	for _, f := range files {
		if filepath.Ext(f) != ".go" || strings.HasSuffix(f, "_test.go") {
			continue
		}
		path := filepath.Join(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		pkgDir := filepath.Dir(f)
		if definedByPkg[pkgDir] == nil {
			definedByPkg[pkgDir] = make(map[string]bool)
		}
		// Extract package name.
		if m := pkgNameRe.FindStringSubmatch(string(data)); len(m) > 1 {
			pkgDirToName[pkgDir] = m[1]
			if definedByName[m[1]] == nil {
				definedByName[m[1]] = make(map[string]bool)
				pkgNameToDirs[m[1]] = nil
			}
			// Track unique dirs for this package name.
			found := false
			for _, d := range pkgNameToDirs[m[1]] {
				if d == pkgDir {
					found = true
					break
				}
			}
			if !found {
				pkgNameToDirs[m[1]] = append(pkgNameToDirs[m[1]], pkgDir)
			}
		}
		// Match "ErrXxx" in var declarations.
		for _, m := range topLevelVarRe.FindAllStringSubmatch(string(data), -1) {
			if strings.HasPrefix(m[1], "Err") {
				definedByPkg[pkgDir][m[1]] = true
				if pn := pkgDirToName[pkgDir]; pn != "" {
					definedByName[pn][m[1]] = true
				}
			}
		}
		for _, block := range varBlockRe.FindAllStringSubmatch(string(data), -1) {
			for _, m := range varBlockEntryRe.FindAllStringSubmatch(block[1], -1) {
				if strings.HasPrefix(m[1], "Err") {
					definedByPkg[pkgDir][m[1]] = true
					if pn := pkgDirToName[pkgDir]; pn != "" {
						definedByName[pn][m[1]] = true
					}
				}
			}
		}
	}

	// Flatten all defined sentinels across all packages for global lookup.
	allDefined := make(map[string]string) // sentinelName → first pkgName that defines it
	for pkgName, sentinels := range definedByName {
		for s := range sentinels {
			if _, exists := allDefined[s]; !exists {
				allDefined[s] = pkgName
			}
		}
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
		newContent := content

		// Build set of imported external package aliases for this file so we
		// don't rewrite references to external library sentinels (e.g. pgx.ErrNoRows).
		externalImports := extractImportAliases(content)

		// Phase 1: Alias rewriting — replace undefined sentinel names with defined aliases.
		// Only rewrite "pkg.ErrX" when pkg is a LOCAL package (not an external import).
		for undefined, candidates := range sentinelAliases {
			if !strings.Contains(newContent, "."+undefined) && !strings.Contains(newContent, " "+undefined+" ") {
				continue
			}
			// Only rewrite if the sentinel is actually undefined across all LOCAL packages.
			if _, globallyDefined := allDefined[undefined]; globallyDefined {
				continue
			}
			for _, candidate := range candidates {
				if _, exists := allDefined[candidate]; exists {
					// Replace only qualified references where the package is local.
					newContent = replaceLocalSentinelRefs(newContent, undefined, candidate, externalImports)
					break
				}
			}
		}

		// Phase 2: Cross-package rewriting — fix wrong package qualifiers.
		// E.g. "domain.ErrNotFound" → "repository.ErrNotFound" when ErrNotFound is in repository pkg.
		// Skip references where the package alias is an external import (e.g. pgx, errors).
		newContent = sentinelRefRe.ReplaceAllStringFunc(newContent, func(match string) string {
			parts := sentinelRefRe.FindStringSubmatch(match)
			if len(parts) < 3 {
				return match
			}
			pkgAlias, errName := parts[1], parts[2]
			// Skip external imports — never rewrite pgx.ErrNoRows, errors.Is, etc.
			if externalImports[pkgAlias] {
				return match
			}
			// If the referenced package defines this sentinel, no fix needed.
			if sentinels, ok := definedByName[pkgAlias]; ok && sentinels[errName] {
				return match
			}
			// The sentinel is not in the referenced package. Find which package defines it.
			if correctPkg, ok := allDefined[errName]; ok && correctPkg != pkgAlias {
				return correctPkg + "." + errName
			}
			return match
		})

		if newContent != content {
			_ = os.WriteFile(path, []byte(newContent), 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("fixed undefined sentinel errors in %d file(s)", fixed)
}

// importLineRe matches import lines: `"pkg/path"` or `alias "pkg/path"`.
var importLineRe = regexp.MustCompile(`(?:(\w+)\s+)?"([^"]+)"`)

// extractImportAliases parses import statements from Go source content and returns
// a set of package aliases for EXTERNAL (third-party / stdlib) packages.
// External packages are identified by having a domain name with a dot in the import
// path (e.g. "github.com/jackc/pgx/v5" → external, "monolith/internal/domain" → local).
// This allows the sentinel fixer to skip external library sentinels like pgx.ErrNoRows
// while still rewriting cross-package references between local packages.
func extractImportAliases(content string) map[string]bool {
	aliases := make(map[string]bool)
	lines := strings.Split(content, "\n")
	inImportBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "import (" {
			inImportBlock = true
			continue
		}
		if inImportBlock && trimmed == ")" {
			inImportBlock = false
			continue
		}
		isImport := inImportBlock || strings.HasPrefix(trimmed, "import ")
		if !isImport {
			continue
		}
		m := importLineRe.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		explicitAlias, pkgPath := m[1], m[2]
		// Only mark as external if the import path contains a dot (domain name).
		// Stdlib packages (like "context", "errors", "fmt") also have no dot but
		// are also external — include those too.
		// Local module imports (like "monolith/internal/domain") have no dot in the
		// first path segment.
		firstSeg := strings.SplitN(pkgPath, "/", 2)[0]
		isExternal := strings.Contains(firstSeg, ".") || !strings.Contains(pkgPath, "/")
		if !isExternal {
			continue
		}
		if explicitAlias != "" && explicitAlias != "_" {
			aliases[explicitAlias] = true
		} else {
			// Default alias is the last path segment.
			parts := strings.Split(pkgPath, "/")
			lastSeg := parts[len(parts)-1]
			// Strip version suffix (e.g. "v5" → use previous segment "pgx").
			if len(lastSeg) > 0 && lastSeg[0] == 'v' && len(parts) > 1 {
				allDigits := true
				for _, c := range lastSeg[1:] {
					if c < '0' || c > '9' {
						allDigits = false
						break
					}
				}
				if allDigits {
					lastSeg = parts[len(parts)-2]
				}
			}
			aliases[lastSeg] = true
		}
	}
	return aliases
}

// replaceLocalSentinelRefs replaces ".undefined" with ".replacement" in qualified
// references, but only when the qualifier is NOT an external import alias.
// E.g. replaces "repository.ErrConflict" → "repository.ErrAlreadyExists" but
// leaves "pgx.ErrNoRows" untouched.
func replaceLocalSentinelRefs(content, undefined, replacement string, externalImports map[string]bool) string {
	return sentinelRefRe.ReplaceAllStringFunc(content, func(match string) string {
		parts := sentinelRefRe.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		pkgAlias, errName := parts[1], parts[2]
		if externalImports[pkgAlias] {
			return match // external import — don't touch
		}
		if errName == undefined {
			return pkgAlias + "." + replacement
		}
		return match
	})
}

// tDotFieldRe matches "t." followed by an identifier.
var tDotFieldRe = regexp.MustCompile(`\bt\.(\w+)`)

// replaceShadowedTDot replaces t.field with tc.field for non-testing.T methods.
func replaceShadowedTDot(line string) string {
	return tDotFieldRe.ReplaceAllStringFunc(line, func(match string) string {
		// Extract field/method name after "t."
		name := match[2:] // skip "t."
		if testingTMethods[name] {
			return match // keep t.Run, t.Fatal, etc.
		}
		return "tc." + name
	})
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

// ── Bare module import fix ─────────────────────────────────────────────────
//
// LLMs sometimes use the app name (e.g. "monolith") as the import path prefix
// instead of the full module path from go.mod (e.g. "github.com/user/monolith").
// This produces "package monolith/internal/... is not in std" errors because Go
// tries to resolve it as a standard library package.
//
// Strategy: read the module path from go.mod, then scan every .go file for import
// paths that look like a suffix of the module path but missing the host prefix.

func fixBareModuleImports(dir string, files []string) string {
	modulePath := extractModulePathFromDir(dir)
	if modulePath == "" {
		return ""
	}

	// Split module path to find components. For "github.com/user/monolith",
	// the bare prefix the LLM might use is "monolith" (the last component).
	parts := strings.Split(modulePath, "/")
	if len(parts) < 2 {
		return "" // single-component module path — can't distinguish from stdlib
	}

	// Build a set of possible bare prefixes: the last component and any sub-path
	// suffix that isn't the full module path. For "github.com/user/monolith",
	// the LLM might write "monolith/internal/..." or "user/monolith/internal/...".
	type barePrefix struct {
		bare string
		full string
	}
	var prefixes []barePrefix
	for i := 1; i < len(parts); i++ {
		bare := strings.Join(parts[i:], "/")
		if bare != modulePath {
			prefixes = append(prefixes, barePrefix{bare: bare, full: modulePath})
		}
	}
	if len(prefixes) == 0 {
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
		newContent := content
		for _, p := range prefixes {
			// Match quoted import paths starting with the bare prefix followed by "/".
			// e.g. "monolith/internal/domain" → "github.com/user/monolith/internal/domain"
			old := `"` + p.bare + `/`
			repl := `"` + p.full + `/`
			newContent = strings.ReplaceAll(newContent, old, repl)

			// Also match the bare prefix as a standalone import (no sub-path).
			// e.g. "monolith" → "github.com/user/monolith"
			oldExact := `"` + p.bare + `"`
			replExact := `"` + p.full + `"`
			newContent = strings.ReplaceAll(newContent, oldExact, replExact)
		}
		if newContent != content {
			_ = os.WriteFile(path, []byte(newContent), 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("rewrote bare module imports in %d file(s) (module: %s)", fixed, modulePath)
}
