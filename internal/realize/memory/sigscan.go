package memory

import (
	"path/filepath"
	"strings"
)

// ErrorSentinel records one exported Err* variable declaration extracted from
// a committed file. Used to build an explicit list of available error sentinels
// in downstream agent prompts, preventing agents from inventing sentinel names.
type ErrorSentinel struct {
	// Name is the sentinel variable name, e.g. "ErrNotFound".
	Name string
	// Package is the relative package directory, e.g. "internal/repository".
	Package string
	// File is the relative source file path, e.g. "internal/repository/errors.go".
	File string
}

// ExtractGoErrorSentinels returns all exported Err* variable declarations from
// a Go source file. These are typically sentinel errors defined as:
//
//	var ErrNotFound = errors.New("not found")
//
// or inside var blocks:
//
//	var (
//	    ErrNotFound = errors.New("not found")
//	    ErrAlreadyExists = errors.New("already exists")
//	)
//
// Test files are skipped.
func ExtractGoErrorSentinels(filePath, content string) []ErrorSentinel {
	lower := strings.ToLower(filePath)
	if !strings.HasSuffix(lower, ".go") || strings.HasSuffix(lower, "_test.go") {
		return nil
	}
	pkg := filepath.Dir(filePath)
	if pkg == "." {
		pkg = ""
	}

	var sentinels []ErrorSentinel
	lines := strings.Split(content, "\n")
	inVarBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track var ( ... ) blocks.
		if trimmed == "var (" {
			inVarBlock = true
			continue
		}
		if inVarBlock && trimmed == ")" {
			inVarBlock = false
			continue
		}

		// Match standalone: var ErrFoo = ...
		if strings.HasPrefix(trimmed, "var Err") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 && strings.HasPrefix(parts[1], "Err") {
				name := parts[1]
				if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
					sentinels = append(sentinels, ErrorSentinel{Name: name, Package: pkg, File: filePath})
				}
			}
			continue
		}

		// Match inside var block: ErrFoo = ...
		if inVarBlock && strings.HasPrefix(trimmed, "Err") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				name := parts[0]
				if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' && strings.HasPrefix(name, "Err") {
					sentinels = append(sentinels, ErrorSentinel{Name: name, Package: pkg, File: filePath})
				}
			}
		}
	}
	return sentinels
}

// ExtractErrorSentinels dispatches to the language-appropriate sentinel extractor.
// Returns nil for unrecognised file types.
func ExtractErrorSentinels(filePath, content string) []ErrorSentinel {
	lower := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(lower, ".go") && !strings.HasSuffix(lower, "_test.go"):
		return ExtractGoErrorSentinels(filePath, content)
	case strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx"):
		return extractTSErrorSentinels(filePath, content)
	case strings.HasSuffix(lower, ".py"):
		return extractPyErrorSentinels(filePath, content)
	default:
		return nil
	}
}

// extractTSErrorSentinels extracts exported error-like constants from TypeScript.
// Patterns matched:
//
//	export const ErrNotFound = new Error("not found")
//	export class NotFoundError extends Error { ... }
func extractTSErrorSentinels(filePath, content string) []ErrorSentinel {
	pkg := filepath.Dir(filePath)
	if pkg == "." {
		pkg = ""
	}
	var sentinels []ErrorSentinel
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		// export const ErrFoo = ...
		if strings.HasPrefix(trimmed, "export const Err") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 4 && parts[0] == "export" && parts[1] == "const" {
				name := parts[2]
				// Strip trailing = or : if present
				name = strings.TrimRight(name, "=:")
				name = strings.TrimSpace(name)
				if strings.HasPrefix(name, "Err") {
					sentinels = append(sentinels, ErrorSentinel{Name: name, Package: pkg, File: filePath})
				}
			}
		}
		// export class FooError extends Error
		if strings.HasPrefix(trimmed, "export class ") && strings.Contains(trimmed, "Error") && strings.Contains(trimmed, "extends") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				name := parts[2]
				if strings.Contains(name, "Error") {
					sentinels = append(sentinels, ErrorSentinel{Name: name, Package: pkg, File: filePath})
				}
			}
		}
	}
	return sentinels
}

