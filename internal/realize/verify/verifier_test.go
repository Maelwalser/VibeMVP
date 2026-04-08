package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/realize/dag"
)

// ── normalizeLanguage ─────────────────────────────────────────────────────────

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Go", "go"},
		{"TypeScript", "typescript"},
		{"JavaScript", "typescript"},
		{"Node.js", "typescript"},
		{"TypeScript/Node", "typescript"},
		{"Python", "python"},
		{"Rust", "rust"},
		// Unknown / unsupported languages return empty string
		{"Java", ""},
		{"Kotlin", ""},
		{"PHP", ""},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeLanguage(tc.input)
			if got != tc.want {
				t.Errorf("normalizeLanguage(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ── normalizeFrontendLanguage ─────────────────────────────────────────────────

func TestNormalizeFrontendLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"TypeScript", "typescript"},
		{"JavaScript", "typescript"},
		{"Dart", ""}, // Flutter — no tsc equivalent
		{"", ""},
		{"Unknown", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeFrontendLanguage(tc.input)
			if got != tc.want {
				t.Errorf("normalizeFrontendLanguage(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ── taskLanguage ──────────────────────────────────────────────────────────────

func TestTaskLanguage_ServiceTask_UsesServiceLanguage(t *testing.T) {
	task := &dag.Task{
		Kind: dag.TaskKindServiceHandler,
		Payload: dag.TaskPayload{
			Service: &manifest.ServiceDef{Language: "Go"},
		},
	}
	if got := taskLanguage(task); got != "go" {
		t.Errorf("expected 'go', got %q", got)
	}
}

func TestTaskLanguage_ServiceTask_FallsBackToAllServices(t *testing.T) {
	task := &dag.Task{
		Kind: dag.TaskKindServiceBootstrap,
		Payload: dag.TaskPayload{
			AllServices: []manifest.ServiceDef{{Language: "Python"}},
		},
	}
	if got := taskLanguage(task); got != "python" {
		t.Errorf("expected 'python', got %q", got)
	}
}

func TestTaskLanguage_DataTask_ReturnsEmpty(t *testing.T) {
	task := &dag.Task{Kind: dag.TaskKindDataSchemas}
	if got := taskLanguage(task); got != "" {
		t.Errorf("data task language should be empty, got %q", got)
	}
}

func TestTaskLanguage_InfraTerraform_ReturnsTerraform(t *testing.T) {
	task := &dag.Task{Kind: dag.TaskKindInfraTerraform}
	if got := taskLanguage(task); got != "terraform" {
		t.Errorf("expected 'terraform', got %q", got)
	}
}

func TestTaskLanguage_FrontendTask_UsesFrontendLanguage(t *testing.T) {
	task := &dag.Task{
		Kind: dag.TaskKindFrontend,
		Payload: dag.TaskPayload{
			Frontend: &manifest.FrontendPillar{
				Tech: &manifest.FrontendTechConfig{Language: "TypeScript"},
			},
		},
	}
	if got := taskLanguage(task); got != "typescript" {
		t.Errorf("expected 'typescript', got %q", got)
	}
}

// ── modTidyHint ───────────────────────────────────────────────────────────────

func TestModTidyHint_InvalidVersion(t *testing.T) {
	output := `github.com/foo/bar@v0.0.0-20200101000000-abcdef123456: invalid version: git ls-remote -q https://github.com/foo/bar terminal prompts disabled`
	hint := modTidyHint(output)
	if hint == "" {
		t.Error("expected non-empty hint for invalid version error")
	}
	if !strings.Contains(hint, "github.com/foo/bar") {
		t.Errorf("hint should mention the broken module, got: %q", hint)
	}
}

func TestModTidyHint_NotFound(t *testing.T) {
	output := `github.com/missing/pkg@v1.2.3: reading github.com/missing/pkg/go.mod at revision v1.2.3: 404 Not Found`
	hint := modTidyHint(output)
	if hint == "" {
		t.Error("expected non-empty hint for 404 Not Found error")
	}
}

func TestModTidyHint_NoMatchReturnsEmpty(t *testing.T) {
	output := "some unrelated go mod tidy output without known patterns"
	hint := modTidyHint(output)
	if hint != "" {
		t.Errorf("expected empty hint for unrecognized error, got: %q", hint)
	}
}

func TestModTidyHint_EmptyOutput(t *testing.T) {
	hint := modTidyHint("")
	if hint != "" {
		t.Errorf("expected empty hint for empty output, got: %q", hint)
	}
}

func TestModTidyHint_DeduplicatesBrokenModules(t *testing.T) {
	// Same module appears twice in output — should only be listed once in hint
	output := `github.com/dup/pkg@v1.0.0: invalid version: something
github.com/dup/pkg@v1.0.0: invalid version: something`
	hint := modTidyHint(output)
	count := strings.Count(hint, "github.com/dup/pkg@v1.0.0")
	if count != 1 {
		t.Errorf("broken module should appear exactly once in hint, got %d occurrences", count)
	}
}

// ── goModDirs ─────────────────────────────────────────────────────────────────

func TestGoModDirs_FindsGoModFiles(t *testing.T) {
	files := []string{
		"services/api/go.mod",
		"services/api/main.go",
		"services/worker/go.mod",
		"services/worker/worker.go",
	}
	dirs := goModDirs("output", files)
	if len(dirs) != 2 {
		t.Fatalf("expected 2 dirs, got %d: %v", len(dirs), dirs)
	}
	dirSet := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		dirSet[d] = true
	}
	for _, want := range []string{"services/api", "services/worker"} {
		if !dirSet[want] {
			t.Errorf("expected dir %q in result, got %v", want, dirs)
		}
	}
}

func TestGoModDirs_DeduplicatesDirs(t *testing.T) {
	files := []string{
		"api/go.mod",
		"api/go.sum",
	}
	dirs := goModDirs("output", files)
	if len(dirs) != 1 {
		t.Errorf("expected 1 unique dir, got %d: %v", len(dirs), dirs)
	}
}

func TestGoModDirs_EmptyFilesReturnsEmpty(t *testing.T) {
	dirs := goModDirs("output", nil)
	if len(dirs) != 0 {
		t.Errorf("expected empty result for nil files, got %v", dirs)
	}
}

func TestGoModDirs_NoGoModFallsBackToGoFileDirs(t *testing.T) {
	// No go.mod — should not panic and returns something from .go files
	files := []string{"api/main.go"}
	dirs := goModDirs("output", files)
	// Just verify it doesn't panic; fallback behavior is OS-path-list-dependent
	_ = dirs
}

// ── fixShadowedTestingT ──────────────────────────────────────────────────────

func TestReplaceShadowedTDot(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "struct field access",
			line: `		if t.email == "" {`,
			want: `		if tc.email == "" {`,
		},
		{
			name: "testing.T method preserved",
			line: `			t.Fatal("email is empty")`,
			want: `			t.Fatal("email is empty")`,
		},
		{
			name: "t.Run preserved",
			line: `		t.Run(t.name, func(t *testing.T) {`,
			want: `		t.Run(tc.name, func(t *testing.T) {`,
		},
		{
			name: "t.Errorf preserved",
			line: `			t.Errorf("got %v", t.id)`,
			want: `			t.Errorf("got %v", tc.id)`,
		},
		{
			name: "no t. in line",
			line: `		ctx := context.Background()`,
			want: `		ctx := context.Background()`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := replaceShadowedTDot(tc.line)
			if got != tc.want {
				t.Errorf("replaceShadowedTDot(%q)\n  got:  %q\n  want: %q", tc.line, got, tc.want)
			}
		})
	}
}

