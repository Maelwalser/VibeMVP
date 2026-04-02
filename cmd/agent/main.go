package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-mvp/internal/manifest"
	"github.com/vibe-mvp/internal/ui"
)

func main() {
	// Determine manifest path: CLI arg > default.
	manifestPath := "manifest.json"
	if len(os.Args) > 1 {
		manifestPath = os.Args[1]
	}

	saveFunc := func(m *manifest.Manifest) error {
		return m.Save(manifestPath)
	}

	model := ui.NewModel(saveFunc)

	// Auto-load if the manifest file already exists on disk.
	if _, err := os.Stat(manifestPath); err == nil {
		loaded, loadErr := model.LoadManifestIntoModel(manifestPath)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load %s: %v\n", manifestPath, loadErr)
		} else {
			loaded.SetFilePath(manifestPath)
			model = loaded
		}
	}

	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// After the TUI exits, print a summary and realize command if triggered.
	if m, ok := finalModel.(ui.Model); ok {
		mf := m.BuildManifest()
		if mf.Backend.ArchPattern != "" {
			fmt.Printf("\nManifest saved to %s\n", manifestPath)
			fmt.Printf("Backend   : %s  [%s]\n", mf.Backend.ArchPattern, mf.Backend.ComputeEnv)
			fmt.Printf("Entities  : %d defined\n", len(mf.Entities))
			fmt.Printf("Databases : %d defined\n", len(mf.Databases))
			fmt.Printf("Services  : %d defined\n", len(mf.Backend.Services))
		}
		if m.RealizeTriggered() {
			r := mf.Realize
			fmt.Printf("\n── Realization ──────────────────────────────────────────\n")
			fmt.Printf("App name    : %s\n", r.AppName)
			fmt.Printf("Output dir  : %s\n", r.OutputDir)
			fmt.Printf("Model       : %s\n", r.Model)
			fmt.Printf("Concurrency : %d\n", r.Concurrency)
			fmt.Printf("Verify      : %v\n", r.Verify)
			fmt.Printf("Dry run     : %v\n", r.DryRun)
			fmt.Printf("\nTo start realization, run:\n")
			fmt.Printf("  realize --manifest %s --app-name %q --output-dir %q --model %s --concurrency %d\n",
				manifestPath, r.AppName, r.OutputDir, r.Model, r.Concurrency)
		}
	}
}
