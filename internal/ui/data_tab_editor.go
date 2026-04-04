package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── sub-tabs ──────────────────────────────────────────────────────────────────

type dataTabIdx int

const (
	dataTabDatabases dataTabIdx = iota
	dataTabDomains
	dataTabCaching
	dataTabFileStorage
	dataTabGovernance
)

var dataTabLabels = []string{"DATABASES", "DOMAINS", "CACHING", "FILE STORAGE", "GOVERNANCE"}

// ── domain list+form types ────────────────────────────────────────────────────

type domainSubView int

const (
	domainViewList domainSubView = iota
	domainViewForm
	domainViewAttrs      // attribute list inside domain form
	domainViewAttrForm   // attribute form
	domainViewRels       // relationship list inside domain form
	domainViewRelForm    // relationship form
)

// ── file storage list+form types ─────────────────────────────────────────────

type fsView int

const (
	fsViewList fsView = iota
	fsViewForm
)

// ── governance list+form types ────────────────────────────────────────────────

type govView int

const (
	govViewList govView = iota
	govViewForm
)

// ── caching list+form types ───────────────────────────────────────────────────

type cachingView int

const (
	cachingViewList cachingView = iota
	cachingViewForm
)


// ── DataTabEditor ─────────────────────────────────────────────────────────────

// DataTabEditor is the composite DATA main-tab editor. It delegates the
// DATABASES sub-tab to DBEditor and DATA sub-tab to DataEditor (the legacy
// entity editor), and adds new DOMAINS, CACHING, and FILE STORAGE sub-tabs.
type DataTabEditor struct {
	activeTab dataTabIdx

	// Delegated sub-editors
	dbEditor   DBEditor
	dataEditor DataEditor

	// DOMAINS sub-tab
	domains        []manifest.DomainDef
	domainSubView  domainSubView
	domainIdx      int
	domainForm     []Field
	domainFormIdx  int
	attrItems      [][]Field
	attrIdx        int
	attrForm       []Field
	attrFormIdx    int
	relItems       [][]Field
	relIdx         int
	relForm        []Field
	relFormIdx     int

	// CACHING sub-tab
	cachings       []manifest.CachingConfig
	cachingSubView cachingView
	cachingIdx     int
	cachingForm    []Field
	cachingFormIdx int

	// FILE STORAGE sub-tab
	fileStorages []manifest.FileStorageDef
	fsSubView    fsView
	fsIdx        int
	fsForm       []Field
	fsFormIdx    int

	// GOVERNANCE sub-tab
	governances []manifest.DataGovernanceConfig
	govSubView  govView
	govIdx      int
	govForm     []Field
	govFormIdx  int

	// Context from backend — used to filter migration tool options.
	backendLangs []string
	// Context from backend — used to populate the service selector in FS forms.
	serviceNames []string
	// Context from infra — used to filter file storage technology options.
	cloudProvider string
	// Context from infra — used to populate the environment selector in FS forms.
	environmentNames []string
	// Context from contracts — used to populate entities multiselect in caching forms.
	availableDTOs []string

	// Shared
	internalMode Mode
	formInput    textinput.Model
	width        int

	// Dropdown state for multiselect fields in domain/caching/fs forms
	dd DropdownState

	cBuf bool
}

func newDataTabEditor() DataTabEditor {
	return DataTabEditor{
		dbEditor:   newDBEditor(),
		dataEditor: newDataEditor(),
		formInput:  newFormInput(),
	}
}

// CacheAliases returns the aliases of all database sources marked as cache
// (IsCache == true), for use by the backend rate_limit_backend dropdown.
func (dt DataTabEditor) CacheAliases() []string {
	var out []string
	for _, src := range dt.dbEditor.Sources {
		if src.IsCache && src.Alias != "" {
			out = append(out, src.Alias)
		}
	}
	return out
}

// AllDBSourceAliases returns the aliases of every configured database source
// (both regular and cache), for use by the backend health_deps multiselect.
func (dt DataTabEditor) AllDBSourceAliases() []string {
	var out []string
	for _, src := range dt.dbEditor.Sources {
		if src.Alias != "" {
			out = append(out, src.Alias)
		}
	}
	return out
}

// dbNames returns the aliases of all created databases for use as dropdown options.
func (dt DataTabEditor) dbNames() []string {
	names := make([]string, 0, len(dt.dbEditor.Sources))
	for _, src := range dt.dbEditor.Sources {
		if src.Alias != "" {
			names = append(names, src.Alias)
		}
	}
	return names
}

