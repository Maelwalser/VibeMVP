package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── sub-tabs ──────────────────────────────────────────────────────────────────

type ccTabIdx int

const (
	ccTabTesting ccTabIdx = iota
	ccTabDocs
	ccTabStandards
)

var ccTabLabels = []string{"TESTING", "DOCS", "STANDARDS"}


// ── CrossCutEditor ────────────────────────────────────────────────────────────

// CrossCutEditor manages the CROSS-CUTTING CONCERNS main-tab.
type CrossCutEditor struct {
	activeTab ccTabIdx

	testingFields  []Field
	testFormIdx    int
	testingEnabled bool

	docsFields  []Field
	docsFormIdx int
	docsEnabled bool

	standardsFields  []Field
	standardsFormIdx int
	standardsEnabled bool

	internalMode Mode
	formInput    textinput.Model
	width        int

	// Dropdown state
	dd DropdownState

	// Vim motion state
	nav  VimNav
	cBuf bool

	// Context from other editors — used to filter testing options.
	backendLangs        []string
	backendProtocols    []string
	backendArchPattern  string
	frontendLang        string
	frontendFramework   string

	// Active API protocols sourced from ContractsEditor — used to build
	// per-protocol documentation format fields.
	docsProtocols []string
}

func (cc CrossCutEditor) activeTabEnabled() bool {
	switch cc.activeTab {
	case ccTabTesting:
		return cc.testingEnabled
	case ccTabDocs:
		return cc.docsEnabled
	case ccTabStandards:
		return cc.standardsEnabled
	}
	return false
}

func (cc *CrossCutEditor) enableActiveTab() {
	switch cc.activeTab {
	case ccTabTesting:
		cc.testingEnabled = true
		cc.testFormIdx = 0
	case ccTabDocs:
		cc.docsEnabled = true
		cc.docsFormIdx = 0
		cc.rebuildDocsFields()
	case ccTabStandards:
		cc.standardsEnabled = true
		cc.standardsFormIdx = 0
	}
}

func (cc *CrossCutEditor) disableActiveTab() {
	switch cc.activeTab {
	case ccTabTesting:
		cc.testingEnabled = false
		cc.testingFields = computeTestingFields(cc.backendLangs, cc.backendProtocols, cc.backendArchPattern, "", cc.frontendLang, cc.frontendFramework, nil)
		cc.testFormIdx = 0
	case ccTabDocs:
		cc.docsEnabled = false
		cc.docsFields = defaultDocsFields()
		cc.docsFormIdx = 0
	case ccTabStandards:
		cc.standardsEnabled = false
		cc.standardsFields = defaultStandardsFields()
		cc.standardsFormIdx = 0
	}
}

