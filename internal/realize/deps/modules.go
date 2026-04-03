package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolvedDeps holds the validated go.mod content and a mapping of
// import path → resolved version for common libraries.
type ResolvedDeps struct {
	GoMod    string            `json:"go_mod"`
	GoSum    string            `json:"go_sum"`
	Versions map[string]string `json:"versions"`
}

// ModuleInfo holds the canonical import path and version for a Go module.
type ModuleInfo struct {
	Module   string
	Version  string
	TestDeps []ModuleDep
}

// ModuleDep is a dependency needed alongside a primary module.
type ModuleDep struct {
	Module  string
	Version string
}

// WellKnownGoModules maps framework/library names used in manifests to their
// actual Go module paths and known-good recent versions.
// This is the SINGLE SOURCE OF TRUTH for dependency versions — agents never
// guess versions; they use these.
//
// To update: change the version string here and re-run.
// To add a new library: add an entry and the pipeline picks it up automatically.
var WellKnownGoModules = map[string]ModuleInfo{
	// ── Web frameworks ─────────────────────────────────────────────
	"Fiber": {
		Module: "github.com/gofiber/fiber/v2", Version: "v2.52.5",
		TestDeps: []ModuleDep{
			{Module: "github.com/stretchr/testify", Version: "v1.9.0"},
		},
	},
	"Gin":  {Module: "github.com/gin-gonic/gin", Version: "v1.10.0"},
	"Echo": {Module: "github.com/labstack/echo/v4", Version: "v4.12.0"},
	"Chi":  {Module: "github.com/go-chi/chi/v5", Version: "v5.1.0"},

	// ── Database drivers ───────────────────────────────────────────
	"pgx": {
		Module: "github.com/jackc/pgx/v5", Version: "v5.7.2",
		TestDeps: []ModuleDep{
			{Module: "github.com/pashagolub/pgxmock/v4", Version: "v4.4.0"},
		},
	},
	"PostgreSQL": {Module: "github.com/jackc/pgx/v5", Version: "v5.7.2"},
	"MySQL":      {Module: "github.com/go-sql-driver/mysql", Version: "v1.8.1"},
	"SQLite":     {Module: "modernc.org/sqlite", Version: "v1.34.4"},
	"MongoDB":    {Module: "go.mongodb.org/mongo-driver", Version: "v1.17.1"},

	// ── Auth ───────────────────────────────────────────────────────
	"JWT":    {Module: "github.com/golang-jwt/jwt/v5", Version: "v5.2.1"},
	"bcrypt": {Module: "golang.org/x/crypto", Version: "v0.31.0"},

	// ── Testing ────────────────────────────────────────────────────
	"testify": {Module: "github.com/stretchr/testify", Version: "v1.9.0"},
	"pgxmock": {Module: "github.com/pashagolub/pgxmock/v4", Version: "v4.4.0"},

	// ── Validation ─────────────────────────────────────────────────
	"validator": {Module: "github.com/go-playground/validator/v10", Version: "v10.22.1"},

	// ── Logging ────────────────────────────────────────────────────
	"zap":     {Module: "go.uber.org/zap", Version: "v1.27.0"},
	"zerolog": {Module: "github.com/rs/zerolog", Version: "v1.33.0"},

	// ── UUID ───────────────────────────────────────────────────────
	"uuid": {Module: "github.com/google/uuid", Version: "v1.6.0"},

	// ── Message brokers ────────────────────────────────────────────
	"NATS": {Module: "github.com/nats-io/nats.go", Version: "v1.37.0"},

	// ── Config ─────────────────────────────────────────────────────
	"envconfig": {Module: "github.com/kelseyhightower/envconfig", Version: "v1.4.0"},
}

