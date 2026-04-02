package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-mvp/internal/manifest"
)

// ── sub-tabs ──────────────────────────────────────────────────────────────────

type ccTabIdx int

const (
	ccTabTesting ccTabIdx = iota
	ccTabDocs
	ccTabStandards
)

var ccTabLabels = []string{"TESTING", "DOCS", "STANDARDS"}

// ── mode ──────────────────────────────────────────────────────────────────────

type ccMode int

const (
	ccNormal ccMode = iota
	ccInsert
)

// ── field definitions ─────────────────────────────────────────────────────────

func defaultTestingFields() []Field {
	return []Field{
		{
			Key: "unit", Label: "unit          ", Kind: KindSelect,
			Options: []string{
				"Jest", "Vitest", "pytest", "Go testing",
				"JUnit", "xUnit", "Other",
			},
			Value: "Go testing", SelIdx: 3,
		},
		{
			Key: "integration", Label: "integration   ", Kind: KindSelect,
			Options: []string{
				"Testcontainers", "Docker Compose", "In-memory fakes", "None",
			},
			Value: "Testcontainers",
		},
		{
			Key: "e2e", Label: "e2e           ", Kind: KindSelect,
			Options: []string{"Playwright", "Cypress", "Selenium", "None"},
			Value:   "Playwright",
		},
		{
			Key: "api", Label: "api           ", Kind: KindSelect,
			Options: []string{"Bruno", "Hurl", "Postman/Newman", "REST Client", "None"},
			Value:   "Hurl", SelIdx: 1,
		},
		{
			Key: "load", Label: "load          ", Kind: KindSelect,
			Options: []string{"k6", "Locust", "Artillery", "JMeter", "None"},
			Value:   "k6",
		},
		{
			Key: "contract", Label: "contract      ", Kind: KindSelect,
			Options: []string{"Pact", "Schemathesis", "Dredd", "None"},
			Value:   "None", SelIdx: 3,
		},
	}
}

