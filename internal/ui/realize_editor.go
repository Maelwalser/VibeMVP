package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
)

// RealizeMsg is emitted when the user presses R to start realization.
type RealizeMsg struct{}


// ── field definitions ─────────────────────────────────────────────────────────

// providerTierOrder lists the tiers for each provider in display order.
// Used to build per-section model option lists from the configured providers.
var providerTierOrder = map[string][]string{
	"Claude":  {"Haiku", "Sonnet", "Opus"},
	"ChatGPT": {"Mini", "4o", "o1"},
	"Gemini":  {"Flash", "Pro", "Ultra"},
	"Mistral": {"Nemo", "Small", "Large"},
	"Llama":   {"8B", "70B", "405B"},
	"Custom":  {"Custom"},
}

// providerOrder is the canonical display order for providers.
var providerOrder = []string{"Claude", "ChatGPT", "Gemini", "Mistral", "Llama", "Custom"}

// sectionModelKeys are the field keys for the per-section model overrides.
var sectionModelKeys = []string{
	"backend_model", "data_model", "contracts_model",
	"frontend_model", "infra_model", "crosscut_model",
}

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
			Options: OptionsOnOff,
			Value:   "true",
		},
		{
			Key: "dry_run", Label: "dry_run       ", Kind: KindSelect,
			Options: OptionsOffOn,
			Value:   "false",
		},
		// Per-section model overrides (options populated dynamically via UpdateProviderOptions)
		{
			Key: "backend_model", Label: "backend_model ", Kind: KindSelect,
			Options: []string{"default"}, Value: "default",
		},
		{
			Key: "data_model", Label: "data_model    ", Kind: KindSelect,
			Options: []string{"default"}, Value: "default",
		},
		{
			Key: "contracts_model", Label: "contracts_mdl ", Kind: KindSelect,
			Options: []string{"default"}, Value: "default",
		},
		{
			Key: "frontend_model", Label: "frontend_model", Kind: KindSelect,
			Options: []string{"default"}, Value: "default",
		},
		{
			Key: "infra_model", Label: "infra_model   ", Kind: KindSelect,
			Options: []string{"default"}, Value: "default",
		},
		{
			Key: "crosscut_model", Label: "crosscut_model", Kind: KindSelect,
			Options: []string{"default"}, Value: "default",
		},
	}
}

// ── RealizeEditor ─────────────────────────────────────────────────────────────

// RealizeEditor manages the REALIZE main-tab.
type RealizeEditor struct {
	fields    []Field
	activeIdx int
	mode      Mode
	formInput textinput.Model
	width     int
	dd        DropdownState
	cBuf      bool
}

func newRealizeEditor() RealizeEditor {
	return RealizeEditor{
		fields:    defaultRealizeFields(),
		formInput: newFormInput(),
	}
}

// ── Provider options sync ─────────────────────────────────────────────────────

// UpdateProviderOptions rebuilds the per-section model field options from the
// currently configured providers. Options are formatted as "Provider · Tier"
// (e.g. "Claude · Sonnet", "Gemini · Flash"). Any section field whose current
// value is no longer in the new option list is reset to "default".
func (r RealizeEditor) UpdateProviderOptions(configured map[string]ProviderSelection) RealizeEditor {
	options := []string{"default"}
	for _, provLabel := range providerOrder {
		sel, ok := configured[provLabel]
		if !ok || !sel.IsSet() {
			continue
		}
		for _, tier := range providerTierOrder[provLabel] {
			options = append(options, provLabel+" · "+tier)
		}
	}

	for i := range r.fields {
		f := &r.fields[i]
		if !isSectionModelKey(f.Key) {
			continue
		}
		f.Options = options
		// Keep value if still valid, else reset to default.
		valid := false
		for j, o := range options {
			if o == f.Value {
				f.SelIdx = j
				valid = true
				break
			}
		}
		if !valid {
			f.Value = "default"
			f.SelIdx = 0
		}
	}
	return r
}

// isSectionModelKey reports whether key is one of the per-section model fields.
func isSectionModelKey(key string) bool {
	for _, k := range sectionModelKeys {
		if k == key {
			return true
		}
	}
	return false
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

	// Collect non-default section model overrides.
	sectionKeys := []string{"backend_model", "data_model", "contracts_model", "frontend_model", "infra_model", "crosscut_model"}
	sectionIDs := []string{"backend", "data", "contracts", "frontend", "infra", "crosscut"}
	var sectionModels map[string]string
	for i, key := range sectionKeys {
		val := fieldGet(r.fields, key)
		if val != "" && val != "default" {
			if sectionModels == nil {
				sectionModels = make(map[string]string)
			}
			sectionModels[sectionIDs[i]] = val
		}
	}

	return manifest.RealizeOptions{
		AppName:       fieldGet(r.fields, "app_name"),
		OutputDir:     fieldGet(r.fields, "output_dir"),
		Model:         fieldGet(r.fields, "model"),
		Concurrency:   concurrency,
		Verify:        fieldGet(r.fields, "verify") == "true",
		DryRun:        fieldGet(r.fields, "dry_run") == "true",
		SectionModels: sectionModels,
	}
}

