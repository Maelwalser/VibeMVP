package contracts

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
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

// ── ContractsEditor ───────────────────────────────────────────────────────────

// ContractsEditor manages the CONTRACTS main-tab: DTOs, Endpoints, API Versioning.
type ContractsEditor struct {
	activeTab contractsTabIdx

	// DTOs
	dtos       []manifest.DTODef
	dtoSubView core.SubView
	dtoIdx     int
	dtoForm    []core.Field
	dtoFormIdx int
	// DTO fields sub-list
	dtoFieldItems   [][]core.Field
	dtoFieldIdx     int
	dtoFieldForm    []core.Field
	dtoFieldFormIdx int

	// Endpoints
	endpoints []manifest.EndpointDef
	epSubView core.SubView
	epIdx     int
	epForm    []core.Field
	epFormIdx int

	// API Versioning (simple field form)
	versioningFields  []core.Field
	verFormIdx        int
	versioningEnabled bool

	// External APIs
	externalAPIs []manifest.ExternalAPIDef
	extSubView   core.SubView
	extIdx       int
	extForm      []core.Field
	extFormIdx   int
	// External API interactions sub-list
	extIntIdx     int
	extIntForm    []core.Field
	extIntFormIdx int

	// Cross-editor reference data (set by model.go before each Update)
	availableDomains     []string              // from DataTabEditor.domainNames()
	availableDomainDefs  []manifest.DomainDef  // from DataTabEditor.domains
	availableServices    []string              // from BackendEditor.ServiceNames()
	availableServiceDefs []manifest.ServiceDef // from BackendEditor.ServiceDefs()
	availableAuthRoles   []string              // from BackendEditor.AuthRoleOptions()
	wafRateLimitStrategy string                // from BackendEditor.WAFRateLimitStrategy()

	// Dropdown state for KindSelect/KindMultiSelect fields
	dd core.DropdownState

	// Shared
	internalMode core.Mode
	formInput    textinput.Model
	width        int

	cBuf bool

	// Per-subtab undo stacks (structural add/delete only)
	dtosUndo core.UndoStack[[]manifest.DTODef]
	epsUndo  core.UndoStack[[]manifest.EndpointDef]
	extUndo  core.UndoStack[[]manifest.ExternalAPIDef]
}

func NewEditor() ContractsEditor {
	return ContractsEditor{
		versioningFields: defaultVersioningFields(),
		formInput:        core.NewFormInput(),
	}
}

// SetDomains updates the list of available domain names for cross-referencing.
func (ce *ContractsEditor) SetDomains(domains []string) {
	ce.availableDomains = domains
}

// SetServices updates the list of available service names for cross-referencing
// and clears stale service references on committed endpoints.
func (ce *ContractsEditor) SetServices(services []string) {
	ce.availableServices = services
	ce.ClearStaleServiceRefs(services)
}

// SetServiceDefs updates full service definitions for technology-based protocol filtering.
func (ce *ContractsEditor) SetServiceDefs(defs []manifest.ServiceDef) {
	ce.availableServiceDefs = defs
}

// SetAuthRoles updates the auth role options used in endpoint forms.
func (ce *ContractsEditor) SetAuthRoles(roles []string) {
	ce.availableAuthRoles = roles
}

// SetWAFRateLimitStrategy updates the backend WAF rate-limit strategy so that
// new endpoint forms can default rate_limit appropriately.
func (ce *ContractsEditor) SetWAFRateLimitStrategy(strategy string) {
	ce.wafRateLimitStrategy = strategy
}

