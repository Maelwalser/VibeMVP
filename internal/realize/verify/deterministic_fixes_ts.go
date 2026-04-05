package verify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// fixTypeScript applies deterministic, zero-LLM fixes to TypeScript/TSX files.
//
// Applied in order:
//  1. prettier or biome format   — enforces consistent style (semicolons, quotes,
//     trailing commas) so tsc errors are not obscured by formatting noise.
//  2. biome check --apply or eslint --fix — removes unused imports, which tsc
//     reports as TS6133 ("declared but its value is never read").
//  3. Regex fallback for TS6133  — when neither biome nor eslint is installed,
//     parse tsc output for TS6133 errors and remove the offending import specifiers.
//  4. Implicit-any annotation    — parse TS7006/TS7031 ("implicitly has an 'any'
//     type") from tsc output and add `: any` to the offending parameter.
//
// All steps are no-ops when no .ts/.tsx files are present.
func fixTypeScript(dir string, files []string) string {
	var msgs []string
	if m := fixTSFormat(dir, files); m != "" {
		msgs = append(msgs, m)
	}
	if m := fixTSUnusedImports(dir, files); m != "" {
		msgs = append(msgs, m)
	}
	if m := fixTSImplicitAny(dir, files); m != "" {
		msgs = append(msgs, m)
	}
	if len(msgs) == 0 {
		return ""
	}
	return strings.Join(msgs, "; ")
}

// fixTSFormat runs biome (preferred) or prettier to auto-format TypeScript/TSX files.
// Both tools are idempotent and safe to run unconditionally.
func fixTSFormat(dir string, files []string) string {
	tsFiles := filterByExt(dir, files, ".ts", ".tsx")
	if len(tsFiles) == 0 {
		return ""
	}

	if biomePath, err := exec.LookPath("biome"); err == nil {
		before := snapshotFiles(tsFiles)
		args := append([]string{"format", "--write"}, tsFiles...)
		cmd := exec.Command(biomePath, args...)
		cmd.Dir = dir
		_ = cmd.Run()
		n := countChanged(tsFiles, before)
		if n > 0 {
			return fmt.Sprintf("biome formatted %d TypeScript file(s)", n)
		}
	}

	if prettierPath, err := exec.LookPath("prettier"); err == nil {
		before := snapshotFiles(tsFiles)
		args := append([]string{"--write", "--log-level", "silent"}, tsFiles...)
		cmd := exec.Command(prettierPath, args...)
		cmd.Dir = dir
		_ = cmd.Run()
		n := countChanged(tsFiles, before)
		if n > 0 {
			return fmt.Sprintf("prettier formatted %d TypeScript file(s)", n)
		}
	}

	return ""
}

// fixTSUnusedImports removes unused TypeScript imports.
//
// Strategy (first tool that succeeds wins):
//  1. biome check --apply — handles TS unused-import lint rule noUnusedImports
//  2. eslint --fix        — handles @typescript-eslint/no-unused-vars
//  3. Regex fallback      — parse tsc TS6133 output directly
func fixTSUnusedImports(dir string, files []string) string {
	tsFiles := filterByExt(dir, files, ".ts", ".tsx")
	if len(tsFiles) == 0 {
		return ""
	}

	if biomePath, err := exec.LookPath("biome"); err == nil {
		before := snapshotFiles(tsFiles)
		args := append([]string{"check", "--apply"}, tsFiles...)
		cmd := exec.Command(biomePath, args...)
		cmd.Dir = dir
		_ = cmd.Run()
		n := countChanged(tsFiles, before)
		if n > 0 {
			return fmt.Sprintf("biome removed unused imports in %d TypeScript file(s)", n)
		}
	}

	if eslintPath, err := exec.LookPath("eslint"); err == nil {
		before := snapshotFiles(tsFiles)
		args := append([]string{
			"--fix",
			"--rule", `{"@typescript-eslint/no-unused-vars": "error"}`,
		}, tsFiles...)
		cmd := exec.Command(eslintPath, args...)
		cmd.Dir = dir
		_ = cmd.Run()
		n := countChanged(tsFiles, before)
		if n > 0 {
			return fmt.Sprintf("eslint --fix removed unused imports in %d TypeScript file(s)", n)
		}
	}

	return fixTSUnusedImportsFallback(dir, files)
}

// tsUnusedRe matches tsc TS6133: 'X' is declared but its value is never read.
// Format: <file>(line,col): error TS6133: 'Name' is declared but its value is never read.
var tsUnusedRe = regexp.MustCompile(`^(.+\.[tj]sx?)\((\d+),\d+\): error TS6133: '([^']+)' is declared but its value is never read`)