// extractPyErrorSentinels extracts custom exception classes from Python.
// Pattern: class FooError(Exception): or class FooError(SomeBaseError):
func extractPyErrorSentinels(filePath, content string) []ErrorSentinel {
	pkg := filepath.Dir(filePath)
	if pkg == "." {
		pkg = ""
	}
	var sentinels []ErrorSentinel
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "class ") {
			continue
		}
		// class FooError(Exception):
		if strings.Contains(trimmed, "Error") || strings.Contains(trimmed, "Exception") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				name := strings.Split(parts[1], "(")[0]
				name = strings.TrimRight(name, ":")
				if name != "" && name[0] >= 'A' && name[0] <= 'Z' {
					sentinels = append(sentinels, ErrorSentinel{Name: name, Package: pkg, File: filePath})
				}
			}
		}
	}
	return sentinels
}

// ExtractGoExportedTypeNames returns a map of exported type name → TypeEntry for
// every exported type, interface, struct alias, or const group declared in the given
// Go source file. Test files (_test.go) are intentionally skipped.
//
// This is used to populate the cross-task type registry so downstream agents know
// which types are already defined and must not be redeclared.
func ExtractGoExportedTypeNames(filePath, content string) map[string]TypeEntry {
	lower := strings.ToLower(filePath)
	if !strings.HasSuffix(lower, ".go") || strings.HasSuffix(lower, "_test.go") {
		return nil
	}
	pkg := filepath.Dir(filePath)
	if pkg == "." {
		pkg = ""
	}
	result := make(map[string]TypeEntry)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "type ") {
			continue
		}
		// "type Foo struct {" / "type Foo interface {" / "type Foo = Bar"
		parts := strings.Fields(trimmed)
		if len(parts) < 2 {
			continue
		}
		name := parts[1]
		// Only export-worthy (capitalised) names.
		if len(name) == 0 || name[0] < 'A' || name[0] > 'Z' {
			continue
		}

		// Capture the full type body so downstream agents see method/field signatures.
		var defBuilder strings.Builder
		defBuilder.WriteString(line + "\n")
		if strings.HasSuffix(trimmed, "{") {
			depth := 1
			for j := i + 1; j < len(lines) && depth > 0; j++ {
				defBuilder.WriteString(lines[j] + "\n")
				depth += strings.Count(lines[j], "{") - strings.Count(lines[j], "}")
			}
		}

		result[name] = TypeEntry{
			Package:    pkg,
			File:       filePath,
			Definition: defBuilder.String(),
		}
	}
	return result
}

// ExtractConstructorSigs returns all exported constructor and factory function
// signature lines from a source file, operating on the full untruncated content.
// This is called at commit time so signatures are never lost to excerpt truncation.
//
// Recognised patterns per language:
//   - Go (.go, excluding _test.go): package-level and method-based funcs whose name
//     starts with New, Make, Create, Build, Open, or Must.
//   - TypeScript (.ts/.tsx): exported class constructors and create*/build* factories.
//   - Python (.py): class __init__ and top-level create_*/build_*/get_* functions.
func ExtractConstructorSigs(filePath, content string) []string {
	lower := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(lower, ".go") && !strings.HasSuffix(lower, "_test.go"):
		return extractGoCtorSigs(content)
	case strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx"):
		return extractTSCtorSigs(content)
	case strings.HasSuffix(lower, ".py"):
		return extractPyCtorSigs(content)
	default:
		return nil
	}
}

