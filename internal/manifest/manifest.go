package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"
)

// ── Sentinel value sanitization ──────────────────────────────────────────────
// clearSentinels walks a struct tree via reflection and replaces sentinel string
// values ("None", "none", "Platform default") with empty strings so that
// omitempty can elide them from the serialized JSON.

func clearSentinels(v interface{}) {
	clearSentinelsValue(reflect.ValueOf(v))
}

// sentinelSet contains exact-match sentinel values that should be treated as
// empty (unset) when serialising the manifest.
var sentinelSet = map[string]bool{
	"None":             true,
	"none":             true,
	"N/A":              true,
	"Platform default": true,
	"None (external)":  true,
}

// isSentinel returns true if s is a known sentinel that should be cleared.
// It matches the static sentinelSet plus any string of the form "(…)" which
// covers all UI placeholder values like "(none)", "(no environments configured)",
// "(no services configured)", etc.
func isSentinel(s string) bool {
	if sentinelSet[s] {
		return true
	}
	if len(s) >= 2 && s[0] == '(' && s[len(s)-1] == ')' {
		return true
	}
	return false
}

func clearSentinelsValue(rv reflect.Value) {
	switch rv.Kind() {
	case reflect.Ptr:
		if rv.IsNil() {
			return
		}
		clearSentinelsValue(rv.Elem())
		// Nil out pointer to struct that became all-zero after clearing.
		if rv.Elem().Kind() == reflect.Struct && rv.Elem().Type() != reflect.TypeOf(time.Time{}) &&
			reflect.DeepEqual(rv.Elem().Interface(), reflect.Zero(rv.Elem().Type()).Interface()) {
			rv.Set(reflect.Zero(rv.Type()))
		}
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			return
		}
		for i := 0; i < rv.NumField(); i++ {
			clearSentinelsValue(rv.Field(i))
		}
	case reflect.String:
		if isSentinel(rv.String()) && rv.CanSet() {
			rv.SetString("")
		}
	case reflect.Slice:
		for i := 0; i < rv.Len(); i++ {
			clearSentinelsValue(rv.Index(i))
		}
		// For []string slices, remove entries that were cleared to "" so
		// placeholder values like "(no databases configured)" don't leave
		// empty strings in the serialized array.
		if rv.Type().Elem().Kind() == reflect.String && rv.CanSet() {
			j := 0
			for i := 0; i < rv.Len(); i++ {
				if rv.Index(i).String() != "" {
					if i != j {
						rv.Index(j).Set(rv.Index(i))
					}
					j++
				}
			}
			rv.SetLen(j)
			if j == 0 {
				rv.Set(reflect.Zero(rv.Type()))
			}
		}
	case reflect.Map:
		if rv.IsNil() {
			return
		}
		for _, key := range rv.MapKeys() {
			val := rv.MapIndex(key)
			if val.Kind() == reflect.String && isSentinel(val.String()) {
				rv.SetMapIndex(key, reflect.Value{}) // delete entry
			}
		}
		// Nil out empty maps so omitempty elides them.
		if rv.Len() == 0 && rv.CanSet() {
			rv.Set(reflect.Zero(rv.Type()))
		}
	}
}

// ── isEmpty helpers ────────────────────────────────────────────────────────────
// These determine whether a pillar has any meaningful user configuration so that
// unconfigured pillars are omitted from the saved manifest.json.

func (p BackendPillar) isEmpty() bool {
	return len(p.Services) == 0 && len(p.StackConfigs) == 0 &&
		p.Auth == nil && p.Messaging == nil && p.APIGateway == nil &&
		len(p.CommLinks) == 0 && len(p.Events) == 0 && len(p.JobQueues) == 0
}

func (p DataPillar) isEmpty() bool {
	return len(p.Databases) == 0 && len(p.Domains) == 0 && len(p.Entities) == 0 &&
		len(p.Cachings) == 0 && len(p.FileStorages) == 0 && len(p.Governances) == 0
}

func (p ContractsPillar) isEmpty() bool {
	return len(p.DTOs) == 0 && len(p.Endpoints) == 0 &&
		p.Versioning == nil && len(p.ExternalAPIs) == 0
}

func (p FrontendPillar) isEmpty() bool {
	return p.Tech == nil && p.Theme == nil && p.Navigation == nil &&
		len(p.Pages) == 0 && len(p.Components) == 0 && len(p.Assets) == 0 &&
		p.I18n == nil && p.A11ySEO == nil
}

func (p InfraPillar) isEmpty() bool {
	return p.Networking == nil && p.CICD == nil && p.Observability == nil &&
		len(p.Environments) == 0
}

