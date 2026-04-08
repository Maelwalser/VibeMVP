// Package codegen provides deterministic code generation for pipeline tasks
// where the output structure is mechanical and benefits from exact signatures.
package codegen

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vibe-menu/internal/realize/memory"
)

// WireBootstrap generates a deterministic Go main.go skeleton from constructor
// signatures extracted by upstream tasks. The skeleton:
//   - Instantiates repositories, services, and handlers in dependency order
//   - Wires constructors with the exact argument count and types
//   - Sets up a framework-appropriate router and server
//
// The LLM receives this skeleton and is instructed to fill in custom parts only
// (middleware logic, env parsing, graceful shutdown). This eliminates the most
// common bootstrap failure: wrong constructor argument counts.
//
// framework is the HTTP framework (e.g. "Fiber", "Gin", "Echo", "Chi", "").
// language is the primary language (e.g. "Go", "TypeScript", "Python").
// Returns "" if there are insufficient constructors to generate a useful skeleton.
func WireBootstrap(
	ctors []memory.ConstructorSig,
	methods []memory.ServiceMethodSig,
	modulePath string,
	framework string,
	language string,
) string {
	if len(ctors) == 0 || modulePath == "" {
		return ""
	}

	// Classify constructors by layer based on package path conventions.
	var repos, services, handlers []memory.ConstructorSig
	for _, c := range ctors {
		lower := strings.ToLower(c.Package)
		switch {
		case strings.Contains(lower, "repository") || strings.Contains(lower, "postgres") || strings.Contains(lower, "mysql"):
			repos = append(repos, c)
		case strings.Contains(lower, "service") || strings.Contains(lower, "logic"):
			services = append(services, c)
		case strings.Contains(lower, "handler") || strings.Contains(lower, "http") || strings.Contains(lower, "api"):
			handlers = append(handlers, c)
		}
	}

	if len(repos) == 0 && len(services) == 0 && len(handlers) == 0 {
		return ""
	}

	// Collect unique import paths.
	importPaths := make(map[string]bool)
	importPaths[`"context"`] = true
	importPaths[`"fmt"`] = true
	importPaths[`"log"`] = true
	importPaths[`"os"`] = true
	for _, c := range ctors {
		if c.Package != "" {
			importPaths[fmt.Sprintf(`"%s/%s"`, modulePath, c.Package)] = true
		}
	}
	// Add framework-specific imports.
	fwImports := frameworkImports(framework)
	for _, imp := range fwImports {
		importPaths[imp] = true
	}
	// Fiber's error handler uses errors.As.
	if strings.ToLower(framework) == "fiber" {
		importPaths[`"errors"`] = true
	}

	// Sort imports for determinism.
	var imports []string
	for imp := range importPaths {
		imports = append(imports, imp)
	}
	sort.Strings(imports)

	var b strings.Builder
	b.WriteString("package main\n\nimport (\n")
	for _, imp := range imports {
		b.WriteString("\t" + imp + "\n")
	}
	b.WriteString(")\n\nfunc main() {\n")
	b.WriteString("\tctx := context.Background()\n\n")

	// Step 1: Database connection (placeholder for LLM to fill).
	b.WriteString("\t// TODO: Initialize database connection pool\n")
	b.WriteString("\t// db, err := pgxpool.New(ctx, os.Getenv(\"DATABASE_URL\"))\n")
	b.WriteString("\t// if err != nil { log.Fatalf(\"connect to database: %v\", err) }\n")
	b.WriteString("\t// defer db.Close()\n")
	b.WriteString("\t_ = ctx // placeholder — replace with actual db pool initialization\n\n")

	// Step 2: Instantiate repositories.
	if len(repos) > 0 {
		b.WriteString("\t// ── Repositories ──\n")
		for _, r := range repos {
			varName := ctorVarName(r.Signature)
			pkgAlias := filepath.Base(r.Package)
			funcName := extractCtorFuncName(r.Signature)
			b.WriteString(fmt.Sprintf("\t// %s\n", r.Signature))
			b.WriteString(fmt.Sprintf("\t%s := %s.%s(db) // wire exact args from constructor signature\n",
				varName, pkgAlias, funcName))
		}
		b.WriteString("\n")
	}

	// Step 3: Instantiate services.
	if len(services) > 0 {
		b.WriteString("\t// ── Services ──\n")
		for _, s := range services {
			varName := ctorVarName(s.Signature)
			pkgAlias := filepath.Base(s.Package)
			funcName := extractCtorFuncName(s.Signature)
			b.WriteString(fmt.Sprintf("\t// %s\n", s.Signature))
			b.WriteString(fmt.Sprintf("\t%s := %s.%s(/* wire repo instances */) // match exact constructor params\n",
				varName, pkgAlias, funcName))
		}
		b.WriteString("\n")
	}

	// Step 4: Instantiate handlers.
	if len(handlers) > 0 {
		b.WriteString("\t// ── Handlers ──\n")
		for _, h := range handlers {
			varName := ctorVarName(h.Signature)
			pkgAlias := filepath.Base(h.Package)
			funcName := extractCtorFuncName(h.Signature)
			b.WriteString(fmt.Sprintf("\t// %s\n", h.Signature))
			b.WriteString(fmt.Sprintf("\t%s := %s.%s(/* wire service instances */) // match exact constructor params\n",
				varName, pkgAlias, funcName))
		}
		b.WriteString("\n")
	}

	// Step 5: Router setup + server startup (framework-specific).
	writeRouterAndServer(&b, framework, handlers)

	return b.String()
}

