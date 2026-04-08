package verify

import (
	"fmt"
	"regexp"
	"strings"
)

// CompilerError is a structured representation of a single compiler error.
// Used by deterministic fixes and error pattern hints to operate on parsed
// errors rather than raw string matching, making the pipeline resilient to
// compiler output format changes across Go/TypeScript/Python versions.
type CompilerError struct {
	File    string // relative file path
	Line    int    // 1-indexed line number
	Col     int    // 1-indexed column number (0 if unknown)
	Code    string // error code (e.g. "TS2304") or category (e.g. "undefined")
	Message string // full error message
	Symbol  string // extracted symbol name when applicable (e.g. the undefined identifier)
}

// String returns a human-readable representation of the error.
func (e CompilerError) String() string {
	if e.Col > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Col, e.Message)
	}
	return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
}

// Go compiler error patterns.
var (
	goErrorRe = regexp.MustCompile(
		`^(.+\.go):(\d+):(\d+): (.+)$`)
	goUndefinedRe = regexp.MustCompile(
		`undefined:\s+(\w+)`)
	goNotImplementRe = regexp.MustCompile(
		`does not implement (\w+)`)
	goRedeclaredRe = regexp.MustCompile(
		`(\w+) redeclared in this block`)
	goImportedNotUsedRe = regexp.MustCompile(
		`"([^"]+)" imported and not used`)
	goTooManyArgsRe = regexp.MustCompile(
		`too many arguments in call to (\S+)`)
	goNotEnoughArgsRe = regexp.MustCompile(
		`not enough arguments in call to (\S+)`)
)

// ParseGoErrors parses Go compiler output into structured errors.
func ParseGoErrors(output string) []CompilerError {
	var errors []CompilerError
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		m := goErrorRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		ce := CompilerError{
			File:    m[1],
			Message: m[4],
		}
		fmt.Sscanf(m[2], "%d", &ce.Line)
		fmt.Sscanf(m[3], "%d", &ce.Col)

		// Classify and extract symbol.
		switch {
		case strings.Contains(ce.Message, "undefined:"):
			ce.Code = "undefined"
			if sm := goUndefinedRe.FindStringSubmatch(ce.Message); sm != nil {
				ce.Symbol = sm[1]
			}
		case strings.Contains(ce.Message, "does not implement"):
			ce.Code = "not_implemented"
			if sm := goNotImplementRe.FindStringSubmatch(ce.Message); sm != nil {
				ce.Symbol = sm[1]
			}
		case strings.Contains(ce.Message, "redeclared in this block"):
			ce.Code = "redeclared"
			if sm := goRedeclaredRe.FindStringSubmatch(ce.Message); sm != nil {
				ce.Symbol = sm[1]
			}
		case strings.Contains(ce.Message, "imported and not used"):
			ce.Code = "unused_import"
			if sm := goImportedNotUsedRe.FindStringSubmatch(ce.Message); sm != nil {
				ce.Symbol = sm[1]
			}
		case strings.Contains(ce.Message, "too many arguments"):
			ce.Code = "too_many_args"
			if sm := goTooManyArgsRe.FindStringSubmatch(ce.Message); sm != nil {
				ce.Symbol = sm[1]
			}
		case strings.Contains(ce.Message, "not enough arguments"):
			ce.Code = "not_enough_args"
			if sm := goNotEnoughArgsRe.FindStringSubmatch(ce.Message); sm != nil {
				ce.Symbol = sm[1]
			}
		case strings.Contains(ce.Message, "no new variables"):
			ce.Code = "no_new_vars"
		case strings.Contains(ce.Message, "cannot use"):
			ce.Code = "type_mismatch"
		default:
			ce.Code = "other"
		}

		errors = append(errors, ce)
	}
	return errors
}

// TypeScript compiler error pattern: file.ts(line,col): error TSXXXX: message
var tsErrorRe = regexp.MustCompile(
	`^(.+\.tsx?)\((\d+),(\d+)\):\s+error\s+(TS\d+):\s+(.+)$`)

// ParseTSErrors parses TypeScript compiler output into structured errors.
func ParseTSErrors(output string) []CompilerError {
	var errors []CompilerError
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		m := tsErrorRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		ce := CompilerError{
			File:    m[1],
			Code:    m[4],
			Message: m[5],
		}
		fmt.Sscanf(m[2], "%d", &ce.Line)
		fmt.Sscanf(m[3], "%d", &ce.Col)
		errors = append(errors, ce)
	}
	return errors
}

// ErrorsByFile groups compiler errors by file path.
func ErrorsByFile(errors []CompilerError) map[string][]CompilerError {
	result := make(map[string][]CompilerError)
	for _, e := range errors {
		result[e.File] = append(result[e.File], e)
	}
	return result
}

// ErrorsByCode groups compiler errors by error code.
func ErrorsByCode(errors []CompilerError) map[string][]CompilerError {
	result := make(map[string][]CompilerError)
	for _, e := range errors {
		result[e.Code] = append(result[e.Code], e)
	}
	return result
}

// HasCode reports whether any error in the slice has the given code.
func HasCode(errors []CompilerError, code string) bool {
	for _, e := range errors {
		if e.Code == code {
			return true
		}
	}
	return false
}

// HasMessageSubstring reports whether any error contains the substring in its message.
func HasMessageSubstring(errors []CompilerError, sub string) bool {
	for _, e := range errors {
		if strings.Contains(e.Message, sub) {
			return true
		}
	}
	return false
}

// WithCode returns errors matching the given code.
func WithCode(errors []CompilerError, code string) []CompilerError {
	var result []CompilerError
	for _, e := range errors {
		if e.Code == code {
			result = append(result, e)
		}
	}
	return result
}

// WithMessageSubstring returns errors whose message contains the substring.
func WithMessageSubstring(errors []CompilerError, sub string) []CompilerError {
	var result []CompilerError
	for _, e := range errors {
		if strings.Contains(e.Message, sub) {
			result = append(result, e)
		}
	}
	return result
}

// ParseErrors dispatches to the language-appropriate parser.
// Returns nil for unrecognised languages.
func ParseErrors(output, language string) []CompilerError {
	switch language {
	case "go", "":
		return ParseGoErrors(output)
	case "typescript":
		return ParseTSErrors(output)
	default:
		return nil
	}
}
