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

	cmdBuffer string
	statusMsg string
	statusErr bool
	modified  bool

	width  int
	height int

	onSave SaveFunc
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
		onSave:          onSave,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// ── Section routing helpers ───────────────────────────────────────────────────

func (m Model) activeSectionID() string {
	return m.sections[m.activeSection].ID
}

func (m Model) isBackendSection() bool    { return m.activeSectionID() == "backend" }
func (m Model) isDataSection() bool       { return m.activeSectionID() == "data" }
func (m Model) isContractsSection() bool  { return m.activeSectionID() == "contracts" }
func (m Model) isFrontendSection() bool   { return m.activeSectionID() == "frontend" }
func (m Model) isInfraSection() bool      { return m.activeSectionID() == "infrastructure" }
func (m Model) isCrossCutSection() bool   { return m.activeSectionID() == "crosscut" }

// activeMode returns the effective mode, delegating to sub-editors when appropriate.
func (m Model) activeMode() Mode {
	switch {
	case m.isBackendSection():
		return m.backendEditor.Mode()
	case m.isDataSection():
		return m.dataTabEditor.Mode()
	case m.isContractsSection():
		return m.contractsEditor.Mode()
	case m.isFrontendSection():
		return m.frontendEditor.Mode()
	case m.isInfraSection():
		return m.infraEditor.Mode()
	case m.isCrossCutSection():
		return m.crossCutEditor.Mode()
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
	m.statusMsg = ""

	// Global keys always processed regardless of section.
	switch key.String() {
	case "ctrl+c":
		// Behave like Escape: exit insert/form/dropdown modes in sub-editors
		escMsg := tea.KeyMsg{Type: tea.KeyEsc}
		return m.delegateUpdate(escMsg)

	case ":":
		m.mode = ModeCommand
		m.cmdBuffer = ""
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

	switch {
	case m.isBackendSection():
		// Inject domain names for event domain dropdowns
		m.backendEditor.SetDomainNames(m.dataTabEditor.domainNames())
		m.backendEditor, cmd = m.backendEditor.Update(msg)
		m.modified = true
	case m.isDataSection():
		m.dataTabEditor, cmd = m.dataTabEditor.Update(msg)
		m.modified = true
	case m.isContractsSection():
		// Inject current domain and service names before processing
		m.contractsEditor.SetDomains(m.dataTabEditor.domainNames())
		m.contractsEditor.SetDomainDefs(m.dataTabEditor.domains)
		m.contractsEditor.SetServices(m.backendEditor.ServiceNames())
		m.contractsEditor.SetServiceDefs(m.backendEditor.ServiceDefs())
		m.contractsEditor, cmd = m.contractsEditor.Update(msg)
		m.modified = true
	case m.isFrontendSection():
		// Inject auth roles from backend before processing
		m.frontendEditor.SetAuthRoles(m.backendEditor.AuthRoleOptions())
		m.frontendEditor, cmd = m.frontendEditor.Update(msg)
		m.modified = true
	case m.isInfraSection():
		m.infraEditor, cmd = m.infraEditor.Update(msg)
		m.modified = true
	case m.isCrossCutSection():
		m.crossCutEditor, cmd = m.crossCutEditor.Update(msg)
		m.modified = true
	}

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
		m.cmdBuffer = ""
	case "enter":
		return m.execCommand(m.cmdBuffer)
	case "backspace":
		if len(m.cmdBuffer) > 0 {
			m.cmdBuffer = m.cmdBuffer[:len(m.cmdBuffer)-1]
		} else {
			m.mode = ModeNormal
		}
	default:
		if len(key.Runes) > 0 {
			m.cmdBuffer += string(key.Runes)
		}
	}
	return m, nil
}

func (m Model) execCommand(cmd string) (tea.Model, tea.Cmd) {
	m.mode = ModeNormal
	m.cmdBuffer = ""

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
		m.statusMsg = "Tab:section  j/k:nav  i:insert  Space:cycle  :w save  :q quit"
		m.statusErr = false
	default:
		if len(cmd) == 1 && cmd[0] >= '1' && cmd[0] <= '9' {
			idx := int(cmd[0] - '1')
			if idx < len(m.sections) {
				m.activeSection = idx
				m.activeField = 0
			}
			return m, nil
		}
		m.statusMsg = fmt.Sprintf("E492: Not an editor command: %s", cmd)
		m.statusErr = true
	}
	return m, nil
}

