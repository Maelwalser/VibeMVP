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
	ModeNormal  Mode = iota
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

	// Input widgets (reused for the active field)
	textInput textinput.Model
	textArea  textarea.Model

	cmdBuffer  string // characters typed after ':'
	statusMsg  string // transient status line message
	statusErr  bool   // true = red, false = green
	modified   bool

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
		sections:  initSections(),
		textInput: ti,
		textArea:  ta,
		onSave:    onSave,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
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
		return m, nil
	}
	m.statusMsg = "" // clear on any keypress in normal mode

	switch key.String() {
	case "ctrl+c":
		return m, tea.Quit

	// ── Vertical navigation ──────────────────────────────────────────────
	case "j", "down":
		sec := m.sections[m.activeSection]
		if m.activeField < len(sec.Fields)-1 {
			m.activeField++
		}

	case "k", "up":
		if m.activeField > 0 {
			m.activeField--
		}

	// ── Section (tab) navigation ─────────────────────────────────────────
	case "tab", "l", "right":
		m.activeSection = (m.activeSection + 1) % len(m.sections)
		m.activeField = 0

	case "shift+tab", "h", "left":
		m.activeSection = (m.activeSection - 1 + len(m.sections)) % len(m.sections)
		m.activeField = 0

	// ── Jump to first / last field ────────────────────────────────────────
	case "g":
		m.activeField = 0

	case "G":
		m.activeField = len(m.sections[m.activeSection].Fields) - 1

	// ── Enter INSERT mode ─────────────────────────────────────────────────
	case "i", "a":
		return m.enterInsert()

	// ── Select fields: cycle with Enter / Space ───────────────────────────
	case "enter", " ":
		sec := &m.sections[m.activeSection]
		f := &sec.Fields[m.activeField]
		if f.Kind == KindSelect {
			f.CycleNext()
			m.modified = true
		} else {
			return m.enterInsert()
		}

	case "shift+left", "H":
		sec := &m.sections[m.activeSection]
		f := &sec.Fields[m.activeField]
		if f.Kind == KindSelect {
			f.CyclePrev()
			m.modified = true
		}

	// ── Command mode ──────────────────────────────────────────────────────
	case ":":
		m.mode = ModeCommand
		m.cmdBuffer = ""

	// ── Quick save ────────────────────────────────────────────────────────
	case "ctrl+s":
		return m.execSave()
	}

	return m, nil
}

func (m Model) enterInsert() (Model, tea.Cmd) {
	sec := m.sections[m.activeSection]
	f := sec.Fields[m.activeField]

	if f.Kind == KindSelect {
		return m, nil // select fields don't use insert mode
	}

	m.mode = ModeInsert

	if f.Kind == KindTextArea {
		m.textArea.SetValue(f.Value)
		m.textArea.SetWidth(m.width - 4)
		m.textArea.SetHeight(m.contentHeight() - 4)
		return m, m.textArea.Focus()
	}

	m.textInput.SetValue(f.Value)
	m.textInput.Width = m.width - 22 // label(14) + " = "(3) + padding
	m.textInput.CursorEnd()
	return m, m.textInput.Focus()
}

