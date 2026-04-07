package memory

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vibe-menu/internal/realize/config"
	"github.com/vibe-menu/internal/realize/dag"
)

func TestIsHighValue(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"services/user/types.go", true},
		{"services/user/models.go", true},
		{"internal/domain/schema.go", true},
		{"proto/user.proto", true},
		{"openapi.yaml", true},
		{"types.ts", true},
		{"models.ts", true},
		{"services/user/handler.go", false},
		{"services/user/main.go", false},
		{"Dockerfile", false},
		{"deploy/main.tf", false},
		{".github/ci.yml", false},
		// Python patterns
		{"app/models.py", true},
		{"app/schemas.py", true},
		{"app/types.py", true},
		{"app/interfaces.py", true},
		{"app/entities.py", true},
		{"app/utils.py", false},
		// Java patterns
		{"src/main/java/UserEntity.java", true},
		{"src/main/java/UserRepository.java", true},
		{"src/main/java/UserModel.java", true},
		{"src/main/java/UserController.java", false},
		// Config/dependency files
		{"go.mod", true},
		{"package.json", true},
		{"pyproject.toml", true},
		{"requirements.txt", true},
		{"tsconfig.json", false},
		// Interface directories
		{"internal/interfaces/user.go", true},
		{"pkg/interfaces/repo.ts", true},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := isHighValue(tc.path); got != tc.want {
				t.Errorf("isHighValue(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestBuildExcerpts_TruncatesLargeFiles(t *testing.T) {
	longContent := strings.Repeat("x", config.MaxFileChars+100)
	files := []dag.GeneratedFile{
		{Path: "types.go", Content: longContent},
	}

	excerpts := buildExcerpts(files)
	if len(excerpts) != 1 {
		t.Fatalf("expected 1 excerpt, got %d", len(excerpts))
	}
	if !excerpts[0].Truncated {
		t.Error("expected Truncated=true for oversized file")
	}
	if len(excerpts[0].Content) >= len(longContent) {
		t.Error("expected content to be shorter than original")
	}
}

func TestBuildExcerpts_PrioritisesHighValueFiles(t *testing.T) {
	files := []dag.GeneratedFile{
		{Path: "handler.go", Content: "handler"},
		{Path: "main.go", Content: "main"},
		{Path: "types.go", Content: "types"},
		{Path: "models.go", Content: "models"},
	}

	excerpts := buildExcerpts(files)
	if len(excerpts) != 4 {
		t.Fatalf("expected 4 excerpts, got %d", len(excerpts))
	}
	// First two should be the high-value files.
	if excerpts[0].Path != "types.go" && excerpts[0].Path != "models.go" {
		t.Errorf("expected high-value file first, got %q", excerpts[0].Path)
	}
	if excerpts[1].Path != "types.go" && excerpts[1].Path != "models.go" {
		t.Errorf("expected high-value file second, got %q", excerpts[1].Path)
	}
}

func TestSharedMemory_RecordAndDepsOf(t *testing.T) {
	mem := New()

	taskA := &dag.Task{
		ID:    "data.schemas",
		Kind:  dag.TaskKindDataSchemas,
		Label: "Data schemas",
	}
	taskB := &dag.Task{
		ID:           "backend.service.api",
		Kind:         dag.TaskKindServiceHandler,
		Label:        "API service",
		Dependencies: []string{"data.schemas"},
	}

	filesA := []dag.GeneratedFile{
		{Path: "internal/models/types.go", Content: "package models\n\ntype User struct{}"},
	}
	mem.Record(taskA, filesA, "")

	deps := mem.DepsOf(taskB)
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency output, got %d", len(deps))
	}
	if deps[0].TaskID != "data.schemas" {
		t.Errorf("expected taskID=data.schemas, got %q", deps[0].TaskID)
	}
	if len(deps[0].Files) != 1 {
		t.Fatalf("expected 1 file in dep output, got %d", len(deps[0].Files))
	}
}