// ActiveDocProtocols returns the distinct endpoint protocols present in the
// current endpoints list, in a stable order. Used by CrossCutEditor to build
// per-protocol documentation format fields. Falls back to ["REST"] when empty.
func (ce ContractsEditor) ActiveDocProtocols() []string {
	order := []string{"REST", "GraphQL", "gRPC", "WebSocket message", "Event"}
	seen := make(map[string]bool)
	for _, ep := range ce.endpoints {
		seen[ep.Protocol] = true
	}
	var result []string
	for _, p := range order {
		if seen[p] {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{"REST"}
	}
	return result
}

// protocolsForService returns the protocol options valid for the named service
// based on its registered technologies. Returns nil when no filter applies.
// SetDomainDefs updates the full domain definitions for attribute injection.
func (ce *ContractsEditor) SetDomainDefs(domains []manifest.DomainDef) {
	ce.availableDomainDefs = domains
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (ce ContractsEditor) ToManifestContractsPillar() manifest.ContractsPillar {
	// Clean protocol-specific fields from endpoints so stale values from a
	// previous protocol selection don't leak into the manifest.
	endpoints := make([]manifest.EndpointDef, len(ce.endpoints))
	for i, ep := range ce.endpoints {
		switch ep.Protocol {
		case "REST":
			ep.GraphQLOpType = ""
			ep.GRPCStreamType = ""
			ep.WSDirection = ""
		case "GraphQL":
			ep.HTTPMethod = ""
			ep.GRPCStreamType = ""
			ep.WSDirection = ""
		case "gRPC":
			ep.HTTPMethod = ""
			ep.GraphQLOpType = ""
			ep.WSDirection = ""
		case "WebSocket message":
			ep.HTTPMethod = ""
			ep.GraphQLOpType = ""
			ep.GRPCStreamType = ""
		case "Event":
			ep.HTTPMethod = ""
			ep.GraphQLOpType = ""
			ep.GRPCStreamType = ""
			ep.WSDirection = ""
			ep.PaginationStrategy = ""
		}
		endpoints[i] = ep
	}

	// Clean protocol-specific fields from DTOs.
	dtos := make([]manifest.DTODef, len(ce.dtos))
	for i, dto := range ce.dtos {
		if dto.Protocol != "Protobuf" {
			dto.ProtoPackage = ""
			dto.ProtoSyntax = ""
			dto.ProtoOptions = ""
		}
		if dto.Protocol != "Avro" {
			dto.AvroNamespace = ""
			dto.SchemaRegistry = ""
		}
		if dto.Protocol != "Thrift" {
			dto.ThriftNamespace = ""
			dto.ThriftLanguage = ""
		}
		if dto.Protocol != "FlatBuffers" && dto.Protocol != "Cap'n Proto" {
			dto.Namespace = ""
		}
		// Clean protocol-specific fields from DTO fields.
		if len(dto.Fields) > 0 {
			fields := make([]manifest.DTOField, len(dto.Fields))
			for j, f := range dto.Fields {
				if dto.Protocol != "Protobuf" {
					f.FieldNumber = ""
					f.ProtoModifier = ""
					f.JsonName = ""
				}
				if dto.Protocol != "Thrift" && dto.Protocol != "Cap'n Proto" {
					f.FieldID = ""
				}
				if dto.Protocol != "Thrift" {
					f.ThriftModifier = ""
				}
				if dto.Protocol != "FlatBuffers" {
					f.Deprecated = false
				}
				fields[j] = f
			}
			dto.Fields = fields
		}
		dtos[i] = dto
	}

	// Clean protocol-specific fields from external APIs and their interactions.
	externalAPIs := make([]manifest.ExternalAPIDef, len(ce.externalAPIs))
	for i, api := range ce.externalAPIs {
		proto := api.Protocol
		if proto == "" {
			proto = "REST"
		}
		// base_url shown for all except Webhook
		if proto == "Webhook" {
			api.BaseURL = ""
		}
		// rate_limit only for REST/GraphQL
		if proto != "REST" && proto != "GraphQL" {
			api.RateLimit = ""
		}
		// webhook_endpoint only for REST/Webhook
		if proto != "REST" && proto != "Webhook" {
			api.WebhookEndpoint = ""
		}
		// gRPC-only
		if proto != "gRPC" {
			api.TLSMode = ""
		}
		// WebSocket-only
		if proto != "WebSocket" {
			api.WSSubprotocol = ""
			api.MessageFormat = ""
		}
		// Webhook-only
		if proto != "Webhook" {
			api.HMACHeader = ""
			api.RetryPolicy = ""
		}
		// SOAP-only
		if proto != "SOAP" {
			api.SOAPVersion = ""
		}
		// Clean protocol-specific fields from interactions.
		if len(api.Interactions) > 0 {
			interactions := make([]manifest.ExternalAPIInteraction, len(api.Interactions))
			for j, it := range api.Interactions {
				if proto != "REST" {
					it.HTTPMethod = ""
				}
				if proto != "GraphQL" {
					it.GQLOperation = ""
				}
				if proto != "gRPC" {
					it.GRPCStreamType = ""
				}
				if proto != "WebSocket" {
					it.WSDirection = ""
				}
				if proto == "Webhook" {
					it.Path = ""
				}
				interactions[j] = it
			}
			api.Interactions = interactions
		}
		externalAPIs[i] = api
	}

	p := manifest.ContractsPillar{
		DTOs:         dtos,
		Endpoints:    endpoints,
		ExternalAPIs: externalAPIs,
	}
	if ce.versioningEnabled {
		strategies := make(map[string]string)
		for _, proto := range []string{"REST", "GraphQL", "gRPC"} {
			key := versioningStrategyFieldKey(proto)
			if v := core.FieldGet(ce.versioningFields, key); v != "" {
				strategies[proto] = v
			}
		}
		p.Versioning = &manifest.APIVersioning{
			PerProtocolStrategies: strategies,
			CurrentVersion:        core.FieldGet(ce.versioningFields, "current_version"),
			DeprecationPolicy:     core.FieldGet(ce.versioningFields, "deprecation"),
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
	if cp.Versioning != nil && (len(cp.Versioning.PerProtocolStrategies) > 0 || cp.Versioning.CurrentVersion != "") {
		ce.versioningEnabled = true
		// Rebuild fields based on stored protocols.
		ce.rebuildVersioningFields()
		for proto, strategy := range cp.Versioning.PerProtocolStrategies {
			ce.versioningFields = core.SetFieldValue(ce.versioningFields, versioningStrategyFieldKey(proto), strategy)
		}
		ce.versioningFields = core.SetFieldValue(ce.versioningFields, "current_version", cp.Versioning.CurrentVersion)
		ce.versioningFields = core.SetFieldValue(ce.versioningFields, "deprecation", cp.Versioning.DeprecationPolicy)
	}

	return ce
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (ce ContractsEditor) Mode() core.Mode {
	if ce.internalMode == core.ModeInsert {
		return core.ModeInsert
	}
	return core.ModeNormal
}

func (ce ContractsEditor) HintLine() string {
	if ce.internalMode == core.ModeInsert {
		return core.StyleInsertMode.Render(" -- INSERT -- ") +
			core.StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case core.ViewList:
			return core.HintBar("j/k", "navigate", "a", "add DTO", "d", "delete", "u", "undo", "Enter", "edit", "h/l", "sub-tab")
		case core.ViewForm:
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "A", "fields", "b/Esc", "back")
		case core.ViewSubList:
			return core.HintBar("j/k", "navigate", "a", "add field", "d", "delete", "Enter", "edit", "b", "back")
		case core.ViewSubForm:
			return core.HintBar("j/k", "navigate", "i", "edit text", "Enter/Space", "dropdown", "b/Esc", "back")
		}
	case contractsTabEndpoints:
		switch ce.epSubView {
		case core.ViewList:
			return core.HintBar("j/k", "navigate", "a", "add endpoint", "d", "delete", "u", "undo", "Enter", "edit", "h/l", "sub-tab")
		case core.ViewForm:
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
	case contractsTabVersioning:
		if !ce.versioningEnabled {
			return core.HintBar("a", "configure", "h/l", "sub-tab")
		}
		return core.HintBar("j/k", "navigate", "a/i/Enter", "edit", "Space", "cycle", "H", "cycle back", "D", "delete config", "h/l", "sub-tab")
	case contractsTabExternal:
		switch ce.extSubView {
		case core.ViewList:
			return core.HintBar("j/k", "navigate", "a", "add provider", "d", "delete", "u", "undo", "Enter", "edit", "h/l", "sub-tab")
		case core.ViewForm:
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "I", "interactions", "b/Esc", "back")
		case core.ViewSubList:
			return core.HintBar("j/k", "navigate", "a", "add", "d", "delete", "Enter", "edit", "b/Esc", "back")
		case core.ViewSubForm:
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
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

// activeCEFieldPtr returns a pointer to the currently focused field that supports dropdown.
func (ce *ContractsEditor) activeCEFieldPtr() *core.Field {
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case core.ViewForm:
			visible := ce.visibleDTOFields()
			if ce.dtoFormIdx < len(visible) {
				return ce.dtoFormFieldByKey(visible[ce.dtoFormIdx].Key)
			}
		case core.ViewSubForm:
			visible := ce.visibleDTOFieldFormFields()
			if ce.dtoFieldFormIdx < len(visible) {
				return ce.dtoFieldFormFieldByKey(visible[ce.dtoFieldFormIdx].Key)
			}
		}
	case contractsTabEndpoints:
		if ce.epSubView == core.ViewForm {
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
		switch ce.extSubView {
		case core.ViewForm:
			visible := ce.visibleExtFormFields()
			if ce.extFormIdx < len(visible) {
				return ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
			}
		case core.ViewSubForm:
			visible := ce.visibleExtIntFormFields()
			if ce.extIntFormIdx < len(visible) {
				return ce.extIntFormFieldByKey(visible[ce.extIntFormIdx].Key)
			}
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
	ce.dd.OptIdx = core.NavigateDropdown(key.String(), ce.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == core.KindMultiSelect {
			f.ToggleMultiSelect(ce.dd.OptIdx)
			f.DDCursor = ce.dd.OptIdx
		} else if f.Kind == core.KindSelect {
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
		if f.Kind == core.KindMultiSelect {
			f.DDCursor = ce.dd.OptIdx
		} else if f.Kind == core.KindSelect {
			f.SelIdx = ce.dd.OptIdx
			if ce.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[ce.dd.OptIdx]
			}
		}
		ce.dd.Open = false
		if f.Kind == core.KindSelect && f.PrepareCustomEntry() {
			ce.updateEPDependentFields()
			return ce.tryEnterInsert()
		}
	case "esc", "b":
		if f.Kind == core.KindMultiSelect {
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
		case core.ViewForm:
			ce.saveDTOForm()
		case core.ViewSubForm:
			if ce.dtoFieldIdx < len(ce.dtoFieldItems) {
				ce.dtoFieldItems[ce.dtoFieldIdx] = core.CopyFields(ce.dtoFieldForm)
			}
		}
	case contractsTabEndpoints:
		if ce.epSubView == core.ViewForm {
			ce.saveEPForm()
		}
	case contractsTabExternal:
		switch ce.extSubView {
		case core.ViewForm:
			ce.saveExtForm()
		case core.ViewSubForm:
			ce.saveExtIntForm()
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
	if ce.internalMode == core.ModeInsert {
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
		ce.activeTab = contractsTabIdx(core.NavigateTab(key.String(), int(ce.activeTab), len(contractsTabLabels)))
		if ce.activeTab == contractsTabVersioning && ce.versioningEnabled {
			ce.rebuildVersioningFields()
		}
		return ce, nil
	}

	// cc detection: clear field and enter insert mode
	if !ce.dd.Open {
		if key.String() == "c" {
			if ce.cBuf {
				ce.cBuf = false
				return ce.clearAndEnterInsert()
			}
			ce.cBuf = true
			return ce, nil
		}
		ce.cBuf = false
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
			ce.internalMode = core.ModeNormal
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
		case core.ViewForm:
			n := len(ce.visibleDTOFields())
			if n > 0 {
				ce.dtoFormIdx = (ce.dtoFormIdx + delta + n) % n
			}
		case core.ViewSubForm:
			n := len(ce.visibleDTOFieldFormFields())
			if n > 0 {
				ce.dtoFieldFormIdx = (ce.dtoFieldFormIdx + delta + n) % n
			}
		}
	case contractsTabEndpoints:
		if ce.epSubView == core.ViewForm {
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
		switch ce.extSubView {
		case core.ViewForm:
			n := len(ce.visibleExtFormFields())
			if n > 0 {
				ce.extFormIdx = (ce.extFormIdx + delta + n) % n
			}
		case core.ViewSubForm:
			n := len(ce.visibleExtIntFormFields())
			if n > 0 {
				ce.extIntFormIdx = (ce.extIntFormIdx + delta + n) % n
			}
		}
	}
}

func (ce *ContractsEditor) saveInput() {
	val := ce.formInput.Value()
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case core.ViewForm:
			if ce.dtoFormIdx < len(ce.dtoForm) && ce.dtoForm[ce.dtoFormIdx].CanEditAsText() {
				ce.dtoForm[ce.dtoFormIdx].SaveTextInput(val)
			}
		case core.ViewSubForm:
			if ce.dtoFieldFormIdx < len(ce.dtoFieldForm) && ce.dtoFieldForm[ce.dtoFieldFormIdx].CanEditAsText() {
				ce.dtoFieldForm[ce.dtoFieldFormIdx].SaveTextInput(val)
			}
		}
	case contractsTabEndpoints:
		visible := ce.visibleEPFields()
		if ce.epSubView == core.ViewForm && ce.epFormIdx < len(visible) {
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
		switch ce.extSubView {
		case core.ViewForm:
			visible := ce.visibleExtFormFields()
			if ce.extFormIdx < len(visible) {
				f := ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
				if f != nil && f.CanEditAsText() {
					f.SaveTextInput(val)
				}
			}
		case core.ViewSubForm:
			visible := ce.visibleExtIntFormFields()
			if ce.extIntFormIdx < len(visible) {
				f := ce.extIntFormFieldByKey(visible[ce.extIntFormIdx].Key)
				if f != nil && f.CanEditAsText() {
					f.SaveTextInput(val)
				}
			}
		}
	}
}

func (ce ContractsEditor) clearAndEnterInsert() (ContractsEditor, tea.Cmd) {
	ce, cmd := ce.tryEnterInsert()
	if ce.internalMode == core.ModeInsert {
		ce.formInput.SetValue("")
	}
	return ce, cmd
}

func (ce ContractsEditor) tryEnterInsert() (ContractsEditor, tea.Cmd) {
	n := 0
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case core.ViewForm:
			n = len(ce.dtoForm)
		case core.ViewSubForm:
			n = len(ce.visibleDTOFieldFormFields())
		}
	case contractsTabEndpoints:
		if ce.epSubView == core.ViewForm {
			n = len(ce.visibleEPFields())
		}
	case contractsTabVersioning:
		n = len(ce.versioningFields)
	case contractsTabExternal:
		switch ce.extSubView {
		case core.ViewForm:
			n = len(ce.visibleExtFormFields())
		case core.ViewSubForm:
			n = len(ce.visibleExtIntFormFields())
		}
	}
	for range n {
		var f *core.Field
		switch ce.activeTab {
		case contractsTabDTOs:
			switch ce.dtoSubView {
			case core.ViewForm:
				if ce.dtoFormIdx < len(ce.dtoForm) {
					f = &ce.dtoForm[ce.dtoFormIdx]
				}
			case core.ViewSubForm:
				if ce.dtoFieldFormIdx < len(ce.dtoFieldForm) {
					f = &ce.dtoFieldForm[ce.dtoFieldFormIdx]
				}
			}
		case contractsTabEndpoints:
			visible := ce.visibleEPFields()
			if ce.epSubView == core.ViewForm && ce.epFormIdx < len(visible) {
				f = ce.epFieldByKey(visible[ce.epFormIdx].Key)
			}
		case contractsTabVersioning:
			if ce.verFormIdx < len(ce.versioningFields) {
				f = &ce.versioningFields[ce.verFormIdx]
			}
		case contractsTabExternal:
			switch ce.extSubView {
			case core.ViewForm:
				visible := ce.visibleExtFormFields()
				if ce.extFormIdx < len(visible) {
					f = ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
				}
			case core.ViewSubForm:
				visible := ce.visibleExtIntFormFields()
				if ce.extIntFormIdx < len(visible) {
					f = ce.extIntFormFieldByKey(visible[ce.extIntFormIdx].Key)
				}
			}
		}
		if f == nil {
			break
		}
		if f.CanEditAsText() {
			ce.internalMode = core.ModeInsert
			ce.formInput.SetValue(f.TextInputValue())
			ce.formInput.Width = ce.width - 22
			ce.formInput.CursorEnd()
			return ce, ce.formInput.Focus()
		}
		ce.advanceField(1)
	}
	return ce, nil
}

// CurrentField returns the currently highlighted form field for the description panel.
// Returns nil when in list view or when no field can be resolved.
func (ce *ContractsEditor) CurrentField() *core.Field {
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case core.ViewForm:
			if ce.dtoFormIdx >= 0 && ce.dtoFormIdx < len(ce.dtoForm) {
				return &ce.dtoForm[ce.dtoFormIdx]
			}
		case core.ViewSubForm:
			if ce.dtoFieldFormIdx >= 0 && ce.dtoFieldFormIdx < len(ce.dtoFieldForm) {
				return &ce.dtoFieldForm[ce.dtoFieldFormIdx]
			}
		}
	case contractsTabEndpoints:
		if ce.epSubView == core.ViewForm {
			visible := ce.visibleEPFields()
			if ce.epFormIdx >= 0 && ce.epFormIdx < len(visible) {
				return ce.epFieldByKey(visible[ce.epFormIdx].Key)
			}
		}
	case contractsTabVersioning:
		if ce.versioningEnabled && ce.verFormIdx >= 0 && ce.verFormIdx < len(ce.versioningFields) {
			return &ce.versioningFields[ce.verFormIdx]
		}
	case contractsTabExternal:
		if ce.extSubView == core.ViewForm {
			visible := ce.visibleExtFormFields()
			if ce.extFormIdx >= 0 && ce.extFormIdx < len(visible) {
				return ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
			}
		}
	}
	return nil
}