// assembleGoFuncSig handles multi-line function declarations by accumulating
// subsequent lines when the initial "func ..." line doesn't contain the opening "{".
// Returns the full assembled signature (without the body) and the number of
// additional lines consumed. This ensures signatures like:
//
//	func NewService(
//	    repo Repository,
//	    logger Logger,
//	) (*Service, error) {
//
// are captured as a single signature string.
func assembleGoFuncSig(lines []string, startIdx int) (sig string, extraLines int) {
	trimmed := strings.TrimSpace(lines[startIdx])
	if strings.Contains(trimmed, "{") {
		// Single-line: strip the opening brace.
		sig = strings.TrimSuffix(strings.TrimSuffix(trimmed, " {"), "{")
		return strings.TrimSpace(sig), 0
	}
	// Multi-line: accumulate until we find "{" or a blank line (bail out).
	var b strings.Builder
	b.WriteString(trimmed)
	for j := startIdx + 1; j < len(lines); j++ {
		extra := strings.TrimSpace(lines[j])
		if extra == "" {
			break // blank line — stop accumulating
		}
		b.WriteString(" " + extra)
		extraLines = j - startIdx
		if strings.Contains(extra, "{") {
			assembled := b.String()
			assembled = strings.TrimSuffix(strings.TrimSuffix(assembled, " {"), "{")
			return strings.TrimSpace(assembled), extraLines
		}
	}
	// No opening brace found — return what we have (may be a forward declaration).
	return strings.TrimSpace(b.String()), extraLines
}

// extractGoCtorSigs extracts exported constructor/factory signatures from Go source.
// It handles both package-level funcs (func NewFoo) and method-based constructors
// (func (r *Repo) NewSomething), and recognises a wider set of prefixes than the
// older prompt-side extractor. Multi-line declarations are assembled into a single
// signature string.
func extractGoCtorSigs(content string) []string {
	lines := strings.Split(content, "\n")
	var sigs []string
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "func ") {
			continue
		}
		name := goFuncName(trimmed)
		if !isGoCtorName(name) {
			continue
		}
		sig, extra := assembleGoFuncSig(lines, i)
		sigs = append(sigs, sig)
		i += extra
	}
	return sigs
}

// goFuncName returns the bare function or method name from a "func ..." declaration.
// For a method receiver "func (r *Repo) Name(" it skips the receiver group.
func goFuncName(trimmed string) string {
	rest := strings.TrimPrefix(trimmed, "func ")
	if strings.HasPrefix(rest, "(") {
		end := strings.Index(rest, ")")
		if end < 0 {
			return ""
		}
		rest = strings.TrimSpace(rest[end+1:])
	}
	if idx := strings.Index(rest, "("); idx > 0 {
		return rest[:idx]
	}
	return rest
}

// isGoCtorName reports whether name is an exported constructor/factory identifier.
func isGoCtorName(name string) bool {
	if len(name) == 0 || name[0] < 'A' || name[0] > 'Z' {
		return false
	}
	for _, p := range []string{"New", "Make", "Create", "Build", "Open", "Must"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// extractTSCtorSigs extracts exported class constructors and factory function
// signatures from TypeScript/TSX source.
func extractTSCtorSigs(content string) []string {
	lines := strings.Split(content, "\n")
	var sigs []string
	inClass := false
	className := ""
	depth := 0

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])

		if (strings.HasPrefix(trimmed, "export class ") ||
			strings.HasPrefix(trimmed, "export default class ")) &&
			strings.Contains(trimmed, "{") {
			parts := strings.Fields(trimmed)
			for j, p := range parts {
				if p == "class" && j+1 < len(parts) {
					className = strings.TrimRight(parts[j+1], "{")
					break
				}
			}
			inClass = true
			depth = 1
			continue
		}
		if inClass {
			depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
			if depth <= 0 {
				inClass = false
				className = ""
				depth = 0
				continue
			}
			if strings.HasPrefix(trimmed, "constructor(") {
				sig := className + " — " + trimmed
				sig = strings.TrimSuffix(strings.TrimSuffix(sig, " {"), "{")
				sigs = append(sigs, sig)
			}
			continue
		}

		isFactory := strings.HasPrefix(trimmed, "export function create") ||
			strings.HasPrefix(trimmed, "export async function create") ||
			strings.HasPrefix(trimmed, "export function build") ||
			strings.HasPrefix(trimmed, "export async function build")
		if isFactory && strings.Contains(trimmed, "(") {
			sig := strings.TrimSuffix(strings.TrimSuffix(trimmed, " {"), "{")
			sigs = append(sigs, sig)
		}
	}
	return sigs
}

