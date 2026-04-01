package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-mvp/internal/manifest"
)

// Mode represents the vim editing mode.
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeCommand
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeCommand:
		return "COMMAND"
	}
	return ""
}

// SaveFunc is called when the user issues :w.
type SaveFunc func(m *manifest.Manifest) error

// cmdState holds vim command-line state.
type cmdState struct {
	buffer string
	status string
	isErr  bool
}

// modalState holds provider-menu modal state.
type modalState struct {
	open bool
	menu ProviderMenu
}

// realizeState holds realization-screen state.
type realizeState struct {
	screen    RealizationScreen
	show      bool
	triggered bool
}

// Model is the root bubbletea model for the declaration UI.
type Model struct {
	sections      []Section
	activeSection int
	activeField   int
	mode          Mode

	// Input widgets (reused for generic sections, not used by delegated editors)
	textInput textinput.Model
	textArea  textarea.Model

	// ── Main tab editors (one per section) ───────────────────────────────────
	backendEditor   BackendEditor
	dataTabEditor   DataTabEditor
	contractsEditor ContractsEditor
	frontendEditor  FrontendEditor
	infraEditor     InfraEditor
	crossCutEditor  CrossCutEditor
	realizeEditor   RealizeEditor

	cmd     cmdState
	modal   modalState
	realize realizeState

	modified      bool
	width, height int
	onSave        SaveFunc
}

// NewModel creates and returns the initial UI model.
func NewModel(onSave SaveFunc) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg))
	ti.CursorStyle = StyleCursor
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))

	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.Prompt = "  "
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBgHL))
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBgHL))
	ta.FocusedStyle.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg))
	ta.BlurredStyle.Base = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBgHL))

	return Model{
		sections:        initSections(),
		textInput:       ti,
		textArea:        ta,
		backendEditor:   newBackendEditor(),
		dataTabEditor:   newDataTabEditor(),
		contractsEditor: newContractsEditor(),
		frontendEditor:  newFrontendEditor(),
		infraEditor:     newInfraEditor(),
		crossCutEditor:  newCrossCutEditor(),
		realizeEditor:   newRealizeEditor(),
		realize:         realizeState{screen: newRealizationScreen()},
		modal:           modalState{menu: newProviderMenu()},
		onSave:          onSave,
	}
}

// Init satisfies tea.Model — starts the animation ticker.
func (m Model) Init() tea.Cmd {
	return uiTick()
}

// ── Section routing helpers ───────────────────────────────────────────────────

func (m Model) activeSectionID() string {
	return m.sections[m.activeSection].ID
}

// activeEditor returns the Editor interface for the currently visible section,
// allowing Mode, View, and HintLine to be dispatched without per-operation switches.
// Returns nil when no delegated editor is active (fallback generic renderer).
func (m Model) activeEditor() Editor {
	switch m.activeSectionID() {
	case "backend":
		return m.backendEditor
	case "data":
		return m.dataTabEditor
	case "contracts":
		return m.contractsEditor
	case "frontend":
		return m.frontendEditor
	case "infrastructure":
		return m.infraEditor
	case "crosscut":
		return m.crossCutEditor
	case "realize":
		if m.realize.show {
			return m.realize.screen
		}
		return m.realizeEditor
	}
	return nil
}