// LibraryAPIDocs holds the exported API surface of commonly-misused libraries.
// Injected into agent prompts to prevent hallucinated types/functions.
//
// Each entry is keyed by a lowercase technology name that matches against
// the task's technology stack.
var LibraryAPIDocs = map[string]string{
	"pgxmock": `## pgxmock/v4 API Reference (github.com/pashagolub/pgxmock/v4)

Creating a mock pool:
  mock, err := pgxmock.NewPool()
  // Returns an interface-satisfying mock. The concrete type is UNEXPORTED.

DO NOT reference any of these — they do not exist:
  pgxmock.PgxPoolMock   ← WRONG, does not exist
  pgxmock.PgxMock       ← WRONG, does not exist
  pgxmock.MockPool      ← WRONG, does not exist
  pgxmock.Pool          ← WRONG, does not exist

Correct usage pattern:
  // Define your own interface for the pool:
  type DBTX interface {
      Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
      Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
      QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
  }

  // In tests:
  mock, err := pgxmock.NewPool()
  repo := NewRepository(mock)  // pass mock as the DBTX interface

Setting up expectations:
  // For queries returning rows:
  rows := pgxmock.NewRows([]string{"id", "name", "email"}).
      AddRow("uuid-1", "Alice", "alice@example.com")
  mock.ExpectQuery("SELECT").
      WithArgs("uuid-1").
      WillReturnRows(rows)

  // For exec (INSERT/UPDATE/DELETE):
  mock.ExpectExec("INSERT INTO users").
      WithArgs("Alice", "alice@example.com").
      WillReturnResult(pgxmock.NewResult("INSERT", 1))

  // Verify all expectations were met:
  if err := mock.ExpectationsWereMet(); err != nil {
      t.Errorf("unmet expectations: %s", err)
  }

WithArgs matching:
  // Use pgxmock.AnyArg() for arguments you don't care about:
  mock.ExpectExec("INSERT").WithArgs(pgxmock.AnyArg(), "alice@example.com")
`,

	"fiber": `## Fiber v2 API Reference (github.com/gofiber/fiber/v2)

IMPORTANT — these do NOT exist in fiber/v2:
  fiber.As()     ← WRONG, does not exist (use errors.As from stdlib)
  fiber.Is()     ← WRONG, does not exist (use errors.Is from stdlib)

App creation:
  app := fiber.New(fiber.Config{
      ErrorHandler: customErrorHandler,
  })

Route handlers — signature is func(c *fiber.Ctx) error:
  app.Get("/users/:id", getUser)
  app.Post("/users", createUser)
  app.Put("/users/:id", updateUser)
  app.Delete("/users/:id", deleteUser)

Context (c *fiber.Ctx) methods:
  c.Params("id")                    // path parameter
  c.Query("page", "1")             // query parameter with default
  c.BodyParser(&req)               // parse JSON body into struct
  c.Status(201).JSON(data)         // respond with status + JSON body
  c.SendStatus(204)                // respond with status only, no body
  c.Locals("user")                 // get value from middleware context
  c.Locals("user", userObj)        // set value in middleware context

Middleware:
  app.Use(logger.New())
  app.Use(recover.New())
  app.Use(cors.New(cors.Config{AllowOrigins: "http://localhost:3000"}))

Route groups:
  api := app.Group("/api/v1")
  api.Use(authMiddleware)
  api.Get("/users", listUsers)

Error responses:
  return fiber.NewError(fiber.StatusNotFound, "user not found")
  return fiber.NewError(fiber.StatusBadRequest, "invalid input")
  return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})

Testing:
  req := httptest.NewRequest("GET", "/api/v1/users", nil)
  req.Header.Set("Content-Type", "application/json")
  resp, err := app.Test(req, -1)  // -1 = no timeout
`,

	"golang-jwt": `## golang-jwt/v5 API Reference (github.com/golang-jwt/jwt/v5)

Creating a token:
  claims := jwt.MapClaims{
      "sub": userID,
      "exp": jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
      "iat": jwt.NewNumericDate(time.Now()),
  }
  token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
  signedString, err := token.SignedString([]byte(secretKey))

Parsing a token:
  token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
      if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
          return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
      }
      return []byte(secretKey), nil
  })
  if err != nil || !token.Valid {
      return fmt.Errorf("invalid token")
  }
  claims, ok := token.Claims.(jwt.MapClaims)
`,
}