// ctorVarName derives a variable name from a constructor signature.
// E.g. "func NewUserRepository(...)" → "userRepo"
// E.g. "func NewUserService(...)" → "userSvc"
// E.g. "func NewUserHandler(...)" → "userHandler"
func ctorVarName(sig string) string {
	name := extractCtorFuncName(sig)
	name = strings.TrimPrefix(name, "New")
	name = strings.TrimPrefix(name, "Make")
	name = strings.TrimPrefix(name, "Create")

	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "repository"):
		base := name[:len(name)-len("Repository")]
		return lcFirst(base) + "Repo"
	case strings.HasSuffix(lower, "service"):
		base := name[:len(name)-len("Service")]
		return lcFirst(base) + "Svc"
	case strings.HasSuffix(lower, "handler"):
		return lcFirst(name)
	default:
		return lcFirst(name)
	}
}

// extractCtorFuncName extracts the function name from a Go constructor signature.
func extractCtorFuncName(sig string) string {
	s := strings.TrimPrefix(sig, "func ")
	paren := strings.Index(s, "(")
	if paren < 0 {
		return s
	}
	return strings.TrimSpace(s[:paren])
}

// lcFirst lowercases the first letter of a string.
func lcFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// frameworkImports returns the import paths required for the given framework.
// Returns stdlib "net/http" as the default.
func frameworkImports(framework string) []string {
	switch strings.ToLower(framework) {
	case "fiber":
		return []string{`"github.com/gofiber/fiber/v2"`}
	case "gin":
		return []string{`"github.com/gin-gonic/gin"`}
	case "echo":
		return []string{`"github.com/labstack/echo/v4"`}
	case "chi":
		return []string{`"net/http"`, `"github.com/go-chi/chi/v5"`}
	default:
		return []string{`"net/http"`}
	}
}

