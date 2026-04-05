package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ProvidersPath returns ~/.vibemenu/providers.json — the global credentials file
// shared across all projects, analogous to ~/.claude/ for Claude Code.
// Credentials are kept here so manifest.json stays credential-free and safe to commit.
func ProvidersPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "providers.json" // fallback to cwd on unusual systems
	}
	return filepath.Join(home, ".vibemenu", "providers.json")
}

// SaveProviders writes the provider assignments to path as indented JSON.
// The parent directory is created if it does not exist.
// If pa is nil or empty the file is left unchanged (not written).
func SaveProviders(path string, pa ProviderAssignments) error {
	if len(pa) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create providers directory: %w", err)
	}
	data, err := json.MarshalIndent(pa, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal providers: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write providers to %s: %w", path, err)
	}
	return nil
}

// LoadProviders reads provider assignments from path. Returns nil (not an error)
// when the file does not exist — callers treat that as "no providers configured".
func LoadProviders(path string) (ProviderAssignments, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read providers %s: %w", path, err)
	}
	var pa ProviderAssignments
	if err := json.Unmarshal(data, &pa); err != nil {
		return nil, fmt.Errorf("failed to parse providers %s: %w", path, err)
	}
	return pa, nil
}
