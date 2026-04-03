package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── sub-tabs ──────────────────────────────────────────────────────────────────

type contractsTabIdx int

const (
	contractsTabDTOs contractsTabIdx = iota
	contractsTabEndpoints
	contractsTabVersioning
	contractsTabExternal
)

var contractsTabLabels = []string{"DTOs", "ENDPOINTS", "API VERSIONING", "EXTERNAL APIS"}


// ── list-item sub-view ────────────────────────────────────────────────────────

type ceSubView int

const (
	ceViewList     ceSubView = iota
	ceViewForm               // top-level form
	ceViewSubList            // sub-list (e.g., DTO fields, endpoint error responses)
	ceViewSubForm            // sub-item form
)


// ── ContractsEditor ───────────────────────────────────────────────────────────

// ContractsEditor manages the CONTRACTS main-tab: DTOs, Endpoints, API Versioning.
type ContractsEditor struct {
	activeTab contractsTabIdx

	// DTOs
	dtos       []manifest.DTODef
	dtoSubView ceSubView
	dtoIdx     int
	dtoForm    []Field
	dtoFormIdx int
	// DTO fields sub-list
	dtoFieldItems   [][]Field
	dtoFieldIdx     int
	dtoFieldForm    []Field
	dtoFieldFormIdx int

	// Endpoints
	endpoints []manifest.EndpointDef
	epSubView ceSubView
	epIdx     int
	epForm    []Field
	epFormIdx int

	// API Versioning (simple field form)
	versioningFields  []Field
	verFormIdx        int
	versioningEnabled bool

	// External APIs
	externalAPIs []manifest.ExternalAPIDef
	extSubView   ceSubView
	extIdx       int
	extForm      []Field
	extFormIdx   int

	// Cross-editor reference data (set by model.go before each Update)
	availableDomains     []string               // from DataTabEditor.domainNames()
	availableDomainDefs  []manifest.DomainDef   // from DataTabEditor.domains
	availableServices    []string               // from BackendEditor.ServiceNames()
	availableServiceDefs []manifest.ServiceDef  // from BackendEditor.ServiceDefs()
	availableAuthRoles   []string               // from BackendEditor.AuthRoleOptions()

	// Dropdown state for KindSelect/KindMultiSelect fields
	dd DropdownState

	// Shared
	internalMode Mode
	formInput    textinput.Model
	width        int
}

func newContractsEditor() ContractsEditor {
	return ContractsEditor{
		versioningFields: defaultVersioningFields(),
		formInput:        newFormInput(),
	}
}

// SetDomains updates the list of available domain names for cross-referencing.
func (ce *ContractsEditor) SetDomains(domains []string) {
	ce.availableDomains = domains
}

// SetServices updates the list of available service names for cross-referencing.
func (ce *ContractsEditor) SetServices(services []string) {
	ce.availableServices = services
}

// SetServiceDefs updates full service definitions for technology-based protocol filtering.
func (ce *ContractsEditor) SetServiceDefs(defs []manifest.ServiceDef) {
	ce.availableServiceDefs = defs
}

// SetAuthRoles updates the auth role options used in endpoint forms.
func (ce *ContractsEditor) SetAuthRoles(roles []string) {
	ce.availableAuthRoles = roles
}