// extractPyCtorSigs extracts class __init__ signatures and top-level factory
// functions from Python source.
func extractPyCtorSigs(content string) []string {
	lines := strings.Split(content, "\n")
	var sigs []string
	inClass := false
	className := ""
	classIndent := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "class ") && strings.Contains(trimmed, ":") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				name := strings.Split(parts[1], "(")[0]
				name = strings.TrimSuffix(name, ":")
				className = name
				classIndent = len(line) - len(strings.TrimLeft(line, " \t"))
				inClass = true
			}
			continue
		}

		if inClass {
			if trimmed == "" {
				continue
			}
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if currentIndent <= classIndent && !strings.HasPrefix(trimmed, "#") {
				inClass = false
				className = ""
			}
			if inClass && strings.HasPrefix(trimmed, "def __init__(") {
				sig := className + ".__init__" + strings.TrimPrefix(trimmed, "def __init__")
				sigs = append(sigs, strings.TrimSuffix(sig, ":"))
				// Do NOT exit inClass — the class may have @classmethod factory methods
				// like create() or build() that are also constructor-like.
			}
			// Capture @classmethod factory methods within the class.
			if inClass {
				isClassFactory := strings.HasPrefix(trimmed, "def create") ||
					strings.HasPrefix(trimmed, "def build") ||
					strings.HasPrefix(trimmed, "async def create") ||
					strings.HasPrefix(trimmed, "async def build")
				if isClassFactory && strings.Contains(trimmed, "(") {
					sig := className + "." + strings.TrimPrefix(strings.TrimPrefix(trimmed, "async "), "def ")
					sigs = append(sigs, strings.TrimSuffix(sig, ":"))
				}
			}
			continue
		}

		isFactory := strings.HasPrefix(trimmed, "def create_") ||
			strings.HasPrefix(trimmed, "def build_") ||
			strings.HasPrefix(trimmed, "def get_") ||
			strings.HasPrefix(trimmed, "async def create_") ||
			strings.HasPrefix(trimmed, "async def build_")
		if isFactory && strings.Contains(trimmed, "(") {
			sigs = append(sigs, strings.TrimSuffix(trimmed, ":"))
		}
	}
	return sigs
}

// ExtractServiceMethodSigs returns exported method signatures on service/repository
// structs from Go source. Unlike ExtractConstructorSigs (which captures New*/Make*/etc.),
// this captures all exported methods with receivers — e.g.:
//
//	func (s *UserService) GetByEmail(ctx context.Context, email string) (*User, error)
//
// These are never truncated by the memory budget, ensuring handler tasks can generate
// compatible method calls even when file excerpts are budget-limited.
func ExtractServiceMethodSigs(filePath, content string) []string {
	lower := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(lower, ".go") && !strings.HasSuffix(lower, "_test.go"):
		// Extract only concrete method signatures (func (s *Svc) Method(...)).
		// Interface method signatures are handled separately by InterfaceContracts
		// and injected into their own prompt section. Mixing them here causes
		// ambiguity (bare "FindByEmail(...)" without the interface prefix).
		return extractGoServiceMethodSigs(content)
	case strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx"):
		return extractTSServiceMethodSigs(content)
	case strings.HasSuffix(lower, ".py"):
		return extractPyServiceMethodSigs(content)
	default:
		return nil
	}
}

// extractGoServiceMethodSigs extracts exported method signatures that have a receiver.
// Constructor signatures (New*/Make*/etc.) are excluded — they are handled separately
// by ExtractConstructorSigs. Multi-line declarations are assembled into a single string.
func extractGoServiceMethodSigs(content string) []string {
	lines := strings.Split(content, "\n")
	var sigs []string
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "func (") {
			continue
		}
		name := goFuncName(trimmed)
		// Only exported methods (capitalised), exclude constructors.
		if len(name) == 0 || name[0] < 'A' || name[0] > 'Z' {
			continue
		}
		if isGoCtorName(name) {
			continue
		}
		sig, extra := assembleGoFuncSig(lines, i)
		sigs = append(sigs, sig)
		i += extra
	}
	return sigs
}

