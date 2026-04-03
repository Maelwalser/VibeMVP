package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
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

// GoDevTool describes a Go tool installed in Dockerfiles (not in go.mod).
type GoDevTool struct {
	Name         string // human-readable name
	ModulePath   string // correct module path (may differ from historical path)
	Version      string // pinned version known to work
	MinGoVersion string // minimum Go version required by this tool version
}

// WellKnownGoDevTools lists dev tools installed via `go install` in Dockerfiles.
// These are NOT added to go.mod — they are installed in the Docker image layer only.
// The Version field is a fallback; InfraPromptContext resolves the actual latest
// compatible version from proxy.golang.org at runtime.
// IMPORTANT: github.com/cosmtrek/air was renamed to github.com/air-verse/air at v1.52.x.
// Never use the old module path.
var WellKnownGoDevTools = []GoDevTool{
	{
		Name:         "air",
		ModulePath:   "github.com/air-verse/air",
		Version:      "v1.61.5", // fallback; resolved dynamically at runtime
		MinGoVersion: "1.23",
	},
}

// WellKnownNpmPackages maps npm package names to fallback versions used when the
// npm registry is unreachable. At runtime, InfraPromptContext resolves the actual
// latest stable versions from registry.npmjs.org and only falls back to these.
var WellKnownNpmPackages = map[string]string{
	"next":                  "15.3.0",
	"react":                 "19.1.0",
	"react-dom":             "19.1.0",
	"typescript":            "5.7.2",
	"@types/react":          "19.1.0",
	"@types/react-dom":      "19.1.0",
	"@types/node":           "22.10.0",
	"tailwindcss":           "4.0.0",
	"@tailwindcss/postcss": "4.0.0",
	"postcss":               "8.5.1",
	"autoprefixer":          "10.4.20",
	"eslint":                "9.17.0",
	"eslint-config-next":    "15.3.0",
	"axios":                 "1.7.9",
	"@tanstack/react-query": "5.62.3",
	"zustand":               "5.0.2",
	"zod":                   "3.24.1",
	"react-hook-form":       "7.54.2",
	"@hookform/resolvers":   "3.9.1",
	"lucide-react":          "0.468.0",
	"clsx":                  "2.1.1",
	"tailwind-merge":        "2.5.5",
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

	"next": `## Next.js Configuration Rules

CRITICAL: next.config file naming:
- Use next.config.mjs (ESM) — works with ALL Next.js versions including 15.x
- next.config.ts is ONLY supported from Next.js 15.3+; default to .mjs to be safe

npm install vs npm ci in Dockerfiles:
- Use 'npm install' NOT 'npm ci' — package-lock.json is not generated by the pipeline
- npm ci requires an existing package-lock.json and will FAIL without one

Correct Dockerfile pattern for Next.js:
  COPY package*.json ./
  RUN npm install        ← NOT npm ci
  COPY . .
  CMD ["npm", "run", "dev"]
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

// ResolveGoVersion resolves the minimum Go runtime version required by all dev
// tools (e.g. air) by querying the Go module proxy. Falls back to the pinned
// MinGoVersion in WellKnownGoDevTools when the proxy is unreachable.
// Call this once at pipeline startup and share the result across all tasks.
func ResolveGoVersion(ctx context.Context) string {
	tools := resolveAllGoDevTools(ctx)
	if len(tools) == 0 {
		return ""
	}
	min := tools[0].MinGoVersion
	for _, t := range tools[1:] {
		if t.MinGoVersion > min {
			min = t.MinGoVersion
		}
	}
	return min
}

// resolveGoModuleVersion fetches the latest tagged version of a single Go module
// from the Go module proxy. Falls back to info.Version on any error.
// TestDeps versions are not changed — they are resolved transitively by go mod tidy.
func resolveGoModuleVersion(ctx context.Context, info ModuleInfo) ModuleInfo {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	encoded := goModProxyPath(info.Module)
	latestURL := "https://proxy.golang.org/" + encoded + "/@latest"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, latestURL, nil)
	if err != nil {
		return info
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return info
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return info
	}
	var latest struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil || latest.Version == "" {
		return info
	}
	resolved := info
	resolved.Version = latest.Version
	return resolved
}

// ResolveGoModuleVersions fetches the latest versions for every module in
// WellKnownGoModules from the Go module proxy, deduplicating by module path so
// each import URL is queried only once. Falls back to static versions on any error.
// Call once at pipeline startup and pass the result to PromptContext / GoModForService.
func ResolveGoModuleVersions(ctx context.Context) map[string]ModuleInfo {
	// Deduplicate: multiple keys may share the same module path (e.g. "pgx" and
	// "PostgreSQL" both use github.com/jackc/pgx/v5). Resolve each module path once.
	type job struct {
		keys []string // all WellKnownGoModules keys that share this module path
		info ModuleInfo
	}
	byModule := make(map[string]*job)
	for key, info := range WellKnownGoModules {
		if j, ok := byModule[info.Module]; ok {
			j.keys = append(j.keys, key)
		} else {
			infoCopy := info
			byModule[info.Module] = &job{keys: []string{key}, info: infoCopy}
		}
	}

	type result struct {
		modulePath string
		resolved   ModuleInfo
	}
	ch := make(chan result, len(byModule))
	var wg sync.WaitGroup
	for _, j := range byModule {
		wg.Add(1)
		go func(jb *job) {
			defer wg.Done()
			ch <- result{jb.info.Module, resolveGoModuleVersion(ctx, jb.info)}
		}(j)
	}
	wg.Wait()
	close(ch)

	// Build resolved-by-module-path map.
	resolvedByPath := make(map[string]ModuleInfo, len(byModule))
	for r := range ch {
		resolvedByPath[r.modulePath] = r.resolved
	}

	// Re-expand back to the original key space.
	out := make(map[string]ModuleInfo, len(WellKnownGoModules))
	for key, info := range WellKnownGoModules {
		if resolved, ok := resolvedByPath[info.Module]; ok {
			out[key] = resolved
		} else {
			out[key] = info
		}
	}
	return out
}

// moduleSource returns the resolved map if non-nil, otherwise falls back to
// the static WellKnownGoModules. Used by PromptContext and GoModForService.
func moduleSource(resolved map[string]ModuleInfo) map[string]ModuleInfo {
	if resolved != nil {
		return resolved
	}
	return WellKnownGoModules
}

// PromptContext generates text to inject into agent prompts that provides
// exact dependency versions and library API docs for the task's technology stack.
// goVersion, if non-empty, is the minimum Go runtime version required by dev tools
// (resolved at pipeline startup via ResolveGoVersion); it is injected so the plan
// task generates the correct `go X.Y` directive in go.mod.
// resolved, if non-nil, is the live-fetched module map from ResolveGoModuleVersions;
// pass nil to use static fallback versions.
func PromptContext(framework string, technologies []string, goVersion string, resolved map[string]ModuleInfo) string {
	modules_ := moduleSource(resolved)
	var b strings.Builder

	b.WriteString("\n## Dependency & API Reference\n\n")
	b.WriteString("Use EXACTLY these module paths and versions in go.mod. Do NOT invent versions.\n\n")

	// List relevant modules with their exact versions.
	seen := make(map[string]bool)
	var modules []ModuleInfo

	if info, ok := modules_[framework]; ok && !seen[info.Module] {
		seen[info.Module] = true
		modules = append(modules, info)
	}
	for _, tech := range technologies {
		if info, ok := modules_[tech]; ok && !seen[info.Module] {
			seen[info.Module] = true
			modules = append(modules, info)
		}
	}

	if goVersion != "" {
		b.WriteString(fmt.Sprintf("### Required Go Version\n\nUse `go %s` in go.mod. This MUST match the Docker base image (`FROM golang:%s-alpine`) generated by the infrastructure task.\n\n", goVersion, goVersion))
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
// goVersion is the minimum Go runtime version (e.g. "1.25") resolved at
// pipeline startup via ResolveGoVersion; it must match the Docker base image.
// resolved, if non-nil, is the live-fetched module map from ResolveGoModuleVersions.
func GoModForService(modulePath, framework, goVersion string, technologies []string, resolved map[string]ModuleInfo) string {
	modules_ := moduleSource(resolved)
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

	if info, ok := modules_[framework]; ok {
		addModule(info)
	}
	for _, tech := range technologies {
		if info, ok := modules_[tech]; ok {
			addModule(info)
		}
	}
	for _, key := range []string{"testify", "uuid"} {
		if info, ok := modules_[key]; ok {
			addModule(info)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("module %s\n\ngo %s\n\n", modulePath, goVersion))
	if len(requires) > 0 {
		b.WriteString("require (\n")
		for _, r := range requires {
			b.WriteString(r + "\n")
		}
		b.WriteString(")\n")
	}
	return b.String()
}

// resolveNpmVersion fetches the latest stable version of a package from the npm registry.
// Falls back to fallback on any error (network failure, registry unavailable, etc.).
func resolveNpmVersion(ctx context.Context, pkg, fallback string) string {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rawURL := "https://registry.npmjs.org/" + url.PathEscape(pkg) + "/latest"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fallback
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fallback
	}
	var result struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Version == "" {
		return fallback
	}
	return result.Version
}

// resolveAllNpmVersions fetches the latest versions for all packages in the fallback
// map concurrently, returning a new map with resolved (or fallback) versions.
func resolveAllNpmVersions(ctx context.Context) map[string]string {
	type result struct {
		pkg, version string
	}
	ch := make(chan result, len(WellKnownNpmPackages))
	var wg sync.WaitGroup
	for pkg, fallback := range WellKnownNpmPackages {
		wg.Add(1)
		go func(p, fb string) {
			defer wg.Done()
			ch <- result{p, resolveNpmVersion(ctx, p, fb)}
		}(pkg, fallback)
	}
	wg.Wait()
	close(ch)
	resolved := make(map[string]string, len(WellKnownNpmPackages))
	for r := range ch {
		resolved[r.pkg] = r.version
	}
	return resolved
}

// resolveGoDevToolVersion fetches the latest version of a Go dev tool from the Go
// module proxy, then reads its go.mod to find the minimum Go version it requires.
// Both version and MinGoVersion are updated from live registry data.
// Falls back to t on any error.
func resolveGoDevToolVersion(ctx context.Context, t GoDevTool) GoDevTool {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	encoded := goModProxyPath(t.ModulePath)

	// Step 1: resolve @latest version tag.
	latestURL := "https://proxy.golang.org/" + encoded + "/@latest"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, latestURL, nil)
	if err != nil {
		return t
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return t
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return t
	}
	var latest struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil || latest.Version == "" {
		return t
	}

	resolved := t
	resolved.Version = latest.Version

	// Step 2: fetch the go.mod for that version to read the `go` directive,
	// which tells us the minimum Go version the tool requires.
	modURL := "https://proxy.golang.org/" + encoded + "/@v/" + latest.Version + ".mod"
	modReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, modURL, nil)
	if err != nil {
		return resolved
	}
	modResp, err := http.DefaultClient.Do(modReq)
	if err != nil || modResp.StatusCode != http.StatusOK {
		if modResp != nil {
			modResp.Body.Close()
		}
		return resolved
	}
	defer modResp.Body.Close()

	// Parse "go X.YY" from the go.mod content.
	if data, err := io.ReadAll(modResp.Body); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "go ") {
				if ver := strings.TrimPrefix(line, "go "); ver != "" {
					resolved.MinGoVersion = ver
				}
				break
			}
		}
	}

	return resolved
}

// goModProxyPath encodes a module path for the Go module proxy API by replacing
// each uppercase letter with "!" followed by the lowercase equivalent.
func goModProxyPath(module string) string {
	var b strings.Builder
	for _, r := range module {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + 32)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// resolveAllGoDevTools resolves the latest version for each dev tool concurrently.
func resolveAllGoDevTools(ctx context.Context) []GoDevTool {
	resolved := make([]GoDevTool, len(WellKnownGoDevTools))
	var wg sync.WaitGroup
	for i, t := range WellKnownGoDevTools {
		wg.Add(1)
		go func(idx int, tool GoDevTool) {
			defer wg.Done()
			resolved[idx] = resolveGoDevToolVersion(ctx, tool)
		}(i, t)
	}
	wg.Wait()
	return resolved
}

// InfraPromptContext generates the dependency & API reference context for infrastructure
// and frontend tasks. Versions are resolved dynamically from the npm registry and Go
// module proxy; the values in WellKnownNpmPackages and WellKnownGoDevTools are used as
// fallbacks only when the registries are unreachable.
func InfraPromptContext(ctx context.Context, hasGoServices bool, hasFrontend bool) string {
	var b strings.Builder
	b.WriteString("\n## Infrastructure & Dependency Reference\n\n")
	b.WriteString("Use EXACTLY these versions. Do NOT invent alternatives.\n\n")

	if hasGoServices {
		// Resolve dev tool versions from the Go module proxy at runtime.
		tools := resolveAllGoDevTools(ctx)

		// Derive the minimum Go version required across all dev tools.
		minGoVersion := "1.23"
		for _, t := range tools {
			if t.MinGoVersion > minGoVersion {
				minGoVersion = t.MinGoVersion
			}
		}

		b.WriteString(fmt.Sprintf("### Go Docker Base Image\n\n```\nFROM golang:%s-alpine\n```\n\n", minGoVersion))
		b.WriteString(fmt.Sprintf("Minimum Go %s required for dev tools.\n\n", minGoVersion))

		b.WriteString("### Go Dev Tools (install via `go install`, NOT in go.mod)\n\n")
		b.WriteString("| Tool | Correct Module Path | Version | Min Go |\n")
		b.WriteString("|------|---------------------|---------|--------|\n")
		for _, t := range tools {
			b.WriteString(fmt.Sprintf("| %s | `%s` | `%s` | %s |\n",
				t.Name, t.ModulePath+"@"+t.Version, t.Version, t.MinGoVersion))
		}
		b.WriteString("\n")

		// Build the Dockerfile example dynamically from resolved tool versions.
		b.WriteString("### Go Dockerfile Rules\n\n")
		b.WriteString(fmt.Sprintf("Go base image: always use `golang:%s-alpine` or newer.\n", minGoVersion))
		for _, t := range tools {
			if t.Name == "air" {
				b.WriteString(fmt.Sprintf("Hot-reload tool (air): CORRECT path is `%s@%s`\n", t.ModulePath, t.Version))
				b.WriteString("  WRONG (old, renamed): `github.com/cosmtrek/air` ← DO NOT USE\n\n")
			}
		}
		b.WriteString("Required Dockerfile layer order (copy go.mod before source for layer caching):\n\n")
		b.WriteString(fmt.Sprintf("```dockerfile\nFROM golang:%s-alpine\nWORKDIR /app\n", minGoVersion))
		for _, t := range tools {
			b.WriteString(fmt.Sprintf("RUN go install %s@%s\n", t.ModulePath, t.Version))
		}
		b.WriteString("COPY go.mod go.sum ./\n")
		b.WriteString("RUN go mod download\n")
		b.WriteString("COPY . .\n")
		b.WriteString("CMD [\"air\", \"-c\", \".air.toml\"]\n```\n\n")
		b.WriteString("Without `go mod download`, air's incremental build will fail with `go: updates to go.mod needed`.\n\n")
	}

	if hasFrontend {
		// Resolve npm package versions from the registry at runtime.
		resolved := resolveAllNpmVersions(ctx)

		b.WriteString("### Node.js Docker Base Image\n\n```dockerfile\nFROM node:20-alpine\n```\n\n")
		b.WriteString("### npm Package Versions\n\n")
		b.WriteString("| Package | Version |\n|---------|--------|\n")
		pkgs := make([]string, 0, len(resolved))
		for pkg := range resolved {
			pkgs = append(pkgs, pkg)
		}
		sort.Strings(pkgs)
		for _, pkg := range pkgs {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", pkg, resolved[pkg]))
		}
		b.WriteString("\n")
		b.WriteString(LibraryAPIDocs["next"])
		b.WriteString("\n")

		// Tailwind CSS v4 breaking changes — injected when tailwindcss >= 4.x is resolved.
		if twVer := resolved["tailwindcss"]; twVer != "" && twVer[0] >= '4' {
			b.WriteString("### Tailwind CSS v4 — CRITICAL Configuration Rules\n\n")
			b.WriteString("Tailwind CSS 4.x moved its PostCSS plugin to a separate package. " +
				"Using `tailwindcss` directly as a PostCSS plugin will crash the build.\n\n")
			b.WriteString("**postcss.config.mjs** — CORRECT for Tailwind v4:\n")
			b.WriteString("```js\n" +
				"const config = {\n" +
				"  plugins: {\n" +
				"    \"@tailwindcss/postcss\": {},\n" +
				"    autoprefixer: {}\n" +
				"  }\n" +
				"};\n" +
				"export default config;\n" +
				"```\n\n")
			b.WriteString("WRONG (v3 syntax — causes build crash in v4):\n")
			b.WriteString("```js\n// DO NOT DO THIS:\nplugins: { tailwindcss: {} }\n```\n\n")
			b.WriteString("**globals.css** — CORRECT for Tailwind v4:\n")
			b.WriteString("```css\n@import \"tailwindcss\";\n```\n\n")
			b.WriteString("WRONG (v3 directives — removed in v4):\n")
			b.WriteString("```css\n// DO NOT DO THIS:\n@tailwind base;\n@tailwind components;\n@tailwind utilities;\n```\n\n")
			b.WriteString("**package.json devDependencies** must include `@tailwindcss/postcss` at the same version as `tailwindcss`.\n\n")
		}
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