// fixTSUnusedImportsFallback removes TS6133-reported import specifiers without
// external tools by running tsc and parsing its output.
func fixTSUnusedImportsFallback(dir string, files []string) string {
	tscPath, err := exec.LookPath("tsc")
	if err != nil {
		return ""
	}

	tsconfigDirs := tsConfigDirs(dir, files)
	if len(tsconfigDirs) == 0 {
		tsconfigDirs = []string{"."}
	}

	type unusedSpec struct {
		file string
		line int
		name string
	}
	var specs []unusedSpec
	seen := make(map[string]bool)

	for _, d := range tsconfigDirs {
		absDir := filepath.Join(dir, d)
		cmd := exec.Command(tscPath, "--noEmit")
		cmd.Dir = absDir
		out, _ := cmd.CombinedOutput()
		for _, line := range strings.Split(string(out), "\n") {
			m := tsUnusedRe.FindStringSubmatch(strings.TrimSpace(line))
			if m == nil {
				continue
			}
			relFile, lineStr, name := m[1], m[2], m[3]
			key := relFile + ":" + lineStr + ":" + name
			if seen[key] {
				continue
			}
			seen[key] = true
			lineNum := 0
			fmt.Sscanf(lineStr, "%d", &lineNum)
			specs = append(specs, unusedSpec{file: relFile, line: lineNum, name: name})
		}
	}
	if len(specs) == 0 {
		return ""
	}

	type fileEdit struct {
		line int
		name string
	}
	byFile := make(map[string][]fileEdit)
	for _, s := range specs {
		path := s.file
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		byFile[path] = append(byFile[path], fileEdit{line: s.line, name: s.name})
	}

	applied := 0
	for path, edits := range byFile {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		changed := false
		for _, e := range edits {
			if e.line < 1 || e.line > len(lines) {
				continue
			}
			l := lines[e.line-1]
			trimmed := strings.TrimSpace(l)
			// Only touch import lines to avoid accidental variable removal.
			if !strings.HasPrefix(trimmed, "import ") && !strings.Contains(trimmed, "from ") {
				continue
			}
			patched := removeImportSpecifier(l, e.name)
			if patched != l {
				lines[e.line-1] = patched
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
	return fmt.Sprintf("removed unused TS import specifiers in %d file(s) (tsc fallback)", applied)
}

// removeImportSpecifier removes a single named specifier from a TypeScript import line.
// Handles: `import { Foo, Bar } from '...'` → `import { Bar } from '...'`
// When the specifier is the only one, blanks the entire line.
func removeImportSpecifier(line, name string) string {
	namedRe := regexp.MustCompile(`\{([^}]*)\}`)
	m := namedRe.FindStringSubmatchIndex(line)
	if m == nil {
		return line
	}
	inner := line[m[2]:m[3]]
	specifiers := strings.Split(inner, ",")
	var kept []string
	for _, s := range specifiers {
		if strings.TrimSpace(s) != name {
			kept = append(kept, s)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	return line[:m[2]] + strings.Join(kept, ",") + line[m[3]:]
}

// tsImplicitAnyRe matches tsc TS7006 and TS7031 errors:
//
//	Parameter 'x' implicitly has an 'any' type.
//	Binding element 'x' implicitly has an 'any' type.
var tsImplicitAnyRe = regexp.MustCompile(`^(.+\.[tj]sx?)\((\d+),(\d+)\): error TS70(?:06|31): (?:Parameter|Binding element) '([^']+)' implicitly has an 'any' type`)

// fixTSImplicitAny parses tsc output for TS7006/TS7031 errors and annotates the
// offending parameter with `: any`. This turns a hard compile error into valid
// (if loose) TypeScript that the LLM can tighten on the next retry.
func fixTSImplicitAny(dir string, files []string) string {
	tscPath, err := exec.LookPath("tsc")
	if err != nil {
		return ""
	}

	if len(filterByExt(dir, files, ".ts", ".tsx")) == 0 {
		return ""
	}

	tsconfigDirs := tsConfigDirs(dir, files)
	if len(tsconfigDirs) == 0 {
		tsconfigDirs = []string{"."}
	}

	type implicitAny struct {
		file string
		line int
		col  int
		name string
	}
	var anies []implicitAny
	seen := make(map[string]bool)

	for _, d := range tsconfigDirs {
		absDir := filepath.Join(dir, d)
		cmd := exec.Command(tscPath, "--noEmit")
		cmd.Dir = absDir
		out, _ := cmd.CombinedOutput()
		for _, line := range strings.Split(string(out), "\n") {
			m := tsImplicitAnyRe.FindStringSubmatch(strings.TrimSpace(line))
			if m == nil {
				continue
			}
			relFile, lineStr, colStr, name := m[1], m[2], m[3], m[4]
			key := relFile + ":" + lineStr + ":" + colStr
			if seen[key] {
				continue
			}
			seen[key] = true
			lineNum, colNum := 0, 0
			fmt.Sscanf(lineStr, "%d", &lineNum)
			fmt.Sscanf(colStr, "%d", &colNum)
			anies = append(anies, implicitAny{file: relFile, line: lineNum, col: colNum, name: name})
		}
	}
	if len(anies) == 0 {
		return ""
	}

	type fileEdit struct {
		line int
		col  int
		name string
	}
	byFile := make(map[string][]fileEdit)
	for _, a := range anies {
		path := a.file
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		byFile[path] = append(byFile[path], fileEdit{line: a.line, col: a.col, name: a.name})
	}

	applied := 0
	for path, edits := range byFile {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		changed := false
		for _, e := range edits {
			if e.line < 1 || e.line > len(lines) {
				continue
			}
			l := lines[e.line-1]
			// Insert `: any` after the parameter name when it is not already annotated.
			paramRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(e.name) + `\b(\s*)([,)=])`)
			if paramRe.MatchString(l) {
				patched := paramRe.ReplaceAllStringFunc(l, func(s string) string {
					nameEnd := strings.Index(s, e.name) + len(e.name)
					rest := strings.TrimSpace(s[nameEnd:])
					if strings.HasPrefix(rest, ":") {
						return s // already annotated
					}
					return e.name + ": any" + s[nameEnd:]
				})
				if patched != l {
					lines[e.line-1] = patched
					changed = true
				}
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
	return fmt.Sprintf("annotated implicit-any parameters in %d TypeScript file(s)", applied)
}
