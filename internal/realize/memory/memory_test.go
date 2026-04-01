package memory

import (
	"strings"
	"testing"

	"github.com/vibe-mvp/internal/realize/dag"
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
	longContent := strings.Repeat("x", maxFileChars+100)
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
		Kind:         dag.TaskKindService,
		Label:        "API service",
		Dependencies: []string{"data.schemas"},
	}

	filesA := []dag.GeneratedFile{
		{Path: "internal/models/types.go", Content: "package models\n\ntype User struct{}"},
	}
	mem.Record(taskA, filesA)

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
		Kind:         dag.TaskKindService,
		Label:        "API service",
		Dependencies: []string{"data.schemas", "nonexistent"},
	}

	// Record only data.schemas.
	mem.Record(&dag.Task{ID: "data.schemas"}, []dag.GeneratedFile{
		{Path: "types.go", Content: "package x"},
	})

	deps := mem.DepsOf(task)
	if len(deps) != 1 {
		t.Errorf("expected 1 dep (nonexistent should be skipped), got %d", len(deps))
	}
}

func TestSharedMemory_TotalBudgetCap(t *testing.T) {
	mem := New()

	// Create many upstream tasks each with content near the budget.
	chunkSize := maxTotalChars / 3
	for i := 0; i < 10; i++ {
		id := strings.Repeat("a", i+1)
		mem.Record(
			&dag.Task{ID: id, Kind: dag.TaskKindDataSchemas},
			[]dag.GeneratedFile{
				{Path: "types.go", Content: strings.Repeat("y", chunkSize)},
			},
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
	if total > maxTotalChars+100 { // allow small overhead from truncation notices
		t.Errorf("total shared context %d exceeds budget %d", total, maxTotalChars)
	}
}

func TestSharedMemory_NoDepsReturnsEmpty(t *testing.T) {
	mem := New()
	task := &dag.Task{ID: "root", Dependencies: nil}
	if deps := mem.DepsOf(task); len(deps) != 0 {
		t.Errorf("expected empty deps for root task, got %d", len(deps))
	}
}