// domainNames returns the names of all created domains (excluding current) for use as dropdown options.
func (dt DataTabEditor) domainNames() []string {
	names := make([]string, 0, len(dt.domains))
	for _, d := range dt.domains {
		if d.Name != "" {
			names = append(names, d.Name)
		}
	}
	return names
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (dt DataTabEditor) ToManifestDataPillar() manifest.DataPillar {
	return manifest.DataPillar{
		Databases:    dt.dbEditor.Sources,
		Domains:      dt.domains,
		Entities:     dt.dataEditor.Entities,
		FileStorages: dt.fileStorages,
		Cachings:     dt.cachings,
		Governances:  dt.governances,
	}
}

// FromDataPillar populates the editor from a saved manifest DataPillar,
// reversing the ToManifestDataPillar() operation.
func (dt DataTabEditor) FromDataPillar(dp manifest.DataPillar) DataTabEditor {
	// Databases — Sources stored directly; dbForm rebuilt lazily on navigation.
	dt.dbEditor.Sources = dp.Databases

	// Entities — stored directly; colForm rebuilt lazily on navigation.
	dt.dataEditor.Entities = dp.Entities

	// Domains — stored directly.
	dt.domains = dp.Domains

	// File storages — stored directly.
	dt.fileStorages = dp.FileStorages

	// Caching strategies.
	dt.cachings = dp.Cachings

	// Governance policies.
	dt.governances = dp.Governances

	return dt
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (dt DataTabEditor) Mode() Mode {
	switch dt.activeTab {
	case dataTabDatabases:
		return dt.dbEditor.Mode()
	case dataTabDomains:
		// data editor used for old entities, but this tab manages domains
		if dt.internalMode == ModeInsert {
			return ModeInsert
		}
		return ModeNormal
	default:
		if dt.internalMode == ModeInsert {
			return ModeInsert
		}
		return ModeNormal
	}
}

func (dt DataTabEditor) HintLine() string {
	switch dt.activeTab {
	case dataTabDatabases:
		return dt.dbEditor.HintLine()
	case dataTabDomains:
		return dt.domainHintLine()
	case dataTabCaching:
		if dt.internalMode == ModeInsert {
			return StyleInsertMode.Render(" -- INSERT -- ") +
				StyleHelpDesc.Render("  Esc: normal  Tab: next field")
		}
		switch dt.cachingSubView {
		case cachingViewList:
			return hintBar("j/k", "navigate", "a", "add strategy", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		case cachingViewForm:
			return hintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "i/a", "edit", "b/Esc", "back")
		}
	case dataTabFileStorage:
		return dt.fsHintLine()
	case dataTabGovernance:
		if dt.internalMode == ModeInsert {
			return StyleInsertMode.Render(" -- INSERT -- ") +
				StyleHelpDesc.Render("  Esc: normal  Tab: next field")
		}
		switch dt.govSubView {
		case govViewList:
			return hintBar("j/k", "navigate", "a", "add policy", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		case govViewForm:
			return hintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "i/a", "edit", "b/Esc", "back")
		}
	}
	return ""
}

func (dt DataTabEditor) domainHintLine() string {
	if dt.internalMode == ModeInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch dt.domainSubView {
	case domainViewList:
		return hintBar("j/k", "navigate", "a", "add domain", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
	case domainViewForm:
		return hintBar("j/k", "navigate", "i", "edit", "A", "attributes", "R", "relationships", "b", "back")
	case domainViewAttrs:
		return hintBar("j/k", "navigate", "a", "add attr", "d", "delete", "Enter", "edit", "b", "back")
	case domainViewAttrForm:
		return hintBar("j/k", "navigate", "i", "edit text", "Enter/Space", "dropdown", "b/Esc", "back")
	case domainViewRels:
		return hintBar("j/k", "navigate", "a", "add rel", "d", "delete", "Enter", "edit", "b", "back")
	case domainViewRelForm:
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	}
	return ""
}

func (dt DataTabEditor) fsHintLine() string {
	if dt.internalMode == ModeInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch dt.fsSubView {
	case fsViewList:
		return hintBar("j/k", "navigate", "a", "add storage", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
	case fsViewForm:
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	}
	return ""
}

// ── Dropdown helpers ──────────────────────────────────────────────────────────

// activeDTFieldPtr returns a pointer to the currently active field that could be a multiselect.
func (dt *DataTabEditor) activeDTFieldPtr() *Field {
	switch dt.activeTab {
	case dataTabDomains:
		switch dt.domainSubView {
		case domainViewForm:
			if dt.domainFormIdx < len(dt.domainForm) {
				return &dt.domainForm[dt.domainFormIdx]
			}
		case domainViewAttrForm:
			if dt.attrFormIdx < len(dt.attrForm) {
				return &dt.attrForm[dt.attrFormIdx]
			}
		case domainViewRelForm:
			if dt.relFormIdx < len(dt.relForm) {
				return &dt.relForm[dt.relFormIdx]
			}
		}
	case dataTabCaching:
		if dt.cachingSubView == cachingViewForm && dt.cachingFormIdx < len(dt.cachingForm) {
			return &dt.cachingForm[dt.cachingFormIdx]
		}
	case dataTabFileStorage:
		if dt.fsSubView == fsViewForm && dt.fsFormIdx < len(dt.fsForm) {
			return &dt.fsForm[dt.fsFormIdx]
		}
	case dataTabGovernance:
		if dt.govSubView == govViewForm && dt.govFormIdx < len(dt.govForm) {
			return &dt.govForm[dt.govFormIdx]
		}
	}
	return nil
}

func (dt DataTabEditor) updateDropdown(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	f := dt.activeDTFieldPtr()
	if f == nil {
		dt.dd.Open = false
		return dt, nil
	}
	fieldKey := f.Key
	dt.dd.OptIdx = NavigateDropdown(key.String(), dt.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == KindMultiSelect {
			f.ToggleMultiSelect(dt.dd.OptIdx)
			f.DDCursor = dt.dd.OptIdx
			// When a custom option is toggled ON, close the dropdown and let the user type.
			if dt.dd.OptIdx < len(f.Options) && isCustomOption(f.Options[dt.dd.OptIdx]) && f.IsMultiSelected(dt.dd.OptIdx) {
				f.CustomText = ""
				dt.dd.Open = false
				return dt.tryEnterInsert()
			}
		} else if f.Kind == KindSelect {
			f.SelIdx = dt.dd.OptIdx
			f.Value = f.Options[dt.dd.OptIdx]
			dt.dd.Open = false
			if f.PrepareCustomEntry() {
				return dt.tryEnterInsert()
			}
		}
	case "enter":
		if f.Kind == KindMultiSelect {
			f.DDCursor = dt.dd.OptIdx
			// Enter on a custom option: toggle it on (if not already) and enter insert mode.
			if dt.dd.OptIdx < len(f.Options) && isCustomOption(f.Options[dt.dd.OptIdx]) {
				if !f.IsMultiSelected(dt.dd.OptIdx) {
					f.ToggleMultiSelect(dt.dd.OptIdx)
				}
				f.CustomText = ""
				dt.dd.Open = false
				return dt.tryEnterInsert()
			}
		} else if f.Kind == KindSelect {
			f.SelIdx = dt.dd.OptIdx
			f.Value = f.Options[dt.dd.OptIdx]
		}
		dt.dd.Open = false
		if f.Kind == KindSelect && f.PrepareCustomEntry() {
			return dt.tryEnterInsert()
		}
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = dt.dd.OptIdx
		}
		dt.dd.Open = false
	}
	if fieldKey == "compliance_frameworks" {
		dt = dt.complianceAutoUpgrade()
	}
	return dt, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (dt DataTabEditor) Update(msg tea.Msg) (DataTabEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		dt.width = wsz.Width
		dt.formInput.Width = wsz.Width - 22
		return dt, nil
	}

	key, ok := msg.(tea.KeyMsg)

	// Insert mode is handled globally for all sub-tabs except db
	if dt.internalMode == ModeInsert {
		return dt.updateInsert(msg)
	}

	// Dropdown mode for multiselect fields
	if dt.dd.Open && ok {
		return dt.updateDropdown(key)
	}

	if !ok {
		// Delegate non-key messages to db editor when on db tab
		if dt.activeTab == dataTabDatabases {
			var cmd tea.Cmd
			dt.dbEditor, cmd = dt.dbEditor.Update(msg)
			return dt, cmd
		}
		return dt, nil
	}

	// Sub-tab switching always available in normal mode
	switch key.String() {
	case "h", "left", "l", "right":
		dt.activeTab = dataTabIdx(NavigateTab(key.String(), int(dt.activeTab), len(dataTabLabels)))
		return dt, nil
	}

	// cc detection: clear field and enter insert mode (only for non-database sub-tabs;
	// DBEditor handles cc for the databases sub-tab itself)
	if dt.activeTab != dataTabDatabases && !dt.dd.Open {
		if key.String() == "c" {
			if dt.cBuf {
				dt.cBuf = false
				return dt.clearAndEnterInsert()
			}
			dt.cBuf = true
			return dt, nil
		}
		dt.cBuf = false
	}

	switch dt.activeTab {
	case dataTabDatabases:
		var cmd tea.Cmd
		dt.dbEditor, cmd = dt.dbEditor.Update(msg)
		// sync databases to data editor and governance search tech options
		dt.dataEditor.availableDbs = dt.dbEditor.Sources
		dt.updateSearchTechOptions()
		return dt, cmd
	case dataTabDomains:
		return dt.updateDomains(key)
	case dataTabCaching:
		return dt.updateCaching(key)
	case dataTabFileStorage:
		return dt.updateFileStorage(key)
	case dataTabGovernance:
		return dt.updateGovernance(key)
	}
	return dt, nil
}

func (dt DataTabEditor) resetFieldIdx() {
	// nothing needed; individual sub-editors reset their own state
}

func (dt DataTabEditor) updateInsert(msg tea.Msg) (DataTabEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			dt.saveInput()
			dt.internalMode = ModeNormal
			dt.formInput.Blur()
			// Auto-process attr_names when exiting insert mode on that field
			if dt.activeTab == dataTabDomains && dt.domainSubView == domainViewForm &&
				dt.domainFormIdx < len(dt.domainForm) && dt.domainForm[dt.domainFormIdx].Key == "attr_names" {
				dt.processAttrNames()
				dt.saveDomainAttrItemsOnly()
			}
			return dt, nil
		case "tab":
			dt.saveInput()
			dt.advanceFormField(1)
			return dt.tryEnterInsert()
		case "shift+tab":
			dt.saveInput()
			dt.advanceFormField(-1)
			return dt.tryEnterInsert()
		}
	}
	var cmd tea.Cmd
	dt.formInput, cmd = dt.formInput.Update(msg)
	return dt, cmd
}

func (dt *DataTabEditor) advanceFormField(delta int) {
	switch dt.activeTab {
	case dataTabDomains:
		switch dt.domainSubView {
		case domainViewForm:
			n := len(dt.domainForm)
			if n > 0 {
				dt.domainFormIdx = (dt.domainFormIdx + delta + n) % n
			}
		case domainViewAttrForm:
			n := len(dt.attrForm)
			if n > 0 {
				dt.attrFormIdx = (dt.attrFormIdx + delta + n) % n
			}
		case domainViewRelForm:
			n := len(dt.relForm)
			if n > 0 {
				dt.relFormIdx = (dt.relFormIdx + delta + n) % n
			}
		}
	case dataTabCaching:
		n := len(dt.cachingForm)
		if n > 0 {
			dt.cachingFormIdx = (dt.cachingFormIdx + delta + n) % n
		}
	case dataTabFileStorage:
		n := len(dt.fsForm)
		if n > 0 {
			dt.fsFormIdx = (dt.fsFormIdx + delta + n) % n
		}
	case dataTabGovernance:
		n := len(dt.govForm)
		if n > 0 {
			dt.govFormIdx = (dt.govFormIdx + delta + n) % n
		}
	}
}

func (dt *DataTabEditor) saveInput() {
	val := dt.formInput.Value()
	switch dt.activeTab {
	case dataTabDomains:
		switch dt.domainSubView {
		case domainViewForm:
			if dt.domainFormIdx < len(dt.domainForm) && dt.domainForm[dt.domainFormIdx].CanEditAsText() {
				dt.domainForm[dt.domainFormIdx].SaveTextInput(val)
			}
		case domainViewAttrForm:
			if dt.attrFormIdx < len(dt.attrForm) && dt.attrForm[dt.attrFormIdx].CanEditAsText() {
				dt.attrForm[dt.attrFormIdx].SaveTextInput(val)
			}
		case domainViewRelForm:
			if dt.relFormIdx < len(dt.relForm) && dt.relForm[dt.relFormIdx].CanEditAsText() {
				dt.relForm[dt.relFormIdx].SaveTextInput(val)
			}
		}
	case dataTabCaching:
		if dt.cachingFormIdx < len(dt.cachingForm) && dt.cachingForm[dt.cachingFormIdx].CanEditAsText() {
			dt.cachingForm[dt.cachingFormIdx].SaveTextInput(val)
		}
	case dataTabFileStorage:
		if dt.fsFormIdx < len(dt.fsForm) && dt.fsForm[dt.fsFormIdx].CanEditAsText() {
			dt.fsForm[dt.fsFormIdx].SaveTextInput(val)
		}
	case dataTabGovernance:
		if dt.govFormIdx < len(dt.govForm) && dt.govForm[dt.govFormIdx].CanEditAsText() {
			dt.govForm[dt.govFormIdx].SaveTextInput(val)
		}
	}
}

func (dt DataTabEditor) tryEnterInsert() (DataTabEditor, tea.Cmd) {
	n := 0
	switch dt.activeTab {
	case dataTabDomains:
		switch dt.domainSubView {
		case domainViewForm:
			n = len(dt.domainForm)
		case domainViewAttrForm:
			n = len(dt.attrForm)
		case domainViewRelForm:
			n = len(dt.relForm)
		}
	case dataTabCaching:
		n = len(dt.cachingForm)
	case dataTabFileStorage:
		n = len(dt.fsForm)
	case dataTabGovernance:
		n = len(dt.govForm)
	}
	for range n {
		var f *Field
		switch dt.activeTab {
		case dataTabDomains:
			switch dt.domainSubView {
			case domainViewForm:
				if dt.domainFormIdx < len(dt.domainForm) {
					f = &dt.domainForm[dt.domainFormIdx]
				}
			case domainViewAttrForm:
				if dt.attrFormIdx < len(dt.attrForm) {
					f = &dt.attrForm[dt.attrFormIdx]
				}
			case domainViewRelForm:
				if dt.relFormIdx < len(dt.relForm) {
					f = &dt.relForm[dt.relFormIdx]
				}
			}
		case dataTabCaching:
			if dt.cachingFormIdx < len(dt.cachingForm) {
				f = &dt.cachingForm[dt.cachingFormIdx]
			}
		case dataTabFileStorage:
			if dt.fsFormIdx < len(dt.fsForm) {
				f = &dt.fsForm[dt.fsFormIdx]
			}
		case dataTabGovernance:
			if dt.govFormIdx < len(dt.govForm) {
				f = &dt.govForm[dt.govFormIdx]
			}
		}
		if f == nil {
			break
		}
		if f.CanEditAsText() {
			dt.internalMode = ModeInsert
			dt.formInput.SetValue(f.TextInputValue())
			dt.formInput.Width = dt.width - 22
			dt.formInput.CursorEnd()
			return dt, dt.formInput.Focus()
		}
		dt.advanceFormField(1)
	}
	return dt, nil
}

func (dt DataTabEditor) clearAndEnterInsert() (DataTabEditor, tea.Cmd) {
	dt, cmd := dt.tryEnterInsert()
	if dt.internalMode == ModeInsert {
		dt.formInput.SetValue("")
	}
	return dt, cmd
}

func (dt DataTabEditor) View(w, h int) string {
	dt.width = w
	dt.formInput.Width = w - 22
	var lines []string

	// Header + sub-tab bar
	lines = append(lines,
		StyleSectionDesc.Render("  # Data — databases, domains, caching, and file storage"),
		"",
		renderSubTabBar(dataTabLabels, int(dt.activeTab), w),
		"",
	)

	headerLines := len(lines)
	contentH := h - headerLines
	if contentH < 2 {
		contentH = 2
	}

	var contentLines []string
	switch dt.activeTab {
	case dataTabDatabases:
		raw := dt.dbEditor.View(w, contentH)
		// dbEditor.View already returns a \n-terminated string with tilde padding
		return strings.Join(lines, "\n") + "\n" + raw
	case dataTabDomains:
		contentLines = dt.viewDomains(w)
		if dt.domainSubView == domainViewList {
			contentLines = appendViewport(contentLines, 2, dt.domainIdx, contentH)
		}
	case dataTabCaching:
		contentLines = dt.viewCaching(w)
		if dt.cachingSubView == cachingViewList {
			contentLines = appendViewport(contentLines, 2, dt.cachingIdx, contentH)
		}
	case dataTabFileStorage:
		contentLines = dt.viewFileStorage(w)
	case dataTabGovernance:
		contentLines = dt.viewGovernance(w)
		if dt.govSubView == govViewList {
			contentLines = appendViewport(contentLines, 2, dt.govIdx, contentH)
		}
	}

	lines = append(lines, contentLines...)
	return fillTildes(lines, h)
}

