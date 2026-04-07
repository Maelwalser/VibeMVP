package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── Load ──────────────────────────────────────────────────────────────────────

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not valid json}"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/manifest.json")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	m := &Manifest{
		Description: "test project",
		Backend: BackendPillar{
			ArchPattern: ArchMonolith,
		},
	}
	if err := m.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Description != "test project" {
		t.Errorf("description mismatch: got %q", loaded.Description)
	}
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestSave_InjectsCreatedAt(t *testing.T) {
	before := time.Now()
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	m := &Manifest{Description: "ts test"}
	if err := m.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	after := time.Now()

	if m.CreatedAt.IsZero() {
		t.Error("Save() should set CreatedAt, but it is zero")
	}
	if m.CreatedAt.Before(before) || m.CreatedAt.After(after) {
		t.Errorf("CreatedAt %v is outside expected range [%v, %v]", m.CreatedAt, before, after)
	}
}

func TestSave_WritesValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	m := &Manifest{Description: "json test"}
	if err := m.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Errorf("saved file is not valid JSON: %v", err)
	}
}

// ── MarshalJSON (isEmpty-based omission) ─────────────────────────────────────

func TestMarshalJSON_EmptyPillarsOmitted(t *testing.T) {
	m := Manifest{Description: "minimal"}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	for _, key := range []string{"backend", "data", "contracts", "frontend", "infrastructure", "cross_cutting"} {
		if _, ok := raw[key]; ok {
			t.Errorf("empty pillar %q should be omitted from JSON, but was present", key)
		}
	}
}

func TestMarshalJSON_BackendIncludedWhenNonEmpty(t *testing.T) {
	m := Manifest{
		Backend: BackendPillar{
			ArchPattern: ArchMonolith,
			Services:    []ServiceDef{{Name: "api", Language: "Go"}},
		},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var raw map[string]any
	if _, ok := raw["backend"]; !ok {
		// Re-unmarshal and check
		json.Unmarshal(data, &raw)
	}
	json.Unmarshal(data, &raw)
	if _, ok := raw["backend"]; !ok {
		t.Error("non-empty backend pillar should be included in JSON")
	}
}

func TestMarshalJSON_DescriptionIncludedWhenSet(t *testing.T) {
	m := Manifest{Description: "my project"}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if _, ok := raw["description"]; !ok {
		t.Error("non-empty description should be present in JSON")
	}
}

// ── Round-trip: Save + Load ───────────────────────────────────────────────────

func TestRoundTrip_Description(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	orig := &Manifest{Description: "round-trip test"}
	if err := orig.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Description != orig.Description {
		t.Errorf("Description mismatch: want %q, got %q", orig.Description, loaded.Description)
	}
}

func TestRoundTrip_BackendPillar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	orig := &Manifest{
		Backend: BackendPillar{
			ArchPattern: ArchMicroservices,
			Services: []ServiceDef{
				{Name: "users", Language: "Go", Framework: "Gin"},
				{Name: "orders", Language: "TypeScript", Framework: "Express"},
			},
		},
	}
	if err := orig.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Backend.ArchPattern != ArchMicroservices {
		t.Errorf("ArchPattern mismatch: want %q, got %q", ArchMicroservices, loaded.Backend.ArchPattern)
	}
	if len(loaded.Backend.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(loaded.Backend.Services))
	}
	if loaded.Backend.Services[0].Name != "users" {
		t.Errorf("first service name mismatch: want 'users', got %q", loaded.Backend.Services[0].Name)
	}
}

func TestRoundTrip_RealizeOptions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	orig := &Manifest{
		Realize: RealizeOptions{
			AppName:   "my-app",
			OutputDir: "./output",
			Verify:    true,
		},
	}
	if err := orig.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Realize.AppName != "my-app" {
		t.Errorf("AppName mismatch: want 'my-app', got %q", loaded.Realize.AppName)
	}
	if loaded.Realize.OutputDir != "./output" {
		t.Errorf("OutputDir mismatch: want './output', got %q", loaded.Realize.OutputDir)
	}
	if !loaded.Realize.Verify {
		t.Error("Verify should be true after round-trip")
	}
}

func TestRoundTrip_SampleManifest(t *testing.T) {
	// Verify the existing sample fixture loads without error and retains basic fields.
	m, err := Load("../../testdata/sample-manifest.json")
	if err != nil {
		t.Fatalf("Load sample-manifest.json: %v", err)
	}
	if m == nil {
		t.Fatal("loaded manifest is nil")
	}
}

// ── Sentinel value stripping ────────────────────────────────────────────────

func TestMarshalJSON_SentinelValuesOmitted(t *testing.T) {
	m := Manifest{
		Contracts: ContractsPillar{
			Versioning: &APIVersioning{
				PerProtocolStrategies: map[string]string{"REST": "URL path (/v1/)"},
				CurrentVersion:        "v1",
				DeprecationPolicy:     "None",
			},
			DTOs: []DTODef{{Name: "Req", Category: "Request"}},
		},
		Backend: BackendPillar{
			Services: []ServiceDef{{Name: "api", ErrorFormat: "Platform default"}},
		},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	jsonStr := string(data)
	for _, sentinel := range []string{`"None"`, `"Platform default"`, `"none"`} {
		if contains(jsonStr, sentinel) {
			t.Errorf("sentinel value %s should be stripped from JSON output", sentinel)
		}
	}

	// Verify non-sentinel values are preserved.
	contracts := raw["contracts"].(map[string]any)
	versioning := contracts["versioning"].(map[string]any)
	if versioning["current_version"] != "v1" {
		t.Errorf("expected current_version=v1, got %v", versioning["current_version"])
	}
	if _, ok := versioning["deprecation_policy"]; ok {
		t.Error("deprecation_policy should be omitted when value is 'None'")
	}
}

func TestMarshalJSON_FalseBoolsOmitted(t *testing.T) {
	m := Manifest{
		Data: DataPillar{
			Databases: []DBSourceDef{{Alias: "db", Type: DBPostgres, IsCache: false}},
		},
		Frontend: FrontendPillar{
			Navigation: &NavigationConfig{NavType: "Top bar", Breadcrumbs: false, AuthAware: true},
		},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]any
	json.Unmarshal(data, &raw)

	// is_cache: false should be omitted
	db := raw["data"].(map[string]any)["databases"].([]any)[0].(map[string]any)
	if _, ok := db["is_cache"]; ok {
		t.Error("is_cache: false should be omitted from JSON")
	}

	// breadcrumbs: false should be omitted, auth_aware: true should remain
	nav := raw["frontend"].(map[string]any)["navigation"].(map[string]any)
	if _, ok := nav["breadcrumbs"]; ok {
		t.Error("breadcrumbs: false should be omitted from JSON")
	}
	if nav["auth_aware"] != true {
		t.Error("auth_aware: true should be preserved in JSON")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