func defaultStandardsFields() []Field {
	return []Field{
		{
			Key: "branch_strategy", Label: "Branch Strat. ", Kind: KindSelect,
			Options: []string{"GitHub Flow", "GitFlow", "Trunk-based", "Custom"},
			Value:   "GitHub Flow",
		},
		{
			Key: "dep_updates", Label: "Dep. Updates  ", Kind: KindSelect,
			Options: []string{"Dependabot", "Renovate", "Manual", "None"},
			Value:   "Dependabot",
		},
		{
			Key: "code_review", Label: "Code Review   ", Kind: KindSelect,
			Options: []string{"Required (1 approval)", "Required (2 approvals)", "Optional", "None"},
			Value:   "Required (1 approval)",
		},
		{
			Key: "feature_flags", Label: "Feature Flags ", Kind: KindSelect,
			Options: []string{"LaunchDarkly", "Unleash", "Flagsmith", "Custom (env vars)", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "uptime_slo", Label: "Uptime SLO    ", Kind: KindSelect,
			Options: []string{"99.9%", "99.95%", "99.99%", "Custom"},
			Value:   "99.9%",
		},
		{
			Key: "latency_p99", Label: "Latency P99   ", Kind: KindSelect,
			Options: []string{"<50ms", "<100ms", "<200ms", "<500ms", "<1s", "Custom"},
			Value:   "<200ms", SelIdx: 2,
		},
	}
}

func defaultDocsFields() []Field {
	return []Field{
		{
			Key: "api_docs", Label: "api_docs      ", Kind: KindSelect,
			Options: []string{
				"OpenAPI/Swagger", "GraphQL Playground",
				"gRPC reflection", "None",
			},
			Value: "OpenAPI/Swagger",
		},
		{
			Key: "auto_generate", Label: "auto_generate ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "true", SelIdx: 1,
		},
		{
			Key: "changelog", Label: "changelog     ", Kind: KindSelect,
			Options: []string{"Conventional Commits", "Manual", "None"},
			Value:   "Conventional Commits",
		},
	}
}

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

	internalMode ccMode
	formInput    textinput.Model
	width        int

	// Dropdown state
	ddOpen   bool
	ddOptIdx int

	// Vim motion state
	nav VimNav
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
	case ccTabStandards:
		cc.standardsEnabled = true
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
			Unit:        fieldGet(cc.testingFields, "unit"),
			Integration: fieldGet(cc.testingFields, "integration"),
			E2E:         fieldGet(cc.testingFields, "e2e"),
			API:         fieldGet(cc.testingFields, "api"),
			Load:        fieldGet(cc.testingFields, "load"),
			Contract:    fieldGet(cc.testingFields, "contract"),
		}
	}
	if cc.docsEnabled {
		p.Docs = manifest.DocsConfig{
			APIDocs:      fieldGet(cc.docsFields, "api_docs"),
			AutoGenerate: fieldGet(cc.docsFields, "auto_generate") == "true",
			Changelog:    fieldGet(cc.docsFields, "changelog"),
		}
	}
	if cc.standardsEnabled {
		p.BranchStrategy = fieldGet(cc.standardsFields, "branch_strategy")
		p.DependencyUpdates = fieldGet(cc.standardsFields, "dep_updates")
		p.CodeReview = fieldGet(cc.standardsFields, "code_review")
		p.FeatureFlags = fieldGet(cc.standardsFields, "feature_flags")
		p.UptimeSLO = fieldGet(cc.standardsFields, "uptime_slo")
		p.LatencyP99 = fieldGet(cc.standardsFields, "latency_p99")
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
		cc.testingFields = setFieldValue(cc.testingFields, "api", t.API)
		cc.testingFields = setFieldValue(cc.testingFields, "load", t.Load)
		cc.testingFields = setFieldValue(cc.testingFields, "contract", t.Contract)
	}

	d := p.Docs
	if d.APIDocs != "" {
		cc.docsEnabled = true
		cc.docsFields = setFieldValue(cc.docsFields, "api_docs", d.APIDocs)
		boolStr := func(b bool) string {
			if b {
				return "true"
			}
			return "false"
		}
		cc.docsFields = setFieldValue(cc.docsFields, "auto_generate", boolStr(d.AutoGenerate))
		cc.docsFields = setFieldValue(cc.docsFields, "changelog", d.Changelog)
	}

	if p.BranchStrategy != "" || p.DependencyUpdates != "" {
		cc.standardsEnabled = true
		cc.standardsFields = setFieldValue(cc.standardsFields, "branch_strategy", p.BranchStrategy)
		cc.standardsFields = setFieldValue(cc.standardsFields, "dep_updates", p.DependencyUpdates)
		cc.standardsFields = setFieldValue(cc.standardsFields, "code_review", p.CodeReview)
		cc.standardsFields = setFieldValue(cc.standardsFields, "feature_flags", p.FeatureFlags)
		cc.standardsFields = setFieldValue(cc.standardsFields, "uptime_slo", p.UptimeSLO)
		cc.standardsFields = setFieldValue(cc.standardsFields, "latency_p99", p.LatencyP99)
	}

	return cc
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (cc CrossCutEditor) Mode() Mode {
	if cc.internalMode == ccInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (cc CrossCutEditor) HintLine() string {
	if cc.internalMode == ccInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	if !cc.activeTabEnabled() {
		return hintBar("a", "configure", "h/l", "sub-tab")
	}
	return hintBar("j/k", "navigate", "gg/G", "top/bottom", "[n]j/k", "jump n lines", "Space/Enter", "cycle", "H", "cycle back", "a/i", "edit text", "h/l", "sub-tab")
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
		cc.ddOpen = false
		return cc, nil
	}
	switch key.String() {
	case "j", "down":
		if cc.ddOptIdx < len(f.Options)-1 {
			cc.ddOptIdx++
		}
	case "k", "up":
		if cc.ddOptIdx > 0 {
			cc.ddOptIdx--
		}
	case "g":
		cc.ddOptIdx = 0
	case "G":
		if len(f.Options) > 0 {
			cc.ddOptIdx = len(f.Options) - 1
		}
	case " ", "enter":
		f.SelIdx = cc.ddOptIdx
		if cc.ddOptIdx < len(f.Options) {
			f.Value = f.Options[cc.ddOptIdx]
		}
		cc.ddOpen = false
	case "esc", "b":
		cc.ddOpen = false
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
	if cc.internalMode == ccInsert {
		return cc.updateInsert(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return cc, nil
	}

	if cc.ddOpen {
		return cc.updateCCDropdown(key)
	}

	switch key.String() {
	case "h", "left", "l", "right":
		cc.activeTab = ccTabIdx(NavigateTab(key.String(), int(cc.activeTab), len(ccTabLabels)))
		return cc, nil
	}

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
					cc.ddOpen = true
					cc.ddOptIdx = f.SelIdx
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
			cc.internalMode = ccNormal
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
		if cc.testFormIdx < len(cc.testingFields) && cc.testingFields[cc.testFormIdx].Kind == KindText {
			cc.testingFields[cc.testFormIdx].Value = val
		}
	case ccTabDocs:
		if cc.docsFormIdx < len(cc.docsFields) && cc.docsFields[cc.docsFormIdx].Kind == KindText {
			cc.docsFields[cc.docsFormIdx].Value = val
		}
	case ccTabStandards:
		if cc.standardsFormIdx < len(cc.standardsFields) && cc.standardsFields[cc.standardsFormIdx].Kind == KindText {
			cc.standardsFields[cc.standardsFormIdx].Value = val
		}
	}
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
		if f.Kind == KindText || f.Kind == KindTextArea {
			cc.internalMode = ccInsert
			cc.formInput.SetValue(f.Value)
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

	switch cc.activeTab {
	case ccTabTesting:
		if cc.testingEnabled {
			lines = append(lines, renderFormFields(w, cc.testingFields, cc.testFormIdx, cc.internalMode == ccInsert, cc.formInput, cc.ddOpen, cc.ddOptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case ccTabDocs:
		if cc.docsEnabled {
			lines = append(lines, renderFormFields(w, cc.docsFields, cc.docsFormIdx, cc.internalMode == ccInsert, cc.formInput, cc.ddOpen, cc.ddOptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case ccTabStandards:
		if cc.standardsEnabled {
			lines = append(lines, renderFormFields(w, cc.standardsFields, cc.standardsFormIdx, cc.internalMode == ccInsert, cc.formInput, cc.ddOpen, cc.ddOptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	}

	return fillTildes(lines, h)
}