// extractTSServiceMethodSigs extracts public method signatures from exported TypeScript classes.
func extractTSServiceMethodSigs(content string) []string {
	lines := strings.Split(content, "\n")
	var sigs []string
	inClass := false
	depth := 0

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])

		if (strings.HasPrefix(trimmed, "export class ") ||
			strings.HasPrefix(trimmed, "export default class ")) &&
			strings.Contains(trimmed, "{") {
			inClass = true
			depth = 1
			continue
		}
		if inClass {
			depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
			if depth <= 0 {
				inClass = false
				depth = 0
				continue
			}
			// Skip constructor (handled by ExtractConstructorSigs), private, protected.
			if strings.HasPrefix(trimmed, "constructor(") ||
				strings.HasPrefix(trimmed, "private ") ||
				strings.HasPrefix(trimmed, "protected ") ||
				strings.HasPrefix(trimmed, "#") {
				continue
			}
			// Match public method signatures: "async methodName(" or "methodName("
			isMethod := false
			if strings.HasPrefix(trimmed, "async ") && strings.Contains(trimmed, "(") {
				isMethod = true
			} else if len(trimmed) > 0 && trimmed[0] >= 'a' && trimmed[0] <= 'z' && strings.Contains(trimmed, "(") {
				isMethod = true
			} else if strings.HasPrefix(trimmed, "public ") && strings.Contains(trimmed, "(") {
				isMethod = true
			}
			if isMethod {
				sig := strings.TrimSuffix(strings.TrimSuffix(trimmed, " {"), "{")
				sigs = append(sigs, strings.TrimSpace(sig))
			}
		}
	}
	return sigs
}

// extractSignatures returns a compact representation of a source file containing
// only type declarations, exported function/method signatures, and package/import
// lines — not implementation bodies. This is used to reduce downstream agent
// context to the minimum needed to stay type-consistent with upstream outputs.
//
// For unrecognised file types the first 500 characters are returned (schema-like
// formats such as YAML, JSON, and .tf are already compact enough).
func extractSignatures(path, content string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".go"):
		return extractGoSignatures(content)
	case strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx"):
		return extractTSSignatures(content)
	case strings.HasSuffix(lower, ".proto"):
		return content // proto definitions are already minimal
	default:
		if len(content) > 500 {
			return content[:500] + "\n// ... [truncated]"
		}
		return content
	}
}

