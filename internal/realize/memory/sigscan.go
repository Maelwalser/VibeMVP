package memory

import (
	"path/filepath"
	"strings"
)

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
	for _, line := range strings.Split(content, "\n") {
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
		result[name] = TypeEntry{Package: pkg, File: filePath}
	}
	return result
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
		if strings.HasPrefix(trimmed, "type ") {
			out = append(out, line)
			if strings.HasSuffix(trimmed, "{") {
				inTypeBody = true
				depth = 1
				out = append(out, "\t// ... [body omitted]")
			}
			continue
		}
		if inTypeBody {
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth <= 0 {
				inTypeBody = false
				depth = 0
				out = append(out, "}")
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
// declarations from TypeScript/TSX source, omitting implementation bodies.
func extractTSSignatures(content string) string {
	lines := strings.Split(content, "\n")
	var out []string

	inBlock := false
	depth := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if inBlock {
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth <= 0 {
				inBlock = false
				depth = 0
				out = append(out, "}")
			}
			continue
		}

		isDecl := strings.HasPrefix(trimmed, "interface ") ||
			strings.HasPrefix(trimmed, "export interface ") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "export type ") ||
			strings.HasPrefix(trimmed, "export enum ") ||
			strings.HasPrefix(trimmed, "enum ") ||
			strings.HasPrefix(trimmed, "export const ") ||
			strings.HasPrefix(trimmed, "export function ") ||
			strings.HasPrefix(trimmed, "export async function ") ||
			strings.HasPrefix(trimmed, "export default function ") ||
			strings.HasPrefix(trimmed, `import `)

		if isDecl {
			out = append(out, line)
			if strings.HasSuffix(trimmed, "{") {
				inBlock = true
				depth = 1
				out = append(out, "  // ... [body omitted]")
			}
			continue
		}

		if trimmed == "" && len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
	}

	return strings.Join(out, "\n")
}
