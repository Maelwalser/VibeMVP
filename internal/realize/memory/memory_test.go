package memory

import (
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
