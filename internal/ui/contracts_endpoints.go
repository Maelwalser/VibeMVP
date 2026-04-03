package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

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
		ce.epForm = defaultEndpointFormFields(ce.availableServices, ce.dtoNames(), ce.availableAuthRoles)
		existing := make([]string, 0, len(ce.endpoints)-1)
		for i, ep := range ce.endpoints {
			if i != ce.epIdx {
				existing = append(existing, ep.NamePath)
			}
		}
		ce.epForm = setFieldValue(ce.epForm, "name_path", uniqueName("endpoint", existing))
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
			ce.epForm = defaultEndpointFormFields(ce.availableServices, ce.dtoNames(), ce.availableAuthRoles)
			ce.epForm = setFieldValue(ce.epForm, "service_unit", ep.ServiceUnit)
			ce.epForm = setFieldValue(ce.epForm, "name_path", ep.NamePath)
			if ep.Protocol != "" {
				ce.epForm = setFieldValue(ce.epForm, "protocol", ep.Protocol)
			}
			ce.epForm = setFieldValue(ce.epForm, "auth_required", ep.AuthRequired)
			if ep.AuthRoles != "" {
				for i := range ce.epForm {
					if ce.epForm[i].Key != "auth_roles" {
						continue
					}
					for _, sel := range strings.Split(ep.AuthRoles, ", ") {
						for j, opt := range ce.epForm[i].Options {
							if opt == strings.TrimSpace(sel) {
								ce.epForm[i].SelectedIdxs = append(ce.epForm[i].SelectedIdxs, j)
							}
						}
					}
					break
				}
			}
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
			if f != nil && (f.Kind == KindSelect || f.Kind == KindMultiSelect) {
				ce.dd.Open = true
				if f.Kind == KindMultiSelect {
					ce.dd.OptIdx = f.DDCursor
				} else {
					ce.dd.OptIdx = f.SelIdx
				}
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
			if f != nil && f.CanEditAsText() {
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
	ep.AuthRoles = fieldGetMulti(ce.epForm, "auth_roles")
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
	if !ce.versioningEnabled {
		if key.String() == "a" {
			ce.versioningEnabled = true
			ce.verFormIdx = 0
		}
		return ce, nil
	}
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
			ce.dd.Open = true
			ce.dd.OptIdx = f.SelIdx
		} else {
			return ce.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &ce.versioningFields[ce.verFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "D":
		ce.versioningEnabled = false
		ce.versioningFields = defaultVersioningFields()
		ce.verFormIdx = 0
	case "i", "a":
		if ce.versioningFields[ce.verFormIdx].CanEditAsText() {
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
		ce.extForm = defaultExternalAPIFormFields(ce.dtoNames())
		existing := make([]string, 0, len(ce.externalAPIs)-1)
		for i, api := range ce.externalAPIs {
			if i != ce.extIdx {
				existing = append(existing, api.Provider)
			}
		}
		ce.extForm = setFieldValue(ce.extForm, "provider", uniqueName("api", existing))
		ce.refreshExtDTOOptions()
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
			ce.extForm = defaultExternalAPIFormFields(ce.dtoNames())
			ce.extForm = setFieldValue(ce.extForm, "provider", api.Provider)
			ce.extForm = setFieldValue(ce.extForm, "responsibility", api.Responsibility)
			if api.Protocol != "" {
				ce.extForm = setFieldValue(ce.extForm, "protocol", api.Protocol)
			}
			ce.extForm = setFieldValue(ce.extForm, "auth_mechanism", api.AuthMechanism)
			ce.extForm = setFieldValue(ce.extForm, "failure_strategy", api.FailureStrategy)
			ce.extForm = setFieldValue(ce.extForm, "request_dto", api.RequestDTO)
			ce.extForm = setFieldValue(ce.extForm, "response_dto", api.ResponseDTO)
			// REST / shared
			ce.extForm = setFieldValue(ce.extForm, "base_url", api.BaseURL)
			if api.HTTPMethod != "" {
				ce.extForm = setFieldValue(ce.extForm, "http_method", api.HTTPMethod)
			}
			if api.ContentType != "" {
				ce.extForm = setFieldValue(ce.extForm, "content_type", api.ContentType)
			}
			ce.extForm = setFieldValue(ce.extForm, "rate_limit", api.RateLimit)
			ce.extForm = setFieldValue(ce.extForm, "webhook_endpoint", api.WebhookEndpoint)
			// GraphQL
			if api.GQLOperation != "" {
				ce.extForm = setFieldValue(ce.extForm, "gql_operation", api.GQLOperation)
			}
			// gRPC
			if api.GRPCStreamType != "" {
				ce.extForm = setFieldValue(ce.extForm, "grpc_stream_type", api.GRPCStreamType)
			}
			if api.TLSMode != "" {
				ce.extForm = setFieldValue(ce.extForm, "tls_mode", api.TLSMode)
			}
			// WebSocket
			ce.extForm = setFieldValue(ce.extForm, "ws_subprotocol", api.WSSubprotocol)
			if api.MessageFormat != "" {
				ce.extForm = setFieldValue(ce.extForm, "message_format", api.MessageFormat)
			}
			// Webhook
			if api.HMACHeader != "" {
				ce.extForm = setFieldValue(ce.extForm, "hmac_header", api.HMACHeader)
			}
			if api.RetryPolicy != "" {
				ce.extForm = setFieldValue(ce.extForm, "retry_policy", api.RetryPolicy)
			}
			// SOAP
			if api.SOAPVersion != "" {
				ce.extForm = setFieldValue(ce.extForm, "soap_version", api.SOAPVersion)
			}
			// Filter DTO options to match the saved protocol.
			ce.refreshExtDTOOptions()
			ce.extFormIdx = 0
			ce.extSubView = ceViewForm
		}
	}
	return ce, nil
}

func (ce ContractsEditor) updateExtForm(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	visible := ce.visibleExtFormFields()
	n := len(visible)
	switch key.String() {
	case "j", "down":
		if ce.extFormIdx < n-1 {
			ce.extFormIdx++
		}
	case "k", "up":
		if ce.extFormIdx > 0 {
			ce.extFormIdx--
		}
	case "enter", " ":
		if ce.extFormIdx < n {
			f := ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
			if f != nil && (f.Kind == KindSelect || f.Kind == KindMultiSelect) {
				ce.dd.Open = true
				if f.Kind == KindMultiSelect {
					ce.dd.OptIdx = f.DDCursor
				} else {
					ce.dd.OptIdx = f.SelIdx
				}
			} else {
				return ce.tryEnterInsert()
			}
		}
	case "H", "shift+left":
		if ce.extFormIdx < n {
			f := ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
			if f != nil && f.Kind == KindSelect {
				f.CyclePrev()
				ce.updateExtDependentFields()
			}
		}
	case "i", "a":
		if ce.extFormIdx < n {
			f := ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
			if f != nil && f.CanEditAsText() {
				return ce.tryEnterInsert()
			}
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
	api.Responsibility = fieldGet(ce.extForm, "responsibility")
	api.Protocol = fieldGet(ce.extForm, "protocol")
	api.AuthMechanism = fieldGet(ce.extForm, "auth_mechanism")
	api.FailureStrategy = fieldGet(ce.extForm, "failure_strategy")
	api.RequestDTO = fieldGet(ce.extForm, "request_dto")
	api.ResponseDTO = fieldGet(ce.extForm, "response_dto")
	// REST / shared
	api.BaseURL = fieldGet(ce.extForm, "base_url")
	api.HTTPMethod = fieldGet(ce.extForm, "http_method")
	api.ContentType = fieldGet(ce.extForm, "content_type")
	api.RateLimit = fieldGet(ce.extForm, "rate_limit")
	api.WebhookEndpoint = fieldGet(ce.extForm, "webhook_endpoint")
	// GraphQL
	api.GQLOperation = fieldGet(ce.extForm, "gql_operation")
	// gRPC
	api.GRPCStreamType = fieldGet(ce.extForm, "grpc_stream_type")
	api.TLSMode = fieldGet(ce.extForm, "tls_mode")
	// WebSocket
	api.WSSubprotocol = fieldGet(ce.extForm, "ws_subprotocol")
	api.MessageFormat = fieldGet(ce.extForm, "message_format")
	// Webhook
	api.HMACHeader = fieldGet(ce.extForm, "hmac_header")
	api.RetryPolicy = fieldGet(ce.extForm, "retry_policy")
	// SOAP
	api.SOAPVersion = fieldGet(ce.extForm, "soap_version")
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
				subtitle := api.Protocol
				if subtitle == "" {
					subtitle = "REST"
				}
				if api.AuthMechanism != "" {
					subtitle += " · " + api.AuthMechanism
				}
				lines = append(lines, renderListItem(w, i == ce.extIdx, "  ▶ ", name, subtitle))
			}
		}
		return lines

	case ceViewForm:
		provider := fieldGet(ce.extForm, "provider")
		if provider == "" {
			provider = "(new external API)"
		}
		proto := fieldGet(ce.extForm, "protocol")
		if proto == "" {
			proto = "REST"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(provider)+" "+StyleSectionDesc.Render("["+proto+"]"), "")
		visible := ce.visibleExtFormFields()
		lines = append(lines, renderFormFields(w, visible, ce.extFormIdx, ce.internalMode == ModeInsert, ce.formInput, ce.dd.Open, ce.dd.OptIdx)...)
		return lines
	}
	return nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (ce ContractsEditor) View(w, h int) string {
	ce.width = w
	ce.formInput.Width = w - 22
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Contracts — DTOs, endpoints, and API versioning"),
		"",
		renderSubTabBar(contractsTabLabels, int(ce.activeTab), w),
		"",
	)
	const ceHeaderH = 4

	switch ce.activeTab {
	case contractsTabDTOs:
		dtoLines := ce.viewDTOs(w)
		switch ce.dtoSubView {
		case ceViewList:
			dtoLines = appendViewport(dtoLines, 2, ce.dtoIdx, h-ceHeaderH)
		case ceViewForm:
			dtoLines = appendViewport(dtoLines, 2, ce.dtoFormIdx, h-ceHeaderH)
		case ceViewSubList:
			dtoLines = appendViewport(dtoLines, 2, ce.dtoFieldIdx, h-ceHeaderH)
		case ceViewSubForm:
			dtoLines = appendViewport(dtoLines, 2, ce.dtoFieldFormIdx, h-ceHeaderH)
		}
		lines = append(lines, dtoLines...)
	case contractsTabEndpoints:
		epLines := ce.viewEndpoints(w)
		switch ce.epSubView {
		case ceViewList:
			epLines = appendViewport(epLines, 2, ce.epIdx, h-ceHeaderH)
		case ceViewForm:
			epLines = appendViewport(epLines, 2, ce.epFormIdx, h-ceHeaderH)
		}
		lines = append(lines, epLines...)
	case contractsTabVersioning:
		verLines := ce.viewVersioning(w)
		if ce.versioningEnabled {
			verLines = appendViewport(verLines, 2, ce.verFormIdx, h-ceHeaderH)
		}
		lines = append(lines, verLines...)
	case contractsTabExternal:
		extLines := ce.viewExternal(w)
		switch ce.extSubView {
		case ceViewList:
			extLines = appendViewport(extLines, 2, ce.extIdx, h-ceHeaderH)
		case ceViewForm:
			extLines = appendViewport(extLines, 2, ce.extFormIdx, h-ceHeaderH)
		}
		lines = append(lines, extLines...)
	}

	return fillTildes(lines, h)
}