// activeMode returns the effective mode, delegating to the active sub-editor.
func (m Model) activeMode() Mode {
	if e := m.activeEditor(); e != nil {
		return e.Mode()
	}
	return m.mode
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsz.Width
		m.height = wsz.Height
		m.textArea.SetWidth(m.width - 4)
		m.textArea.SetHeight(m.contentHeight() - 4)
		return m, nil
	}
	if _, ok := msg.(uiTickMsg); ok {
		AnimFrame = (AnimFrame + 1) % 2
		return m, uiTick()
	}

	if _, ok := msg.(RealizeMsg); ok {
		m.realize.triggered = true
		m2, saveCmd := m.execSave()
		m = m2.(Model)
		mf := m.BuildManifest()
		var startCmd tea.Cmd
		m.realize.screen, startCmd = m.realize.screen.Start("manifest.json", mf)
		m.realize.show = true
		return m, tea.Sequence(saveCmd, startCmd)
	}

	// Route all messages to the realization screen while it is active.
	if m.realize.show {
		var cmd tea.Cmd
		m.realize.screen, cmd = m.realize.screen.Update(msg)
		if m.realize.screen.WantsQuit() {
			return m, tea.Quit
		}
		return m, cmd
	}

	switch m.mode {
	case ModeNormal:
		return m.updateNormal(msg)
	case ModeInsert:
		return m.updateInsert(msg)
	case ModeCommand:
		return m.updateCommand(msg)
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		// Pass non-key messages to active delegated editor.
		return m.delegateUpdate(msg)
	}
	m.cmd.status = ""

	// Provider menu intercepts all input when open.
	if m.modal.open {
		switch key.String() {
		case "M":
			if m.modal.menu.focus != pmFocusCredential {
				m.modal.open = false
				return m, nil
			}
		case "esc":
			// Esc closes the version dropdown first; a second Esc closes the modal.
			// But if in credential step, let the menu handle it (steps back to auth).
			if !m.modal.menu.dropdownOpen && m.modal.menu.focus == pmFocusSections {
				m.modal.open = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.modal.menu, cmd = m.modal.menu.Update(msg)
		return m, cmd
	}

	// Global keys always processed regardless of section.
	switch key.String() {
	case "M":
		m.modal.open = true
		return m, nil

	case "ctrl+c":
		// Behave like Escape: exit insert/form/dropdown modes in sub-editors
		escMsg := tea.KeyMsg{Type: tea.KeyEsc}
		return m.delegateUpdate(escMsg)

	case ":":
		m.mode = ModeCommand
		m.cmd.buffer = ""
		return m, nil

	case "ctrl+s":
		return m.execSave()

	// Section (tab) navigation with Tab/Shift+Tab only
	case "tab":
		m.activeSection = (m.activeSection + 1) % len(m.sections)
		m.activeField = 0
		return m, nil

	case "shift+tab":
		m.activeSection = (m.activeSection - 1 + len(m.sections)) % len(m.sections)
		m.activeField = 0
		return m, nil
	}

	// Delegate all remaining input to the active section editor.
	return m.delegateUpdate(msg)
}

func (m Model) delegateUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.activeSectionID() {
	case "backend":
		m.backendEditor.SetDomainNames(m.dataTabEditor.domainNames())
		m.backendEditor, cmd = m.backendEditor.Update(msg)
	case "data":
		m.dataTabEditor, cmd = m.dataTabEditor.Update(msg)
	case "contracts":
		m.contractsEditor.SetDomains(m.dataTabEditor.domainNames())
		m.contractsEditor.SetDomainDefs(m.dataTabEditor.domains)
		m.contractsEditor.SetServices(m.backendEditor.ServiceNames())
		m.contractsEditor.SetServiceDefs(m.backendEditor.ServiceDefs())
		m.contractsEditor, cmd = m.contractsEditor.Update(msg)
	case "frontend":
		m.frontendEditor.SetAuthRoles(m.backendEditor.AuthRoleOptions())
		m.frontendEditor, cmd = m.frontendEditor.Update(msg)
	case "infrastructure":
		m.infraEditor, cmd = m.infraEditor.Update(msg)
	case "crosscut":
		m.crossCutEditor, cmd = m.crossCutEditor.Update(msg)
	case "realize":
		m.realizeEditor, cmd = m.realizeEditor.Update(msg)
	default:
		return m, nil
	}
	m.modified = true
	return m, cmd
}

func (m Model) updateInsert(msg tea.Msg) (tea.Model, tea.Cmd) {
	// All editors handle their own insert mode internally;
	// the root model's insert mode is only used if no sub-editor is active.
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc", "ctrl+c":
			return m.exitInsert()
		case "tab":
			m = m.saveActiveInput()
			sec := m.sections[m.activeSection]
			if m.activeField < len(sec.Fields)-1 {
				m.activeField++
			}
			return m.enterInsert()
		case "shift+tab":
			m = m.saveActiveInput()
			if m.activeField > 0 {
				m.activeField--
			}
			return m.enterInsert()
		}
	}

	sec := m.sections[m.activeSection]
	f := sec.Fields[m.activeField]
	var cmd tea.Cmd
	if f.Kind == KindTextArea {
		m.textArea, cmd = m.textArea.Update(msg)
	} else {
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m Model) enterInsert() (Model, tea.Cmd) {
	sec := m.sections[m.activeSection]
	f := sec.Fields[m.activeField]
	if f.Kind == KindSelect {
		return m, nil
	}
	m.mode = ModeInsert
	if f.Kind == KindTextArea {
		m.textArea.SetValue(f.Value)
		m.textArea.SetWidth(m.width - 4)
		m.textArea.SetHeight(m.contentHeight() - 4)
		return m, m.textArea.Focus()
	}
	m.textInput.SetValue(f.Value)
	m.textInput.Width = m.width - 22
	m.textInput.CursorEnd()
	return m, m.textInput.Focus()
}

