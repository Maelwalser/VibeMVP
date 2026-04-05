package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
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
	descriptionEditor DescriptionEditor
	backendEditor     BackendEditor
	dataTabEditor     DataTabEditor
	contractsEditor   ContractsEditor
	frontendEditor    FrontendEditor
	infraEditor       InfraEditor
	crossCutEditor    CrossCutEditor
	realizeEditor     RealizeEditor

	cmd    cmdState
	modal  modalState
	realize realizeState

	filePath      string // active save path; empty = use onSave callback default
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

	menu := newProviderMenu()
	m := Model{
		sections:          initSections(),
		textInput:         ti,
		textArea:          ta,
		descriptionEditor: newDescriptionEditor(),
		backendEditor:     newBackendEditor(),
		dataTabEditor:     newDataTabEditor(),
		contractsEditor:   newContractsEditor(),
		frontendEditor:    newFrontendEditor(),
		infraEditor:       newInfraEditor(),
		crossCutEditor:    newCrossCutEditor(),
		realizeEditor:     newRealizeEditor(),
		realize:           realizeState{screen: newRealizationScreen()},
		modal:             modalState{menu: menu},
		onSave:            onSave,
	}
	// Sync realize editor with credentials already loaded from disk.
	m.realizeEditor = m.realizeEditor.UpdateProviderOptions(menu.GetConfiguredProviders())
	return m
}

// SetFilePath sets the active save path (used when loading an existing manifest).
func (m *Model) SetFilePath(path string) { m.filePath = path }

// FilePath returns the active save path.
func (m Model) FilePath() string { return m.filePath }

// SetSaveFunc sets the save callback.
func (m *Model) SetSaveFunc(fn SaveFunc) { m.onSave = fn }

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
	id := m.activeSectionID()
	// "realize" swaps in the RealizationScreen while it is running.
	if id == "realize" && m.realize.show {
		return m.realize.screen
	}
	if entry, ok := sectionRegistry[id]; ok {
		return entry.editor(&m)
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
		// Propagate to all sub-editors so insert-mode inputs resize immediately.
		resizeAllEditors(&m, wsz)
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
		realizePath := "manifest.json"
		if m.filePath != "" {
			realizePath = m.filePath
		}
		var startCmd tea.Cmd
		m.realize.screen, startCmd = m.realize.screen.Start(realizePath, mf)
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
		case "M", "ctrl+c":
			if m.modal.menu.focus != pmFocusCredential {
				m.modal.open = false
				return m, nil
			}
		case "esc":
			// Esc closes the modal when focus is on the provider list.
			// Otherwise let the menu handle it (steps back through auth/credential).
			if m.modal.menu.focus == pmFocusProviders {
				m.modal.open = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.modal.menu, cmd = m.modal.menu.Update(msg)
		// Sync configured providers to realize editor whenever modal state changes.
		m.realizeEditor = m.realizeEditor.UpdateProviderOptions(m.modal.menu.GetConfiguredProviders())
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

	// Section (tab) navigation with Tab/Shift+Tab/Shift+L/Shift+H only when not in insert mode.
	case "tab", "L":
		if e := m.activeEditor(); e != nil && e.Mode() == ModeInsert {
			return m.delegateUpdate(msg)
		}
		m.activeSection = (m.activeSection + 1) % len(m.sections)
		m.activeField = 0
		return m, nil

	case "shift+tab", "H":
		if e := m.activeEditor(); e != nil && e.Mode() == ModeInsert {
			return m.delegateUpdate(msg)
		}
		m.activeSection = (m.activeSection - 1 + len(m.sections)) % len(m.sections)
		m.activeField = 0
		return m, nil
	}

	// Delegate all remaining input to the active section editor.
	return m.delegateUpdate(msg)
}

// LoadManifestIntoModel reads the manifest at path and returns a new Model
// with all pillar editors populated from the manifest data.
func (m Model) LoadManifestIntoModel(path string) (Model, error) {
	mf, err := manifest.Load(path)
	if err != nil {
		return m, err
	}
	m.descriptionEditor.SetValue(mf.Description)
	m.backendEditor = m.backendEditor.FromBackendPillar(mf.Backend)
	m.dataTabEditor = m.dataTabEditor.FromDataPillar(mf.Data)
	m.contractsEditor = m.contractsEditor.FromContractsPillar(mf.Contracts)
	m.frontendEditor = m.frontendEditor.FromFrontendPillar(mf.Frontend)
	m.infraEditor = m.infraEditor.FromInfraPillar(mf.Infra)
	m.crossCutEditor = m.crossCutEditor.FromCrossCutPillar(mf.CrossCut)
	m.realizeEditor = m.realizeEditor.FromRealizeOptions(mf.Realize)
	// Restore configured provider selections.
	if len(mf.ConfiguredProviders) > 0 {
		if m.modal.menu.configured == nil {
			m.modal.menu.configured = make(map[string]ProviderSelection)
		}
		for label, pa := range mf.ConfiguredProviders {
			m.modal.menu.configured[label] = ProviderSelection{
				Provider:   pa.Provider,
				Model:      pa.Model,
				Version:    pa.Version,
				Auth:       pa.Auth,
				Credential: pa.Credential,
			}
		}
		m.realizeEditor = m.realizeEditor.UpdateProviderOptions(m.modal.menu.GetConfiguredProviders())
	}
	m.modified = false
	return m, nil
}

func (m Model) delegateUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	entry, ok := sectionRegistry[m.activeSectionID()]
	if !ok {
		return m, nil
	}
	cmd := entry.update(&m, msg)
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
	savePath := "manifest.json"
	if m.filePath != "" {
		savePath = m.filePath
	}
	m.cmd.status = fmt.Sprintf("%q written", savePath)
	m.cmd.isErr = false
	return m, nil
}

// BuildManifest converts the form state into a Manifest struct.
func (m Model) BuildManifest() *manifest.Manifest {
	dataPillar := m.dataTabEditor.ToManifestDataPillar()
	return &manifest.Manifest{
		Description: m.descriptionEditor.Value(),
		Data:        dataPillar,
		Backend:   m.backendEditor.ToManifest(),
		Contracts: m.contractsEditor.ToManifestContractsPillar(),
		Frontend:  m.frontendEditor.ToManifestFrontendPillar(),
		Infra:     m.infraEditor.ToManifestInfraPillar(),
		CrossCut:  m.crossCutEditor.ToManifestCrossCutPillar(),
		Realize:             m.realizeEditor.ToManifestRealizeOptions(),
		ConfiguredProviders: m.modal.menu.ToManifestConfiguredProviders(),

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

const minTermWidth = 60
const minTermHeight = 12

