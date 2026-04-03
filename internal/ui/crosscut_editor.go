package ui

import (
	"strings"

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

// ── mode ──────────────────────────────────────────────────────────────────────

type ccMode int

const (
	ccNormal ccMode = iota
	ccInsert
)

// ── field definitions ─────────────────────────────────────────────────────────

// unitOptionsForLanguages returns unit-testing framework options relevant to
// the given set of backend languages. Falls back to all options when empty.
func unitOptionsForLanguages(langs []string) []string {
	if len(langs) == 0 {
		return []string{"Jest", "Vitest", "pytest", "Go testing", "JUnit", "xUnit", "Other"}
	}
	seen := make(map[string]bool)
	var opts []string
	add := func(o string) {
		if !seen[o] {
			seen[o] = true
			opts = append(opts, o)
		}
	}
	for _, lang := range langs {
		switch strings.ToLower(lang) {
		case "go", "golang":
			add("Go testing")
			add("Testify")
		case "typescript", "javascript", "ts", "js":
			add("Jest")
			add("Vitest")
		case "python":
			add("pytest")
			add("unittest")
		case "java":
			add("JUnit")
			add("TestNG")
		case "kotlin":
			add("JUnit")
			add("Kotest")
		case "c#", "csharp", "dotnet", ".net":
			add("xUnit")
			add("NUnit")
			add("MSTest")
		case "rust":
			add("cargo test")
		case "ruby":
			add("RSpec")
			add("minitest")
		case "php":
			add("PHPUnit")
			add("Pest")
		default:
			add("Jest")
			add("pytest")
			add("Go testing")
			add("JUnit")
		}
	}
	add("Other")
	return opts
}

// e2eOptionsForFrontend returns E2E framework options suitable for the given
// frontend language and framework.
func e2eOptionsForFrontend(frontendLang, frontendFramework string) []string {
	lang := strings.ToLower(frontendLang)
	fw := strings.ToLower(frontendFramework)
	switch {
	case lang == "dart" || fw == "flutter":
		return []string{"Flutter Driver", "Integration Test", "None"}
	case lang == "kotlin" || fw == "compose multiplatform" || fw == "jetpack compose":
		return []string{"Espresso", "UI Automator", "None"}
	case lang == "swift" || fw == "swiftui" || fw == "uikit":
		return []string{"XCUITest", "EarlGrey", "None"}
	case lang == "" && fw == "":
		return []string{"None"}
	default:
		// Web frameworks
		return []string{"Playwright", "Cypress", "Selenium", "None"}
	}
}

// loadOptionsForLanguages returns load-testing tools relevant to the backend langs.
func loadOptionsForLanguages(langs []string) []string {
	base := []string{"k6", "Artillery", "JMeter", "None"}
	for _, lang := range langs {
		if strings.ToLower(lang) == "python" {
			return []string{"k6", "Locust", "Artillery", "JMeter", "None"}
		}
	}
	return base
}

// apiOptionsForProtocols returns API testing tool options relevant to the given
// communication protocols. Falls back to REST tools when no protocols are configured.
func apiOptionsForProtocols(protocols []string) []string {
	var hasREST, hasGraphQL, hasGRPC bool
	for _, p := range protocols {
		switch p {
		case "REST (HTTP)", "REST":
			hasREST = true
		case "GraphQL":
			hasGraphQL = true
		case "gRPC":
			hasGRPC = true
		}
	}
	// No relevant protocols — default to REST tools.
	if !hasREST && !hasGraphQL && !hasGRPC {
		return []string{"Bruno", "Hurl", "Postman/Newman", "REST Client", "None"}
	}
	// Mixed (more than one protocol present).
	activeCount := 0
	for _, v := range []bool{hasREST, hasGraphQL, hasGRPC} {
		if v {
			activeCount++
		}
	}
	if activeCount > 1 {
		return []string{"Bruno", "Postman/Newman", "None"}
	}
	if hasGraphQL {
		return []string{"Bruno", "Postman/Newman", "GraphQL Playground", "None"}
	}
	if hasGRPC {
		return []string{"grpcurl", "Postman/Newman", "BloomRPC", "None"}
	}
	// REST only.
	return []string{"Bruno", "Hurl", "Postman/Newman", "REST Client", "None"}
}

// contractOptionsForArchPattern returns contract-testing tool options based on
// the selected backend architecture pattern.
func contractOptionsForArchPattern(archPattern string) []string {
	switch archPattern {
	case "microservices":
		return []string{"Pact", "Schemathesis", "Dredd", "None"}
	case "event-driven":
		return []string{"Pact", "AsyncAPI validator", "None"}
	case "hybrid":
		return []string{"Pact", "Schemathesis", "Dredd", "None"}
	default: // monolith, modular-monolith, or unset
		return []string{"None", "Schemathesis"}
	}
}

// computeTestingFields builds testing Field definitions filtered to the given
// backend languages, protocols, arch pattern, and frontend tech. Existing values
// are preserved when the option is still available; otherwise the first option is selected.
func computeTestingFields(backendLangs, backendProtocols []string, backendArchPattern, frontendLang, frontendFramework string, existing []Field) []Field {
	unitOpts := unitOptionsForLanguages(backendLangs)
	e2eOpts := e2eOptionsForFrontend(frontendLang, frontendFramework)
	loadOpts := loadOptionsForLanguages(backendLangs)
	apiOpts := apiOptionsForProtocols(backendProtocols)
	contractOpts := contractOptionsForArchPattern(backendArchPattern)

	template := []struct {
		key, label string
		opts       []string
	}{
		{"unit", "unit          ", unitOpts},
		{"integration", "integration   ", []string{"Testcontainers", "Docker Compose", "In-memory fakes", "None"}},
		{"e2e", "e2e           ", e2eOpts},
		{"api", "api           ", apiOpts},
		{"load", "load          ", loadOpts},
		{"contract", "contract      ", contractOpts},
	}

	// Build lookup of existing values.
	existingVals := make(map[string]string, len(existing))
	for _, f := range existing {
		existingVals[f.Key] = f.Value
	}

	fields := make([]Field, 0, len(template))
	for _, t := range template {
		selIdx := 0
		val := t.opts[0]
		// Preserve current value when still valid.
		if prev, ok := existingVals[t.key]; ok {
			for i, o := range t.opts {
				if o == prev {
					selIdx = i
					val = o
					break
				}
			}
		}
		// Default contract to "None".
		if t.key == "contract" && val == t.opts[0] && existingVals[t.key] == "" {
			for i, o := range t.opts {
				if o == "None" {
					selIdx = i
					val = o
					break
				}
			}
		}
		fields = append(fields, Field{
			Key:    t.key,
			Label:  t.label,
			Kind:   KindSelect,
			Options: t.opts,
			Value:  val,
			SelIdx: selIdx,
		})
	}
	return fields
}

func defaultTestingFields() []Field {
	return computeTestingFields(nil, nil, "", "", "", nil)
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

	// Context from other editors — used to filter testing options.
	backendLangs        []string
	backendProtocols    []string
	backendArchPattern  string
	frontendLang        string
	frontendFramework   string
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

func (cc *CrossCutEditor) disableActiveTab() {
	switch cc.activeTab {
	case ccTabTesting:
		cc.testingEnabled = false
		cc.testingFields = computeTestingFields(cc.backendLangs, cc.backendProtocols, cc.backendArchPattern, cc.frontendLang, cc.frontendFramework, nil)
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

// SetTestingContext updates the backend languages, protocols, and frontend tech
// context used to filter testing framework options. If the testing tab is already
// enabled, the field options are recomputed immediately (preserving current selections).
func (cc *CrossCutEditor) SetTestingContext(backendLangs, backendProtocols []string, backendArchPattern, frontendLang, frontendFramework string) {
	// Nothing changed — skip expensive recompute.
	if stringSlicesEqual(cc.backendLangs, backendLangs) &&
		stringSlicesEqual(cc.backendProtocols, backendProtocols) &&
		cc.backendArchPattern == backendArchPattern &&
		cc.frontendLang == frontendLang &&
		cc.frontendFramework == frontendFramework {
		return
	}
	cc.backendLangs = backendLangs
	cc.backendProtocols = backendProtocols
	cc.backendArchPattern = backendArchPattern
	cc.frontendLang = frontendLang
	cc.frontendFramework = frontendFramework
	cc.testingFields = computeTestingFields(backendLangs, backendProtocols, backendArchPattern, frontendLang, frontendFramework, cc.testingFields)
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
		if f.PrepareCustomEntry() {
			return cc.tryEnterInsert()
		}
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
			cc.internalMode = ccInsert
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
