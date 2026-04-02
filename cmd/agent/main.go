package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/ui"
)

func main() {
	app := ui.NewApp()
	p := tea.NewProgram(app, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	a, ok := finalModel.(ui.AppModel)
	if !ok {
		return
	}
	m := a.MainModel()
	mf := m.BuildManifest()

	if mf.Backend.ArchPattern != "" {
		manifestPath := a.MainModel().FilePath()
		fmt.Printf("\nManifest saved to %s\n", manifestPath)
		fmt.Printf("Backend   : %s  [%s]\n", mf.Backend.ArchPattern, mf.Backend.ComputeEnv)
		fmt.Printf("Entities  : %d defined\n", len(mf.Entities))
		fmt.Printf("Databases : %d defined\n", len(mf.Databases))
		fmt.Printf("Services  : %d defined\n", len(mf.Backend.Services))
	}
	if m.RealizeTriggered() {
		r := mf.Realize
		manifestPath := a.MainModel().FilePath()
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
