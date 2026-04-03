package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DescriptionEditor is the first tab — a large free-text area for describing
// the project in natural language before filling in the structured pillars.
type DescriptionEditor struct {
	ta   textarea.Model
	mode Mode
}

func newDescriptionEditor() DescriptionEditor {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.Prompt = "  "
	ta.Placeholder = "Describe your project…\n\nWhat kind of system are you building?\nWhat are the main goals and constraints?\nWhat users does it serve?"
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBgHL))
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBgHL))
	ta.FocusedStyle.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg))
	ta.BlurredStyle.Base = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBgHL))
	ta.BlurredStyle.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFgDim))
	return DescriptionEditor{ta: ta}
}

// Mode implements Editor.
func (e DescriptionEditor) Mode() Mode { return e.mode }

// HintLine implements Editor.
func (e DescriptionEditor) HintLine() string {
	if e.mode == ModeInsert {
		return StyleInsertMode.Render(" ▷ INSERT ◁ ") +
			StyleHelpDesc.Render("  Esc: normal mode  │  Type freely to describe your project")
	}
	hints := []string{
		StyleHelpKey.Render("i") + StyleHelpDesc.Render(" edit"),
		StyleHelpKey.Render("Tab") + StyleHelpDesc.Render(" next section"),
		StyleHelpKey.Render(":w") + StyleHelpDesc.Render(" save"),
		StyleHelpKey.Render(":q") + StyleHelpDesc.Render(" quit"),
	}
	sep := StyleHelpDesc.Render("  │  ")
	return "  " + strings.Join(hints, sep)
}

// View implements Editor.
func (e DescriptionEditor) View(w, h int) string {
	// Textarea height: content area minus header line and blank separator.
	taH := h - 2
	if taH < 3 {
		taH = 3
	}

	// Use a local copy so we don't mutate the receiver in View.
	ta := e.ta
	ta.SetWidth(w - 4)
	ta.SetHeight(taH)

	// Split textarea output into individual lines.
	taLines := strings.Split(ta.View(), "\n")
	// Trim to taH lines in case the widget emits extras.
	if len(taLines) > taH {
		taLines = taLines[:taH]
	}

	// Indent every textarea line by 2 spaces to match the rest of the UI.
	indented := make([]string, len(taLines))
	for i, l := range taLines {
		indented[i] = "  " + l
	}

	// Collect all output lines: header, blank, textarea rows.
	lines := make([]string, 0, h)
	lines = append(lines, StyleSectionDesc.Render("  # Describe your project — what are you building?"))
	lines = append(lines, "")
	lines = append(lines, indented...)

	// fillTildes ensures exactly h lines so the tab bar is never pushed off-screen.
	return fillTildes(lines, h)
}

// Update handles keyboard input for the description editor.
func (e DescriptionEditor) Update(msg tea.Msg) (DescriptionEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		e.ta.SetWidth(wsz.Width - 4)
		e.ta.SetHeight(wsz.Height - 7)
		return e, nil
	}

	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch e.mode {
		case ModeNormal:
			switch key.String() {
			case "i", "a", "enter":
				e.mode = ModeInsert
				return e, e.ta.Focus()
			}
			return e, nil
		case ModeInsert:
			if key.String() == "esc" {
				e.mode = ModeNormal
				e.ta.Blur()
				return e, nil
			}
		}
	}

	if e.mode == ModeInsert {
		var cmd tea.Cmd
		e.ta, cmd = e.ta.Update(msg)
		return e, cmd
	}
	return e, nil
}

// Value returns the current description text.
func (e DescriptionEditor) Value() string { return e.ta.Value() }

// SetValue sets the description text (used when loading a manifest).
func (e *DescriptionEditor) SetValue(v string) { e.ta.SetValue(v) }