func TestSharedMemory_MissingDepIsSkipped(t *testing.T) {
	mem := New()

	task := &dag.Task{
		ID:           "backend.service.api",
		Kind:         dag.TaskKindServiceLogic,
		Label:        "API service",
		Dependencies: []string{"data.schemas", "nonexistent"},
	}

	// Record only data.schemas.
	mem.Record(&dag.Task{ID: "data.schemas"}, []dag.GeneratedFile{
		{Path: "types.go", Content: "package x"},
	}, "")

	deps := mem.DepsOf(task)
	if len(deps) != 1 {
		t.Errorf("expected 1 dep (nonexistent should be skipped), got %d", len(deps))
	}
}

func TestSharedMemory_TotalBudgetCap(t *testing.T) {
	mem := New()

	// Create many upstream tasks each with content near the budget.
	chunkSize := config.MaxTotalChars / 3
	for i := 0; i < 10; i++ {
		id := strings.Repeat("a", i+1)
		mem.Record(
			&dag.Task{ID: id, Kind: dag.TaskKindDataSchemas},
			[]dag.GeneratedFile{
				{Path: "types.go", Content: strings.Repeat("y", chunkSize)},
			},
			"",
		)
	}

	depIDs := make([]string, 10)
	for i := range depIDs {
		depIDs[i] = strings.Repeat("a", i+1)
	}
	consumer := &dag.Task{
		ID:           "consumer",
		Dependencies: depIDs,
	}

	deps := mem.DepsOf(consumer)

	// Tally total content chars across all returned deps.
	total := 0
	for _, d := range deps {
		for _, f := range d.Files {
			total += len(f.Content)
		}
	}
	if total > config.MaxTotalChars+100 { // allow small overhead from truncation notices
		t.Errorf("total shared context %d exceeds budget %d", total, config.MaxTotalChars)
	}
}

func TestSharedMemory_NoDepsReturnsEmpty(t *testing.T) {
	mem := New()
	task := &dag.Task{ID: "root", Dependencies: nil}
	if deps := mem.DepsOf(task); len(deps) != 0 {
		t.Errorf("expected empty deps for root task, got %d", len(deps))
	}
}

// TestSharedMemory_PriorityWeightedBudget verifies that DepsOf allocates more
// budget to high-priority dependencies (auth, service logic) than low-priority
// ones (data schemas), preventing verbose but low-priority outputs from crowding
// out critical signatures.
func TestSharedMemory_PriorityWeightedBudget(t *testing.T) {
	mem := New()

	// Create two upstream tasks with multiple files that collectively exceed the
	// total DepsOf budget for a handler task (20000 chars). Each individual file
	// must stay under MaxFileChars (4000) so buildExcerpts doesn't truncate first.
	// Use Go type declarations so extractSignatures preserves content.
	makeFiles := func(pkg string, count int) []dag.GeneratedFile {
		files := make([]dag.GeneratedFile, count)
		for i := 0; i < count; i++ {
			files[i] = dag.GeneratedFile{
				Path:    fmt.Sprintf("internal/%s/type_%d.go", pkg, i),
				Content: fmt.Sprintf("package %s\n\ntype T%d struct {\n%s}\n", pkg, i, strings.Repeat("\tF string\n", 200)),
			}
		}
		return files
	}

	// Each task: 5 files * ~2800 chars = ~14000 chars total per task.
	// Combined: ~28000 chars, well above the 20000 handler budget.
	mem.Record(
		&dag.Task{ID: "data.schemas", Kind: dag.TaskKindDataSchemas},
		makeFiles("domain", 5),
		"",
	)
	mem.Record(
		&dag.Task{ID: "backend.auth", Kind: dag.TaskKindAuth},
		makeFiles("auth", 5),
		"",
	)

	// Consumer task depends on both, data listed first (would exhaust budget under flat approach).
	consumer := &dag.Task{
		ID:           "svc.monolith.handler",
		Kind:         dag.TaskKindServiceHandler,
		Dependencies: []string{"data.schemas", "backend.auth"},
	}

	deps := mem.DepsOf(consumer)
	if len(deps) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(deps))
	}

	// Find auth dep and data dep.
	var authChars, dataChars int
	for _, d := range deps {
		for _, f := range d.Files {
			if d.TaskID == "backend.auth" {
				authChars += len(f.Content)
			} else {
				dataChars += len(f.Content)
			}
		}
	}

	// Auth (priority 0.9) should get more budget than data schemas (priority 0.3).
	if authChars <= dataChars {
		t.Errorf("auth (%d chars) should get more budget than data schemas (%d chars) due to higher priority",
			authChars, dataChars)
	}
}