// protocolsForService returns the protocol options valid for the named service
// based on its registered technologies. Returns nil when no filter applies.
// SetDomainDefs updates the full domain definitions for attribute injection.
func (ce *ContractsEditor) SetDomainDefs(domains []manifest.DomainDef) {
	ce.availableDomainDefs = domains
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (ce ContractsEditor) ToManifestContractsPillar() manifest.ContractsPillar {
	p := manifest.ContractsPillar{
		DTOs:         ce.dtos,
		Endpoints:    ce.endpoints,
		ExternalAPIs: ce.externalAPIs,
	}
	if ce.versioningEnabled {
		p.Versioning = manifest.APIVersioning{
			Strategy:          fieldGet(ce.versioningFields, "strategy"),
			CurrentVersion:    fieldGet(ce.versioningFields, "current_version"),
			DeprecationPolicy: fieldGet(ce.versioningFields, "deprecation"),
		}
	}
	return p
}

// FromContractsPillar populates the editor from a saved manifest ContractsPillar,
// reversing the ToManifestContractsPillar() operation.
func (ce ContractsEditor) FromContractsPillar(cp manifest.ContractsPillar) ContractsEditor {
	// Collections stored directly; per-item forms rebuilt lazily on navigation.
	ce.dtos = cp.DTOs
	ce.endpoints = cp.Endpoints
	ce.externalAPIs = cp.ExternalAPIs

	// Versioning fields.
	if cp.Versioning.Strategy != "" {
		ce.versioningEnabled = true
		ce.versioningFields = setFieldValue(ce.versioningFields, "strategy", cp.Versioning.Strategy)
		ce.versioningFields = setFieldValue(ce.versioningFields, "current_version", cp.Versioning.CurrentVersion)
		ce.versioningFields = setFieldValue(ce.versioningFields, "deprecation", cp.Versioning.DeprecationPolicy)
	}

	return ce
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (ce ContractsEditor) Mode() Mode {
	if ce.internalMode == ModeInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (ce ContractsEditor) HintLine() string {
	if ce.internalMode == ModeInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case ceViewList:
			return hintBar("j/k", "navigate", "a", "add DTO", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		case ceViewForm:
			return hintBar("j/k", "navigate", "i/Enter", "edit", "F", "fields", "b/Esc", "back")
		case ceViewSubList:
			return hintBar("j/k", "navigate", "a", "add field", "d", "delete", "Enter", "edit", "b", "back")
		case ceViewSubForm:
			return hintBar("j/k", "navigate", "i", "edit text", "Enter/Space", "dropdown", "b/Esc", "back")
		}
	case contractsTabEndpoints:
		switch ce.epSubView {
		case ceViewList:
			return hintBar("j/k", "navigate", "a", "add endpoint", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		case ceViewForm:
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
	case contractsTabVersioning:
		if !ce.versioningEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "a/i/Enter", "edit", "Space", "cycle", "H", "cycle back", "D", "delete config", "h/l", "sub-tab")
	case contractsTabExternal:
		switch ce.extSubView {
		case ceViewList:
			return hintBar("j/k", "navigate", "a", "add provider", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		case ceViewForm:
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
	}
	return ""
}

// dtoNames returns the names of all created DTOs for use as dropdown options.
func (ce ContractsEditor) dtoNames() []string {
	names := make([]string, 0, len(ce.dtos))
	for _, d := range ce.dtos {
		if d.Name != "" {
			names = append(names, d.Name)
		}
	}
	return names
}

// dtoNamesForProtocol returns DTO names whose protocol matches the given
// external API protocol. DTOs with no protocol set are included for all
// protocols (backwards compatibility with manifests saved before this feature).
func (ce ContractsEditor) dtoNamesForProtocol(protocol string) []string {
	names := make([]string, 0, len(ce.dtos))
	for _, d := range ce.dtos {
		if d.Name == "" {
			continue
		}
		if d.Protocol == "" || d.Protocol == protocol {
			names = append(names, d.Name)
		}
	}
	return names
}

// refreshExtDTOOptions updates the request_dto and response_dto option lists
// in the ext form to match the currently selected protocol, preserving the
// current selection when it is still valid.
func (ce *ContractsEditor) refreshExtDTOOptions() {
	proto := fieldGet(ce.extForm, "protocol")
	opts := ce.dtoNamesForProtocol(proto)
	placeholder := placeholderFor(opts, "(no matching DTOs)")
	for i := range ce.extForm {
		key := ce.extForm[i].Key
		if key != "request_dto" && key != "response_dto" {
			continue
		}
		f := &ce.extForm[i]
		prev := f.Value
		f.Options = opts
		// Try to preserve current selection.
		found := false
		for j, o := range opts {
			if o == prev {
				f.SelIdx = j
				f.Value = o
				found = true
				break
			}
		}
		if !found {
			f.SelIdx = 0
			if len(opts) > 0 {
				f.Value = opts[0]
			} else {
				f.Value = placeholder
			}
		}
	}
}

// activeCEFieldPtr returns a pointer to the currently focused field that supports dropdown.
func (ce *ContractsEditor) activeCEFieldPtr() *Field {
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case ceViewForm:
			visible := ce.visibleDTOFields()
			if ce.dtoFormIdx < len(visible) {
				return ce.dtoFormFieldByKey(visible[ce.dtoFormIdx].Key)
			}
		case ceViewSubForm:
			visible := ce.visibleDTOFieldFormFields()
			if ce.dtoFieldFormIdx < len(visible) {
				return ce.dtoFieldFormFieldByKey(visible[ce.dtoFieldFormIdx].Key)
			}
		}
	case contractsTabEndpoints:
		if ce.epSubView == ceViewForm {
			visible := ce.visibleEPFields()
			if ce.epFormIdx < len(visible) {
				return ce.epFieldByKey(visible[ce.epFormIdx].Key)
			}
		}
	case contractsTabVersioning:
		if ce.verFormIdx < len(ce.versioningFields) {
			return &ce.versioningFields[ce.verFormIdx]
		}
	case contractsTabExternal:
		visible := ce.visibleExtFormFields()
		if ce.extSubView == ceViewForm && ce.extFormIdx < len(visible) {
			return ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
		}
	}
	return nil
}

func (ce ContractsEditor) updateDropdown(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	f := ce.activeCEFieldPtr()
	if f == nil {
		ce.dd.Open = false
		return ce, nil
	}
	ce.dd.OptIdx = NavigateDropdown(key.String(), ce.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == KindMultiSelect {
			f.ToggleMultiSelect(ce.dd.OptIdx)
			f.DDCursor = ce.dd.OptIdx
		} else if f.Kind == KindSelect {
			f.SelIdx = ce.dd.OptIdx
			if ce.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[ce.dd.OptIdx]
			}
			ce.dd.Open = false
			if f.PrepareCustomEntry() {
				ce.updateEPDependentFields()
				return ce.tryEnterInsert()
			}
		}
	case "enter":
		if f.Kind == KindMultiSelect {
			f.DDCursor = ce.dd.OptIdx
		} else if f.Kind == KindSelect {
			f.SelIdx = ce.dd.OptIdx
			if ce.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[ce.dd.OptIdx]
			}
		}
		ce.dd.Open = false
		if f.Kind == KindSelect && f.PrepareCustomEntry() {
			ce.updateEPDependentFields()
			return ce.tryEnterInsert()
		}
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = ce.dd.OptIdx
		}
		ce.dd.Open = false
	}
	// After any dropdown interaction, refresh dependent fields for DTO, EP, and ext forms.
	ce.updateDTODependentFields()
	ce.updateEPDependentFields()
	ce.updateExtDependentFields()
	// Auto-save the active form so changes persist without requiring b/esc.
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case ceViewForm:
			ce.saveDTOForm()
		case ceViewSubForm:
			if ce.dtoFieldIdx < len(ce.dtoFieldItems) {
				ce.dtoFieldItems[ce.dtoFieldIdx] = copyFields(ce.dtoFieldForm)
			}
		}
	case contractsTabEndpoints:
		if ce.epSubView == ceViewForm {
			ce.saveEPForm()
		}
	case contractsTabExternal:
		if ce.extSubView == ceViewForm {
			ce.saveExtForm()
		}
	}
	return ce, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (ce ContractsEditor) Update(msg tea.Msg) (ContractsEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		ce.width = wsz.Width
		ce.formInput.Width = wsz.Width - 22
		return ce, nil
	}
	if ce.internalMode == ModeInsert {
		return ce.updateInsert(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return ce, nil
	}

	// Handle dropdown if open
	if ce.dd.Open && ok {
		return ce.updateDropdown(key)
	}

	// Sub-tab switching always available in normal mode
	switch key.String() {
	case "h", "left", "l", "right":
		ce.activeTab = contractsTabIdx(NavigateTab(key.String(), int(ce.activeTab), len(contractsTabLabels)))
		return ce, nil
	}

	switch ce.activeTab {
	case contractsTabDTOs:
		return ce.updateDTOs(key)
	case contractsTabEndpoints:
		return ce.updateEndpoints(key)
	case contractsTabVersioning:
		return ce.updateVersioning(key)
	case contractsTabExternal:
		return ce.updateExternal(key)
	}
	return ce, nil
}

