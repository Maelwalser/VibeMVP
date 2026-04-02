package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
)

type welcomePhase int

const (
	welcomePhaseMenu     welcomePhase = iota
	welcomePhaseOpenPath              // user entering path to existing manifest
	welcomePhaseNewName               // user entering new project name
)

// WelcomeCompleteMsg is emitted as a Cmd when the welcome flow finishes.
type WelcomeCompleteMsg struct {
	Path     string             // file path to save/load
	IsNew    bool               // true = new project, false = open existing
	Manifest *manifest.Manifest // populated when opening an existing manifest
}

// WelcomeModel is the initial screen presented on startup.
type WelcomeModel struct {
	phase  welcomePhase
	cursor int // 0 = Open Existing, 1 = New Project
	input  textinput.Model
	errMsg string
	width  int
	height int
}

func newWelcomeModel() WelcomeModel {
	inp := newFormInput()
	inp.Width = 48
	return WelcomeModel{input: inp}
}

// Init satisfies tea.Model — starts the animation ticker.
func (w WelcomeModel) Init() tea.Cmd {
	return uiTick()
}

// Update satisfies tea.Model.
func (w WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height
		return w, nil
	case uiTickMsg:
		AnimFrame = (AnimFrame + 1) % 2
		return w, uiTick()
	case tea.KeyMsg:
		return w.handleKey(msg)
	}
	// Forward non-key events to the text input when active.
	if w.phase != welcomePhaseMenu {
		var cmd tea.Cmd
		w.input, cmd = w.input.Update(msg)
		return w, cmd
	}
	return w, nil
}

func (w WelcomeModel) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch w.phase {
	case welcomePhaseMenu:
		switch key.String() {
		case "j", "down":
			w.cursor = (w.cursor + 1) % 2
		case "k", "up":
			w.cursor = (w.cursor - 1 + 2) % 2
		case "enter", " ":
			if w.cursor == 0 {
				w.phase = welcomePhaseOpenPath
				w.input.Placeholder = "e.g. ./project/manifest.json"
				w.input.SetValue("")
				w.errMsg = ""
				return w, w.input.Focus()
			}
			w.phase = welcomePhaseNewName
			w.input.Placeholder = "e.g. my-app"
			w.input.SetValue("")
			w.errMsg = ""
			return w, w.input.Focus()
		case "ctrl+c", "q":
			return w, tea.Quit
		}

	case welcomePhaseOpenPath:
		switch key.String() {
		case "esc":
			w.phase = welcomePhaseMenu
			w.input.Blur()
			w.errMsg = ""
			return w, nil
		case "enter":
			path := strings.TrimSpace(w.input.Value())
			if path == "" {
				w.errMsg = "path cannot be empty"
				return w, nil
			}
			mf, err := manifest.Load(path)
			if err != nil {
				w.errMsg = fmt.Sprintf("error: %v", err)
				return w, nil
			}
			w.input.Blur()
			return w, func() tea.Msg {
				return WelcomeCompleteMsg{Path: path, IsNew: false, Manifest: mf}
			}
		default:
			var cmd tea.Cmd
			w.input, cmd = w.input.Update(key)
			return w, cmd
		}

	case welcomePhaseNewName:
		switch key.String() {
		case "esc":
			w.phase = welcomePhaseMenu
			w.input.Blur()
			w.errMsg = ""
			return w, nil
		case "enter":
			name := strings.TrimSpace(w.input.Value())
			if name == "" {
				w.errMsg = "project name cannot be empty"
				return w, nil
			}
			path := name + ".json"
			w.input.Blur()
			return w, func() tea.Msg {
				return WelcomeCompleteMsg{Path: path, IsNew: true, Manifest: nil}
			}
		default:
			var cmd tea.Cmd
			w.input, cmd = w.input.Update(key)
			return w, cmd
		}
	}
	return w, nil
}

// View satisfies tea.Model.
func (w WelcomeModel) View() string {
	if w.width == 0 {
		return "Loading…"
	}

	const boxWidth = 54

	var b strings.Builder

	logo := StyleNeonCyan.Bold(true).Render("VibeMenu")
	subtitle := StyleHelpDesc.Render("declarative system architecture")
	b.WriteString(lipgloss.NewStyle().Width(boxWidth - 4).Align(lipgloss.Center).Render(logo))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Width(boxWidth - 4).Align(lipgloss.Center).Render(subtitle))
	b.WriteString("\n\n")
	b.WriteString(StyleHelpDesc.Render(strings.Repeat("─", boxWidth-4)))
	b.WriteString("\n\n")

	switch w.phase {
	case welcomePhaseMenu:
		options := []string{"Open Existing Manifest", "New Project"}
		for i, opt := range options {
			if i == w.cursor {
				b.WriteString(StyleNeonViolet.Render("❯ ") + StyleFieldValActive.Render(opt))
			} else {
				b.WriteString(StyleHelpDesc.Render("  ") + StyleFieldVal.Render(opt))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(StyleHelpDesc.Render("j/k: navigate  Enter: select  q: quit"))

	case welcomePhaseOpenPath:
		b.WriteString(StyleNeonCyan.Render("Open Existing Manifest"))
		b.WriteString("\n\n")
		b.WriteString(StyleHelpKey.Render("Path: ") + w.input.View())
		if w.errMsg != "" {
			b.WriteString("\n\n")
			b.WriteString(StyleMsgErr.Render(w.errMsg))
		}
		b.WriteString("\n\n")
		b.WriteString(StyleHelpDesc.Render("Enter: load   Esc: back"))

	case welcomePhaseNewName:
		b.WriteString(StyleNeonCyan.Render("New Project"))
		b.WriteString("\n\n")
		b.WriteString(StyleHelpKey.Render("Name: ") + w.input.View())
		if w.errMsg != "" {
			b.WriteString("\n\n")
			b.WriteString(StyleMsgErr.Render(w.errMsg))
		}
		b.WriteString("\n\n")
		b.WriteString(StyleHelpDesc.Render("Enter: create   Esc: back"))
	}

	box := StyleModalBorder.Width(boxWidth).Render(b.String())
	return lipgloss.Place(w.width, w.height, lipgloss.Center, lipgloss.Center, box)
}
