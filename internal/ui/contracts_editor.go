package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-mvp/internal/manifest"
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

// ── mode ──────────────────────────────────────────────────────────────────────

type ceMode int

const (
	ceNormal ceMode = iota
	ceInsert
)

// ── list-item sub-view ────────────────────────────────────────────────────────

type ceSubView int

const (
	ceViewList     ceSubView = iota
	ceViewForm               // top-level form
	ceViewSubList            // sub-list (e.g., DTO fields, endpoint error responses)
	ceViewSubForm            // sub-item form
)

// ── field definitions ─────────────────────────────────────────────────────────

func defaultDTOFormFields(domainOptions []string) []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{
			Key: "category", Label: "category      ", Kind: KindSelect,
			Options: []string{"Request", "Response", "Event Payload", "Shared/Common"},
			Value:   "Request",
		},
		{
			Key: "source_domains", Label: "source_domains", Kind: KindMultiSelect,
			Options: domainOptions,
		},
		{Key: "description", Label: "description   ", Kind: KindText},
	}
}

func defaultDTOFieldForm() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{
			Key: "type", Label: "type          ", Kind: KindSelect,
			Options: []string{
				"string", "int", "float", "boolean", "datetime",
				"uuid", "enum(values)", "array(type)", "nested(DTO)", "map(key,value)",
			},
			Value: "string",
		},
		{
			Key: "required", Label: "required      ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{
			Key: "nullable", Label: "nullable      ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{Key: "validation", Label: "validation    ", Kind: KindText},
		{Key: "notes", Label: "notes         ", Kind: KindText},
	}
}

