package ui

import (
	"fmt"
	"path/filepath"
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
	phase      welcomePhase
	cursor     int    // menu index
	recentPath string // most recently used manifest path, or ""
	input      textinput.Model
	errMsg     string
	width      int
	height     int
}

func newWelcomeModel() WelcomeModel {
	inp := newFormInput()
	inp.Width = 46
	recent := manifest.LoadRecentPaths()
	var recentPath string
	if len(recent) > 0 {
		recentPath = recent[0]
	}
	return WelcomeModel{input: inp, recentPath: recentPath}
}

// numMenuOptions returns how many options appear in the main menu.
func (w WelcomeModel) numMenuOptions() int {
	if w.recentPath != "" {
		return 3
	}
	return 2
}

// shortPath returns up to the last two path components for display.
func shortPath(p string) string {
	dir, file := filepath.Split(filepath.Clean(p))
	parent := filepath.Base(filepath.Clean(dir))
	if parent == "." || parent == "" {
		return file
	}
	return filepath.Join(parent, file)
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
		n := w.numMenuOptions()
		switch key.String() {
		case "j", "down":
			w.cursor = (w.cursor + 1) % n
		case "k", "up":
			w.cursor = (w.cursor - 1 + n) % n
		case "enter", " ":
			openIdx := 0
			newIdx := 1
			if w.recentPath != "" {
				openIdx = 1
				newIdx = 2
			}
			switch {
			case w.recentPath != "" && w.cursor == 0:
				mf, err := manifest.Load(w.recentPath)
				if err != nil {
					w.errMsg = fmt.Sprintf("error: %v", err)
					return w, nil
				}
				path := w.recentPath
				return w, func() tea.Msg {
					return WelcomeCompleteMsg{Path: path, IsNew: false, Manifest: mf}
				}
			case w.cursor == openIdx:
				w.phase = welcomePhaseOpenPath
				w.input.Placeholder = "e.g. ./project/manifest.json"
				w.input.SetValue("")
				w.errMsg = ""
				return w, w.input.Focus()
			case w.cursor == newIdx:
				w.phase = welcomePhaseNewName
				w.input.Placeholder = "e.g. my-app"
				w.input.SetValue("")
				w.errMsg = ""
				return w, w.input.Focus()
			}
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

// vibeBanner — ASCII art wordmark, 7 lines (no trailing spaces).
var vibeBanner = [7]string{
	`  _     _   __     _____     _____  __    __    _____  __   __   __    __`,
	` /_/\ /\_\ /\_\  /\  __/\  /\_____\/_/\  /\_\ /\_____\/_/\ /\_\ /\_\  /_/\`,
	` ) ) ) ( ( \/_/  ) )(_ ) )( (_____/) ) \/ ( (( (_____/) ) \ ( (( ( (  ) ) )`,
	`/_/ / \ \_\ /\_\/ / __/ /  \ \__\ /_/ \  / \_\\ \__\ /_/   \ \_\\ \ \/ / /`,
	`\ \ \_/ / // / /\ \  _\ \  / /__/_\ \ \\// / // /__/_\ \ \   / / \ \  / /`,
	` \ \   / /( (_(  ) )(__) )( (_____\)_) )( (_(( (_____\)_) \ (_(  ( (__) )`,
	`  \_\_/_/  \/_/  \/____\/  \/_____/\_\/  \/_/ \/_____/\_\/ \/_/   \/__\/`,
}

// View satisfies tea.Model.
func (w WelcomeModel) View() string {
	if w.width == 0 {
		return "Loading…"
	}

	modalBg := lipgloss.Color(clrBg2)
	dim := lipgloss.Color(clrFgDim)
	border := lipgloss.Color(clrComment)

	// innerW is derived from the widest banner line plus 2 so the bounding box
	// of the art has exactly 1 space of margin on each side (left and right).
	innerW := 0
	for _, line := range vibeBanner {
		if len(line) > innerW {
			innerW = len(line)
		}
	}
	innerW += 2
	boxWidth := innerW + 4 // border (1+1) + padding (1+1)

	center := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Background(modalBg)
	left := lipgloss.NewStyle().Width(innerW).Background(modalBg)

	bannerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrYellow)).
		Bold(true).
		Background(modalBg)

	var b strings.Builder

	// ── Wordmark ─────────────────────────────────────────────────────────────
	b.WriteString("\n")
	for _, line := range vibeBanner {
		b.WriteString(bannerStyle.Render(" " + line))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Subtitle.
	b.WriteString(center.Render(
		lipgloss.NewStyle().Foreground(dim).Background(modalBg).
			Render("declarative system architecture"),
	))
	b.WriteString("\n\n")

	// Thin rule.
	b.WriteString(lipgloss.NewStyle().Foreground(border).Background(modalBg).
		Render(strings.Repeat("─", innerW)))
	b.WriteString("\n\n")

	// ── Phase content ─────────────────────────────────────────────────────────
	switch w.phase {
	case welcomePhaseMenu:
		var options []string
		var icons []string
		if w.recentPath != "" {
			options = append(options, "Continue  "+shortPath(w.recentPath))
			icons = append(icons, "▶")
		}
		options = append(options, "Open Existing Manifest", "New Project")
		icons = append(icons, "◈", "+")

		for i, opt := range options {
			if i == w.cursor {
				row := lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Bold(true).Background(modalBg).Render("❯ "+icons[i]+" ") +
					lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg)).Bold(true).Background(modalBg).Render(opt)
				b.WriteString(left.Render(row))
			} else {
				row := lipgloss.NewStyle().Foreground(dim).Background(modalBg).Render("  "+icons[i]+" ") +
					lipgloss.NewStyle().Foreground(dim).Background(modalBg).Render(opt)
				b.WriteString(left.Render(row))
			}
			b.WriteString("\n")
		}
		if w.errMsg != "" {
			b.WriteString("\n")
			b.WriteString(StyleMsgErr.Background(modalBg).Render("  ✗ " + w.errMsg))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(dim).Background(modalBg).
			Render("  j/k  navigate   Enter  select   q  quit"))

	case welcomePhaseOpenPath:
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Bold(true).Background(modalBg).
			Render("◈  Open Existing Manifest"))
		b.WriteString("\n\n")
		b.WriteString(StyleFieldKeyActive.Background(modalBg).Render("  Path ") +
			StyleEquals.Background(modalBg).Render(" = ") +
			w.input.View())
		if w.errMsg != "" {
			b.WriteString("\n\n")
			b.WriteString(StyleMsgErr.Background(modalBg).Render("  ✗ " + w.errMsg))
		}
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(dim).Background(modalBg).
			Render("  Enter  load     Esc  back"))

	case welcomePhaseNewName:
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Bold(true).Background(modalBg).
			Render("+  New Project"))
		b.WriteString("\n\n")
		b.WriteString(StyleFieldKeyActive.Background(modalBg).Render("  Name ") +
			StyleEquals.Background(modalBg).Render(" = ") +
			w.input.View())
		if w.errMsg != "" {
			b.WriteString("\n\n")
			b.WriteString(StyleMsgErr.Background(modalBg).Render("  ✗ " + w.errMsg))
		}
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(dim).Background(modalBg).
			Render("  Enter  create   Esc  back"))
	}

	b.WriteString("\n")

	box := lipgloss.NewStyle().
		Border(sharpBorder).
		BorderForeground(border).
		Background(modalBg).
		Width(boxWidth).
		Padding(0, 1).
		Render(b.String())

	return lipgloss.Place(w.width, w.height, lipgloss.Center, lipgloss.Center, box)
}