func (m Model) exitInsert() (Model, tea.Cmd) {
	m = m.saveActiveInput()
	m.mode = ModeNormal
	m.textInput.Blur()
	m.textArea.Blur()
	return m, nil
}

func (m Model) saveActiveInput() Model {
	sec := m.sections[m.activeSection]
	f := sec.Fields[m.activeField]
	if f.Kind == KindTextArea {
		sec.Fields[m.activeField].Value = m.textArea.Value()
	} else {
		sec.Fields[m.activeField].Value = m.textInput.Value()
	}
	m.sections[m.activeSection] = sec
	m.modified = true
	return m
}

func (m Model) updateCommand(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "esc", "ctrl+c":
		m.mode = ModeNormal
		m.cmd.buffer = ""
	case "enter":
		return m.execCommand(m.cmd.buffer)
	case "backspace":
		if len(m.cmd.buffer) > 0 {
			m.cmd.buffer = m.cmd.buffer[:len(m.cmd.buffer)-1]
		} else {
			m.mode = ModeNormal
		}
	default:
		if len(key.Runes) > 0 {
			m.cmd.buffer += string(key.Runes)
		}
	}
	return m, nil
}

func (m Model) execCommand(cmd string) (tea.Model, tea.Cmd) {
	m.mode = ModeNormal
	m.cmd.buffer = ""

	switch strings.TrimSpace(cmd) {
	case "q", "quit":
		return m, tea.Quit
	case "q!", "quit!":
		return m, tea.Quit
	case "w", "write":
		return m.execSave()
	case "wq", "x":
		m2, saveCmd := m.execSave()
		model := m2.(Model)
		return model, tea.Sequence(saveCmd, tea.Quit)
	case "tabn", "bn":
		m.activeSection = (m.activeSection + 1) % len(m.sections)
		m.activeField = 0
	case "tabp", "bp":
		m.activeSection = (m.activeSection - 1 + len(m.sections)) % len(m.sections)
		m.activeField = 0
	case "help", "h":
		m.cmd.status = "Tab:section  j/k:nav  i:insert  Space:cycle  :w save  :q quit"
		m.cmd.isErr = false
	default:
		if len(cmd) == 1 && cmd[0] >= '1' && cmd[0] <= '9' {
			idx := int(cmd[0] - '1')
			if idx < len(m.sections) {
				m.activeSection = idx
				m.activeField = 0
			}
			return m, nil
		}
		m.cmd.status = fmt.Sprintf("E492: Not an editor command: %s", cmd)
		m.cmd.isErr = true
	}
	return m, nil
}

func (m Model) execSave() (tea.Model, tea.Cmd) {
	if m.onSave == nil {
		m.cmd.status = "No save handler configured."
		m.cmd.isErr = true
		return m, nil
	}
	mf := m.BuildManifest()
	if err := m.onSave(mf); err != nil {
		m.cmd.status = fmt.Sprintf("Error: %v", err)
		m.cmd.isErr = true
		return m, nil
	}
	m.modified = false
	m.cmd.status = `"manifest.json" written`
	m.cmd.isErr = false
	return m, nil
}

// BuildManifest converts the form state into a Manifest struct.
func (m Model) BuildManifest() *manifest.Manifest {
	dataPillar := m.dataTabEditor.ToManifestDataPillar()
	return &manifest.Manifest{
		Data:      dataPillar,
		Backend:   m.backendEditor.ToManifest(),
		Contracts: m.contractsEditor.ToManifestContractsPillar(),
		Frontend:  m.frontendEditor.ToManifestFrontendPillar(),
		Infra:     m.infraEditor.ToManifestInfraPillar(),
		CrossCut:  m.crossCutEditor.ToManifestCrossCutPillar(),
		Realize:   m.realizeEditor.ToManifestRealizeOptions(),
		Providers: m.modal.menu.ToManifestProviderAssignments(),

		// Legacy flat fields for backward compatibility
		Databases: dataPillar.Databases,
		Entities:  dataPillar.Entities,
	}
}

