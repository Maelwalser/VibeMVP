package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-mvp/internal/manifest"
)

// RealizeMsg is emitted when the user presses R to start realization.
type RealizeMsg struct{}

// ── mode ──────────────────────────────────────────────────────────────────────

type realizeMode int

const (
	realizeNormal realizeMode = iota
	realizeInsert
)

// ── field definitions ─────────────────────────────────────────────────────────

func defaultRealizeFields() []Field {
	return []Field{
		{
			Key: "app_name", Label: "app_name      ", Kind: KindText,
			Value: "my-app",
		},
		{
			Key: "output_dir", Label: "output_dir    ", Kind: KindText,
			Value: ".",
		},
		{
			Key: "model", Label: "model         ", Kind: KindSelect,
			Options: []string{
				"claude-haiku-4-5-20251001",
				"claude-sonnet-4-6",
				"claude-opus-4-6",
			},
			Value: "claude-sonnet-4-6", SelIdx: 1,
		},
		{
			Key: "concurrency", Label: "concurrency   ", Kind: KindSelect,
			Options: []string{"1", "2", "4", "8"},
			Value:   "4", SelIdx: 2,
		},
		{
			Key: "verify", Label: "verify        ", Kind: KindSelect,
			Options: []string{"true", "false"},
			Value:   "true",
		},
		{
			Key: "dry_run", Label: "dry_run       ", Kind: KindSelect,
			Options: []string{"false", "true"},
			Value:   "false",
		},
	}
}

// ── RealizeEditor ─────────────────────────────────────────────────────────────

// RealizeEditor manages the REALIZE main-tab.
type RealizeEditor struct {
	fields    []Field
	activeIdx int
	mode      realizeMode
	formInput textinput.Model
	width     int
	ddOpen    bool
	ddOptIdx  int
}

func newRealizeEditor() RealizeEditor {
	return RealizeEditor{
		fields:    defaultRealizeFields(),
		formInput: newFormInput(),
	}
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (r RealizeEditor) ToManifestRealizeOptions() manifest.RealizeOptions {
	concurrency := 4
	switch fieldGet(r.fields, "concurrency") {
	case "1":
		concurrency = 1
	case "2":
		concurrency = 2
	case "4":
		concurrency = 4
	case "8":
		concurrency = 8
	}
	return manifest.RealizeOptions{
		AppName:     fieldGet(r.fields, "app_name"),
		OutputDir:   fieldGet(r.fields, "output_dir"),
		Model:       fieldGet(r.fields, "model"),
		Concurrency: concurrency,
		Verify:      fieldGet(r.fields, "verify") == "true",
		DryRun:      fieldGet(r.fields, "dry_run") == "true",
	}
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (r RealizeEditor) Mode() Mode {
	if r.mode == realizeInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (r RealizeEditor) HintLine() string {
	if r.mode == realizeInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	return hintBar("j/k", "navigate", "Space/Enter", "cycle", "i", "edit text", "R", "start realization")
}

// ── Update ────────────────────────────────────────────────────────────────────

func (r RealizeEditor) Update(msg tea.Msg) (RealizeEditor, tea.Cmd) {
	if r.mode == realizeInsert {
		return r.updateInsert(msg)
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return r, nil
	}
	if r.ddOpen {
		return r.updateDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if r.activeIdx < len(r.fields)-1 {
			r.activeIdx++
		}
	case "k", "up":
		if r.activeIdx > 0 {
			r.activeIdx--
		}
	case "g":
		r.activeIdx = 0
	case "G":
		r.activeIdx = len(r.fields) - 1
	case "enter", " ":
		if r.activeIdx < len(r.fields) {
			f := &r.fields[r.activeIdx]
			if f.Kind == KindSelect {
				r.ddOpen = true
				r.ddOptIdx = f.SelIdx
			} else {
				return r.tryEnterInsert()
			}
		}
	case "H", "shift+left":
		if r.activeIdx < len(r.fields) {
			f := &r.fields[r.activeIdx]
			if f.Kind == KindSelect {
				f.CyclePrev()
			}
		}
	case "i":
		return r.tryEnterInsert()
	case "R":
		return r, func() tea.Msg { return RealizeMsg{} }
	}
	return r, nil
}

func (r RealizeEditor) updateDropdown(key tea.KeyMsg) (RealizeEditor, tea.Cmd) {
	if r.activeIdx >= len(r.fields) {
		r.ddOpen = false
		return r, nil
	}
	f := &r.fields[r.activeIdx]
	switch key.String() {
	case "j", "down":
		if r.ddOptIdx < len(f.Options)-1 {
			r.ddOptIdx++
		}
	case "k", "up":
		if r.ddOptIdx > 0 {
			r.ddOptIdx--
		}
	case " ", "enter":
		f.SelIdx = r.ddOptIdx
		if r.ddOptIdx < len(f.Options) {
			f.Value = f.Options[r.ddOptIdx]
		}
		r.ddOpen = false
	case "esc", "b":
		r.ddOpen = false
	}
	return r, nil
}

func (r RealizeEditor) updateInsert(msg tea.Msg) (RealizeEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			r.saveInput()
			r.mode = realizeNormal
			r.formInput.Blur()
			return r, nil
		case "tab":
			r.saveInput()
			if r.activeIdx < len(r.fields)-1 {
				r.activeIdx++
			}
			return r.tryEnterInsert()
		case "shift+tab":
			r.saveInput()
			if r.activeIdx > 0 {
				r.activeIdx--
			}
			return r.tryEnterInsert()
		}
	}
	var cmd tea.Cmd
	r.formInput, cmd = r.formInput.Update(msg)
	return r, cmd
}

func (r *RealizeEditor) saveInput() {
	if r.activeIdx < len(r.fields) && r.fields[r.activeIdx].Kind == KindText {
		r.fields[r.activeIdx].Value = r.formInput.Value()
	}
}

func (r RealizeEditor) tryEnterInsert() (RealizeEditor, tea.Cmd) {
	if r.activeIdx < len(r.fields) && r.fields[r.activeIdx].Kind == KindText {
		r.mode = realizeInsert
		r.formInput.SetValue(r.fields[r.activeIdx].Value)
		r.formInput.Width = r.width - 22
		r.formInput.CursorEnd()
		return r, r.formInput.Focus()
	}
	return r, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

var (
	styleRealizeBtn = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrBg)).
			Background(lipgloss.Color(clrGreen)).
			Bold(true).
			Padding(0, 2)

	styleRealizeBtnHint = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrComment))
)

func (r RealizeEditor) View(w, h int) string {
	r.width = w
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Realization — configure output directory, app name, and agent options"),
		"",
	)
	lines = append(lines, renderFormFieldsWithDropdown(w, r.fields, r.activeIdx, r.mode == realizeInsert, r.formInput, r.ddOpen, r.ddOptIdx)...)
	lines = append(lines, "")

	// Start button row
	btn := styleRealizeBtn.Render(" R  Start Realization ")
	hint := styleRealizeBtnHint.Render("  saves manifest then launches the realize agent")
	btnLine := "  " + btn + hint
	pad := w - lipgloss.Width(btnLine)
	if pad > 0 {
		btnLine += strings.Repeat(" ", pad)
	}
	lines = append(lines, btnLine)

	return fillTildes(lines, h)
}