// extractGoSignatures extracts package declarations, import blocks, exported type
// definitions, const blocks, and exported function/method signatures from Go source.
// Implementation bodies are replaced with a one-line placeholder to keep the output
// compact while preserving all structural information downstream agents need.
func extractGoSignatures(content string) string {
	lines := strings.Split(content, "\n")
	var out []string

	inImportBlock := false // inside `import ( ... )`
	inTypeBody := false    // inside a type struct/interface body
	inFuncBody := false    // inside a function body
	depth := 0             // brace depth

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Always keep package declaration.
		if strings.HasPrefix(trimmed, "package ") {
			out = append(out, line)
			continue
		}

		// Handle single-line import.
		if strings.HasPrefix(trimmed, `import "`) {
			out = append(out, line)
			continue
		}

		// Handle import block start.
		if trimmed == "import (" {
			inImportBlock = true
			out = append(out, line)
			continue
		}
		if inImportBlock {
			out = append(out, line)
			if trimmed == ")" {
				inImportBlock = false
			}
			continue
		}

		// Handle type declarations (struct / interface / alias).
		// Struct/interface bodies are preserved in full — field declarations are
		// part of the type signature and downstream agents must see them to know
		// what fields exist. Only function bodies are stripped.
		if strings.HasPrefix(trimmed, "type ") {
			out = append(out, line)
			if strings.HasSuffix(trimmed, "{") {
				inTypeBody = true
				depth = 1
			}
			continue
		}
		if inTypeBody {
			out = append(out, line) // keep field/method declarations
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth <= 0 {
				inTypeBody = false
				depth = 0
			}
			continue
		}

		// Handle var blocks (sentinel errors, package-level vars).
		if strings.HasPrefix(trimmed, "var ") {
			out = append(out, line)
			if trimmed == "var (" {
				for i++; i < len(lines); i++ {
					out = append(out, lines[i])
					if strings.TrimSpace(lines[i]) == ")" {
						break
					}
				}
			}
			continue
		}

		// Handle const blocks.
		if strings.HasPrefix(trimmed, "const ") {
			out = append(out, line)
			if strings.HasSuffix(trimmed, "(") || trimmed == "const (" {
				// multi-line const block
				for i++; i < len(lines); i++ {
					out = append(out, lines[i])
					if strings.TrimSpace(lines[i]) == ")" {
						break
					}
				}
			}
			continue
		}

		// Handle exported function/method signatures (keep signature, skip body).
		if strings.HasPrefix(trimmed, "func ") && !inFuncBody {
			out = append(out, line)
			if strings.HasSuffix(trimmed, "{") {
				inFuncBody = true
				depth = 1
				out = append(out, "\t// ... [body omitted]")
			}
			continue
		}
		if inFuncBody {
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth <= 0 {
				inFuncBody = false
				depth = 0
				out = append(out, "}")
			}
			continue
		}

		// Keep blank lines between declarations for readability.
		if trimmed == "" && len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
	}

	return strings.Join(out, "\n")
}

// extractTSSignatures extracts interface, type alias, and exported function
// declarations from TypeScript/TSX source. Interface and type bodies are
// preserved in full (field declarations are part of the signature). Only
// function implementation bodies are stripped.
func extractTSSignatures(content string) string {
	lines := strings.Split(content, "\n")
	var out []string

	// inTypeBlock: inside an interface/type/enum body — keep all lines
	// inFuncBlock: inside a function body — strip lines
	inTypeBlock := false
	inFuncBlock := false
	depth := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if inTypeBlock {
			out = append(out, line) // preserve field declarations
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth <= 0 {
				inTypeBlock = false
				depth = 0
			}
			continue
		}

		if inFuncBlock {
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth <= 0 {
				inFuncBlock = false
				depth = 0
				out = append(out, "}")
			}
			continue
		}

		isTypeDecl := strings.HasPrefix(trimmed, "interface ") ||
			strings.HasPrefix(trimmed, "export interface ") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "export type ") ||
			strings.HasPrefix(trimmed, "export enum ") ||
			strings.HasPrefix(trimmed, "enum ")

		isFuncDecl := strings.HasPrefix(trimmed, "export function ") ||
			strings.HasPrefix(trimmed, "export async function ") ||
			strings.HasPrefix(trimmed, "export default function ")

		isOther := strings.HasPrefix(trimmed, "export const ") ||
			strings.HasPrefix(trimmed, `import `)

		if isTypeDecl {
			out = append(out, line)
			if strings.HasSuffix(trimmed, "{") {
				inTypeBlock = true
				depth = 1
			}
			continue
		}

		if isFuncDecl {
			out = append(out, line)
			if strings.HasSuffix(trimmed, "{") {
				inFuncBlock = true
				depth = 1
				out = append(out, "  // ... [body omitted]")
			}
			continue
		}

		if isOther {
			out = append(out, line)
			continue
		}

		if trimmed == "" && len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
	}

	return strings.Join(out, "\n")
}

