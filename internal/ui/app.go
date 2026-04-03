package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

type appPhase int

const (
	appPhaseWelcome appPhase = iota
	appPhaseMain
)

// AppModel is the top-level bubbletea model. It shows the welcome screen first,
// then transitions to the main editor Model once the user has chosen a project.
type AppModel struct {
	phase   appPhase
	welcome WelcomeModel
	main    Model
}

// NewApp creates the initial application model, starting at the welcome screen.
func NewApp() AppModel {
	return AppModel{
		phase:   appPhaseWelcome,
		welcome: newWelcomeModel(),
	}
}

// Init satisfies tea.Model.
func (a AppModel) Init() tea.Cmd {
	return a.welcome.Init()
}

// Update satisfies tea.Model.
func (a AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Window resize must reach whichever phase is active.
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		a.welcome.width = wsz.Width
		a.welcome.height = wsz.Height
		if a.phase == appPhaseMain {
			updated, cmd := a.main.Update(wsz)
			a.main = updated.(Model)
			return a, cmd
		}
		return a, nil
	}

	// Welcome complete: build save func and transition to main editor.
	if wc, ok := msg.(WelcomeCompleteMsg); ok {
		manifest.RecordRecentPath(wc.Path)
		saveFn := func(mf *manifest.Manifest) error {
			return mf.Save(wc.Path)
		}
		m := NewModel(saveFn)
		m.SetFilePath(wc.Path)
		if !wc.IsNew && wc.Manifest != nil {
			loaded, err := m.LoadManifestIntoModel(wc.Path)
			if err == nil {
				loaded.SetFilePath(wc.Path)
				loaded.SetSaveFunc(saveFn)
				m = loaded
			}
		} else if wc.IsNew {
			// Create the manifest file immediately so it exists on disk.
			empty := m.BuildManifest()
			_ = saveFn(empty)
		}
		// Inject the current terminal size so View() doesn't show "Loading…".
		sized, sizeCmd := m.Update(tea.WindowSizeMsg{
			Width:  a.welcome.width,
			Height: a.welcome.height,
		})
		a.main = sized.(Model)
		a.phase = appPhaseMain
		return a, tea.Batch(a.main.Init(), sizeCmd)
	}

	switch a.phase {
	case appPhaseWelcome:
		updated, cmd := a.welcome.Update(msg)
		a.welcome = updated.(WelcomeModel)
		return a, cmd
	case appPhaseMain:
		updated, cmd := a.main.Update(msg)
		a.main = updated.(Model)
		return a, cmd
	}
	return a, nil
}

// View satisfies tea.Model.
func (a AppModel) View() string {
	switch a.phase {
	case appPhaseWelcome:
		return a.welcome.View()
	case appPhaseMain:
		return a.main.View()
	}
	return ""
}

// MainModel returns the underlying editor model (valid after welcome completes).
func (a AppModel) MainModel() Model { return a.main }