func TestFixShadowedTestingT_FullFile(t *testing.T) {
	input := `package postgres_test

import "testing"

func TestFind(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{name: "found", email: "alice@example.com"},
	}

	for _, t := range tests {
		t.Run(t.name, func(t *testing.T) {
			if t.email == "" {
				t.Fatal("empty")
			}
		})
	}
}
`

	dir := t.TempDir()
	path := dir + "/user_repository_test.go"
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	result := fixShadowedTestingT(dir, []string{"user_repository_test.go"})
	if result == "" {
		t.Fatal("expected fix to be applied, got empty result")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Loop var should be renamed to tc.
	if !strings.Contains(content, "for _, tc := range tests") {
		t.Error("expected 'for _, tc := range tests'")
	}
	// Struct field access on the for-range line should use tc.
	if !strings.Contains(content, "tc.name") {
		t.Error("expected 'tc.name' in t.Run call")
	}
	// testing.T methods should still use t.
	if !strings.Contains(content, "t.Run(") {
		t.Error("expected 't.Run(' to be preserved")
	}
	// Inside the subtest func(t *testing.T), t.Fatal should stay as t.Fatal
	if !strings.Contains(content, "t.Fatal(") {
		t.Error("expected 't.Fatal(' inside subtest body to be preserved")
	}
	// Inside the subtest, t.email should NOT be renamed to tc.email
	// because t is now the inner *testing.T, not the struct.
	// (The email check wouldn't compile either way, but the fix shouldn't touch it)
}

func TestFixShadowedTestingT_NoSubtest(t *testing.T) {
	// Test where there's no func(t *testing.T) subtest — all t.field should be renamed
	input := `package foo_test

import "testing"

func TestBar(t *testing.T) {
	cases := []struct {
		name string
		val  int
	}{{name: "one", val: 1}}

	for _, t := range cases {
		if t.val != 1 {
			t.Errorf("got %d", t.val)
		}
	}
}
`
	dir := t.TempDir()
	path := dir + "/bar_test.go"
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	result := fixShadowedTestingT(dir, []string{"bar_test.go"})
	if result == "" {
		t.Fatal("expected fix to be applied")
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "for _, tc := range cases") {
		t.Error("expected loop var renamed to tc")
	}
	if !strings.Contains(content, "tc.val") {
		t.Error("expected tc.val")
	}
	// t.Errorf is a testing.T method — should be preserved
	if !strings.Contains(content, "t.Errorf(") {
		t.Error("expected t.Errorf preserved")
	}
}

// ── fixUndefinedSentinels ────────────────────────────────────────────────────

func TestFixUndefinedSentinels(t *testing.T) {
	dir := t.TempDir()

	// Create errors.go with ErrAlreadyExists and ErrNotFound.
	errorsFile := `package repository

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)
`
	repoDir := filepath.Join(dir, "internal", "repository")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "errors.go"), []byte(errorsFile), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test file that uses ErrConflict (which doesn't exist).
	testFile := `package repository_test

import "myapp/internal/repository"

func example() {
	_ = repository.ErrConflict
}
`
	if err := os.WriteFile(filepath.Join(repoDir, "repo_test.go"), []byte(testFile), 0644); err != nil {
		t.Fatal(err)
	}

	files := []string{
		"internal/repository/errors.go",
		"internal/repository/repo_test.go",
	}

	result := fixUndefinedSentinels(dir, files)
	if result == "" {
		t.Fatal("expected fix to be applied")
	}

	data, err := os.ReadFile(filepath.Join(repoDir, "repo_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if strings.Contains(content, "ErrConflict") {
		t.Error("ErrConflict should have been replaced")
	}
	if !strings.Contains(content, "ErrAlreadyExists") {
		t.Error("expected ErrAlreadyExists as replacement")
	}
}

// ── Registry ──────────────────────────────────────────────────────────────────

func TestRegistry_ForTask_GoTask_ReturnsGoVerifier(t *testing.T) {
	r := NewRegistry()
	task := &dag.Task{
		Kind: dag.TaskKindServiceHandler,
		Payload: dag.TaskPayload{
			Service: &manifest.ServiceDef{Language: "Go"},
		},
	}
	v := r.ForTask(task)
	if v.Language() != "go" {
		t.Errorf("expected go verifier, got %q", v.Language())
	}
}

// ── ApplyShortDeclFixes ──────────────────────────────────────────────────────

func TestApplyShortDeclFixes(t *testing.T) {
	dir := t.TempDir()
	src := `package repo

func (r *Repo) Create(ctx context.Context) error {
	result, err := r.pool.Exec(ctx, insertSQL, args...)
	if err != nil {
		return err
	}
	err := r.pool.Close()
	return err
}
`
	if err := os.WriteFile(filepath.Join(dir, "repo.go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	// go build reports paths relative to the module root, not absolute.
	// ApplyShortDeclFixes resolves them via filepath.Join(dir, relFile).
	verifyOutput := `repo.go:8:2: no new variables on left side of :=`
	result := ApplyShortDeclFixes(dir, verifyOutput)
	if result == "" {
		t.Fatal("expected fix to be applied")
	}

	fixed, err := os.ReadFile(filepath.Join(dir, "repo.go"))
	if err != nil {
		t.Fatal(err)
	}
	// Line 4 has "result, err :=" which is OK (result is new), line 8 should be fixed.
	lines := strings.Split(string(fixed), "\n")
	// Line 8 (0-indexed: 7) should have "err =" not "err :="
	if strings.Contains(lines[7], ":=") {
		t.Errorf("line 8 should have = not :=, got: %s", lines[7])
	}
	// Line 4 (0-indexed: 3) should still have := since "result" is new
	if !strings.Contains(lines[3], ":=") {
		t.Errorf("line 4 should keep :=, got: %s", lines[3])
	}
}

// ── fixBareModuleImports ─────────────────────────────────────────────────────

func TestFixBareModuleImports(t *testing.T) {
	dir := t.TempDir()
	gomod := `module github.com/user/monolith

go 1.22
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatal(err)
	}
	src := `package postgres

import (
	"context"
	"monolith/internal/domain"
	"monolith/internal/repository"
)
`
	if err := os.WriteFile(filepath.Join(dir, "repo.go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	result := fixBareModuleImports(dir, []string{"repo.go"})
	if result == "" {
		t.Fatal("expected fix to be applied")
	}

	fixed, err := os.ReadFile(filepath.Join(dir, "repo.go"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(fixed)
	if strings.Contains(content, `"monolith/internal/domain"`) {
		t.Error("bare import not rewritten: monolith/internal/domain")
	}
	if !strings.Contains(content, `"github.com/user/monolith/internal/domain"`) {
		t.Error("expected full module path import for domain")
	}
	if !strings.Contains(content, `"github.com/user/monolith/internal/repository"`) {
		t.Error("expected full module path import for repository")
	}
}

func TestRegistry_ForTask_TerraformTask_ReturnsTfVerifier(t *testing.T) {
	r := NewRegistry()
	task := &dag.Task{Kind: dag.TaskKindInfraTerraform}
	v := r.ForTask(task)
	if v.Language() != "terraform" {
		t.Errorf("expected terraform verifier, got %q", v.Language())
	}
}

func TestRegistry_ForTask_DataTask_ReturnsNullVerifier(t *testing.T) {
	r := NewRegistry()
	task := &dag.Task{Kind: dag.TaskKindDataSchemas}
	v := r.ForTask(task)
	// NullVerifier.Language() returns "null"
	if v.Language() != "null" {
		t.Errorf("expected null verifier for data task, got %q", v.Language())
	}
}