func (m Model) updateInsert(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			return m.exitInsert()

		case "tab":
			// Save current field and move to next, staying in INSERT mode.
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
	case "esc":
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
		// chain quit after save
		return model, tea.Sequence(saveCmd, tea.Quit)

	case "tabn", "bn":
		m.activeSection = (m.activeSection + 1) % len(m.sections)
		m.activeField = 0

	case "tabp", "bp":
		m.activeSection = (m.activeSection - 1 + len(m.sections)) % len(m.sections)
		m.activeField = 0

	case "help", "h":
		m.statusMsg = "j/k:nav  i:insert  Tab:section  Enter:cycle  :w save  :q quit  :wq save+quit"
		m.statusErr = false

	default:
		// :1-9 to jump to section by number
		if len(cmd) == 1 && cmd[0] >= '1' && cmd[0] <= '9' {
			idx := int(cmd[0]-'1')
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
	get := func(secID, key string) string {
		for _, s := range m.sections {
			if s.ID != secID {
				continue
			}
			for _, f := range s.Fields {
				if f.Key == key {
					return f.DisplayValue()
				}
			}
		}
		return ""
	}

	return &manifest.Manifest{
		// Phase 1 – Universal Global Constants
		Domain: manifest.DomainPillar{
			EntityRelationships: get("domain", "er_model"),
			Cardinality:         get("domain", "cardinality"),
			CascadingRules:      get("domain", "cascading"),
			RBACMatrix:          get("domain", "rbac_matrix"),
			Compliance:          get("domain", "compliance"),
		},
		Topology: manifest.TopologyPillar{
			ArchPattern:   manifest.ArchPattern(get("topology", "arch_pattern")),
			CommProtocol:  manifest.CommProtocol(get("topology", "comm_protocol")),
			Serialization: manifest.SerializationFmt(get("topology", "serialization")),
			DomainNotes:   get("topology", "domain_notes"),
		},
		GlobalNFR: manifest.GlobalNFRPillar{
			UptimeSLO:      get("gnfr", "uptime_slo"),
			ConcurrentConn: get("gnfr", "concurrent_conn"),
			RTO:            get("gnfr", "rto"),
			RPO:            get("gnfr", "rpo"),
			NFRNotes:       get("gnfr", "nfr_notes"),
		},

		// Phase 2 – Domain-Specific Execution Paths
		Backend: manifest.BackendPillar{
			ComputeEnv:    manifest.ComputeEnv(get("backend", "compute_env")),
			CloudProvider: get("backend", "cloud_provider"),
			Runtime:       get("backend", "runtime"),
			Framework:     get("backend", "be_framework"),
			PrimaryDB:     manifest.DatabaseType(get("backend", "primary_db")),
			CacheStore:    manifest.CacheStore(get("backend", "cache_store")),
			CacheStrategy: get("backend", "cache_strategy"),
			MessageBroker: get("backend", "msg_broker"),
			ExternalAPIs:  get("backend", "external_apis"),
		},
		Frontend: manifest.FrontendPillar{
			Rendering:     manifest.RenderingMode(get("frontend", "rendering")),
			Framework:     get("frontend", "fe_framework"),
			ServerState:   get("frontend", "server_state"),
			ClientState:   get("frontend", "client_state"),
			Styling:       get("frontend", "styling"),
			BrowserMatrix: get("frontend", "browser_matrix"),
		},
		Desktop: manifest.DesktopPillar{
			TargetOS:     get("desktop", "target_os"),
			AppFramework: manifest.DesktopFramework(get("desktop", "app_framework")),
			HWAccess:     get("desktop", "hw_access"),
			IPCModel:     get("desktop", "ipc_model"),
			Distribution: get("desktop", "distribution"),
		},

		// Phase 3 – Lifecycle Operations & Tooling
		Testing: manifest.TestingPillar{
			UnitCoverage:    get("testing", "unit_coverage"),
			IntegCoverage:   get("testing", "integ_coverage"),
			E2EFramework:    manifest.E2EFramework(get("testing", "e2e_framework")),
			E2ECoverage:     get("testing", "e2e_coverage"),
			TestingStrategy: get("testing", "test_strategy"),
		},
		CICD: manifest.CICDPillar{
			CIPlatform:    manifest.CIPlatform(get("cicd", "ci_platform")),
			PipelineGates: get("cicd", "pipeline_gates"),
			EnvStrategy:   get("cicd", "env_strategy"),
			SecretsMgmt:   manifest.SecretsBackend(get("cicd", "secrets_mgmt")),
		},
		Telemetry: manifest.TelemetryPillar{
			LogSolution: manifest.LogSolution(get("telemetry", "log_solution")),
			LogFormat:   get("telemetry", "log_format"),
			Metrics:     get("telemetry", "metrics"),
			Tracing:     get("telemetry", "tracing"),
			Alerting:    get("telemetry", "alerting"),
		},
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

// renderHeader renders the top bar: filename + section counter.
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

// renderContent renders the main editing area.
func (m Model) renderContent(w int) string {
	sec := m.sections[m.activeSection]
	ch := m.contentHeight()

	// When a textarea field is active in INSERT mode, give it the full area.
	if m.mode == ModeInsert {
		f := sec.Fields[m.activeField]
		if f.Kind == KindTextArea {
			return m.renderTextAreaFull(w, ch, f)
		}
	}

	return m.renderFieldList(w, ch, sec)
}

func (m Model) renderTextAreaFull(w, h int, f Field) string {
	var b strings.Builder
	label := StyleTextAreaLabel.Render(fmt.Sprintf("  ── %s ─── (Esc to exit INSERT) ", f.Key))
	b.WriteString(label + "\n")
	b.WriteString(StyleTextAreaBorder.
		Width(w-2).
		Height(h-2).
		Render(m.textArea.View()))
	b.WriteString("\n")
	return b.String()
}

func (m Model) renderFieldList(w, h int, sec Section) string {
	const lineNumW = 4   // e.g. " 1  "
	const labelW = 14    // matches the padded Label strings
	const eqW = 3        // " = "

	valW := w - lineNumW - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	var lines []string

	// Section description as a comment line
	descLine := StyleSectionDesc.Render(fmt.Sprintf("  # %s", sec.Desc))
	lines = append(lines, descLine)
	lines = append(lines, "")

	for i, f := range sec.Fields {
		lineNo := i + 1
		isCur := i == m.activeField

		// ── Line number ───────────────────────────────────────────────────
		var numStr string
		if isCur {
			numStr = StyleCurLineNum.Render(fmt.Sprintf("%3d ", lineNo))
		} else {
			numStr = StyleLineNum.Render(fmt.Sprintf("%3d ", lineNo))
		}

		// ── Key label ─────────────────────────────────────────────────────
		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}

		// ── Equals sign ───────────────────────────────────────────────────
		eq := StyleEquals.Render(" = ")

		// ── Value ─────────────────────────────────────────────────────────
		var valStr string
		if m.mode == ModeInsert && isCur && f.Kind == KindText {
			// Show live textinput cursor
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
			// Show cursor placeholder when empty
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
			// Pad and highlight the entire row
			rawW := lipgloss.Width(row)
			if rawW < w {
				row += strings.Repeat(" ", w-rawW)
			}
			row = StyleCurLine.Render(row)
		}

		lines = append(lines, row)
	}

	// Fill with tilde lines
	for len(lines) < h {
		lines = append(lines, StyleTilde.Render("~"))
	}

	// Trim to content height
	if len(lines) > h {
		lines = lines[:h]
	}

	return strings.Join(lines, "\n") + "\n"
}

// renderTabBar renders the vim-style tab bar.
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

// renderStatusLine renders the bottom status bar.
func (m Model) renderStatusLine(w int) string {
	var modeLabel string
	switch m.mode {
	case ModeNormal:
		modeLabel = StyleNormalMode.Render("NORMAL")
	case ModeInsert:
		modeLabel = StyleInsertMode.Render("INSERT")
	case ModeCommand:
		modeLabel = StyleCommandMode.Render("COMMAND")
	}

	sec := m.sections[m.activeSection]
	pos := fmt.Sprintf("%d,%d", m.activeField+1, len(sec.Fields))
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

// renderCmdLine renders the very bottom command / hint line.
func (m Model) renderCmdLine(w int) string {
	switch m.mode {
	case ModeCommand:
		cursor := StyleCursor.Render(" ")
		return StyleCmdLine.Render(":"+m.cmdBuffer) + cursor

	case ModeNormal:
		hints := []string{
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
			StyleHelpKey.Render("i") + StyleHelpDesc.Render(" insert"),
			StyleHelpKey.Render("Tab") + StyleHelpDesc.Render(" section"),
			StyleHelpKey.Render("Enter") + StyleHelpDesc.Render(" cycle"),
			StyleHelpKey.Render(":w") + StyleHelpDesc.Render(" save"),
			StyleHelpKey.Render(":q") + StyleHelpDesc.Render(" quit"),
		}
		line := "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
		if lipgloss.Width(line) > w {
			line = line[:w-1]
		}
		return line

	case ModeInsert:
		return StyleInsertMode.Render(" -- INSERT -- ") + StyleHelpDesc.Render("  Esc: normal mode  Tab: next field")
	}

	return ""
}
