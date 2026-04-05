package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

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
	"PostgreSQL":  {Module: "github.com/jackc/pgx/v5", Version: "v5.7.2"},
	"MySQL":       {Module: "github.com/go-sql-driver/mysql", Version: "v1.8.1"},
	"SQLite":      {Module: "modernc.org/sqlite", Version: "v1.34.4"},
	"MongoDB":     {Module: "go.mongodb.org/mongo-driver", Version: "v1.17.1"},
	"Redis":       {Module: "github.com/redis/go-redis/v9", Version: "v9.7.3"},
	"sqlx":        {Module: "github.com/jmoiron/sqlx", Version: "v1.4.0"},
	"CockroachDB": {Module: "github.com/jackc/pgx/v5", Version: "v5.7.2"}, // CockroachDB uses pgx wire protocol

	// ── ORM ────────────────────────────────────────────────────────
	"GORM":          {Module: "gorm.io/gorm", Version: "v1.25.12"},
	"gorm-postgres": {Module: "gorm.io/driver/postgres", Version: "v1.5.11"},
	"gorm-mysql":    {Module: "gorm.io/driver/mysql", Version: "v1.5.7"},
	"gorm-sqlite":   {Module: "gorm.io/driver/sqlite", Version: "v1.5.7"},
	"ent":           {Module: "entgo.io/ent", Version: "v0.14.1"},

	// ── Messaging / event streaming ────────────────────────────────
	"NATS":     {Module: "github.com/nats-io/nats.go", Version: "v1.37.0"},
	"Kafka":    {Module: "github.com/segmentio/kafka-go", Version: "v0.4.47"},
	"RabbitMQ": {Module: "github.com/rabbitmq/amqp091-go", Version: "v1.10.0"},

	// ── RPC / API ──────────────────────────────────────────────────
	"gRPC":       {Module: "google.golang.org/grpc", Version: "v1.70.0"},
	"protobuf":   {Module: "google.golang.org/protobuf", Version: "v1.36.5"},
	"ConnectRPC": {Module: "connectrpc.com/connect", Version: "v1.18.1"},

	// ── Auth ───────────────────────────────────────────────────────
	"JWT":    {Module: "github.com/golang-jwt/jwt/v5", Version: "v5.2.1"},
	"bcrypt": {Module: "golang.org/x/crypto", Version: "v0.31.0"},

	// ── Testing ────────────────────────────────────────────────────
	"testify":  {Module: "github.com/stretchr/testify", Version: "v1.9.0"},
	"pgxmock":  {Module: "github.com/pashagolub/pgxmock/v4", Version: "v4.4.0"},
	"gomock":   {Module: "go.uber.org/mock", Version: "v0.5.0"},
	"httptest": {Module: "net/http/httptest", Version: ""}, // stdlib — no version needed

	// ── Validation ─────────────────────────────────────────────────
	"validator": {Module: "github.com/go-playground/validator/v10", Version: "v10.22.1"},

	// ── Logging ────────────────────────────────────────────────────
	"zap":     {Module: "go.uber.org/zap", Version: "v1.27.0"},
	"zerolog": {Module: "github.com/rs/zerolog", Version: "v1.33.0"},
	"slog":    {Module: "log/slog", Version: ""}, // stdlib — no version needed

	// ── Observability ──────────────────────────────────────────────
	"prometheus": {Module: "github.com/prometheus/client_golang", Version: "v1.21.1"},
	"otel":       {Module: "go.opentelemetry.io/otel", Version: "v1.34.0"},
	"otel-http":  {Module: "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp", Version: "v1.34.0"},

	// ── UUID ───────────────────────────────────────────────────────
	"uuid": {Module: "github.com/google/uuid", Version: "v1.6.0"},

	// ── Config ─────────────────────────────────────────────────────
	"envconfig": {Module: "github.com/kelseyhightower/envconfig", Version: "v1.4.0"},
	"viper":     {Module: "github.com/spf13/viper", Version: "v1.19.0"},
	"godotenv":  {Module: "github.com/joho/godotenv", Version: "v1.5.1"},

	// ── Scheduling ─────────────────────────────────────────────────
	"cron": {Module: "github.com/robfig/cron/v3", Version: "v3.0.1"},

	// ── HTTP routing ───────────────────────────────────────────────
	"gorilla/mux": {Module: "github.com/gorilla/mux", Version: "v1.8.1"},

	// ── Serialisation ──────────────────────────────────────────────
	"sonic": {Module: "github.com/bytedance/sonic", Version: "v1.13.2"},
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
		// Skip stdlib sentinels (no module path version needed).
		if info.Version == "" {
			return
		}
		if !seen[info.Module] {
			seen[info.Module] = true
			requires = append(requires, fmt.Sprintf("\t%s %s", info.Module, info.Version))
		}
		for _, td := range info.TestDeps {
			if td.Version == "" {
				continue
			}
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

	sort.Strings(requires)

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
	type job struct {
		keys []string
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

	resolvedByPath := make(map[string]ModuleInfo, len(byModule))
	for r := range ch {
		resolvedByPath[r.modulePath] = r.resolved
	}

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
		_ = os.MkdirAll(filepath.Dir(full), 0755)
		_ = os.WriteFile(full, []byte(content), 0644)
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
		// Single-line form: require github.com/foo/bar v1.2.3
		if strings.HasPrefix(line, "require ") && !strings.HasSuffix(strings.TrimSpace(line), "(") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				versions[parts[1]] = parts[2]
			}
			continue
		}
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
			// parts[0]=module, parts[1]=version, optional parts[2]="//" parts[3]="indirect"
			if len(parts) >= 2 {
				versions[parts[0]] = parts[1]
			}
		}
	}
	return versions
}

// resolveGoDevToolVersion fetches the latest version of a Go dev tool from the Go
// module proxy, then reads its go.mod to find the minimum Go version it requires.
// Both version and MinGoVersion are updated from live registry data.
// Falls back to t on any error.
func resolveGoDevToolVersion(ctx context.Context, t GoDevTool) GoDevTool {
	encoded := goModProxyPath(t.ModulePath)

	// Step 1: resolve @latest version tag — fresh timeout so this request gets
	// its full budget regardless of any prior work done by the caller.
	latestURL := "https://proxy.golang.org/" + encoded + "/@latest"
	req1Ctx, cancel1 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel1()
	req, err := http.NewRequestWithContext(req1Ctx, http.MethodGet, latestURL, nil)
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
	// Use a fresh timeout — the first request has already consumed some of any
	// shared deadline, so a new context ensures step 2 gets its own full window.
	modURL := "https://proxy.golang.org/" + encoded + "/@v/" + latest.Version + ".mod"
	req2Ctx, cancel2 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel2()
	modReq, err := http.NewRequestWithContext(req2Ctx, http.MethodGet, modURL, nil)
	if err != nil {
		return resolved
	}
	modResp, err := http.DefaultClient.Do(modReq)
	if err != nil || modResp.StatusCode != http.StatusOK {
		if modResp != nil {
			_ = modResp.Body.Close()
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