func (m Model) execSave() (tea.Model, tea.Cmd) {
	if m.onSave == nil {
		m.statusMsg = "No save handler configured."
		m.statusErr = true
		return m, nil
	}
	mf := m.BuildManifest()
	if err := m.onSave(mf); err != nil {
		m.statusMsg = fmt.Sprintf("Error: %v", err)
		m.statusErr = true
		return m, nil
	}
	m.modified = false
	m.statusMsg = `"manifest.json" written`
	m.statusErr = false
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

		// Legacy flat fields for backward compatibility
		Databases: dataPillar.Databases,
		Entities:  dataPillar.Entities,
	}
}

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
	title := StyleSectionTitle.Render(sec.ID+".manifest") + modMark
	counter := StyleHeaderTitle.Render(fmt.Sprintf("[%d/%d]", m.activeSection+1, len(m.sections)))
	gap := w - lipgloss.Width(title) - lipgloss.Width(counter) - 2
	if gap < 1 {
		gap = 1
	}
	line := " " + title + strings.Repeat(" ", gap) + counter
	return StyleHeaderBar.Width(w).Render(line)
}

func (m Model) renderContent(w int) string {
	ch := m.contentHeight()

	switch {
	case m.isBackendSection():
		return m.backendEditor.View(w, ch)
	case m.isDataSection():
		return m.dataTabEditor.View(w, ch)
	case m.isContractsSection():
		return m.contractsEditor.View(w, ch)
	case m.isFrontendSection():
		return m.frontendEditor.View(w, ch)
	case m.isInfraSection():
		return m.infraEditor.View(w, ch)
	case m.isCrossCutSection():
		return m.crossCutEditor.View(w, ch)
	}

	// Fallback: should not be reached for 6-section layout
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
			row = StyleCurLine.Render(row)
		}
		lines = append(lines, row)
	}

	return fillTildes(lines, h)
}

func (m Model) renderTabBar(w int) string {
	var parts []string
	for i, s := range m.sections {
		if i == m.activeSection {
			parts = append(parts, StyleTabActive.Render(s.Abbr))
		} else {
			parts = append(parts, StyleTabInactive.Render(s.Abbr))
		}
	}
	tabs := strings.Join(parts, "")
	rawW := lipgloss.Width(tabs)
	if rawW < w {
		tabs += StyleTabBar.Render(strings.Repeat(" ", w-rawW))
	}
	return tabs
}

func (m Model) renderStatusLine(w int) string {
	var modeLabel string
	switch m.activeMode() {
	case ModeNormal:
		modeLabel = StyleNormalMode.Render("NORMAL")
	case ModeInsert:
		modeLabel = StyleInsertMode.Render("INSERT")
	case ModeCommand:
		modeLabel = StyleCommandMode.Render("COMMAND")
	}

	sec := m.sections[m.activeSection]
	pos := fmt.Sprintf("%d/%d", m.activeSection+1, len(m.sections))
	right := StyleStatusRight.Render(fmt.Sprintf(" %s.manifest  %s  All ", sec.ID, pos))

	msg := ""
	if m.statusMsg != "" {
		if m.statusErr {
			msg = StyleMsgErr.Render(m.statusMsg)
		} else {
			msg = StyleMsgOK.Render(m.statusMsg)
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
		return StyleCmdLine.Render(":"+m.cmdBuffer) + cursor
	}

	// Delegate hint line to the active sub-editor.
	var line string
	switch {
	case m.isBackendSection():
		line = m.backendEditor.HintLine()
	case m.isDataSection():
		line = m.dataTabEditor.HintLine()
	case m.isContractsSection():
		line = m.contractsEditor.HintLine()
	case m.isFrontendSection():
		line = m.frontendEditor.HintLine()
	case m.isInfraSection():
		line = m.infraEditor.HintLine()
	case m.isCrossCutSection():
		line = m.crossCutEditor.HintLine()
	default:
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
			line = "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
		case ModeInsert:
			line = StyleInsertMode.Render(" -- INSERT -- ") + StyleHelpDesc.Render("  Esc: normal mode  Tab: next field")
		}
	}

	if lipgloss.Width(line) > w {
		line = line[:w-1]
	}
	return line
}