func newCrossCutEditor() CrossCutEditor {
	return CrossCutEditor{
		testingFields:   defaultTestingFields(),
		docsFields:      defaultDocsFields(),
		standardsFields: defaultStandardsFields(),
		formInput:       newFormInput(),
	}
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (cc CrossCutEditor) ToManifestCrossCutPillar() manifest.CrossCutPillar {
	var p manifest.CrossCutPillar
	if cc.testingEnabled {
		p.Testing = manifest.TestingConfig{
			Unit:            fieldGet(cc.testingFields, "unit"),
			Integration:     fieldGet(cc.testingFields, "integration"),
			E2E:             fieldGet(cc.testingFields, "e2e"),
			FrontendTesting: fieldGet(cc.testingFields, "fe_testing"),
			API:             fieldGet(cc.testingFields, "api"),
			Load:            fieldGet(cc.testingFields, "load"),
			Contract:        fieldGet(cc.testingFields, "contract"),
		}
	}
	if cc.docsEnabled {
		protos := cc.docsProtocols
		if len(protos) == 0 {
			protos = []string{"REST"}
		}
		formats := make(map[string]string, len(protos))
		for _, proto := range protos {
			key := docsFormatFieldKey(proto)
			if v := fieldGet(cc.docsFields, key); v != "" && v != "None" {
				formats[proto] = v
			}
		}
		p.Docs = manifest.DocsConfig{
			PerProtocolFormats: formats,
			AutoGenerate:       fieldGet(cc.docsFields, "auto_generate") == "true",
			Changelog:          fieldGet(cc.docsFields, "changelog"),
		}
	}
	if cc.standardsEnabled {
		p.DependencyUpdates = fieldGet(cc.standardsFields, "dep_updates")
		p.FeatureFlags = fieldGet(cc.standardsFields, "feature_flags")
		p.UptimeSLO = fieldGet(cc.standardsFields, "uptime_slo")
		p.LatencyP99 = fieldGet(cc.standardsFields, "latency_p99")
		p.BackendLinter = fieldGet(cc.standardsFields, "be_linter")
		p.FrontendLinter = fieldGet(cc.standardsFields, "fe_linter")
	}
	return p
}

// FromCrossCutPillar populates the editor from a saved manifest CrossCutPillar,
// reversing the ToManifestCrossCutPillar() operation.
func (cc CrossCutEditor) FromCrossCutPillar(p manifest.CrossCutPillar) CrossCutEditor {
	t := p.Testing
	if t.Unit != "" || t.Integration != "" || t.E2E != "" {
		cc.testingEnabled = true
		cc.testingFields = setFieldValue(cc.testingFields, "unit", t.Unit)
		cc.testingFields = setFieldValue(cc.testingFields, "integration", t.Integration)
		cc.testingFields = setFieldValue(cc.testingFields, "e2e", t.E2E)
		cc.testingFields = setFieldValue(cc.testingFields, "fe_testing", t.FrontendTesting)
		cc.testingFields = setFieldValue(cc.testingFields, "api", t.API)
		cc.testingFields = setFieldValue(cc.testingFields, "load", t.Load)
		cc.testingFields = setFieldValue(cc.testingFields, "contract", t.Contract)
	}

	d := p.Docs
	if len(d.PerProtocolFormats) > 0 || d.APIDocs != "" || d.Changelog != "" {
		cc.docsEnabled = true
		boolStr := func(b bool) string {
			if b {
				return "true"
			}
			return "false"
		}
		if len(d.PerProtocolFormats) > 0 {
			// Rebuild fields for the saved protocols so per-protocol keys exist.
			order := []string{"REST", "GraphQL", "gRPC", "WebSocket message", "Event"}
			var protos []string
			for _, proto := range order {
				if _, ok := d.PerProtocolFormats[proto]; ok {
					protos = append(protos, proto)
				}
			}
			cc.docsProtocols = protos
			cc.rebuildDocsFields()
			for proto, format := range d.PerProtocolFormats {
				cc.docsFields = setFieldValue(cc.docsFields, docsFormatFieldKey(proto), format)
			}
		} else if d.APIDocs != "" {
			// Migrate legacy single-format field to REST.
			cc.docsFields = setFieldValue(cc.docsFields, docsFormatFieldKey("REST"), d.APIDocs)
		}
		cc.docsFields = setFieldValue(cc.docsFields, "auto_generate", boolStr(d.AutoGenerate))
		if d.Changelog != "" {
			cc.docsFields = setFieldValue(cc.docsFields, "changelog", d.Changelog)
		}
	}

	if p.DependencyUpdates != "" || p.BackendLinter != "" || p.FrontendLinter != "" {
		cc.standardsEnabled = true
		cc.standardsFields = setFieldValue(cc.standardsFields, "dep_updates", p.DependencyUpdates)
		cc.standardsFields = setFieldValue(cc.standardsFields, "feature_flags", p.FeatureFlags)
		cc.standardsFields = setFieldValue(cc.standardsFields, "uptime_slo", p.UptimeSLO)
		cc.standardsFields = setFieldValue(cc.standardsFields, "latency_p99", p.LatencyP99)
		cc.standardsFields = setFieldValue(cc.standardsFields, "be_linter", p.BackendLinter)
		cc.standardsFields = setFieldValue(cc.standardsFields, "fe_linter", p.FrontendLinter)
	}

	return cc
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (cc CrossCutEditor) Mode() Mode {
	if cc.internalMode == ModeInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (cc CrossCutEditor) HintLine() string {
	if cc.internalMode == ModeInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	if !cc.activeTabEnabled() {
		return hintBar("a", "configure", "h/l", "sub-tab")
	}
	return hintBar("j/k", "navigate", "gg/G", "top/bottom", "[n]j/k", "jump n lines", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit text", "h/l", "sub-tab")
}

// activeCCFieldPtr returns a pointer to the currently focused field.
func (cc *CrossCutEditor) activeCCFieldPtr() *Field {
	switch cc.activeTab {
	case ccTabTesting:
		if cc.testFormIdx < len(cc.testingFields) {
			return &cc.testingFields[cc.testFormIdx]
		}
	case ccTabDocs:
		if cc.docsFormIdx < len(cc.docsFields) {
			return &cc.docsFields[cc.docsFormIdx]
		}
	case ccTabStandards:
		if cc.standardsFormIdx < len(cc.standardsFields) {
			return &cc.standardsFields[cc.standardsFormIdx]
		}
	}
	return nil
}

func (cc CrossCutEditor) updateCCDropdown(key tea.KeyMsg) (CrossCutEditor, tea.Cmd) {
	f := cc.activeCCFieldPtr()
	if f == nil {
		cc.dd.Open = false
		return cc, nil
	}
	cc.dd.OptIdx = NavigateDropdown(key.String(), cc.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = cc.dd.OptIdx
		if cc.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[cc.dd.OptIdx]
		}
		cc.dd.Open = false
		if f.PrepareCustomEntry() {
			return cc.tryEnterInsert()
		}
	case "esc", "b":
		cc.dd.Open = false
	}
	return cc, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (cc CrossCutEditor) Update(msg tea.Msg) (CrossCutEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		cc.width = wsz.Width
		cc.formInput.Width = wsz.Width - 22
		return cc, nil
	}
	if cc.internalMode == ModeInsert {
		return cc.updateInsert(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return cc, nil
	}

	if cc.dd.Open {
		return cc.updateCCDropdown(key)
	}

	switch key.String() {
	case "h", "left", "l", "right":
		cc.activeTab = ccTabIdx(NavigateTab(key.String(), int(cc.activeTab), len(ccTabLabels)))
		return cc, nil
	}

	// cc detection: clear field and enter insert mode
	if key.String() == "c" {
		if cc.cBuf {
			cc.cBuf = false
			return cc.clearAndEnterInsert()
		}
		cc.cBuf = true
		return cc, nil
	}
	cc.cBuf = false

	switch cc.activeTab {
	case ccTabTesting:
		return cc.updateFields(key)
	case ccTabDocs:
		return cc.updateFields(key)
	case ccTabStandards:
		return cc.updateFields(key)
	}
	return cc, nil
}

func (cc CrossCutEditor) updateFields(key tea.KeyMsg) (CrossCutEditor, tea.Cmd) {
	if !cc.activeTabEnabled() {
		if key.String() == "a" {
			cc.enableActiveTab()
		}
		return cc, nil
	}
	var fields []Field
	var idx int
	switch cc.activeTab {
	case ccTabTesting:
		fields, idx = cc.testingFields, cc.testFormIdx
	case ccTabDocs:
		fields, idx = cc.docsFields, cc.docsFormIdx
	case ccTabStandards:
		fields, idx = cc.standardsFields, cc.standardsFormIdx
	default:
		return cc, nil
	}
	n := len(fields)
	k := key.String()
	wantsInsert := false

	if newIdx, consumed := cc.nav.Handle(k, idx, n); consumed {
		idx = newIdx
	} else {
		cc.nav.Reset()
		switch k {
		case "enter", " ":
			if idx < n {
				f := &fields[idx]
				if f.Kind == KindSelect {
					cc.dd.Open = true
					cc.dd.OptIdx = f.SelIdx
				} else {
					wantsInsert = true
				}
			}
		case "H", "shift+left":
			if idx < n {
				f := &fields[idx]
				if f.Kind == KindSelect {
					f.CyclePrev()
				}
			}
		case "D":
			cc.disableActiveTab()
			return cc, nil
		case "i", "a":
			wantsInsert = true
		}
	}
	// Write back updated fields and index
	switch cc.activeTab {
	case ccTabTesting:
		cc.testingFields = fields
		cc.testFormIdx = idx
	case ccTabDocs:
		cc.docsFields = fields
		cc.docsFormIdx = idx
	case ccTabStandards:
		cc.standardsFields = fields
		cc.standardsFormIdx = idx
	}
	if wantsInsert {
		return cc.tryEnterInsert()
	}
	return cc, nil
}

func (cc CrossCutEditor) updateInsert(msg tea.Msg) (CrossCutEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			cc.saveInput()
			cc.internalMode = ModeNormal
			cc.formInput.Blur()
			return cc, nil
		case "tab":
			cc.saveInput()
			cc.advanceField(1)
			return cc.tryEnterInsert()
		case "shift+tab":
			cc.saveInput()
			cc.advanceField(-1)
			return cc.tryEnterInsert()
		}
	}
	var cmd tea.Cmd
	cc.formInput, cmd = cc.formInput.Update(msg)
	return cc, cmd
}

func (cc *CrossCutEditor) advanceField(delta int) {
	switch cc.activeTab {
	case ccTabTesting:
		n := len(cc.testingFields)
		if n > 0 {
			cc.testFormIdx = (cc.testFormIdx + delta + n) % n
		}
	case ccTabDocs:
		n := len(cc.docsFields)
		if n > 0 {
			cc.docsFormIdx = (cc.docsFormIdx + delta + n) % n
		}
	case ccTabStandards:
		n := len(cc.standardsFields)
		if n > 0 {
			cc.standardsFormIdx = (cc.standardsFormIdx + delta + n) % n
		}
	}
}

func (cc *CrossCutEditor) saveInput() {
	val := cc.formInput.Value()
	switch cc.activeTab {
	case ccTabTesting:
		if cc.testFormIdx < len(cc.testingFields) && cc.testingFields[cc.testFormIdx].CanEditAsText() {
			cc.testingFields[cc.testFormIdx].SaveTextInput(val)
		}
	case ccTabDocs:
		if cc.docsFormIdx < len(cc.docsFields) && cc.docsFields[cc.docsFormIdx].CanEditAsText() {
			cc.docsFields[cc.docsFormIdx].SaveTextInput(val)
		}
	case ccTabStandards:
		if cc.standardsFormIdx < len(cc.standardsFields) && cc.standardsFields[cc.standardsFormIdx].CanEditAsText() {
			cc.standardsFields[cc.standardsFormIdx].SaveTextInput(val)
		}
	}
}

func (cc CrossCutEditor) clearAndEnterInsert() (CrossCutEditor, tea.Cmd) {
	cc, cmd := cc.tryEnterInsert()
	if cc.internalMode == ModeInsert {
		cc.formInput.SetValue("")
	}
	return cc, cmd
}

func (cc CrossCutEditor) tryEnterInsert() (CrossCutEditor, tea.Cmd) {
	n := 0
	switch cc.activeTab {
	case ccTabTesting:
		n = len(cc.testingFields)
	case ccTabDocs:
		n = len(cc.docsFields)
	case ccTabStandards:
		n = len(cc.standardsFields)
	}
	for range n {
		var f *Field
		switch cc.activeTab {
		case ccTabTesting:
			if cc.testFormIdx < len(cc.testingFields) {
				f = &cc.testingFields[cc.testFormIdx]
			}
		case ccTabDocs:
			if cc.docsFormIdx < len(cc.docsFields) {
				f = &cc.docsFields[cc.docsFormIdx]
			}
		case ccTabStandards:
			if cc.standardsFormIdx < len(cc.standardsFields) {
				f = &cc.standardsFields[cc.standardsFormIdx]
			}
		}
		if f == nil {
			break
		}
		if f.CanEditAsText() {
			cc.internalMode = ModeInsert
			cc.formInput.SetValue(f.TextInputValue())
			cc.formInput.Width = cc.width - 22
			cc.formInput.CursorEnd()
			return cc, cc.formInput.Focus()
		}
		cc.advanceField(1)
	}
	return cc, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (cc CrossCutEditor) View(w, h int) string {
	cc.width = w
	cc.formInput.Width = w - 22
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Cross-Cutting Concerns — testing strategy and documentation"),
		"",
		renderSubTabBar(ccTabLabels, int(cc.activeTab), w),
		"",
	)

	const ccHeaderH = 4
	switch cc.activeTab {
	case ccTabTesting:
		if cc.testingEnabled {
			fl := renderFormFields(w, cc.testingFields, cc.testFormIdx, cc.internalMode == ModeInsert, cc.formInput, cc.dd.Open, cc.dd.OptIdx)
			lines = append(lines, appendViewport(fl, 0, cc.testFormIdx, h-ccHeaderH)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case ccTabDocs:
		if cc.docsEnabled {
			fl := renderFormFields(w, cc.docsFields, cc.docsFormIdx, cc.internalMode == ModeInsert, cc.formInput, cc.dd.Open, cc.dd.OptIdx)
			lines = append(lines, appendViewport(fl, 0, cc.docsFormIdx, h-ccHeaderH)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case ccTabStandards:
		if cc.standardsEnabled {
			fl := renderFormFields(w, cc.standardsFields, cc.standardsFormIdx, cc.internalMode == ModeInsert, cc.formInput, cc.dd.Open, cc.dd.OptIdx)
			lines = append(lines, appendViewport(fl, 0, cc.standardsFormIdx, h-ccHeaderH)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	}

	return fillTildes(lines, h)
}