// writeRouterAndServer emits the framework-specific router setup and server
// startup code into the bootstrap skeleton.
func writeRouterAndServer(b *strings.Builder, framework string, handlers []memory.ConstructorSig) {
	b.WriteString("\t// ── Router ──\n")

	switch strings.ToLower(framework) {
	case "fiber":
		b.WriteString("\tapp := fiber.New(fiber.Config{\n")
		b.WriteString("\t\tErrorHandler: func(c *fiber.Ctx, err error) error {\n")
		b.WriteString("\t\t\tcode := fiber.StatusInternalServerError\n")
		b.WriteString("\t\t\tvar e *fiber.Error\n")
		b.WriteString("\t\t\tif errors.As(err, &e) { code = e.Code }\n")
		b.WriteString("\t\t\treturn c.Status(code).JSON(fiber.Map{\"error\": err.Error()})\n")
		b.WriteString("\t\t},\n")
		b.WriteString("\t})\n")
		b.WriteString("\tapi := app.Group(\"/api\")\n")
		for _, h := range handlers {
			varName := ctorVarName(h.Signature)
			b.WriteString(fmt.Sprintf("\t// TODO: Register %s routes on api group\n", varName))
			_ = varName
		}
		b.WriteString("\n")
		b.WriteString("\tport := os.Getenv(\"PORT\")\n")
		b.WriteString("\tif port == \"\" { port = \"8080\" }\n")
		b.WriteString("\tlog.Printf(\"server starting on :%s\", port)\n")
		b.WriteString("\tlog.Fatal(app.Listen(\":\" + port))\n")

	case "gin":
		b.WriteString("\tr := gin.Default()\n")
		b.WriteString("\tapi := r.Group(\"/api\")\n")
		for _, h := range handlers {
			varName := ctorVarName(h.Signature)
			b.WriteString(fmt.Sprintf("\t// TODO: Register %s routes on api group\n", varName))
			_ = varName
		}
		b.WriteString("\n")
		b.WriteString("\tport := os.Getenv(\"PORT\")\n")
		b.WriteString("\tif port == \"\" { port = \"8080\" }\n")
		b.WriteString("\tlog.Printf(\"server starting on :%s\", port)\n")
		b.WriteString("\tlog.Fatal(r.Run(\":\" + port))\n")

	case "echo":
		b.WriteString("\te := echo.New()\n")
		b.WriteString("\tapi := e.Group(\"/api\")\n")
		for _, h := range handlers {
			varName := ctorVarName(h.Signature)
			b.WriteString(fmt.Sprintf("\t// TODO: Register %s routes on api group\n", varName))
			_ = varName
		}
		b.WriteString("\n")
		b.WriteString("\tport := os.Getenv(\"PORT\")\n")
		b.WriteString("\tif port == \"\" { port = \"8080\" }\n")
		b.WriteString("\tlog.Printf(\"server starting on :%s\", port)\n")
		b.WriteString("\te.Logger.Fatal(e.Start(\":\" + port))\n")

	case "chi":
		b.WriteString("\tr := chi.NewRouter()\n")
		for _, h := range handlers {
			varName := ctorVarName(h.Signature)
			b.WriteString(fmt.Sprintf("\t// TODO: Register %s routes on r\n", varName))
			_ = varName
		}
		b.WriteString("\n")
		b.WriteString("\tport := os.Getenv(\"PORT\")\n")
		b.WriteString("\tif port == \"\" { port = \"8080\" }\n")
		b.WriteString("\tlog.Printf(\"server starting on :%s\", port)\n")
		b.WriteString("\tif err := http.ListenAndServe(\":\"+port, r); err != nil {\n")
		b.WriteString("\t\tlog.Fatalf(\"server error: %v\", err)\n")
		b.WriteString("\t}\n")

	default:
		// stdlib net/http
		b.WriteString("\tmux := http.NewServeMux()\n")
		for _, h := range handlers {
			varName := ctorVarName(h.Signature)
			b.WriteString(fmt.Sprintf("\t// TODO: Register %s routes on mux\n", varName))
			_ = varName
		}
		b.WriteString("\n")
		b.WriteString("\tport := os.Getenv(\"PORT\")\n")
		b.WriteString("\tif port == \"\" { port = \"8080\" }\n")
		b.WriteString("\tlog.Printf(\"server starting on :%s\", port)\n")
		b.WriteString("\tif err := http.ListenAndServe(\":\"+port, mux); err != nil {\n")
		b.WriteString("\t\tlog.Fatalf(\"server error: %v\", err)\n")
		b.WriteString("\t}\n")
	}

	b.WriteString("}\n")
}
