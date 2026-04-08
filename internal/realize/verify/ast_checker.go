package verify

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibe-menu/internal/realize/memory"
)

// CheckImportConsistency scans all .go files in dir and checks that internal
// imports use the correct module prefix. Returns a list of human-readable error
// messages for each invalid import found.
func CheckImportConsistency(dir string) []string {
	modulePath := readModulePath(dir)
	if modulePath == "" {
		return nil
	}

	fset := token.NewFileSet()
	var errors []string

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return nil // skip unparseable files
		}
		rel, _ := filepath.Rel(dir, path)
		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			// Check for placeholder imports that should use the module path.
			if strings.HasPrefix(importPath, "github.com/your-org/") ||
				strings.HasPrefix(importPath, "github.com/example/") {
				errors = append(errors, fmt.Sprintf(
					"%s: import %q should use module path %q",
					rel, importPath, modulePath))
			}
			// Check for bare "internal/" imports missing the module prefix.
			if strings.HasPrefix(importPath, "internal/") {
				errors = append(errors, fmt.Sprintf(
					"%s: import %q is missing module prefix — should be %q",
					rel, importPath, modulePath+"/"+importPath))
			}
		}
		return nil
	})
	return errors
}

// CheckConstructorArity parses Go source files in dir and checks that calls to
// known constructor functions (New*, Make*, etc.) have the correct argument count.
// Returns human-readable error messages for each arity mismatch found.
func CheckConstructorArity(dir string, knownCtors []memory.ConstructorSig) []string {
	if len(knownCtors) == 0 {
		return nil
	}

	// Build a map of constructor name → expected arg count.
	ctorArity := make(map[string]int, len(knownCtors))
	for _, c := range knownCtors {
		name := extractFuncName(c.Signature)
		if name == "" {
			continue
		}
		arity := countParams(c.Signature)
		ctorArity[name] = arity
	}

	fset := token.NewFileSet()
	var errors []string

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, parseErr := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if parseErr != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			var funcName string
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				funcName = fn.Name
			case *ast.SelectorExpr:
				funcName = fn.Sel.Name
			}
			if funcName == "" {
				return true
			}
			expected, known := ctorArity[funcName]
			if !known {
				return true
			}
			actual := len(call.Args)
			if actual != expected {
				pos := fset.Position(call.Pos())
				errors = append(errors, fmt.Sprintf(
					"%s:%d: %s called with %d args, expected %d",
					rel, pos.Line, funcName, actual, expected))
			}
			return true
		})
		return nil
	})
	return errors
}

// readModulePath reads the module path from go.mod in dir.
func readModulePath(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
}

// extractFuncName extracts the function name from a Go function signature.
// E.g. "func NewUserRepository(db *pgxpool.Pool) (*UserRepository, error)" → "NewUserRepository"
func extractFuncName(sig string) string {
	sig = strings.TrimPrefix(sig, "func ")
	// Skip receiver if present: "func (s *X) Method(...)"
	if strings.HasPrefix(sig, "(") {
		closing := strings.Index(sig, ")")
		if closing < 0 {
			return ""
		}
		sig = strings.TrimSpace(sig[closing+1:])
	}
	paren := strings.Index(sig, "(")
	if paren < 0 {
		return ""
	}
	return strings.TrimSpace(sig[:paren])
}

// countParams counts the number of parameters in a Go function signature.
// E.g. "func New(a int, b string) error" → 2
func countParams(sig string) int {
	// Find the first parameter list.
	funcStart := strings.Index(sig, "(")
	if funcStart < 0 {
		return 0
	}
	// Skip receiver parens.
	if strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(sig, "func ")), "(") {
		// Has receiver — find the second opening paren.
		afterReceiver := strings.Index(sig[funcStart+1:], ")")
		if afterReceiver < 0 {
			return 0
		}
		funcStart = strings.Index(sig[funcStart+afterReceiver+2:], "(")
		if funcStart < 0 {
			return 0
		}
		funcStart += afterReceiver + funcStart + 2
	}

	// Extract parameter list content.
	depth := 0
	start := -1
	for i := funcStart; i < len(sig); i++ {
		if sig[i] == '(' {
			depth++
			if depth == 1 {
				start = i + 1
			}
		} else if sig[i] == ')' {
			depth--
			if depth == 0 {
				params := strings.TrimSpace(sig[start:i])
				if params == "" {
					return 0
				}
				// Count commas, handling nested types like func(a, b int) as 2 params.
				// A rough but sufficient heuristic: count top-level commas + 1.
				count := 1
				parenDepth := 0
				for _, ch := range params {
					if ch == '(' {
						parenDepth++
					} else if ch == ')' {
						parenDepth--
					} else if ch == ',' && parenDepth == 0 {
						count++
					}
				}
				return count
			}
		}
	}
	return 0
}