// PromptContext generates text to inject into agent prompts that provides
// exact dependency versions and library API docs for the task's technology stack.
func PromptContext(framework string, technologies []string) string {
	var b strings.Builder

	b.WriteString("\n## Dependency & API Reference\n\n")
	b.WriteString("Use EXACTLY these module paths and versions in go.mod. Do NOT invent versions.\n\n")

	// List relevant modules with their exact versions.
	seen := make(map[string]bool)
	var modules []ModuleInfo

	if info, ok := WellKnownGoModules[framework]; ok && !seen[info.Module] {
		seen[info.Module] = true
		modules = append(modules, info)
	}
	for _, tech := range technologies {
		if info, ok := WellKnownGoModules[tech]; ok && !seen[info.Module] {
			seen[info.Module] = true
			modules = append(modules, info)
		}
	}

	if len(modules) > 0 {
		b.WriteString("### Exact Module Versions\n\n| Module | Version |\n|--------|--------|\n")
		for _, m := range modules {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", m.Module, m.Version))
			for _, td := range m.TestDeps {
				b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", td.Module, td.Version))
			}
		}
		b.WriteString("\n")
	}

	// Inject library API docs for relevant technologies.
	injected := make(map[string]bool)
	allTechs := append([]string{framework}, technologies...)
	for _, tech := range allTechs {
		lower := strings.ToLower(tech)
		for key, doc := range LibraryAPIDocs {
			if injected[key] {
				continue
			}
			if strings.Contains(lower, key) || strings.Contains(key, lower) {
				b.WriteString(doc)
				b.WriteString("\n")
				injected[key] = true
			}
		}
	}

	// Special case: if PostgreSQL is in the stack, always inject pgxmock docs.
	for _, tech := range allTechs {
		if strings.Contains(strings.ToLower(tech), "postgre") || strings.Contains(strings.ToLower(tech), "pgx") {
			if !injected["pgxmock"] {
				b.WriteString(LibraryAPIDocs["pgxmock"])
				injected["pgxmock"] = true
			}
		}
	}

	return b.String()
}

// GoModForService generates a go.mod for a service based on its declared
// framework and database dependencies, using only known-good versions.
func GoModForService(modulePath, framework string, technologies []string) string {
	var requires []string
	seen := make(map[string]bool)

	addModule := func(info ModuleInfo) {
		if !seen[info.Module] {
			seen[info.Module] = true
			requires = append(requires, fmt.Sprintf("\t%s %s", info.Module, info.Version))
		}
		for _, td := range info.TestDeps {
			if !seen[td.Module] {
				seen[td.Module] = true
				requires = append(requires, fmt.Sprintf("\t%s %s", td.Module, td.Version))
			}
		}
	}

	if info, ok := WellKnownGoModules[framework]; ok {
		addModule(info)
	}
	for _, tech := range technologies {
		if info, ok := WellKnownGoModules[tech]; ok {
			addModule(info)
		}
	}
	for _, key := range []string{"testify", "uuid"} {
		if info, ok := WellKnownGoModules[key]; ok {
			addModule(info)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("module %s\n\ngo 1.22\n\n", modulePath))
	if len(requires) > 0 {
		b.WriteString("require (\n")
		for _, r := range requires {
			b.WriteString(r + "\n")
		}
		b.WriteString(")\n")
	}
	return b.String()
}

// ValidateGoMod runs go mod tidy in a temp directory to resolve real versions.
func ValidateGoMod(ctx context.Context, goModContent string, goFiles map[string]string) (*ResolvedDeps, error) {
	tmpDir, err := os.MkdirTemp("", "deps-resolve-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return nil, fmt.Errorf("write go.mod: %w", err)
	}
	for path, content := range goFiles {
		full := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(full), 0755)
		os.WriteFile(full, []byte(content), 0644)
	}

	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go mod tidy: %s\n%s", err, string(out))
	}

	modData, _ := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	sumData, _ := os.ReadFile(filepath.Join(tmpDir, "go.sum"))

	return &ResolvedDeps{
		GoMod:    string(modData),
		GoSum:    string(sumData),
		Versions: parseRequires(string(modData)),
	}, nil
}

func parseRequires(gomod string) map[string]string {
	versions := make(map[string]string)
	inRequire := false
	for _, line := range strings.Split(gomod, "\n") {
		line = strings.TrimSpace(line)
		if line == "require (" {
			inRequire = true
			continue
		}
		if line == ")" {
			inRequire = false
			continue
		}
		if inRequire {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				versions[parts[0]] = parts[1]
			}
		}
	}
	return versions
}

// SaveResolvedDeps persists resolved deps for downstream tasks.
func SaveResolvedDeps(dir, taskID string, d *ResolvedDeps) error {
	depsDir := filepath.Join(dir, ".realize", "deps")
	if err := os.MkdirAll(depsDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(depsDir, taskID+".json"), data, 0644)
}

// LoadResolvedDeps reads previously resolved dependency info.
func LoadResolvedDeps(dir, taskID string) (*ResolvedDeps, error) {
	path := filepath.Join(dir, ".realize", "deps", taskID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d ResolvedDeps
	return &d, json.Unmarshal(data, &d)
}