// RealizeTriggered reports whether the user requested realization.
func (m Model) RealizeTriggered() bool { return m.realize.triggered }

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) contentHeight() int {
	// total - header(1) - divider(1) - tabbar(1) - statusline(1) - cmdline(1)
	h := m.height - 5
	if h < 4 {
		return 4
	}
	return h
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	base := m.renderBaseView()

	if m.modal.open {
		modal := m.modal.menu.View()
		modalLines := strings.Split(modal, "\n")
		modalH := len(modalLines)
		modalW := 0
		for _, l := range modalLines {
			if w := lipgloss.Width(l); w > modalW {
				modalW = w
			}
		}
		x := (m.width - modalW) / 2
		y := (m.height - modalH) / 2
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		return placeOverlay(base, modal, x, y)
	}

	return base
}

func (m Model) renderBaseView() string {
	var b strings.Builder
	w := m.width
	b.WriteString(m.renderHeader(w))
	b.WriteString("\n")
	b.WriteString(m.renderContent(w))
	b.WriteString(m.renderTabBar(w))
	b.WriteString("\n")
	b.WriteString(m.renderStatusLine(w))
	b.WriteString("\n")
	b.WriteString(m.renderCmdLine(w))
	return b.String()
}

func (m Model) renderHeader(w int) string {
	sec := m.sections[m.activeSection]
	modMark := ""
	if m.modified {
		modMark = StyleHeaderMod.Render(" [+]")
	}

	deco := StyleHeaderDeco.Render(headerDecoFrames[AnimFrame])
	title := deco + " " + StyleSectionTitle.Render(sec.ID+".manifest") + modMark

	counter := StyleHeaderDeco.Render(headerDecoFrames[1-AnimFrame]) + " " +
		StyleHeaderTitle.Render(fmt.Sprintf("[%02d/%02d]", m.activeSection+1, len(m.sections)))

	titleW := lipgloss.Width(title)
	counterW := lipgloss.Width(counter)
	gap := w - titleW - counterW - 2
	if gap < 1 {
		gap = 1
	}
	line := " " + title + strings.Repeat(" ", gap) + counter + " "
	return StyleHeaderBar.Width(w).Render(line)
}

func (m Model) renderContent(w int) string {
	ch := m.contentHeight()
	if e := m.activeEditor(); e != nil {
		return e.View(w, ch)
	}
	// Fallback: generic field list for sections without a delegated editor.
	sec := m.sections[m.activeSection]
	return m.renderFieldList(w, ch, sec)
}

func (m Model) renderFieldList(w, h int, sec Section) string {
	const lineNumW = 4
	const labelW = 14
	const eqW = 3
	valW := w - lineNumW - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	var lines []string
	descLine := StyleSectionDesc.Render(fmt.Sprintf("  # %s", sec.Desc))
	lines = append(lines, descLine, "")

	for i, f := range sec.Fields {
		lineNo := i + 1
		isCur := i == m.activeField

		var numStr string
		if isCur {
			numStr = StyleCurLineNum.Render(fmt.Sprintf("%3d ", lineNo))
		} else {
			numStr = StyleLineNum.Render(fmt.Sprintf("%3d ", lineNo))
		}

		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}

		eq := StyleEquals.Render(" = ")

		var valStr string
		if m.mode == ModeInsert && isCur && f.Kind == KindText {
			valStr = m.textInput.View()
		} else if f.Kind == KindSelect {
			arrow := StyleSelectArrow.Render(" ▾")
			val := f.DisplayValue()
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			valStr = val + arrow
		} else {
			dv := f.DisplayValue()
			if len(dv) > valW {
				dv = dv[:valW-1] + "…"
			}
			if dv == "" && !isCur {
				dv = StyleFieldVal.Foreground(lipgloss.Color(clrFgDim)).Render("_")
			} else if isCur {
				valStr = StyleFieldValActive.Render(dv)
			} else {
				valStr = StyleFieldVal.Render(dv)
			}
			if valStr == "" {
				valStr = StyleFieldVal.Render(dv)
			}
		}

		row := numStr + keyStr + eq + valStr
		if isCur {
			rawW := lipgloss.Width(row)
			if rawW < w {
				row += strings.Repeat(" ", w-rawW)
			}
			row = activeCurLineStyle().Render(row)
		}
		lines = append(lines, row)
	}

	return fillTildes(lines, h)
}