func defaultEndpointFormFields(serviceOptions, dtoOptions []string) []Field {
	// Ensure at least empty slice so KindSelect works
	if serviceOptions == nil {
		serviceOptions = []string{}
	}
	if dtoOptions == nil {
		dtoOptions = []string{}
	}
	serviceKind := KindText
	if len(serviceOptions) > 0 {
		serviceKind = KindSelect
	}
	dtoKind := KindText
	if len(dtoOptions) > 0 {
		dtoKind = KindSelect
	}
	fields := []Field{
		{Key: "service_unit", Label: "service_unit  ", Kind: serviceKind, Options: serviceOptions},
		{Key: "name_path", Label: "name_path     ", Kind: KindText},
		{
			Key: "protocol", Label: "protocol      ", Kind: KindSelect,
			Options: []string{"REST", "GraphQL", "gRPC", "WebSocket message", "Event"},
			Value:   "REST",
		},
		{
			Key: "auth_required", Label: "auth_required ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{Key: "request_dto", Label: "request_dto   ", Kind: dtoKind, Options: dtoOptions},
		{Key: "response_dto", Label: "response_dto  ", Kind: dtoKind, Options: dtoOptions},
		{
			Key: "http_method", Label: "http_method   ", Kind: KindSelect,
			Options: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			Value:   "GET",
		},
		{
			Key: "graphql_op_type", Label: "Operation     ", Kind: KindSelect,
			Options: []string{"Query", "Mutation", "Subscription"},
			Value:   "Query",
		},
		{
			Key: "grpc_stream_type", Label: "Stream Type   ", Kind: KindSelect,
			Options: []string{"Unary", "Server stream", "Client stream", "Bidirectional"},
			Value:   "Unary",
		},
		{
			Key: "ws_direction", Label: "WS Direction  ", Kind: KindSelect,
			Options: []string{"Client→Server", "Server→Client", "Bidirectional"},
			Value:   "Bidirectional", SelIdx: 2,
		},
		{
			Key: "pagination", Label: "Pagination    ", Kind: KindSelect,
			Options: []string{"Cursor-based", "Offset/limit", "Keyset", "Page number", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "rate_limit", Label: "Rate Limit    ", Kind: KindSelect,
			Options: []string{"Default (global)", "Strict", "Relaxed", "None"},
			Value:   "Default (global)",
		},
		{Key: "description", Label: "description   ", Kind: KindText},
	}
	return fields
}

func defaultVersioningFields() []Field {
	return []Field{
		{
			Key: "strategy", Label: "strategy      ", Kind: KindSelect,
			Options: []string{
				"URL path (/v1/)", "Header (Accept-Version)", "Query param", "None",
			},
			Value: "URL path (/v1/)",
		},
		{Key: "current_version", Label: "current_ver   ", Kind: KindText, Value: "v1"},
		{
			Key: "deprecation", Label: "deprecation   ", Kind: KindSelect,
			Options: []string{
				"None", "Sunset header", "Versioned removal notice", "Changelog entry", "Custom",
			},
			Value: "None",
		},
	}
}

func defaultExternalAPIFormFields() []Field {
	return []Field{
		{Key: "provider", Label: "provider      ", Kind: KindText},
		{
			Key: "auth_mechanism", Label: "auth_mechanism", Kind: KindSelect,
			Options: []string{"API Key", "OAuth2 Client Credentials", "OAuth2 PKCE", "Bearer Token", "Basic Auth", "mTLS", "None"},
			Value:   "API Key",
		},
		{Key: "base_url", Label: "base_url      ", Kind: KindText},
		{Key: "rate_limit", Label: "rate_limit    ", Kind: KindText, Value: ""},
		{Key: "webhook_endpoint", Label: "webhook_path  ", Kind: KindText},
		{
			Key: "failure_strategy", Label: "failure_strat ", Kind: KindSelect,
			Options: []string{"Circuit Breaker", "Retry with backoff", "Fallback value", "Timeout + fail", "None"},
			Value:   "Circuit Breaker",
		},
	}
}

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
	versioningFields []Field
	verFormIdx       int

	// External APIs
	externalAPIs []manifest.ExternalAPIDef
	extSubView   ceSubView
	extIdx       int
	extForm      []Field
	extFormIdx   int

	// Cross-editor reference data (set by model.go before each Update)
	availableDomains    []string               // from DataTabEditor.domainNames()
	availableDomainDefs []manifest.DomainDef   // from DataTabEditor.domains
	availableServices   []string               // from BackendEditor.ServiceNames()
	availableServiceDefs []manifest.ServiceDef // from BackendEditor.ServiceDefs()

	// Dropdown state for KindSelect/KindMultiSelect fields
	ddOpen   bool
	ddOptIdx int

	// Shared
	internalMode ceMode
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

// protocolsForService returns the protocol options valid for the named service
// based on its registered technologies. Returns nil when no filter applies.
func (ce ContractsEditor) protocolsForService(serviceName string) []string {
	techToProto := map[string]string{
		"REST":           "REST",
		"GraphQL":        "GraphQL",
		"gRPC":           "gRPC",
		"WebSocket":      "WebSocket message",
		"SSE":            "REST",
		"tRPC":           "REST",
		"MQTT":           "Event",
		"Kafka consumer": "Event",
	}
	for _, svc := range ce.availableServiceDefs {
		if svc.Name != serviceName {
			continue
		}
		if len(svc.Technologies) == 0 {
			return nil
		}
		seen := make(map[string]bool)
		var protos []string
		for _, tech := range svc.Technologies {
			if proto, ok := techToProto[tech]; ok && !seen[proto] {
				seen[proto] = true
				protos = append(protos, proto)
			}
		}
		if len(protos) == 0 {
			return nil
		}
		return protos
	}
	return nil
}

// updateEPDependentFields refreshes the protocol options based on the selected
// service unit and clamps epFormIdx to the visible field range.
func (ce *ContractsEditor) updateEPDependentFields() {
	if ce.activeTab != contractsTabEndpoints || ce.epSubView != ceViewForm {
		return
	}
	svcName := fieldGet(ce.epForm, "service_unit")
	protos := ce.protocolsForService(svcName)
	if protos == nil {
		protos = []string{"REST", "GraphQL", "gRPC", "WebSocket message", "Event"}
	}
	for i := range ce.epForm {
		if ce.epForm[i].Key != "protocol" {
			continue
		}
		current := ce.epForm[i].Value
		ce.epForm[i].Options = protos
		found := false
		for j, opt := range protos {
			if opt == current {
				ce.epForm[i].SelIdx = j
				found = true
				break
			}
		}
		if !found {
			ce.epForm[i].SelIdx = 0
			ce.epForm[i].Value = protos[0]
		}
		break
	}
	// Clamp cursor to visible range
	visible := ce.visibleEPFields()
	if len(visible) > 0 && ce.epFormIdx >= len(visible) {
		ce.epFormIdx = len(visible) - 1
	}
}

// visibleEPFields returns only the endpoint form fields relevant to the
// currently selected protocol, hiding the other protocol-specific fields.
func (ce ContractsEditor) visibleEPFields() []Field {
	if len(ce.epForm) == 0 {
		return nil
	}
	proto := fieldGet(ce.epForm, "protocol")
	var visible []Field
	for _, f := range ce.epForm {
		switch f.Key {
		case "http_method":
			if proto != "REST" {
				continue
			}
		case "graphql_op_type":
			if proto != "GraphQL" {
				continue
			}
		case "grpc_stream_type":
			if proto != "gRPC" {
				continue
			}
		case "ws_direction":
			if proto != "WebSocket message" {
				continue
			}
		}
		visible = append(visible, f)
	}
	return visible
}

// epFieldByKey returns a pointer to the endpoint form field with the given key.
func (ce *ContractsEditor) epFieldByKey(key string) *Field {
	for i := range ce.epForm {
		if ce.epForm[i].Key == key {
			return &ce.epForm[i]
		}
	}
	return nil
}

// SetDomainDefs updates the full domain definitions for attribute injection.
func (ce *ContractsEditor) SetDomainDefs(domains []manifest.DomainDef) {
	ce.availableDomainDefs = domains
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (ce ContractsEditor) ToManifestContractsPillar() manifest.ContractsPillar {
	return manifest.ContractsPillar{
		DTOs:      ce.dtos,
		Endpoints: ce.endpoints,
		Versioning: manifest.APIVersioning{
			Strategy:          fieldGet(ce.versioningFields, "strategy"),
			CurrentVersion:    fieldGet(ce.versioningFields, "current_version"),
			DeprecationPolicy: fieldGet(ce.versioningFields, "deprecation"),
		},
		ExternalAPIs: ce.externalAPIs,
	}
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (ce ContractsEditor) Mode() Mode {
	if ce.internalMode == ceInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (ce ContractsEditor) HintLine() string {
	if ce.internalMode == ceInsert {
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
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
	case contractsTabEndpoints:
		switch ce.epSubView {
		case ceViewList:
			return hintBar("j/k", "navigate", "a", "add endpoint", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		case ceViewForm:
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
	case contractsTabVersioning:
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "H", "cycle back", "h/l", "sub-tab")
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

// activeCEFieldPtr returns a pointer to the currently focused field that supports dropdown.
func (ce *ContractsEditor) activeCEFieldPtr() *Field {
	switch ce.activeTab {
	case contractsTabDTOs:
		switch ce.dtoSubView {
		case ceViewForm:
			if ce.dtoFormIdx < len(ce.dtoForm) {
				return &ce.dtoForm[ce.dtoFormIdx]
			}
		case ceViewSubForm:
			if ce.dtoFieldFormIdx < len(ce.dtoFieldForm) {
				return &ce.dtoFieldForm[ce.dtoFieldFormIdx]
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
		if ce.extSubView == ceViewForm && ce.extFormIdx < len(ce.extForm) {
			return &ce.extForm[ce.extFormIdx]
		}
	}
	return nil
}

func (ce ContractsEditor) updateDropdown(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	f := ce.activeCEFieldPtr()
	if f == nil {
		ce.ddOpen = false
		return ce, nil
	}
	switch key.String() {
	case "j", "down":
		if ce.ddOptIdx < len(f.Options)-1 {
			ce.ddOptIdx++
		}
	case "k", "up":
		if ce.ddOptIdx > 0 {
			ce.ddOptIdx--
		}
	case "g":
		ce.ddOptIdx = 0
	case "G":
		if len(f.Options) > 0 {
			ce.ddOptIdx = len(f.Options) - 1
		}
	case " ":
		if f.Kind == KindMultiSelect {
			f.ToggleMultiSelect(ce.ddOptIdx)
			f.DDCursor = ce.ddOptIdx
		} else if f.Kind == KindSelect {
			f.SelIdx = ce.ddOptIdx
			if ce.ddOptIdx < len(f.Options) {
				f.Value = f.Options[ce.ddOptIdx]
			}
			ce.ddOpen = false
		}
	case "enter":
		if f.Kind == KindMultiSelect {
			f.DDCursor = ce.ddOptIdx
		} else if f.Kind == KindSelect {
			f.SelIdx = ce.ddOptIdx
			if ce.ddOptIdx < len(f.Options) {
				f.Value = f.Options[ce.ddOptIdx]
			}
		}
		ce.ddOpen = false
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = ce.ddOptIdx
		}
		ce.ddOpen = false
	}
	// After any dropdown interaction, refresh EP dependent fields (protocol options, visibility)
	ce.updateEPDependentFields()
	return ce, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (ce ContractsEditor) Update(msg tea.Msg) (ContractsEditor, tea.Cmd) {
	if ce.internalMode == ceInsert {
		return ce.updateInsert(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return ce, nil
	}

	// Handle dropdown if open
	if ce.ddOpen && ok {
		return ce.updateDropdown(key)
	}

	// Sub-tab switching (only from top-level lists)
	canSwitch := (ce.activeTab == contractsTabDTOs && ce.dtoSubView == ceViewList) ||
		(ce.activeTab == contractsTabEndpoints && ce.epSubView == ceViewList) ||
		ce.activeTab == contractsTabVersioning ||
		(ce.activeTab == contractsTabExternal && ce.extSubView == ceViewList)

	if canSwitch {
		switch key.String() {
		case "h", "left":
			if ce.activeTab > 0 {
				ce.activeTab--
			}
			return ce, nil
		case "l", "right":
			if int(ce.activeTab) < len(contractsTabLabels)-1 {
				ce.activeTab++
			}
			return ce, nil
		}
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
			ce.internalMode = ceNormal
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
			n := len(ce.dtoForm)
			if n > 0 {
				ce.dtoFormIdx = (ce.dtoFormIdx + delta + n) % n
			}
		case ceViewSubForm:
			n := len(ce.dtoFieldForm)
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
			n := len(ce.extForm)
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
			if ce.dtoFormIdx < len(ce.dtoForm) && ce.dtoForm[ce.dtoFormIdx].Kind == KindText {
				ce.dtoForm[ce.dtoFormIdx].Value = val
			}
		case ceViewSubForm:
			if ce.dtoFieldFormIdx < len(ce.dtoFieldForm) && ce.dtoFieldForm[ce.dtoFieldFormIdx].Kind == KindText {
				ce.dtoFieldForm[ce.dtoFieldFormIdx].Value = val
			}
		}
	case contractsTabEndpoints:
		visible := ce.visibleEPFields()
		if ce.epSubView == ceViewForm && ce.epFormIdx < len(visible) {
			f := ce.epFieldByKey(visible[ce.epFormIdx].Key)
			if f != nil && f.Kind == KindText {
				f.Value = val
			}
		}
	case contractsTabVersioning:
		if ce.verFormIdx < len(ce.versioningFields) && ce.versioningFields[ce.verFormIdx].Kind == KindText {
			ce.versioningFields[ce.verFormIdx].Value = val
		}
	case contractsTabExternal:
		if ce.extSubView == ceViewForm && ce.extFormIdx < len(ce.extForm) && ce.extForm[ce.extFormIdx].Kind == KindText {
			ce.extForm[ce.extFormIdx].Value = val
		}
	}
}

func (ce ContractsEditor) tryEnterInsert() (ContractsEditor, tea.Cmd) {
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
		if ce.extSubView == ceViewForm && ce.extFormIdx < len(ce.extForm) {
			f = &ce.extForm[ce.extFormIdx]
		}
	}
	if f == nil || f.Kind != KindText {
		return ce, nil
	}
	ce.internalMode = ceInsert
	ce.formInput.SetValue(f.Value)
	ce.formInput.Width = ce.width - 22
	ce.formInput.CursorEnd()
	return ce, ce.formInput.Focus()
}

// ── DTO updates ───────────────────────────────────────────────────────────────

func (ce ContractsEditor) updateDTOs(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	switch ce.dtoSubView {
	case ceViewList:
		return ce.updateDTOList(key)
	case ceViewForm:
		return ce.updateDTOForm(key)
	case ceViewSubList:
		return ce.updateDTOFieldList(key)
	case ceViewSubForm:
		return ce.updateDTOFieldForm(key)
	}
	return ce, nil
}

func (ce ContractsEditor) updateDTOList(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	n := len(ce.dtos)
	switch key.String() {
	case "j", "down":
		if n > 0 && ce.dtoIdx < n-1 {
			ce.dtoIdx++
		}
	case "k", "up":
		if ce.dtoIdx > 0 {
			ce.dtoIdx--
		}
	case "a":
		ce.dtos = append(ce.dtos, manifest.DTODef{})
		ce.dtoIdx = len(ce.dtos) - 1
		ce.dtoForm = defaultDTOFormFields(ce.availableDomains)
		ce.dtoFormIdx = 0
		ce.dtoFieldItems = nil
		ce.dtoSubView = ceViewForm
		return ce.tryEnterInsert()
	case "d":
		if n > 0 {
			ce.dtos = append(ce.dtos[:ce.dtoIdx], ce.dtos[ce.dtoIdx+1:]...)
			if ce.dtoIdx > 0 && ce.dtoIdx >= len(ce.dtos) {
				ce.dtoIdx = len(ce.dtos) - 1
			}
		}
	case "enter":
		if n > 0 {
			d := ce.dtos[ce.dtoIdx]
			ce.dtoForm = defaultDTOFormFields(ce.availableDomains)
			ce.dtoForm = setFieldValue(ce.dtoForm, "name", d.Name)
			ce.dtoForm = setFieldValue(ce.dtoForm, "category", d.Category)
			// Restore multiselect for source_domains
			if d.SourceDomains != "" {
				for i := range ce.dtoForm {
					if ce.dtoForm[i].Key == "source_domains" {
						for _, sel := range splitCSV(d.SourceDomains) {
							for j, opt := range ce.dtoForm[i].Options {
								if opt == sel {
									ce.dtoForm[i].SelectedIdxs = append(ce.dtoForm[i].SelectedIdxs, j)
								}
							}
						}
						break
					}
				}
			}
			ce.dtoForm = setFieldValue(ce.dtoForm, "description", d.Description)
			ce.dtoFormIdx = 0
			// Rebuild field items
			ce.dtoFieldItems = make([][]Field, len(d.Fields))
			for i, df := range d.Fields {
				f := defaultDTOFieldForm()
				f = setFieldValue(f, "name", df.Name)
				f = setFieldValue(f, "type", df.Type)
				if df.Required {
					f = setFieldValue(f, "required", "true")
				}
				if df.Nullable {
					f = setFieldValue(f, "nullable", "true")
				}
				f = setFieldValue(f, "validation", df.Validation)
				f = setFieldValue(f, "notes", df.Notes)
				ce.dtoFieldItems[i] = f
			}
			ce.dtoSubView = ceViewForm
		}
	}
	return ce, nil
}

// splitCSV splits a comma-separated string into trimmed parts.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ", ")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func (ce ContractsEditor) updateDTOForm(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if ce.dtoFormIdx < len(ce.dtoForm)-1 {
			ce.dtoFormIdx++
		}
	case "k", "up":
		if ce.dtoFormIdx > 0 {
			ce.dtoFormIdx--
		}
	case "enter", " ":
		f := &ce.dtoForm[ce.dtoFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			ce.ddOpen = true
			if f.Kind == KindSelect {
				ce.ddOptIdx = f.SelIdx
			} else {
				ce.ddOptIdx = f.DDCursor
			}
		} else {
			return ce.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &ce.dtoForm[ce.dtoFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if ce.dtoForm[ce.dtoFormIdx].Kind == KindText {
			return ce.tryEnterInsert()
		}
	case "F":
		ce.saveDTOForm()
		ce.populateDTOFieldsFromDomains()
		ce.dtoFieldIdx = 0
		ce.dtoSubView = ceViewSubList
	case "b", "esc":
		ce.saveDTOForm()
		ce.dtoSubView = ceViewList
	}
	return ce, nil
}

// populateDTOFieldsFromDomains auto-populates DTO fields from selected source domains
// when navigating to the fields sub-list. Only runs when the field list is empty.
func (ce *ContractsEditor) populateDTOFieldsFromDomains() {
	if len(ce.dtoFieldItems) > 0 {
		return
	}
	sourceDomains := fieldGetMulti(ce.dtoForm, "source_domains")
	if sourceDomains == "" {
		return
	}
	for _, domainName := range strings.Split(sourceDomains, ", ") {
		domainName = strings.TrimSpace(domainName)
		if domainName == "" {
			continue
		}
		for _, domainDef := range ce.availableDomainDefs {
			if domainDef.Name != domainName {
				continue
			}
			for _, attr := range domainDef.Attributes {
				f := defaultDTOFieldForm()
				f = setFieldValue(f, "name", attr.Name)
				f = setFieldValue(f, "type", domainTypeToDTOType(attr.Type))
				if attr.Sensitive {
					f = setFieldValue(f, "nullable", "true")
				}
				if attr.Validation != "" {
					f = setFieldValue(f, "validation", attr.Validation)
				}
				ce.dtoFieldItems = append(ce.dtoFieldItems, f)
			}
			break
		}
	}
	// Fallback placeholder if no domain attrs found
	if len(ce.dtoFieldItems) == 0 {
		placeholder := defaultDTOFieldForm()
		placeholder = setFieldValue(placeholder, "name", "id")
		placeholder = setFieldValue(placeholder, "type", "uuid")
		placeholder = setFieldValue(placeholder, "required", "true")
		ce.dtoFieldItems = append(ce.dtoFieldItems, placeholder)
	}
}

func domainTypeToDTOType(t string) string {
	switch t {
	case "String":
		return "string"
	case "Int":
		return "int"
	case "Float":
		return "float"
	case "Boolean":
		return "boolean"
	case "DateTime":
		return "datetime"
	case "UUID":
		return "uuid"
	case "Enum(values)":
		return "enum(values)"
	case "JSON/Map":
		return "map(key,value)"
	case "Array(type)":
		return "array(type)"
	case "Ref(Domain)":
		return "nested(DTO)"
	default:
		return "string"
	}
}

func (ce *ContractsEditor) saveDTOForm() {
	if ce.dtoIdx >= len(ce.dtos) {
		return
	}
	d := &ce.dtos[ce.dtoIdx]
	d.Name = fieldGet(ce.dtoForm, "name")
	d.Category = fieldGet(ce.dtoForm, "category")
	d.SourceDomains = fieldGetMulti(ce.dtoForm, "source_domains")
	d.Description = fieldGet(ce.dtoForm, "description")

	d.Fields = make([]manifest.DTOField, len(ce.dtoFieldItems))
	for i, item := range ce.dtoFieldItems {
		d.Fields[i] = manifest.DTOField{
			Name:       fieldGet(item, "name"),
			Type:       fieldGet(item, "type"),
			Required:   fieldGet(item, "required") == "true",
			Nullable:   fieldGet(item, "nullable") == "true",
			Validation: fieldGet(item, "validation"),
			Notes:      fieldGet(item, "notes"),
		}
	}
}

func (ce ContractsEditor) updateDTOFieldList(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	n := len(ce.dtoFieldItems)
	switch key.String() {
	case "j", "down":
		if n > 0 && ce.dtoFieldIdx < n-1 {
			ce.dtoFieldIdx++
		}
	case "k", "up":
		if ce.dtoFieldIdx > 0 {
			ce.dtoFieldIdx--
		}
	case "a":
		ce.dtoFieldItems = append(ce.dtoFieldItems, defaultDTOFieldForm())
		ce.dtoFieldIdx = len(ce.dtoFieldItems) - 1
		ce.dtoFieldForm = copyFields(ce.dtoFieldItems[ce.dtoFieldIdx])
		ce.dtoFieldFormIdx = 0
		ce.dtoSubView = ceViewSubForm
		return ce.tryEnterInsert()
	case "d":
		if n > 0 {
			ce.dtoFieldItems = append(ce.dtoFieldItems[:ce.dtoFieldIdx], ce.dtoFieldItems[ce.dtoFieldIdx+1:]...)
			if ce.dtoFieldIdx > 0 && ce.dtoFieldIdx >= len(ce.dtoFieldItems) {
				ce.dtoFieldIdx = len(ce.dtoFieldItems) - 1
			}
		}
	case "enter":
		if n > 0 {
			ce.dtoFieldForm = copyFields(ce.dtoFieldItems[ce.dtoFieldIdx])
			ce.dtoFieldFormIdx = 0
			ce.dtoSubView = ceViewSubForm
		}
	case "b", "esc":
		ce.dtoSubView = ceViewForm
	}
	return ce, nil
}

func (ce ContractsEditor) updateDTOFieldForm(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if ce.dtoFieldFormIdx < len(ce.dtoFieldForm)-1 {
			ce.dtoFieldFormIdx++
		}
	case "k", "up":
		if ce.dtoFieldFormIdx > 0 {
			ce.dtoFieldFormIdx--
		}
	case "enter", " ":
		f := &ce.dtoFieldForm[ce.dtoFieldFormIdx]
		if f.Kind == KindSelect {
			ce.ddOpen = true
			ce.ddOptIdx = f.SelIdx
		} else {
			return ce.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &ce.dtoFieldForm[ce.dtoFieldFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if ce.dtoFieldForm[ce.dtoFieldFormIdx].Kind == KindText {
			return ce.tryEnterInsert()
		}
	case "b", "esc":
		if ce.dtoFieldIdx < len(ce.dtoFieldItems) {
			ce.dtoFieldItems[ce.dtoFieldIdx] = copyFields(ce.dtoFieldForm)
		}
		ce.dtoSubView = ceViewSubList
	}
	return ce, nil
}

// ── Endpoint updates ──────────────────────────────────────────────────────────

func (ce ContractsEditor) updateEndpoints(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	switch ce.epSubView {
	case ceViewList:
		return ce.updateEPList(key)
	case ceViewForm:
		return ce.updateEPForm(key)
	}
	return ce, nil
}

func (ce ContractsEditor) updateEPList(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	n := len(ce.endpoints)
	switch key.String() {
	case "j", "down":
		if n > 0 && ce.epIdx < n-1 {
			ce.epIdx++
		}
	case "k", "up":
		if ce.epIdx > 0 {
			ce.epIdx--
		}
	case "a":
		ce.endpoints = append(ce.endpoints, manifest.EndpointDef{})
		ce.epIdx = len(ce.endpoints) - 1
		ce.epForm = defaultEndpointFormFields(ce.availableServices, ce.dtoNames())
		ce.epFormIdx = 0
		ce.epSubView = ceViewForm
		return ce.tryEnterInsert()
	case "d":
		if n > 0 {
			ce.endpoints = append(ce.endpoints[:ce.epIdx], ce.endpoints[ce.epIdx+1:]...)
			if ce.epIdx > 0 && ce.epIdx >= len(ce.endpoints) {
				ce.epIdx = len(ce.endpoints) - 1
			}
		}
	case "enter":
		if n > 0 {
			ep := ce.endpoints[ce.epIdx]
			ce.epForm = defaultEndpointFormFields(ce.availableServices, ce.dtoNames())
			ce.epForm = setFieldValue(ce.epForm, "service_unit", ep.ServiceUnit)
			ce.epForm = setFieldValue(ce.epForm, "name_path", ep.NamePath)
			if ep.Protocol != "" {
				ce.epForm = setFieldValue(ce.epForm, "protocol", ep.Protocol)
			}
			ce.epForm = setFieldValue(ce.epForm, "auth_required", ep.AuthRequired)
			ce.epForm = setFieldValue(ce.epForm, "request_dto", ep.RequestDTO)
			ce.epForm = setFieldValue(ce.epForm, "response_dto", ep.ResponseDTO)
			if ep.HTTPMethod != "" {
				ce.epForm = setFieldValue(ce.epForm, "http_method", ep.HTTPMethod)
			}
			if ep.GraphQLOpType != "" {
				ce.epForm = setFieldValue(ce.epForm, "graphql_op_type", ep.GraphQLOpType)
			}
			if ep.GRPCStreamType != "" {
				ce.epForm = setFieldValue(ce.epForm, "grpc_stream_type", ep.GRPCStreamType)
			}
			if ep.WSDirection != "" {
				ce.epForm = setFieldValue(ce.epForm, "ws_direction", ep.WSDirection)
			}
			if ep.PaginationStrategy != "" {
				ce.epForm = setFieldValue(ce.epForm, "pagination", ep.PaginationStrategy)
			}
			if ep.RateLimit != "" {
				ce.epForm = setFieldValue(ce.epForm, "rate_limit", ep.RateLimit)
			}
			ce.epForm = setFieldValue(ce.epForm, "description", ep.Description)
			ce.epFormIdx = 0
			ce.epSubView = ceViewForm
		}
	}
	return ce, nil
}

func (ce ContractsEditor) updateEPForm(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	visible := ce.visibleEPFields()
	n := len(visible)
	switch key.String() {
	case "j", "down":
		if ce.epFormIdx < n-1 {
			ce.epFormIdx++
		}
	case "k", "up":
		if ce.epFormIdx > 0 {
			ce.epFormIdx--
		}
	case "enter", " ":
		if ce.epFormIdx < n {
			f := ce.epFieldByKey(visible[ce.epFormIdx].Key)
			if f != nil && f.Kind == KindSelect {
				ce.ddOpen = true
				ce.ddOptIdx = f.SelIdx
			} else {
				return ce.tryEnterInsert()
			}
		}
	case "H", "shift+left":
		if ce.epFormIdx < n {
			f := ce.epFieldByKey(visible[ce.epFormIdx].Key)
			if f != nil && f.Kind == KindSelect {
				f.CyclePrev()
				ce.updateEPDependentFields()
			}
		}
	case "i", "a":
		if ce.epFormIdx < n {
			f := ce.epFieldByKey(visible[ce.epFormIdx].Key)
			if f != nil && f.Kind == KindText {
				return ce.tryEnterInsert()
			}
		}
	case "b", "esc":
		ce.saveEPForm()
		ce.epSubView = ceViewList
	}
	return ce, nil
}

func (ce *ContractsEditor) saveEPForm() {
	if ce.epIdx >= len(ce.endpoints) {
		return
	}
	ep := &ce.endpoints[ce.epIdx]
	ep.ServiceUnit = fieldGet(ce.epForm, "service_unit")
	ep.NamePath = fieldGet(ce.epForm, "name_path")
	ep.Protocol = fieldGet(ce.epForm, "protocol")
	ep.AuthRequired = fieldGet(ce.epForm, "auth_required")
	ep.RequestDTO = fieldGet(ce.epForm, "request_dto")
	ep.ResponseDTO = fieldGet(ce.epForm, "response_dto")
	ep.HTTPMethod = fieldGet(ce.epForm, "http_method")
	ep.GraphQLOpType = fieldGet(ce.epForm, "graphql_op_type")
	ep.GRPCStreamType = fieldGet(ce.epForm, "grpc_stream_type")
	ep.WSDirection = fieldGet(ce.epForm, "ws_direction")
	ep.PaginationStrategy = fieldGet(ce.epForm, "pagination")
	ep.RateLimit = fieldGet(ce.epForm, "rate_limit")
	ep.Description = fieldGet(ce.epForm, "description")
}

// ── Versioning update ─────────────────────────────────────────────────────────

func (ce ContractsEditor) updateVersioning(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if ce.verFormIdx < len(ce.versioningFields)-1 {
			ce.verFormIdx++
		}
	case "k", "up":
		if ce.verFormIdx > 0 {
			ce.verFormIdx--
		}
	case "enter", " ":
		f := &ce.versioningFields[ce.verFormIdx]
		if f.Kind == KindSelect {
			ce.ddOpen = true
			ce.ddOptIdx = f.SelIdx
		} else {
			return ce.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &ce.versioningFields[ce.verFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i":
		if ce.versioningFields[ce.verFormIdx].Kind == KindText {
			return ce.tryEnterInsert()
		}
	}
	return ce, nil
}

// ── External APIs updates ─────────────────────────────────────────────────────

func (ce ContractsEditor) updateExternal(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	switch ce.extSubView {
	case ceViewList:
		return ce.updateExtList(key)
	case ceViewForm:
		return ce.updateExtForm(key)
	}
	return ce, nil
}

func (ce ContractsEditor) updateExtList(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	n := len(ce.externalAPIs)
	switch key.String() {
	case "j", "down":
		if n > 0 && ce.extIdx < n-1 {
			ce.extIdx++
		}
	case "k", "up":
		if ce.extIdx > 0 {
			ce.extIdx--
		}
	case "a":
		ce.externalAPIs = append(ce.externalAPIs, manifest.ExternalAPIDef{})
		ce.extIdx = len(ce.externalAPIs) - 1
		ce.extForm = defaultExternalAPIFormFields()
		ce.extFormIdx = 0
		ce.extSubView = ceViewForm
		return ce.tryEnterInsert()
	case "d":
		if n > 0 {
			ce.externalAPIs = append(ce.externalAPIs[:ce.extIdx], ce.externalAPIs[ce.extIdx+1:]...)
			if ce.extIdx > 0 && ce.extIdx >= len(ce.externalAPIs) {
				ce.extIdx = len(ce.externalAPIs) - 1
			}
		}
	case "enter":
		if n > 0 {
			api := ce.externalAPIs[ce.extIdx]
			ce.extForm = defaultExternalAPIFormFields()
			ce.extForm = setFieldValue(ce.extForm, "provider", api.Provider)
			ce.extForm = setFieldValue(ce.extForm, "auth_mechanism", api.AuthMechanism)
			ce.extForm = setFieldValue(ce.extForm, "base_url", api.BaseURL)
			ce.extForm = setFieldValue(ce.extForm, "rate_limit", api.RateLimit)
			ce.extForm = setFieldValue(ce.extForm, "webhook_endpoint", api.WebhookEndpoint)
			ce.extForm = setFieldValue(ce.extForm, "failure_strategy", api.FailureStrategy)
			ce.extFormIdx = 0
			ce.extSubView = ceViewForm
		}
	}
	return ce, nil
}

func (ce ContractsEditor) updateExtForm(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if ce.extFormIdx < len(ce.extForm)-1 {
			ce.extFormIdx++
		}
	case "k", "up":
		if ce.extFormIdx > 0 {
			ce.extFormIdx--
		}
	case "enter", " ":
		f := &ce.extForm[ce.extFormIdx]
		if f.Kind == KindSelect {
			ce.ddOpen = true
			ce.ddOptIdx = f.SelIdx
		} else {
			return ce.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &ce.extForm[ce.extFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if ce.extForm[ce.extFormIdx].Kind == KindText {
			return ce.tryEnterInsert()
		}
	case "b", "esc":
		ce.saveExtForm()
		ce.extSubView = ceViewList
	}
	return ce, nil
}

func (ce *ContractsEditor) saveExtForm() {
	if ce.extIdx >= len(ce.externalAPIs) {
		return
	}
	api := &ce.externalAPIs[ce.extIdx]
	api.Provider = fieldGet(ce.extForm, "provider")
	api.AuthMechanism = fieldGet(ce.extForm, "auth_mechanism")
	api.BaseURL = fieldGet(ce.extForm, "base_url")
	api.RateLimit = fieldGet(ce.extForm, "rate_limit")
	api.WebhookEndpoint = fieldGet(ce.extForm, "webhook_endpoint")
	api.FailureStrategy = fieldGet(ce.extForm, "failure_strategy")
}

func (ce ContractsEditor) viewExternal(w int) []string {
	switch ce.extSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # External APIs — a: add  d: delete  Enter: edit"), "")
		if len(ce.externalAPIs) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no external APIs yet — press 'a' to add one)"))
		} else {
			for i, api := range ce.externalAPIs {
				name := api.Provider
				if name == "" {
					name = fmt.Sprintf("(api #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == ce.extIdx, "  ▶ ", name, api.AuthMechanism))
			}
		}
		return lines

	case ceViewForm:
		provider := fieldGet(ce.extForm, "provider")
		if provider == "" {
			provider = "(new external API)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(provider), "")
		lines = append(lines, renderFormFieldsWithDropdown(w, ce.extForm, ce.extFormIdx, ce.internalMode == ceInsert, ce.formInput, ce.ddOpen, ce.ddOptIdx)...)
		return lines
	}
	return nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (ce ContractsEditor) View(w, h int) string {
	ce.width = w
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Contracts — DTOs, endpoints, and API versioning"),
		"",
		renderSubTabBar(contractsTabLabels, int(ce.activeTab)),
		"",
	)

	switch ce.activeTab {
	case contractsTabDTOs:
		lines = append(lines, ce.viewDTOs(w)...)
	case contractsTabEndpoints:
		lines = append(lines, ce.viewEndpoints(w)...)
	case contractsTabVersioning:
		lines = append(lines, ce.viewVersioning(w)...)
	case contractsTabExternal:
		lines = append(lines, ce.viewExternal(w)...)
	}

	return fillTildes(lines, h)
}

func (ce ContractsEditor) viewDTOs(w int) []string {
	switch ce.dtoSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # DTOs — a: add  d: delete  Enter: edit"), "")
		if len(ce.dtos) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no DTOs yet — press 'a' to add one)"))
		} else {
			for i, dto := range ce.dtos {
				cat := dto.Category
				lines = append(lines, renderListItem(w, i == ce.dtoIdx, "  ▶ ", dto.Name, cat))
			}
		}
		return lines

	case ceViewForm:
		name := fieldGet(ce.dtoForm, "name")
		if name == "" {
			name = "(new DTO)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFieldsWithDropdown(w, ce.dtoForm, ce.dtoFormIdx, ce.internalMode == ceInsert, ce.formInput, ce.ddOpen, ce.ddOptIdx)...)
		lines = append(lines, "", StyleSectionDesc.Render(fmt.Sprintf("  F: edit fields  (%d field(s))", len(ce.dtoFieldItems))))
		return lines

	case ceViewSubList:
		var lines []string
		dtoName := ""
		if ce.dtoIdx < len(ce.dtos) {
			dtoName = ce.dtos[ce.dtoIdx].Name
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(dtoName)+StyleSectionDesc.Render(" › Fields"), "")
		if len(ce.dtoFieldItems) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no fields — press 'a' to add)"))
		} else {
			for i, item := range ce.dtoFieldItems {
				fname := fieldGet(item, "name")
				ftype := fieldGet(item, "type")
				req := fieldGet(item, "required")
				extra := ftype
				if req == "true" {
					extra += " *required"
				}
				lines = append(lines, renderListItem(w, i == ce.dtoFieldIdx, "  ▶ ", fname, extra))
			}
		}
		return lines

	case ceViewSubForm:
		fname := fieldGet(ce.dtoFieldForm, "name")
		if fname == "" {
			fname = "(new field)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(fname), "")
		lines = append(lines, renderFormFieldsWithDropdown(w, ce.dtoFieldForm, ce.dtoFieldFormIdx, ce.internalMode == ceInsert, ce.formInput, ce.ddOpen, ce.ddOptIdx)...)
		return lines
	}
	return nil
}

func (ce ContractsEditor) viewEndpoints(w int) []string {
	switch ce.epSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Endpoints — a: add  d: delete  Enter: edit"), "")
		if len(ce.endpoints) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no endpoints yet — press 'a' to add one)"))
		} else {
			for i, ep := range ce.endpoints {
				proto := ep.Protocol
				if proto == "" {
					proto = "?"
				}
				name := ep.NamePath
				if name == "" {
					name = fmt.Sprintf("(endpoint #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == ce.epIdx, "  ▶ ", name, ep.ServiceUnit+" / "+proto))
			}
		}
		return lines

	case ceViewForm:
		visible := ce.visibleEPFields()
		title := fieldGet(ce.epForm, "name_path")
		if title == "" {
			title = "(new endpoint)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(title), "")
		lines = append(lines, renderFormFieldsWithDropdown(w, visible, ce.epFormIdx, ce.internalMode == ceInsert, ce.formInput, ce.ddOpen, ce.ddOptIdx)...)
		return lines
	}
	return nil
}

func (ce ContractsEditor) viewVersioning(w int) []string {
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  # API Versioning"), "")
	lines = append(lines, renderFormFieldsWithDropdown(w, ce.versioningFields, ce.verFormIdx, ce.internalMode == ceInsert, ce.formInput, ce.ddOpen, ce.ddOptIdx)...)
	return lines
}

// Expose endpoint names for cross-reference in other editors.
func (ce ContractsEditor) EndpointNames() []string {
	names := make([]string, len(ce.endpoints))
	for i, ep := range ce.endpoints {
		names[i] = ep.NamePath
	}
	return names
}

// DTONames returns the names of all DTOs for cross-reference.
func (ce ContractsEditor) DTONames() []string {
	names := make([]string, len(ce.dtos))
	for i, dto := range ce.dtos {
		names[i] = dto.Name
	}
	return names
}