func (ce ContractsEditor) updateInsert(msg tea.Msg) (ContractsEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			ce.saveInput()
			ce.internalMode = ModeNormal
			ce.formInput.Blur()
			return ce, nil
		case "tab":
			ce.saveInput()
			ce.advanceField(1)
			return ce.tryEnterInsert()
		case "shift+tab":
			ce.saveInput()
			ce.advanceField(-1)
			return ce.tryEnterInsert()
		}
	}
	var cmd tea.Cmd
	ce.formInput, cmd = ce.formInput.Update(msg)
	return ce, cmd
}

func (ce *ContractsEditor) advanceField(delta int) {
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case ceViewForm:
			n := len(ce.visibleDTOFields())
			if n > 0 {
				ce.dtoFormIdx = (ce.dtoFormIdx + delta + n) % n
			}
		case ceViewSubForm:
			n := len(ce.visibleDTOFieldFormFields())
			if n > 0 {
				ce.dtoFieldFormIdx = (ce.dtoFieldFormIdx + delta + n) % n
			}
		}
	case contractsTabEndpoints:
		if ce.epSubView == ceViewForm {
			n := len(ce.visibleEPFields())
			if n > 0 {
				ce.epFormIdx = (ce.epFormIdx + delta + n) % n
			}
		}
	case contractsTabVersioning:
		n := len(ce.versioningFields)
		if n > 0 {
			ce.verFormIdx = (ce.verFormIdx + delta + n) % n
		}
	case contractsTabExternal:
		if ce.extSubView == ceViewForm {
			n := len(ce.visibleExtFormFields())
			if n > 0 {
				ce.extFormIdx = (ce.extFormIdx + delta + n) % n
			}
		}
	}
}

