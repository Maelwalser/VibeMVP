package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibe-menu/internal/realize/memory"
)

// ApplyConstructorArityFixes parses go compiler output for constructor arity
// mismatches and applies deterministic fixes:
//   - "too many arguments": remove trailing arguments from the call site
//   - "not enough arguments": add nil/zero-value placeholders
//
// The knownCtors slice provides the expected parameter count. Returns a
// description of fixes applied, or "" if none were needed.
func ApplyConstructorArityFixes(dir string, verifyOutput string, knownCtors []memory.ConstructorSig) string {
	if len(knownCtors) == 0 {
		return ""
	}

	// Build a lookup: function name → expected param count.
	expectedArity := make(map[string]int)
	for _, c := range knownCtors {
		name := extractFuncNameFromSig(c.Signature)
		if name != "" {
			expectedArity[name] = countSigParams(c.Signature)
		}
	}

	// Use structured error parser to find arity errors.
	parsed := ParseGoErrors(verifyOutput)
	tooManyErrors := WithCode(parsed, "too_many_args")
	notEnoughErrors := WithCode(parsed, "not_enough_args")

	type arityFix struct {
		file     string
		line     int
		funcName string
		kind     string // "too_many" or "not_enough"
	}

	var fixes []arityFix
	seen := make(map[string]bool)

	for _, e := range tooManyErrors {
		key := fmt.Sprintf("%s:%d", e.File, e.Line)
		if seen[key] {
			continue
		}
		funcName := lastComponent(e.Symbol)
		if _, ok := expectedArity[funcName]; ok {
			seen[key] = true
			fixes = append(fixes, arityFix{file: e.File, line: e.Line, funcName: funcName, kind: "too_many"})
		}
	}
	for _, e := range notEnoughErrors {
		key := fmt.Sprintf("%s:%d", e.File, e.Line)
		if seen[key] {
			continue
		}
		funcName := lastComponent(e.Symbol)
		if _, ok := expectedArity[funcName]; ok {
			seen[key] = true
			fixes = append(fixes, arityFix{file: e.File, line: e.Line, funcName: funcName, kind: "not_enough"})
		}
	}

	if len(fixes) == 0 {
		return ""
	}

	applied := 0
	for _, fx := range fixes {
		path := filepath.Join(dir, fx.file)
		if _, err := os.Stat(path); err != nil {
			path = fx.file
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		if fx.line <= 0 || fx.line > len(lines) {
			continue
		}

		srcLine := lines[fx.line-1]
		expected := expectedArity[fx.funcName]

		switch fx.kind {
		case "too_many":
			patched := trimExtraArgs(srcLine, fx.funcName, expected)
			if patched != srcLine {
				lines[fx.line-1] = patched
				applied++
			}
		case "not_enough":
			patched := addMissingArgs(srcLine, fx.funcName, expected)
			if patched != srcLine {
				lines[fx.line-1] = patched
				applied++
			}
		}

		if applied > 0 {
			_ = os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
		}
	}

	if applied == 0 {
		return ""
	}
	return fmt.Sprintf("fixed constructor arity in %d call site(s)", applied)
}

// extractFuncNameFromSig extracts the function name from "func NewX(...) ..."
func extractFuncNameFromSig(sig string) string {
	sig = strings.TrimPrefix(sig, "func ")
	paren := strings.Index(sig, "(")
	if paren < 0 {
		return ""
	}
	return strings.TrimSpace(sig[:paren])
}

// countSigParams counts the number of parameters in a function signature.
// "func New(a int, b string) ..." → 2
func countSigParams(sig string) int {
	open := strings.Index(sig, "(")
	if open < 0 {
		return 0
	}
	close := strings.Index(sig[open:], ")")
	if close < 0 {
		return 0
	}
	params := sig[open+1 : open+close]
	params = strings.TrimSpace(params)
	if params == "" {
		return 0
	}
	return len(strings.Split(params, ","))
}

// lastComponent extracts the function name from a qualified call like "pkg.NewX".
func lastComponent(s string) string {
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

// trimExtraArgs removes trailing arguments from a function call to match expected arity.
func trimExtraArgs(line, funcName string, expected int) string {
	// Find the call: funcName( ... )
	idx := strings.Index(line, funcName+"(")
	if idx < 0 {
		// Try with package prefix: pkg.funcName(
		dotIdx := strings.Index(line, "."+funcName+"(")
		if dotIdx < 0 {
			return line
		}
		idx = dotIdx + 1
	}

	callStart := idx + len(funcName)
	// Find matching paren.
	depth, argStart := 0, callStart
	var argPositions []int
	for i := callStart; i < len(line); i++ {
		switch line[i] {
		case '(':
			depth++
			if depth == 1 {
				argStart = i + 1
			}
		case ')':
			depth--
			if depth == 0 {
				argPositions = append(argPositions, i)
				// Rebuild with only expected args.
				argsStr := line[argStart:i]
				args := splitArgs(argsStr)
				if len(args) > expected && expected >= 0 {
					trimmed := strings.Join(args[:expected], ", ")
					return line[:argStart] + trimmed + line[i:]
				}
				return line
			}
		case ',':
			if depth == 1 {
				argPositions = append(argPositions, i)
			}
		}
	}
	return line
}

// addMissingArgs adds nil placeholders for missing arguments.
func addMissingArgs(line, funcName string, expected int) string {
	idx := strings.Index(line, funcName+"(")
	if idx < 0 {
		dotIdx := strings.Index(line, "."+funcName+"(")
		if dotIdx < 0 {
			return line
		}
		idx = dotIdx + 1
	}

	callStart := idx + len(funcName)
	depth, argStart := 0, callStart
	for i := callStart; i < len(line); i++ {
		switch line[i] {
		case '(':
			depth++
			if depth == 1 {
				argStart = i + 1
			}
		case ')':
			depth--
			if depth == 0 {
				argsStr := strings.TrimSpace(line[argStart:i])
				args := splitArgs(argsStr)
				actual := len(args)
				if argsStr == "" {
					actual = 0
				}
				if actual < expected {
					missing := expected - actual
					for j := 0; j < missing; j++ {
						args = append(args, "nil /* TODO: wire correct dependency */")
					}
					return line[:argStart] + strings.Join(args, ", ") + line[i:]
				}
				return line
			}
		}
	}
	return line
}

// splitArgs splits a comma-separated argument list, respecting nested parens.
func splitArgs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var args []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	args = append(args, strings.TrimSpace(s[start:]))
	return args
}
