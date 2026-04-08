package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/vibe-menu/internal/realize/dag"
	"github.com/vibe-menu/internal/realize/skills"
	"github.com/vibe-menu/internal/realize/verify"
)

// SystemPrompt builds the stable system prompt for a task kind.
// language is the backend language from the task payload (e.g. "Go", "TypeScript", "Python").
// When non-Go, a language adaptation preamble is injected so the Go-specific role
// descriptions are correctly translated to the target language's idioms and conventions.
// depsContext, if non-empty, is injected after the output format instructions
// to provide exact module versions and library API docs for the task's stack.
// The prompt is stable across retries so it benefits from prompt caching.
func SystemPrompt(kind dag.TaskKind, skillDocs []skills.Doc, depsContext, language string) string {
	var b strings.Builder

	b.WriteString(roleDescription(kind))
	b.WriteString("\n\n")

	// When the target language is not Go, inject an adaptation preamble that tells
	// the LLM to translate Go-specific patterns to the target language. This is more
	// robust than maintaining parallel role descriptions per language.
	if adaptation := languageAdaptation(language); adaptation != "" {
		b.WriteString(adaptation)
		b.WriteString("\n\n")
	}

	b.WriteString(outputFormatInstructions())

	if depsContext != "" {
		b.WriteString("\n")
		b.WriteString(depsContext)
	}

	if len(skillDocs) > 0 {
		b.WriteString("\n\n## Technology Skill Guides\n\n")
		for _, doc := range skillDocs {
			b.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", doc.Technology, doc.Content))
		}
	}

	return b.String()
}