func (m Model) renderTabBar(w int) string {
	sep := StyleTabSep.Render("│")
	var parts []string
	for i, s := range m.sections {
		badge := m.providerBadge(s.ID)
		label := s.Abbr
		if badge != "" {
			label = s.Abbr + " " + badge
		}
		if i == m.activeSection {
			parts = append(parts, StyleTabActive.Render(" "+label+" "))
		} else {
			parts = append(parts, StyleTabInactive.Render(" "+label+" "))
		}
	}
	tabs := strings.Join(parts, sep)
	rawW := lipgloss.Width(tabs)
	if rawW < w {
		tabs += StyleTabBar.Render(strings.Repeat(" ", w-rawW))
	}
	return tabs
}

// providerBadge returns a short colored indicator for the provider assigned to
// the given section ID, or an empty string if none is assigned.
func (m Model) providerBadge(sectionID string) string {
	sel, ok := m.modal.menu.SectionAssignment(sectionID)
	if !ok {
		return ""
	}
	// One-letter abbreviations per provider.
	abbrs := map[string]string{
		"Claude":  "C",
		"ChatGPT": "G",
		"Gemini":  "Ge",
		"Mistral": "Mi",
		"Llama":   "L",
		"Custom":  "?",
	}
	letter, ok := abbrs[sel.Provider]
	if !ok {
		letter = sel.Provider[:1]
	}
	return StyleNeonGreen.Render("◆" + letter + "◆")
}

func (m Model) renderStatusLine(w int) string {
	spin := modeSpinFrames[AnimFrame]
	var modeLabel string
	switch m.activeMode() {
	case ModeNormal:
		modeLabel = StyleNormalMode.Render(spin[0] + " NRM " + spin[1])
	case ModeInsert:
		modeLabel = StyleInsertMode.Render(spin[0] + " INS " + spin[1])
	case ModeCommand:
		modeLabel = StyleCommandMode.Render(spin[0] + " CMD " + spin[1])
	}

	sec := m.sections[m.activeSection]
	pos := fmt.Sprintf("%02d/%02d", m.activeSection+1, len(m.sections))
	right := StyleStatusRight.Render(fmt.Sprintf("  %s.manifest  %s  ▪ ", sec.ID, pos))

	msg := ""
	if m.cmd.status != "" {
		if m.cmd.isErr {
			msg = StyleMsgErr.Render("✗ " + m.cmd.status)
		} else {
			msg = StyleMsgOK.Render("✓ " + m.cmd.status)
		}
	}

	leftW := lipgloss.Width(modeLabel)
	rightW := lipgloss.Width(right)
	msgW := lipgloss.Width(msg)
	gapW := w - leftW - rightW - msgW
	if gapW < 1 {
		gapW = 1
	}

	line := modeLabel + strings.Repeat(" ", gapW/2) + msg + StyleStatusLine.Render(strings.Repeat(" ", gapW-gapW/2)) + right
	return line
}

func (m Model) renderCmdLine(w int) string {
	if m.mode == ModeCommand {
		cursor := StyleCursor.Render(" ")
		return StyleCmdLine.Render(":"+m.cmd.buffer) + cursor
	}

	// Delegate hint line to the active sub-editor, with a fallback for the
	// generic field-list renderer (which has no delegated editor).
	var line string
	if e := m.activeEditor(); e != nil {
		line = e.HintLine()
	} else {
		switch m.mode {
		case ModeNormal:
			hints := []string{
				StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
				StyleHelpKey.Render("i") + StyleHelpDesc.Render(" insert"),
				StyleHelpKey.Render("Tab") + StyleHelpDesc.Render(" section"),
				StyleHelpKey.Render("Enter") + StyleHelpDesc.Render(" cycle"),
				StyleHelpKey.Render(":w") + StyleHelpDesc.Render(" save"),
				StyleHelpKey.Render(":q") + StyleHelpDesc.Render(" quit"),
			}
			sep := StyleHelpDesc.Render("  │  ")
			line = "  " + strings.Join(hints, sep)
		case ModeInsert:
			line = StyleInsertMode.Render(" ▷ INSERT ◁ ") + StyleHelpDesc.Render("  Esc: normal  │  Tab: next field")
		}
	}

	if lipgloss.Width(line) > w {
		line = line[:w-1]
	}
	return line
}
