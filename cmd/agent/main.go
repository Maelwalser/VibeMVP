package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-mvp/internal/manifest"
	"github.com/vibe-mvp/internal/ui"
)

const manifestPath = "manifest.json"

func main() {
	saveFunc := func(m *manifest.Manifest) error {
		return m.Save(manifestPath)
	}

	model := ui.NewModel(saveFunc)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// After the TUI exits, print a summary if the manifest was populated.
	if m, ok := finalModel.(ui.Model); ok {
		mf := m.BuildManifest()
		if mf.Topology.ArchPattern != "" {
			fmt.Printf("\nManifest saved to %s\n", manifestPath)
			fmt.Printf("Topology  : %s · %s\n", mf.Topology.ArchPattern, mf.Topology.CommProtocol)
			fmt.Printf("Backend   : %s  [%s]\n", mf.Backend.Runtime, mf.Backend.PrimaryDB)
			fmt.Printf("SLO       : %s uptime  RTO=%s  RPO=%s\n",
				mf.GlobalNFR.UptimeSLO, mf.GlobalNFR.RTO, mf.GlobalNFR.RPO)
		}
	}
}