// ReferenceContext builds the stable cross-task reference sections (type
// registry, constructor signatures, service methods, error sentinels,
// interface contracts) as a separate string suitable for a cached system block.
// These sections are stable across retries (they come from completed upstream
// tasks), so caching them separately from the user message saves ~30-40% on
// retries. Returns "" when there's no reference content to inject.
func ReferenceContext(ac *Context) string {
	var b strings.Builder

	modulePath := ac.Task.Payload.ModulePath

	if len(ac.InterfaceContracts) > 0 {
		b.WriteString("## Interface Contract — MUST IMPLEMENT EXACTLY\n\n")
		b.WriteString("These interfaces were defined by the plan task and are the **binding contract**.\n")
		b.WriteString("Your implementation MUST satisfy every method with the **exact** parameter types and return types shown.\n\n")
		for _, ic := range ac.InterfaceContracts {
			b.WriteString("### " + ic.InterfaceName + " (from: " + ic.File + ")\n```go\n")
			b.WriteString("type " + ic.InterfaceName + " interface {\n")
			for _, m := range ic.Methods {
				b.WriteString("\t" + m + "\n")
			}
			b.WriteString("}\n```\n\n")
		}
	}

	if len(ac.AllConstructors) > 0 {
		b.WriteString("## Critical Constructor Signatures\n\n")
		b.WriteString("Call these constructors/factories with the **exact** signature shown.\n\n")
		b.WriteString("```\n")
		prevFile := ""
		for _, c := range ac.AllConstructors {
			if c.File != prevFile {
				if prevFile != "" {
					b.WriteString("\n")
				}
				b.WriteString("// from: " + c.File + "\n")
				prevFile = c.File
			}
			b.WriteString(c.Signature + "\n")
		}
		b.WriteString("```\n\n")
	}

	if len(ac.AllServiceMethods) > 0 {
		b.WriteString("## Service Method Signatures\n\n")
		b.WriteString("Call these methods with the **exact** parameter and return types shown.\n\n")
		b.WriteString("```\n")
		prevFile := ""
		for _, m := range ac.AllServiceMethods {
			if m.File != prevFile {
				if prevFile != "" {
					b.WriteString("\n")
				}
				b.WriteString("// from: " + m.File + "\n")
				prevFile = m.File
			}
			b.WriteString(m.Signature + "\n")
		}
		b.WriteString("```\n\n")
	}

	if len(ac.AllErrorSentinels) > 0 {
		b.WriteString("## Available Error Sentinels — USE ONLY THESE\n\n")
		byPkg := make(map[string][]string)
		var pkgOrder []string
		for _, s := range ac.AllErrorSentinels {
			importPath := s.Package
			if modulePath != "" && s.Package != "" {
				importPath = modulePath + "/" + s.Package
			}
			key := importPath
			if _, seen := byPkg[key]; !seen {
				pkgOrder = append(pkgOrder, key)
			}
			byPkg[key] = append(byPkg[key], s.Name)
		}
		for _, pkg := range pkgOrder {
			b.WriteString(fmt.Sprintf("**`%s`**:\n", pkg))
			for _, name := range byPkg[pkg] {
				b.WriteString(fmt.Sprintf("- `%s`\n", name))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// UserMessage builds the user turn message for an agent invocation.
func UserMessage(ac *Context) (string, error) {
	payloadJSON, err := json.Marshal(ac.Task.Payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Task: %s\n\n", ac.Task.Label))
	b.WriteString(fmt.Sprintf("Task ID: %s\nKind: %s\n", ac.Task.ID, ac.Task.Kind))
	modulePath := ac.Task.Payload.ModulePath
	outputDir := ac.Task.Payload.OutputDir
	if modulePath != "" {
		b.WriteString(fmt.Sprintf("Module path: **%s**  ← REQUIRED: use this EXACTLY as the Go module name in go.mod and ALL internal import paths. Never substitute a placeholder like \"github.com/your-org/\" or \"github.com/your-company/\".\n", modulePath))
		// Explicitly warn when output directory differs from module path.
		// Agents sometimes use the output directory name as the import prefix, which
		// causes "package X is not in std" errors (e.g. "backend/internal/domain"
		// when the correct import is "monolith/internal/domain").
		if outputDir != "" && outputDir != "." && outputDir != modulePath {
			b.WriteString(fmt.Sprintf("⚠ Output directory (**%s**) is the FILESYSTEM location only — it is NOT the Go module name. Import internal packages as `\"%s/internal/...\"`, NEVER as `\"%s/internal/...\"`.\n", outputDir, modulePath, outputDir))
		}
	}
	if desc := ac.Task.Payload.Description; desc != "" {
		b.WriteString(fmt.Sprintf("\n## Project Context\n\n%s\n", desc))
	}

	b.WriteString("\n## Manifest Payload\n\n```json\n")
	b.Write(payloadJSON)
	b.WriteString("\n```\n")

	// For data.schemas tasks, inject a structured attribute checklist and repository
	// operation summary so the agent cannot accidentally truncate domain fields or
	// miss deriving input/update structs from repository operations.
	if ac.Task.Kind == dag.TaskKindDataSchemas {
		if len(ac.Task.Payload.Domains) > 0 {
			b.WriteString("\n## Domain Attribute Checklist — ALL must be included\n\n")
			for _, domain := range ac.Task.Payload.Domains {
				b.WriteString(fmt.Sprintf("### %s entity — %d attributes\n\n", domain.Name, len(domain.Attributes)))
				for i, attr := range domain.Attributes {
					line := fmt.Sprintf("%d. `%s` (%s)", i+1, attr.Name, attr.Type)
					if attr.Constraints != "" {
						line += " — " + attr.Constraints
					}
					if attr.Default != "" {
						line += fmt.Sprintf(" [default: %s]", attr.Default)
					}
					b.WriteString(line + "\n")
				}
				b.WriteString("\n")
			}
		}
		// List repository operations that require input structs.
		hasOps := false
		for _, svc := range ac.Task.Payload.AllServices {
			for _, repo := range svc.Repositories {
				if len(repo.Operations) > 0 {
					if !hasOps {
						b.WriteString("\n## Repository Operations — derive input/update structs\n\n")
						hasOps = true
					}
					b.WriteString(fmt.Sprintf("### %s (entity: %s)\n\n", repo.Name, repo.EntityRef))
					for _, op := range repo.Operations {
						// Build operation line with derivation hint.
						filterBy := ""
						if len(op.FilterBy) > 0 {
							filterBy = fmt.Sprintf(", filter_by=[%s]", strings.Join(op.FilterBy, ", "))
						}
						desc := ""
						if op.Description != "" {
							desc = fmt.Sprintf(", description: %q", op.Description)
						}
						hint := ""
						switch {
						case op.OpType == "insert" || strings.HasPrefix(op.Name, "Create"):
							hint = fmt.Sprintf(" → generate `Create%sInput` struct", repo.EntityRef)
						case op.OpType == "update":
							// Use the operation name directly to avoid doubling
							// the "Update" prefix (e.g. "UpdateRefreshToken" → "UpdateRefreshTokenInput",
							// not "UpdateUpdateRefreshTokenInput").
							hint = fmt.Sprintf(" → generate `%sInput` struct", op.Name)
						}
						b.WriteString(fmt.Sprintf("- **%s** (op_type=%s%s%s)%s\n",
							op.Name, op.OpType, filterBy, desc, hint))
					}
					b.WriteString("\n")
				}
			}
		}
	}

	// Cross-task type registry: list all types already defined by upstream tasks.
	// Always injected (including retries) so agents can reference authoritative type
	// definitions when fixing type-mismatch or undefined-symbol errors.
	if len(ac.ExistingTypeRegistry) > 0 {
		b.WriteString("\n## Cross-Task Type Registry — DO NOT REDEFINE\n\n")
		b.WriteString("These types are already defined by upstream tasks. ")
		b.WriteString("**Import them from the listed package — do NOT redeclare them in your output.**\n")
		b.WriteString("Redefining these types will cause a `redeclared in this block` compilation error.\n\n")
		b.WriteString("**AUTHORITATIVE SOURCE**: The struct/interface definitions shown here are the COMPLETE, ")
		b.WriteString("untruncated type bodies from upstream tasks. If a type appears both here and in the ")
		b.WriteString("Shared Team Context below (which may be truncated), use THIS section as the source of ")
		b.WriteString("truth for field names, field types, and method signatures.\n\n")
		names := make([]string, 0, len(ac.ExistingTypeRegistry))
		for name := range ac.ExistingTypeRegistry {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			entry := ac.ExistingTypeRegistry[name]
			if entry.Package != "" {
				importPath := entry.Package
				if modulePath != "" {
					importPath = modulePath + "/" + entry.Package
				}
				b.WriteString(fmt.Sprintf("- `%s` — `%s` — import as `\"%s\"`\n", name, entry.File, importPath))
				if entry.Definition != "" {
					b.WriteString("  ```go\n")
					for _, dl := range strings.Split(strings.TrimRight(entry.Definition, "\n"), "\n") {
						b.WriteString("  " + dl + "\n")
					}
					b.WriteString("  ```\n")
				}
			}
		}
		b.WriteString("\n")
	}

	// Inject shared memory from completed upstream tasks.
	// Always inject full context — each LLM API call is stateless and cannot
	// "refer to earlier context" from attempt 0. Without the full type signatures,
	// struct field layouts, and interface definitions, the retry LLM invents wrong
	// type names and import paths, causing cascading failures.
	if len(ac.DependencyOutputs) > 0 {
		b.WriteString("\n## Shared Team Context\n\n")
		b.WriteString("The following type signatures and interfaces were generated by upstream agents.\n")
		b.WriteString("You MUST use the same type names, module paths, import paths, and data shapes —\n")
		b.WriteString("do not redefine anything already declared below.\n")
		b.WriteString("**Note:** These excerpts may be truncated for large files. If a type's fields appear ")
		b.WriteString("incomplete here, refer to the **Cross-Task Type Registry** section above for the ")
		b.WriteString("complete, untruncated struct/interface definition.\n\n")
		// Add language-specific import path guidance so agents don't confuse the
		// filesystem output directory with the language module/package namespace.
		if modulePath != "" {
			b.WriteString(fmt.Sprintf("**Go import paths:** All upstream files belong to module `%s`.\n", modulePath))
			b.WriteString(fmt.Sprintf("Import by prepending the module name: `internal/domain/user.go` → `import \"%s/internal/domain\"`,\n", modulePath))
			b.WriteString(fmt.Sprintf("`internal/repository/interfaces.go` → `import \"%s/internal/repository\"`.\n", modulePath))
			if outputDir != "" && outputDir != "." && outputDir != modulePath {
				b.WriteString(fmt.Sprintf("NEVER use the output directory `%s` as an import prefix — `\"%s/internal/...\"` is WRONG.\n", outputDir, outputDir))
			}
			b.WriteString("\n")
		}
		for _, dep := range ac.DependencyOutputs {
			b.WriteString(fmt.Sprintf("### From task: %s (%s)\n\n", dep.Label, dep.Kind))
			for _, f := range dep.Files {
				b.WriteString(fmt.Sprintf("#### `%s`\n\n", f.Path))
				b.WriteString("```")
				b.WriteString(fenceLanguage(f.Path))
				b.WriteString("\n")
				b.WriteString(f.Content)
				if f.Truncated {
					b.WriteString("\n// ... [truncated — use the types shown above]")
				}
				b.WriteString("\n```\n\n")
			}
		}
	}

	// NOTE: Interface contracts, constructor signatures, service method signatures,
	// and error sentinels are injected via ReferenceContext() into a separate cached
	// system block (for Claude) or appended to the system prompt (for other providers).
	// This saves ~30-40% on retry input costs since these sections are stable across
	// retries and benefit from prompt caching.

	// Inject deterministic bootstrap skeleton when available. This gives the LLM
	// a compilable starting point with correct constructor calls and import paths.
	if ac.BootstrapSkeleton != "" {
		b.WriteString("\n## Deterministic Bootstrap Skeleton\n\n")
		b.WriteString("The following skeleton was generated deterministically from upstream constructor signatures.\n")
		b.WriteString("It has the **correct import paths and constructor call sites**.\n")
		b.WriteString("Use this as your starting point — fill in the TODO sections, add middleware, env parsing,\n")
		b.WriteString("graceful shutdown, and any custom wiring. Do NOT change the constructor call argument counts.\n\n")
		b.WriteString("```go\n")
		b.WriteString(ac.BootstrapSkeleton)
		b.WriteString("```\n")
	}

	// Inject known cross-task build errors from incremental compilation. These are
	// advisory: the downstream task can compensate for known upstream issues.
	if ac.CrossTaskIssues != "" {
		b.WriteString("\n## Known Cross-Task Build Issues (Advisory)\n\n")
		b.WriteString("Incremental compilation detected these cross-task errors. ")
		b.WriteString("Your generated code should be compatible and avoid triggering similar issues.\n\n")
		b.WriteString("```\n")
		b.WriteString(ac.CrossTaskIssues)
		b.WriteString("\n```\n")
	}

	if ac.PreviousErrors != "" {
		b.WriteString("\n## Previous Attempt Failed — Verification Errors\n\n")
		b.WriteString("The previous code generation attempt failed the following verification checks. ")
		b.WriteString("Analyze the errors, fix the issues, and regenerate all files completely.\n\n")
		b.WriteString("```\n")
		b.WriteString(ac.PreviousErrors)
		b.WriteString("\n```\n")
		if hints := errorPatternHints(ac.PreviousErrors); hints != "" {
			b.WriteString(hints)
		}
	}

	b.WriteString("\nGenerate the complete files for this task now.")

	msg := b.String()

	// Progressive context pruning on retries: if the message exceeds 80% of
	// the model's context window, strip the least essential sections to make
	// room for the error context (which is the most important part on retry).
	if ac.PreviousErrors != "" && ac.MaxContextTokens > 0 {
		msg = pruneForRetry(msg, ac.MaxContextTokens)
	}

	return msg, nil
}

// estimateTokens returns a rough token count (4 chars ≈ 1 token).
func estimateTokens(s string) int {
	return len(s) / 4
}

// pruneForRetry progressively removes sections from the user message when
// it exceeds 80% of the context window. Sections are removed in order of
// least importance for error repair.
func pruneForRetry(msg string, maxTokens int) string {
	threshold := int(float64(maxTokens) * 0.8)

	if estimateTokens(msg) <= threshold {
		return msg
	}

	// Level 1: Remove bootstrap skeleton (LLM saw it on attempt 0).
	msg = removeSectionBetween(msg, "## Deterministic Bootstrap Skeleton", "## ")
	if estimateTokens(msg) <= threshold {
		return msg
	}

	// Level 2: Remove service method signatures (keep constructors + contracts).
	msg = removeSectionBetween(msg, "## Service Method Signatures", "## ")
	if estimateTokens(msg) <= threshold {
		return msg
	}

	// Level 3: Truncate PreviousErrors to first 50 lines.
	msg = truncateSection(msg, "## Previous Attempt Failed", 50)
	if estimateTokens(msg) <= threshold {
		return msg
	}

	// Level 4: Remove dependency output file bodies (keep only headers).
	msg = removeSectionBetween(msg, "## Shared Team Context", "## ")
	return msg
}

// removeSectionBetween removes everything from startMarker to the next
// section header (## ) or end of string. Returns the original if markers not found.
func removeSectionBetween(msg, startMarker, nextPrefix string) string {
	startIdx := strings.Index(msg, startMarker)
	if startIdx < 0 {
		return msg
	}
	// Find the next section after the start marker.
	rest := msg[startIdx+len(startMarker):]
	endIdx := strings.Index(rest, "\n"+nextPrefix)
	if endIdx < 0 {
		// Section goes to end — remove it entirely.
		return msg[:startIdx]
	}
	return msg[:startIdx] + rest[endIdx+1:]
}

// truncateSection limits the content within a section to maxLines lines
// after the section header.
func truncateSection(msg, sectionMarker string, maxLines int) string {
	idx := strings.Index(msg, sectionMarker)
	if idx < 0 {
		return msg
	}
	// Find the code block within the section.
	codeStart := strings.Index(msg[idx:], "```\n")
	if codeStart < 0 {
		return msg
	}
	codeStart += idx + 4 // skip "```\n"
	codeEnd := strings.Index(msg[codeStart:], "\n```")
	if codeEnd < 0 {
		return msg
	}
	codeEnd += codeStart

	codeBlock := msg[codeStart:codeEnd]
	lines := strings.Split(codeBlock, "\n")
	if len(lines) <= maxLines {
		return msg
	}
	truncated := strings.Join(lines[:maxLines], "\n") + "\n// ... [truncated — " + fmt.Sprintf("%d", len(lines)-maxLines) + " more error lines]"
	return msg[:codeStart] + truncated + msg[codeEnd:]
}

// fenceLanguage returns the markdown code fence language tag for a file path.
func fenceLanguage(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".go"):
		return "go"
	case strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx"):
		return "typescript"
	case strings.HasSuffix(lower, ".proto"):
		return "protobuf"
	case strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml"):
		return "yaml"
	case strings.HasSuffix(lower, ".json"):
		return "json"
	case strings.HasSuffix(lower, ".py"):
		return "python"
	case strings.HasSuffix(lower, ".tf"):
		return "hcl"
	default:
		return ""
	}
}

// errorPatternHints inspects the verification error output and returns targeted
// guidance for common, well-understood failure modes. Returns empty string when
// no patterns are recognised.
//
// Uses the structured CompilerError parser for Go errors so hint generation is
// resilient to compiler output format changes across Go versions.
func errorPatternHints(errors string) string {
	var matched []string

	// Use the structured error parser for precise symbol extraction.
	parsed := verify.ParseGoErrors(errors)

	// Extract specific missing method names from "does not implement" errors.
	for _, e := range verify.WithCode(parsed, "not_implemented") {
		// Extract concrete type and missing method from message.
		// Format: "*ConcreteType does not implement Interface (missing method MethodName)"
		missingMethodRe := regexp.MustCompile(`(\*?\w+) does not implement (\w+) \(missing method (\w+)\)`)
		if m := missingMethodRe.FindStringSubmatch(e.Message); m != nil {
			concrete, iface, method := m[1], m[2], m[3]
			matched = append(matched,
				fmt.Sprintf("- Interface not satisfied: `%s` is missing method `%s` required by `%s`. "+
					"Find the interface definition in the Shared Team Context (interfaces.go) and implement "+
					"**every** method listed — do not truncate or omit any. Add `%s` with the EXACT "+
					"signature shown in the interface.", concrete, method, iface, method))
		}
	}

	// Extract specific undefined symbol names from "undefined:" errors.
	seen := make(map[string]bool)
	for _, e := range verify.WithCode(parsed, "undefined") {
		sym := e.Symbol
		if sym == "" || seen[sym] {
			continue
		}
		seen[sym] = true
		matched = append(matched,
			fmt.Sprintf("- Symbol `%s` is undefined. Add a top-level import for the package that "+
				"exports `%s` (e.g. `\"net/http\"` for `http.*`, `\"context\"` for `context.*`, "+
				"`\"fmt\"` for `fmt.*`). Every referenced symbol must have a corresponding `import` "+
				"line in the same file — the compiler never resolves imports automatically.", sym, sym))
	}

	type pattern struct {
		needle string
		hint   string
	}
	patterns := []pattern{
		{
			"missing go.sum entry",
			"Go dependency issue: your go.mod `require` block is incomplete or uses wrong module paths. " +
				"The verifier runs `go mod tidy` automatically, but tidy needs the internet — ensure every " +
				"imported package appears in go.mod `require`. Double-check module paths (e.g. " +
				"`github.com/gofiber/fiber/v2`, not `github.com/gofiber/fiber`). " +
				"Regenerate go.mod with the exact, correct module paths for all imports in your generated files.",
		},
		{
			"cannot find module",
			"Module resolution failure: a package import path does not match any module in go.mod. " +
				"Verify the exact import path and version for every dependency, then include a complete go.mod.",
		},
		{
			"undefined:",
			"Undefined symbol: check that all referenced identifiers are either declared in the same " +
				"package or imported. Every file must have a complete import block — the compiler does not " +
				"auto-resolve missing imports. See the specific symbol hints above for targeted guidance.",
		},
		{
			"cannot use",
			"Type mismatch: a value is being passed where an incompatible type is expected. " +
				"Verify struct field types and function signatures are consistent across files. " +
				"Pay special attention to pointer vs value types and interface implementations.",
		},
		{
			"declared and not used",
			"Unused variable: Go forbids declared-but-unused variables. Remove or use every declared variable.",
		},
		{
			"no new variables on left side of :=",
			"Short declaration error: you used := but ALL variables on the left are already declared. " +
				"Use = instead of := when reassigning existing variables. " +
				"WRONG: err := db.Close() (when err was declared earlier). " +
				"CORRECT: err = db.Close(). " +
				"Only use := when at least one variable on the left is NEW.",
		},
		{
			"imported and not used",
			"Unused import: Go forbids imported-but-unused packages. Remove every import that is not referenced.",
		},
		{
			"does not implement",
			"Interface not satisfied: a concrete type is missing one or more required methods. " +
				"Find the interface definition in Shared Team Context and implement EVERY method listed — " +
				"never truncate or omit any. See the specific missing-method hints above for details.",
		},
		{
			"syntax error",
			"Syntax error: the generated Go code is not valid. Check for mismatched braces, missing commas " +
				"in struct literals, or incomplete function bodies. Regenerate the affected files in full.",
		},
		{
			"non-declaration statement outside function body",
			"Truncated interface declaration: a fragment like `, error)` at the top level means the " +
				"`type InterfaceName interface {` opening line and first method signature were cut off. " +
				"Regenerate the affected file in FULL:\n" +
				"  1. Every interface MUST start with `type X interface {`\n" +
				"  2. ALL methods must have COMPLETE signatures (no truncation, no `...` placeholders)\n" +
				"  3. interfaces.go contains ONLY interface type declarations — " +
				"NO `func` implementations, NO struct bodies, NO constructor functions.\n" +
				"Use the EXACT method signatures and return types from the " +
				"'Dependency & API Reference' section — never substitute `interface{}` " +
				"for a library-specific return type.",
		},
		{
			"gofmt",
			"Formatting issue: one or more files are not gofmt-clean. Use standard Go indentation (tabs) " +
				"and ensure the generated source passes `gofmt -l` without listing any files.",
		},
		{
			"has no field named",
			"Struct field mismatch: you are accessing a field that does not exist on the type. " +
				"Check the exact struct definition in the Shared Team Context above — do not guess field names. " +
				"Common cause: using a shorthand like resp.Email when the actual field is nested (resp.User.Email), " +
				"or the struct was renamed by an upstream task.",
		},
		{
			"not enough return values",
			"Return arity mismatch: a function returns fewer values than the caller expects. " +
				"Check the Critical Constructor Signatures and Shared Team Context — a New* constructor " +
				"likely changed to return an additional error value. Update the call site to handle all return values.",
		},
		{
			"too many return values",
			"Return arity mismatch: a function returns more values than the caller expects. " +
				"Check the function's actual signature in the Shared Team Context above.",
		},
		{
			"too many arguments in call to",
			"Argument count mismatch: you are passing MORE arguments than the function accepts. " +
				"Check the EXACT constructor/function signature in 'Critical Constructor Signatures'. " +
				"Common causes: (1) passing an extra config string, logger, or dependency not in the real " +
				"constructor signature; (2) using a different version of the constructor. " +
				"Use ONLY the parameters listed — do not add extras that are not shown.",
		},
		{
			"not enough arguments in call to",
			"Argument count mismatch: you are passing FEWER arguments than the function requires. " +
				"Check the EXACT constructor/function signature in 'Critical Constructor Signatures'. " +
				"The function likely requires more parameters than you are providing.",
		},
		{
			"array type uuid.UUID",
			"UUID type mismatch: uuid.UUID is an array type ([16]byte) and cannot be used directly as string. " +
				"Convert to string using the .String() method: `user.ID.String()` instead of `user.ID`. " +
				"This applies everywhere a uuid.UUID value is passed to a function expecting string, " +
				"including JWT claims (`claims[\"sub\"] = user.ID.String()`), DB queries, and helper functions.",
		},
		{
			"variable of type bool) as string",
			"Type mismatch in struct literal: a bool field is being assigned where string is expected. " +
				"PREFERRED FIX: change the response struct field type to bool — Go's encoding/json handles bool correctly. " +
				"If you must keep string: convert with strconv.FormatBool(value). " +
				"Response structs should mirror domain field types (bool for bool, time.Time for timestamps) — " +
				"do NOT convert every field to string.",
		},
		{
			`variable of struct type "time".Time) as string`,
			"Type mismatch in struct literal: a time.Time field is being assigned where string is expected. " +
				"PREFERRED FIX: change the response struct field type to time.Time — encoding/json serializes " +
				"time.Time as RFC 3339 automatically, which is correct for API responses. " +
				"If you must keep string: convert with value.Format(time.RFC3339).",
		},
		{
			"unexpected keyword import",
			"Syntax error: an import statement appeared inside a function body — this is not valid Go. " +
				"Import statements must appear at the top of the file in the import block, never inside functions. " +
				"Move any needed imports to the top-level import block and remove them from inside functions.",
		},
		{
			"has no field or method Run",
			"Shadowed *testing.T: in a table-driven test loop you used `for _, t := range tests` — " +
				"this shadows the *testing.T parameter so `t.Run(...)` fails because `t` is now the " +
				"test-case struct. Fix: rename the loop variable to `tc`: " +
				"`for _, tc := range tests { t.Run(tc.name, func(t *testing.T) { ... }) }`. " +
				"Use `tc.` to access struct fields and `t.` for testing.T methods (t.Run, t.Fatal, etc.).",
		},
		{
			"wrong type for method",
			"Interface type mismatch: a method's return or parameter type comes from the WRONG package. " +
				"The most common cause is importing \"github.com/jackc/pgconn\" (standalone v4-era package) " +
				"instead of \"github.com/jackc/pgx/v5/pgconn\" (bundled with pgx v5). These packages define " +
				"DIFFERENT types with the SAME name (e.g. pgconn.CommandTag). When pgx v5 is in use, " +
				"ALL pgconn types must come from \"github.com/jackc/pgx/v5/pgconn\". " +
				"Fix: change the import in interfaces.go from \"github.com/jackc/pgconn\" to " +
				"\"github.com/jackc/pgx/v5/pgconn\" and remove the standalone pgconn from go.mod.",
		},
		{
			"ConnectTimeout undefined",
			"pgxpool v5 API change: pgxpool.Config does NOT have a ConnectTimeout field — it was removed in v5. " +
				"Remove any `config.ConnectTimeout = ...` line. To limit connection time, pass a context with " +
				"deadline to pgxpool.NewWithConfig: `ctx, cancel := context.WithTimeout(ctx, 10*time.Second)`. " +
				"Valid pool config fields: MaxConns, MinConns, MaxConnLifetime, MaxConnIdleTime, HealthCheckPeriod.",
		},
		{
			"multiple-value",
			"Multi-return handling: a function returning multiple values (e.g. value, error) is being " +
				"used in a single-value context. Assign all return values explicitly: " +
				"svc, err := NewService(...); if err != nil { ... }",
		},
		{
			"cannot find module providing",
			"Unknown import path: the imported package does not match any module in go.mod. " +
				"Verify you are using the exact module path from the 'Module path:' field at the top of this task, " +
				"not a placeholder like 'github.com/your-org/' or 'github.com/your-company/'. " +
				"All internal packages must be imported as '{module_path}/internal/...'.",
		},
		{
			"does not implement context.Context",
			"Context interface error: you created a mock context using an interface literal " +
				"(e.g. interface{Value(any) any}{}) that does not implement the full context.Context " +
				"interface (needs Deadline, Done, Err, Value). " +
				"FIX: use context.WithValue(context.Background(), key, value) to set values on a real context. " +
				"Never construct mock contexts with interface literals.",
		},
		{
			"is not in std",
			"Module path resolution error: Go is looking for a local package in the standard library, " +
				"which means the import path is INCOMPLETE — it uses a bare app name instead of the full " +
				"module path. WRONG: import \"monolith/internal/repository\" (bare name → Go looks in stdlib). " +
				"CORRECT: import \"<full-module-path>/internal/repository\" where <full-module-path> is the " +
				"EXACT value from the 'Module path:' field (e.g. \"github.com/user/monolith/internal/repository\"). " +
				"Fix EVERY internal import to use the full module path from go.mod.",
		},
		{
			"second argument to errors.As should not be *error",
			"errors.As type error: the second argument must be a pointer to a CONCRETE error type, " +
				"not *error. WRONG: var err error; errors.As(e, &err). " +
				"CORRECT: var pgErr *pgconn.PgError; errors.As(e, &pgErr). " +
				"Use the specific error struct type you want to extract.",
		},
		{
			"could not match actual sql",
			"pgxmock SQL mismatch: the SQL string in ExpectQuery/ExpectExec does not match the " +
				"SQL executed by the implementation. The most common cause is using regex metacharacters " +
				"(\\s+, $1, .+) in the expected SQL string — pgxmock compiles it as a regex. " +
				"FIX: define SQL as a package-level const shared between implementation and test, " +
				"then use regexp.QuoteMeta(sqlConst) in ExpectQuery. " +
				"Example: `mock.ExpectQuery(regexp.QuoteMeta(findByEmailSQL)).WithArgs(...)` " +
				"Make sure to import \"regexp\" in the test file.",
		},
		{
			"unmet mock expectations",
			"pgxmock unmet expectations: one or more mock.Expect* calls were set up but never " +
				"matched by the code under test. This usually means the SQL or arguments in the " +
				"expectation differ from what the implementation actually calls. " +
				"Use the EXACT same SQL const from the implementation in your test expectations, " +
				"wrapped in regexp.QuoteMeta() to escape regex metacharacters.",
		},
		// ── TypeScript / Node.js patterns ────────────────────────────────────
		{
			"Cannot find module",
			"TypeScript module not found: the imported module path does not resolve. " +
				"Check that the module is listed in package.json dependencies with the correct name " +
				"and that tsconfig.json paths are configured correctly. For internal imports, use " +
				"relative paths (./service/user.service) or tsconfig path aliases (@/...).",
		},
		{
			"TS2307",
			"TypeScript cannot find module (TS2307): verify the import path and that the " +
				"package is installed. For @types/ packages, add them to devDependencies.",
		},
		{
			"TS2345",
			"TypeScript type mismatch (TS2345): argument type does not match parameter type. " +
				"Check the function signature in the Shared Team Context and use the exact types shown.",
		},
		{
			"TS2339",
			"TypeScript property does not exist (TS2339): you are accessing a property that " +
				"is not defined on the type. Check the type/interface definition in Shared Team Context.",
		},
		{
			"Cannot redeclare block-scoped variable",
			"TypeScript re-declaration error: a variable declared with let/const is being re-declared " +
				"in the same scope. Use assignment (=) instead of a new declaration (let/const) when " +
				"the variable already exists. WRONG: let err = await fn1(); let err = await fn2(); " +
				"CORRECT: let err = await fn1(); err = await fn2();",
		},
		// ── Python patterns ──────────────────────────────────────────────────
		{
			"ModuleNotFoundError",
			"Python module not found: the imported module is not installed or the path is wrong. " +
				"Check that the module is in requirements.txt / pyproject.toml and the import path " +
				"matches the package structure (e.g. from app.domain.user import User).",
		},
		{
			"ImportError",
			"Python import error: the module exists but the name cannot be imported from it. " +
				"Check that the class/function name is spelled correctly and is exported from __init__.py.",
		},
		{
			"NameError",
			"Python undefined name: a variable or function is used before being defined or imported. " +
				"Add the missing import statement at the top of the file.",
		},
		{
			"IndentationError",
			"Python indentation error: inconsistent use of tabs and spaces. " +
				"Use 4 spaces per indentation level consistently throughout the file.",
		},
	}

	for _, p := range patterns {
		if strings.Contains(errors, p.needle) {
			matched = append(matched, "- "+p.hint)
		}
	}
	if len(matched) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n## Targeted Fix Guidance\n\n")
	b.WriteString("Based on the errors above, pay special attention to:\n\n")
	for _, m := range matched {
		b.WriteString(m + "\n")
	}
	return b.String()
}

// outputFormatInstructions describes the required response format to the agent.
func outputFormatInstructions() string {
	return "## Module/Package Namespace vs Output Directory\n\n" +
		"CRITICAL: The filesystem output directory and the language module/package namespace are DIFFERENT things.\n" +
		"The OutputDir in the manifest payload tells you WHERE to put files on disk — it is NOT the import namespace.\n\n" +
		"Language rules:\n" +
		"- Go: module name is in go.mod (e.g. \"monolith\"). Import as \"monolith/internal/domain\", NEVER as \"backend/internal/domain\".\n" +
		"- Python: package root is the name in setup.py/pyproject.toml (e.g. \"app\"). Import as \"from app.domain import User\", not \"from backend.domain\".\n" +
		"- TypeScript: use tsconfig path aliases (e.g. \"@/\"). Import as \"@/domain/User\", not relative paths through the output dir.\n" +
		"- Java: use the groupId.artifactId base package (e.g. \"com.example.app\"). Import \"com.example.app.domain.User\", not \"backend.domain.User\".\n\n" +
		`## Output Format

You MUST respond with a <files> block containing a JSON array of file objects.
Each file object has a "path" (relative to YOUR component's directory) and "content" (complete file content).
The pipeline places your files under the correct subdirectory automatically — use component-relative paths
like "go.mod", "internal/service/user.go", "src/components/Button.tsx". Do NOT prefix paths with
directory names like "backend/", "frontend/", or "services/user-api/" — those are added by the pipeline.

EXCEPTION — infra.docker task: this task outputs files for the entire project from the project root.
It has no component directory, so paths MUST include the service subdirectory prefix:
  "backend/Dockerfile", "frontend/Dockerfile", "backend/.air.toml", "docker-compose.yml", etc.
See the Role section for full path rules when you are the infra.docker task.

Example:
<files>
[
  {
    "path": "main.go",
    "content": "package main\n\nimport ..."
  },
  {
    "path": "internal/domain/user.go",
    "content": "package domain\n\n..."
  }
]
</files>

Rules:
- Always use the <files>...</files> XML tags — do NOT use markdown code fences for the file list.
- Include ALL files needed for the task to be complete and buildable.
- File paths must use forward slashes and be relative (no leading slash).
- File content must be complete — no placeholders, no TODO comments for required logic.
- For TypeScript/JS: include package.json with pinned dependency versions and all necessary config files.
- For Terraform: include all .tf files needed to apply successfully.
- Generated code must pass the relevant linter/build check for its language.

### Go-Specific Rules (CRITICAL)
- If a go.mod was provided in the "Shared Team Context" or "LOCKED DEPENDENCIES" section,
  DO NOT include go.mod or go.sum in your output. The build system manages dependencies.
- If you MUST generate go.mod (plan/skeleton task only), use EXACTLY the module paths and
  versions from the "Dependency & API Reference" section. Never invent version strings.
- All Go regex patterns MUST use backtick raw strings, not double-quoted strings:
  CORRECT: regexp.MustCompile(` + "`" + `\d{4}-\d{2}-\d{2}` + "`" + `)
  WRONG:   regexp.MustCompile("\d{4}-\d{2}-\d{2}")    // invalid escape sequences
  WRONG:   regexp.MustCompile("\\d{4}-\\d{2}-\\d{2}") // fragile double-escaping
- All Go code MUST be gofmt-clean — use tabs for indentation, not spaces.
- Always generate _test.go files alongside every service, handler, and repository file.
  Use table-driven tests covering happy path, error cases, and edge cases.
- No hardcoded secrets: read all credentials from environment variables with a startup check.
- Apply idiomatic Go: constructor injection, small focused interfaces, error wrapping with
  fmt.Errorf("context: %w", err). Never ignore errors.
- Use the EXACT library APIs documented in the "Dependency & API Reference" section.
  Do NOT invent types or functions that are not listed there.

### npm / Node.js Rules (CRITICAL)
- Use 'npm install' NOT 'npm ci' in Dockerfiles and README instructions — package-lock.json
  is generated at runtime, not by the pipeline. npm ci will FAIL without a pre-existing lockfile.
- Config files: use next.config.mjs (ESM) — works universally across Next.js versions.
  next.config.ts is only supported from Next.js 15.3+.
- Always use EXACTLY the versions from the "Infrastructure & Dependency Reference" section.
  Do NOT guess or use @latest for any package.

### Architecture-Specific Directory Layout

The arch_pattern in your payload determines the expected project structure. Use this as a
reference when choosing file paths — your OutputDir prefix is added by the pipeline automatically.

Monolith (with frontend):
  backend/          ← OutputDir for backend tasks; go.mod lives here
    main.go
    Dockerfile      ← infra.docker task (path: "backend/Dockerfile")
    .air.toml       ← infra.docker task (path: "backend/.air.toml")
    .dockerignore   ← infra.docker task (path: "backend/.dockerignore")
    openapi.yaml    ← contracts task (spec, alongside Go module)
    internal/
      domain/       ← data schemas task
      contracts/    ← contracts task (Go types)
      repository/
      service/
      handler/
    db/migrations/
  frontend/         ← OutputDir for frontend task
    Dockerfile      ← infra.docker task (path: "frontend/Dockerfile")
  docker-compose.yml  ← infra.docker task (path: "docker-compose.yml")

Monolith (no frontend):
  .                 ← OutputDir is "." for all backend tasks
    main.go
    go.mod
    internal/
      domain/
      contracts/
      repository/
      service/
      handler/

Modular Monolith:
  backend/ (or root)
    internal/
      modules/{module-name}/
        domain/
        repository/
        service/
        handler/
      router/       ← single router wiring all modules

Microservices / Event-Driven / Hybrid:
  services/{name}/  ← OutputDir per service; each has its own go.mod
    internal/
      domain/
      repository/
      service/
      handler/
  shared/           ← OutputDir for contracts task
    contracts/      ← shared Go types (package contracts)
    contracts/openapi.yaml
  frontend/
  docker-compose.yml`
}