func (p CrossCutPillar) isEmpty() bool {
	return p.Testing == nil && p.Docs == nil &&
		p.DependencyUpdates == "" && p.FeatureFlags == "" &&
		p.UptimeSLO == "" && p.LatencyP99 == "" &&
		p.BackendLinter == "" && p.FrontendLinter == ""
}

func (r RealizeOptions) isEmpty() bool {
	return r.AppName == "" && r.OutputDir == ""
}

// ── Realize options ───────────────────────────────────────────────────────────

// RealizeOptions holds configuration for the code-generation agent run.
type RealizeOptions struct {
	AppName       string            `json:"app_name,omitempty"`
	OutputDir     string            `json:"output_dir,omitempty"`
	Model         string            `json:"model,omitempty"` // kept for CLI backward compat
	Concurrency   int               `json:"concurrency,omitempty"`
	Verify        bool              `json:"verify,omitempty"`
	DryRun        bool              `json:"dry_run,omitempty"`
	SectionModels map[string]string `json:"section_models,omitempty"` // kept for backward compat
	// Provider and tier model assignments (set via the Realize tab UI).
	Provider   string `json:"provider,omitempty"`    // provider label (e.g. "Claude", "Gemini")
	TierFast   string `json:"tier_fast,omitempty"`   // model ID for low-complexity tasks
	TierMedium string `json:"tier_medium,omitempty"` // model ID for medium-complexity tasks
	TierSlow   string `json:"tier_slow,omitempty"`   // model ID for high-complexity / escalation
}

// ── Provider assignments ──────────────────────────────────────────────────────

// ProviderAssignment maps a pillar section to a specific AI provider and auth config.
type ProviderAssignment struct {
	Provider   string `json:"provider,omitempty"`
	Model      string `json:"model,omitempty"`
	Version    string `json:"version,omitempty"`
	Auth       string `json:"auth,omitempty"`
	Credential string `json:"credential,omitempty"` // API key or OAuth token
}

// ProviderAssignments maps section IDs (backend, data, etc.) to their provider config.
type ProviderAssignments map[string]ProviderAssignment

// ── Legacy pillars (preserved for existing code compatibility) ────────────────

// ── Root manifest ─────────────────────────────────────────────────────────────

// Manifest is the root document holding all configuration.
type Manifest struct {
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description,omitempty"`

	// Structured pillars
	Data      DataPillar      `json:"data"`
	Backend   BackendPillar   `json:"backend"`
	Contracts ContractsPillar `json:"contracts"`
	Frontend  FrontendPillar  `json:"frontend"`
	Infra     InfraPillar     `json:"infrastructure"`
	CrossCut  CrossCutPillar  `json:"cross_cutting"`
	Realize   RealizeOptions  `json:"realize,omitempty"`
}

// Load reads and parses a Manifest from a JSON file at path.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest %s: %w", path, err)
	}
	return &m, nil
}

// MarshalJSON serializes the Manifest, omitting any pillar that has no
// meaningful configuration so the output stays clean.
func (m Manifest) MarshalJSON() ([]byte, error) {
	// Strip sentinel values ("None", "Platform default") so omitempty can elide them.
	clearSentinels(&m)

	// shadow uses pointer pillar fields so encoding/json's omitempty works.
	type shadow struct {
		CreatedAt      time.Time        `json:"created_at"`
		Description    string           `json:"description,omitempty"`
		Data           *DataPillar      `json:"data,omitempty"`
		Backend        *BackendPillar   `json:"backend,omitempty"`
		Contracts      *ContractsPillar `json:"contracts,omitempty"`
		Frontend       *FrontendPillar  `json:"frontend,omitempty"`
		Infrastructure *InfraPillar     `json:"infrastructure,omitempty"`
		CrossCutting   *CrossCutPillar  `json:"cross_cutting,omitempty"`
		Realize        *RealizeOptions  `json:"realize,omitempty"`
	}

	s := shadow{
		CreatedAt:   m.CreatedAt,
		Description: m.Description,
	}

	if !m.Backend.isEmpty() {
		s.Backend = &m.Backend
	}
	if !m.Data.isEmpty() {
		s.Data = &m.Data
	}
	if !m.Contracts.isEmpty() {
		s.Contracts = &m.Contracts
	}
	if !m.Frontend.isEmpty() {
		s.Frontend = &m.Frontend
	}
	if !m.Infra.isEmpty() {
		s.Infrastructure = &m.Infra
	}
	if !m.CrossCut.isEmpty() {
		s.CrossCutting = &m.CrossCut
	}
	if !m.Realize.isEmpty() {
		s.Realize = &m.Realize
	}

	return json.Marshal(s)
}

// Save writes the manifest to path as indented JSON.
func (m *Manifest) Save(path string) error {
	m.CreatedAt = time.Now()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest to %s: %w", path, err)
	}
	return nil
}