// TestSharedMemory_TypeConflicts verifies that RegisterTypes detects when the same
// type name is declared in different packages.
func TestSharedMemory_TypeConflicts(t *testing.T) {
	mem := New()

	// Register User from domain package.
	mem.RegisterTypes(map[string]TypeEntry{
		"User": {Package: "internal/domain", File: "internal/domain/user.go", Definition: "type User struct{}"},
	})
	// Register User from a different package — should trigger conflict.
	mem.RegisterTypes(map[string]TypeEntry{
		"User": {Package: "internal/dto", File: "internal/dto/user.go", Definition: "type User struct{}"},
	})
	// Register Order from domain — no conflict.
	mem.RegisterTypes(map[string]TypeEntry{
		"Order": {Package: "internal/domain", File: "internal/domain/order.go", Definition: "type Order struct{}"},
	})

	conflicts := mem.TypeConflicts()
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].TypeName != "User" {
		t.Errorf("expected conflict on User, got %q", conflicts[0].TypeName)
	}
	if conflicts[0].First.Package != "internal/domain" || conflicts[0].Second.Package != "internal/dto" {
		t.Errorf("unexpected conflict packages: %q vs %q", conflicts[0].First.Package, conflicts[0].Second.Package)
	}
}

// TestSharedMemory_TypeConflicts_SamePackageNoConflict verifies that re-registering
// a type from the same package does not create a conflict.
func TestSharedMemory_TypeConflicts_SamePackageNoConflict(t *testing.T) {
	mem := New()

	mem.RegisterTypes(map[string]TypeEntry{
		"User": {Package: "internal/domain", File: "internal/domain/user.go"},
	})
	mem.RegisterTypes(map[string]TypeEntry{
		"User": {Package: "internal/domain", File: "internal/domain/user.go"},
	})

	if len(mem.TypeConflicts()) != 0 {
		t.Error("same-package re-registration should not create a conflict")
	}
}

// TestSharedMemory_RecordStripsOutputDir verifies that Record removes the outputDir
// prefix from file paths before storing excerpts, so agents see module-relative paths
// like "internal/domain/user.go" rather than "backend/internal/domain/user.go".
// This prevents agents from constructing wrong import paths using the output directory
// name instead of the Go module name (or equivalent in other languages).
func TestSharedMemory_RecordStripsOutputDir(t *testing.T) {
	mem := New()

	taskA := &dag.Task{
		ID:    "data.schemas",
		Kind:  dag.TaskKindDataSchemas,
		Label: "Data schemas",
	}
	taskB := &dag.Task{
		ID:           "svc.monolith.repository",
		Kind:         dag.TaskKindServiceRepository,
		Label:        "Repository",
		Dependencies: []string{"data.schemas"},
	}

	// Simulate what runner.go commit() does: apply outputDir prefix to files before
	// passing to Record (these are the disk-relative paths).
	prefixedFiles := []dag.GeneratedFile{
		{Path: "backend/internal/domain/user.go", Content: "package domain\n\ntype User struct { ID string }"},
	}
	mem.Record(taskA, prefixedFiles, "backend")

	deps := mem.DepsOf(taskB)
	if len(deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(deps))
	}
	if len(deps[0].Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(deps[0].Files))
	}

	// The excerpt path must NOT contain the "backend/" prefix.
	gotPath := deps[0].Files[0].Path
	if gotPath != "internal/domain/user.go" {
		t.Errorf("excerpt path = %q, want %q (outputDir prefix should be stripped)", gotPath, "internal/domain/user.go")
	}

	// rawPaths must still use the prefixed path for disk staging.
	rawPaths := mem.CommittedPaths([]string{"data.schemas"})
	if len(rawPaths) != 1 || rawPaths[0] != "backend/internal/domain/user.go" {
		t.Errorf("rawPaths = %v, want [backend/internal/domain/user.go]", rawPaths)
	}
}