// extractPyServiceMethodSigs extracts public method signatures from Python classes.
// Excludes __init__, __str__, and other dunder methods — only captures business-logic
// methods that downstream tasks need to call.
func extractPyServiceMethodSigs(content string) []string {
	lines := strings.Split(content, "\n")
	var sigs []string
	inClass := false
	className := ""
	classIndent := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "class ") && strings.Contains(trimmed, ":") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				name := strings.Split(parts[1], "(")[0]
				name = strings.TrimRight(name, ":")
				className = name
				classIndent = len(line) - len(strings.TrimLeft(line, " \t"))
				inClass = true
			}
			continue
		}

		if inClass {
			if trimmed == "" {
				continue
			}
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if currentIndent <= classIndent && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "@") {
				inClass = false
				className = ""
				continue
			}

			// Skip dunder methods, private methods, and constructors.
			isMethod := strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ")
			if !isMethod {
				continue
			}
			methodLine := strings.TrimPrefix(strings.TrimPrefix(trimmed, "async "), "def ")
			methodName := strings.Split(methodLine, "(")[0]
			if strings.HasPrefix(methodName, "_") {
				continue // skip private and dunder methods
			}

			sig := className + "." + strings.TrimSuffix(trimmed, ":")
			sigs = append(sigs, sig)
		}
	}
	return sigs
}

// ExtractExportedTypeNames dispatches to the language-appropriate type registry
// extractor for non-Go languages. Returns nil for unrecognised file types.
// Go types are handled by ExtractGoExportedTypeNames (called separately).
func ExtractExportedTypeNames(filePath, content string) map[string]TypeEntry {
	lower := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx"):
		return extractTSExportedTypeNames(filePath, content)
	case strings.HasSuffix(lower, ".py"):
		return extractPyExportedTypeNames(filePath, content)
	default:
		return nil
	}
}

// extractTSExportedTypeNames extracts exported interface, type alias, enum, and class
// declarations from TypeScript source.
func extractTSExportedTypeNames(filePath, content string) map[string]TypeEntry {
	pkg := filepath.Dir(filePath)
	if pkg == "." {
		pkg = ""
	}
	result := make(map[string]TypeEntry)
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])

		isExported := strings.HasPrefix(trimmed, "export interface ") ||
			strings.HasPrefix(trimmed, "export type ") ||
			strings.HasPrefix(trimmed, "export class ") ||
			strings.HasPrefix(trimmed, "export enum ") ||
			strings.HasPrefix(trimmed, "export default class ")

		if !isExported {
			continue
		}

		// Extract the type name — skip keywords until we find a non-keyword token.
		parts := strings.Fields(trimmed)
		keywords := map[string]bool{"export": true, "default": true, "interface": true, "type": true, "class": true, "enum": true, "abstract": true}
		var name string
		for _, p := range parts {
			if keywords[p] {
				continue
			}
			name = strings.TrimRight(p, "{=<")
			name = strings.TrimSpace(name)
			break
		}
		if name == "" {
			continue
		}

		var defBuilder strings.Builder
		defBuilder.WriteString(lines[i] + "\n")
		if strings.HasSuffix(trimmed, "{") {
			depth := 1
			for j := i + 1; j < len(lines) && depth > 0; j++ {
				defBuilder.WriteString(lines[j] + "\n")
				depth += strings.Count(lines[j], "{") - strings.Count(lines[j], "}")
			}
		}

		result[name] = TypeEntry{
			Package:    pkg,
			File:       filePath,
			Definition: defBuilder.String(),
		}
	}
	return result
}