// FromRealizeOptions populates the editor from saved manifest RealizeOptions,
// reversing the ToManifestRealizeOptions() operation.
func (r RealizeEditor) FromRealizeOptions(ro manifest.RealizeOptions) RealizeEditor {
	r.fields = setFieldValue(r.fields, "app_name", ro.AppName)
	r.fields = setFieldValue(r.fields, "output_dir", ro.OutputDir)
	r.fields = setFieldValue(r.fields, "model", ro.Model)
	switch ro.Concurrency {
	case 1, 2, 4, 8:
		r.fields = setFieldValue(r.fields, "concurrency", fmt.Sprintf("%d", ro.Concurrency))
	}
	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}
	r.fields = setFieldValue(r.fields, "verify", boolStr(ro.Verify))
	r.fields = setFieldValue(r.fields, "dry_run", boolStr(ro.DryRun))

	sectionKeys := []string{"backend_model", "data_model", "contracts_model", "frontend_model", "infra_model", "crosscut_model"}
	sectionIDs := []string{"backend", "data", "contracts", "frontend", "infra", "crosscut"}
	for i, sectionID := range sectionIDs {
		if val, ok := ro.SectionModels[sectionID]; ok && val != "" {
			r.fields = setFieldValue(r.fields, sectionKeys[i], val)
		}
	}

	return r
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (r RealizeEditor) Mode() Mode {
	if r.mode == ModeInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (r RealizeEditor) HintLine() string {
	if r.mode == ModeInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	return hintBar("j/k", "navigate", "Space/Enter", "cycle", "i", "edit text", "R", "start realization")
}

// ── Update ────────────────────────────────────────────────────────────────────

func (r RealizeEditor) Update(msg tea.Msg) (RealizeEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		r.width = wsz.Width
		r.formInput.Width = wsz.Width - 22
		return r, nil
	}
	if r.mode == ModeInsert {
		return r.updateInsert(msg)
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return r, nil
	}
	if r.dd.Open {
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
				r.dd.Open = true
				r.dd.OptIdx = f.SelIdx
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
	case "c":
		if r.cBuf {
			r.cBuf = false
			return r.clearAndEnterInsert()
		}
		r.cBuf = true
		return r, nil
	case "R":
		return r, func() tea.Msg { return RealizeMsg{} }
	default:
		r.cBuf = false
	}
	return r, nil
}

func (r RealizeEditor) updateDropdown(key tea.KeyMsg) (RealizeEditor, tea.Cmd) {
	if r.activeIdx >= len(r.fields) {
		r.dd.Open = false
		return r, nil
	}
	f := &r.fields[r.activeIdx]
	r.dd.OptIdx = NavigateDropdown(key.String(), r.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = r.dd.OptIdx
		if r.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[r.dd.OptIdx]
		}
		r.dd.Open = false
		if f.PrepareCustomEntry() {
			return r.tryEnterInsert()
		}
	case "esc", "b":
		r.dd.Open = false
	}
	return r, nil
}

func (r RealizeEditor) updateInsert(msg tea.Msg) (RealizeEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			r.saveInput()
			r.mode = ModeNormal
			r.formInput.Blur()
			return r, nil
		case "tab":
			r.saveInput()
			if next := nextEditableIdx(r.fields, r.activeIdx); next >= 0 {
				r.activeIdx = next
			}
			return r.tryEnterInsert()
		case "shift+tab":
			r.saveInput()
			if prev := prevEditableIdx(r.fields, r.activeIdx); prev >= 0 {
				r.activeIdx = prev
			}
			return r.tryEnterInsert()
		}
	}
	var cmd tea.Cmd
	r.formInput, cmd = r.formInput.Update(msg)
	return r, cmd
}

func (r *RealizeEditor) saveInput() {
	if r.activeIdx < len(r.fields) && r.fields[r.activeIdx].CanEditAsText() {
		r.fields[r.activeIdx].SaveTextInput(r.formInput.Value())
	}
}

func (r RealizeEditor) clearAndEnterInsert() (RealizeEditor, tea.Cmd) {
	r, cmd := r.tryEnterInsert()
	if r.mode == ModeInsert {
		r.formInput.SetValue("")
	}
	return r, cmd
}

func (r RealizeEditor) tryEnterInsert() (RealizeEditor, tea.Cmd) {
	if r.activeIdx < len(r.fields) && r.fields[r.activeIdx].CanEditAsText() {
		r.mode = ModeInsert
		r.formInput.SetValue(r.fields[r.activeIdx].TextInputValue())
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
	r.formInput.Width = w - 22
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Realization — configure output directory, app name, and agent options"),
		"",
	)

	// Split fields into two groups: base options (first 6) and section models (rest).
	baseFields := r.fields
	var sectionFields []Field
	if len(r.fields) > 6 {
		baseFields = r.fields[:6]
		sectionFields = r.fields[6:]
	}

	// Build scrollable content block, tracking the active field's line position.
	var content []string
	activeLine := r.activeIdx // line index within content for the active field

	content = append(content, renderFormFields(w, baseFields, r.activeIdx, r.mode == ModeInsert, r.formInput, r.dd.Open, r.dd.OptIdx)...)
	content = append(content, "")

	if len(sectionFields) > 0 {
		content = append(content, StyleSectionDesc.Render("  # Section model overrides — set per-pillar model (default = use global model above)"))
		content = append(content, "")
		adjIdx := r.activeIdx - 6
		// Active field is in the section group: offset by base lines + empty + 2 section header lines.
		if r.activeIdx >= 6 {
			activeLine = len(baseFields) + 1 + 2 + adjIdx
		}
		content = append(content, renderFormFields(w, sectionFields, adjIdx, r.mode == ModeInsert, r.formInput, r.dd.Open, r.dd.OptIdx)...)
		content = append(content, "")
	}

	// Start button row
	btn := styleRealizeBtn.Render(" R  Start Realization ")
	hint := styleRealizeBtnHint.Render("  saves manifest then launches the realize agent")
	btnLine := "  " + btn + hint
	pad := w - lipgloss.Width(btnLine)
	if pad > 0 {
		btnLine += strings.Repeat(" ", pad)
	}
	content = append(content, btnLine)

	const realizeHeaderH = 2
	lines = append(lines, appendViewport(content, 0, activeLine, h-realizeHeaderH)...)

	return fillTildes(lines, h)
}