func (ce *ContractsEditor) saveInput() {
	val := ce.formInput.Value()
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case ceViewForm:
			if ce.dtoFormIdx < len(ce.dtoForm) && ce.dtoForm[ce.dtoFormIdx].CanEditAsText() {
				ce.dtoForm[ce.dtoFormIdx].SaveTextInput(val)
			}
		case ceViewSubForm:
			if ce.dtoFieldFormIdx < len(ce.dtoFieldForm) && ce.dtoFieldForm[ce.dtoFieldFormIdx].CanEditAsText() {
				ce.dtoFieldForm[ce.dtoFieldFormIdx].SaveTextInput(val)
			}
		}
	case contractsTabEndpoints:
		visible := ce.visibleEPFields()
		if ce.epSubView == ceViewForm && ce.epFormIdx < len(visible) {
			f := ce.epFieldByKey(visible[ce.epFormIdx].Key)
			if f != nil && f.CanEditAsText() {
				f.SaveTextInput(val)
			}
		}
	case contractsTabVersioning:
		if ce.verFormIdx < len(ce.versioningFields) && ce.versioningFields[ce.verFormIdx].CanEditAsText() {
			ce.versioningFields[ce.verFormIdx].SaveTextInput(val)
		}
	case contractsTabExternal:
		visible := ce.visibleExtFormFields()
		if ce.extSubView == ceViewForm && ce.extFormIdx < len(visible) {
			f := ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
			if f != nil && f.CanEditAsText() {
				f.SaveTextInput(val)
			}
		}
	}
}

func (ce ContractsEditor) tryEnterInsert() (ContractsEditor, tea.Cmd) {
	n := 0
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case ceViewForm:
			n = len(ce.dtoForm)
		case ceViewSubForm:
			n = len(ce.visibleDTOFieldFormFields())
		}
	case contractsTabEndpoints:
		if ce.epSubView == ceViewForm {
			n = len(ce.visibleEPFields())
		}
	case contractsTabVersioning:
		n = len(ce.versioningFields)
	case contractsTabExternal:
		if ce.extSubView == ceViewForm {
			n = len(ce.visibleExtFormFields())
		}
	}
	for range n {
		var f *Field
		switch ce.activeTab {
		case contractsTabDTOs:
			switch ce.dtoSubView {
			case ceViewForm:
				if ce.dtoFormIdx < len(ce.dtoForm) {
					f = &ce.dtoForm[ce.dtoFormIdx]
				}
			case ceViewSubForm:
				if ce.dtoFieldFormIdx < len(ce.dtoFieldForm) {
					f = &ce.dtoFieldForm[ce.dtoFieldFormIdx]
				}
			}
		case contractsTabEndpoints:
			visible := ce.visibleEPFields()
			if ce.epSubView == ceViewForm && ce.epFormIdx < len(visible) {
				f = ce.epFieldByKey(visible[ce.epFormIdx].Key)
			}
		case contractsTabVersioning:
			if ce.verFormIdx < len(ce.versioningFields) {
				f = &ce.versioningFields[ce.verFormIdx]
			}
		case contractsTabExternal:
			visible := ce.visibleExtFormFields()
			if ce.extSubView == ceViewForm && ce.extFormIdx < len(visible) {
				f = ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
			}
		}
		if f == nil {
			break
		}
		if f.CanEditAsText() {
			ce.internalMode = ModeInsert
			ce.formInput.SetValue(f.TextInputValue())
			ce.formInput.Width = ce.width - 22
			ce.formInput.CursorEnd()
			return ce, ce.formInput.Focus()
		}
		ce.advanceField(1)
	}
	return ce, nil
}