// extractPyExportedTypeNames extracts class definitions (including dataclasses and
// Pydantic models) from Python source.
func extractPyExportedTypeNames(filePath, content string) map[string]TypeEntry {
	pkg := filepath.Dir(filePath)
	if pkg == "." {
		pkg = ""
	}
	result := make(map[string]TypeEntry)
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "class ") {
			continue
		}
		// Only top-level classes (no indentation).
		indent := len(lines[i]) - len(strings.TrimLeft(lines[i], " \t"))
		if indent > 0 {
			continue
		}

		parts := strings.Fields(trimmed)
		if len(parts) < 2 {
			continue
		}
		name := strings.Split(parts[1], "(")[0]
		name = strings.TrimRight(name, ":")
		if name == "" || name[0] < 'A' || name[0] > 'Z' {
			continue
		}

		var defBuilder strings.Builder
		defBuilder.WriteString(lines[i] + "\n")
		for j := i + 1; j < len(lines); j++ {
			jLine := lines[j]
			jTrimmed := strings.TrimSpace(jLine)
			if jTrimmed == "" {
				defBuilder.WriteString(jLine + "\n")
				continue
			}
			jIndent := len(jLine) - len(strings.TrimLeft(jLine, " \t"))
			if jIndent <= indent && !strings.HasPrefix(jTrimmed, "#") && !strings.HasPrefix(jTrimmed, "@") {
				break
			}
			defBuilder.WriteString(jLine + "\n")
		}

		result[name] = TypeEntry{
			Package:    pkg,
			File:       filePath,
			Definition: defBuilder.String(),
		}
	}
	return result
}

// extractGoInterfaceMethodSigs extracts method signatures from Go interface declarations.
// Returns signatures prefixed with the interface name: "InterfaceName.MethodName(params) returns".
func extractGoInterfaceMethodSigs(filePath, content string) []string {
	contracts := ExtractGoInterfaceContracts(filePath, content)
	var sigs []string
	for _, c := range contracts {
		for _, m := range c.Methods {
			sigs = append(sigs, c.InterfaceName+"."+m)
		}
	}
	return sigs
}

// InterfaceContract records one Go interface definition extracted from a committed file.
// Used to build a hard checklist for downstream implementation tasks so they match
// the exact method signatures defined in the plan task's interfaces.go.
type InterfaceContract struct {
	InterfaceName string
	Package       string
	File          string
	Methods       []string // full method signature lines, e.g. "FindByEmail(ctx context.Context, email string) (*User, error)"
}

// ExtractGoInterfaceContracts parses Go source for interface declarations and returns
// one InterfaceContract per exported interface found. Only processes .go files that are
// not test files. Uses line-by-line parsing consistent with the rest of sigscan.
func ExtractGoInterfaceContracts(filePath, content string) []InterfaceContract {
	lower := strings.ToLower(filePath)
	if !strings.HasSuffix(lower, ".go") || strings.HasSuffix(lower, "_test.go") {
		return nil
	}

	pkg := filepath.Dir(filePath)
	if pkg == "." {
		pkg = ""
	}

	lines := strings.Split(content, "\n")
	var contracts []InterfaceContract

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])

		// Match "type <Name> interface {" (exported interfaces only).
		if !strings.HasPrefix(trimmed, "type ") {
			continue
		}
		rest := strings.TrimPrefix(trimmed, "type ")
		spaceIdx := strings.IndexByte(rest, ' ')
		if spaceIdx < 0 {
			continue
		}
		name := rest[:spaceIdx]
		if len(name) == 0 || name[0] < 'A' || name[0] > 'Z' {
			continue // unexported
		}
		after := strings.TrimSpace(rest[spaceIdx+1:])
		if !strings.HasPrefix(after, "interface") {
			continue
		}
		// Check for opening brace on same or next line.
		if !strings.Contains(after, "{") {
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "{" {
				i++
			} else {
				continue
			}
		}

		// Collect method signatures until closing brace.
		var methods []string
		depth := 1
		for j := i + 1; j < len(lines) && depth > 0; j++ {
			line := strings.TrimSpace(lines[j])
			if line == "}" {
				depth--
				continue
			}
			if strings.HasSuffix(line, "{") {
				depth++
				continue
			}
			// Skip blank lines, comments, and embedded interfaces.
			if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
				continue
			}
			// A method signature line contains a parenthesis (params) and no "type " prefix.
			if strings.Contains(line, "(") && !strings.HasPrefix(line, "type ") {
				methods = append(methods, line)
			}
			i = j
		}

		if len(methods) > 0 {
			contracts = append(contracts, InterfaceContract{
				InterfaceName: name,
				Package:       pkg,
				File:          filePath,
				Methods:       methods,
			})
		}
	}
	return contracts
}
