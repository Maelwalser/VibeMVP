package ui

import (
	"fmt"
	"strings"

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
			Value:   placeholderFor(domainOptions, "(no domains configured)"),
		},
		{Key: "description", Label: "description   ", Kind: KindText},
		{
			Key: "protocol", Label: "protocol      ", Kind: KindSelect,
			Options: []string{"REST/JSON", "Protobuf", "Avro", "MessagePack", "Thrift", "FlatBuffers", "Cap'n Proto"},
			Value:   "REST/JSON",
		},
		// ── Protobuf-specific ────────────────────────────────────────────────────
		{Key: "proto_package", Label: "proto_package ", Kind: KindText},
		{
			Key: "proto_syntax", Label: "proto_syntax  ", Kind: KindSelect,
			Options: []string{"proto3", "proto2"}, Value: "proto3",
		},
		{Key: "proto_options", Label: "proto_options ", Kind: KindText},
		// ── Avro-specific ────────────────────────────────────────────────────────
		{Key: "avro_namespace", Label: "avro_namespace", Kind: KindText},
		{Key: "schema_registry", Label: "schema_reg    ", Kind: KindText},
		// ── Thrift-specific ──────────────────────────────────────────────────────
		{Key: "thrift_namespace", Label: "thrift_ns     ", Kind: KindText},
		{
			Key: "thrift_language", Label: "thrift_lang   ", Kind: KindSelect,
			Options: []string{"go", "java", "python", "cpp", "js", "php", "ruby"},
			Value:   "go",
		},
		// ── FlatBuffers / Cap'n Proto ────────────────────────────────────────────
		{Key: "namespace", Label: "namespace     ", Kind: KindText},
	}
}

// typeOptionsForDTOProtocol returns the native types for a given DTO serialisation protocol.
func typeOptionsForDTOProtocol(proto string) []string {
	switch proto {
	case "Protobuf":
		return []string{
			"string", "bool", "bytes",
			"int32", "int64", "uint32", "uint64", "sint32", "sint64",
			"fixed32", "fixed64", "sfixed32", "sfixed64",
			"float", "double",
			"enum", "message", "repeated", "map", "oneof",
			"google.Any", "google.Timestamp", "google.Duration",
		}
	case "Avro":
		return []string{
			"null", "boolean", "int", "long", "float", "double",
			"bytes", "string",
			"record", "enum", "array", "map", "union", "fixed",
		}
	case "MessagePack":
		return []string{
			"string", "int", "float", "bool", "binary",
			"array", "map", "nil", "timestamp", "ext",
		}
	case "Thrift":
		return []string{
			"bool", "byte", "i16", "i32", "i64", "double",
			"string", "binary",
			"list", "set", "map", "enum", "struct", "void",
		}
	case "FlatBuffers":
		return []string{
			"bool",
			"int8", "int16", "int32", "int64",
			"uint8", "uint16", "uint32", "uint64",
			"float32", "float64",
			"string", "[type]", "struct", "table", "enum", "union",
		}
	case "Cap'n Proto":
		return []string{
			"Bool",
			"Int8", "Int16", "Int32", "Int64",
			"UInt8", "UInt16", "UInt32", "UInt64",
			"Float32", "Float64",
			"Text", "Data",
			"List", "Struct", "Enum", "Union", "AnyPointer",
		}
	default: // REST/JSON
		return []string{
			"string", "int", "float", "boolean", "datetime",
			"uuid", "enum(values)", "array(type)", "nested(DTO)", "map(key,value)",
		}
	}
}

func defaultDTOFieldForm(protocol string) []Field {
	typeOpts := typeOptionsForDTOProtocol(protocol)
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{
			Key: "type", Label: "type          ", Kind: KindSelect,
			Options: typeOpts, Value: typeOpts[0],
		},
		// ── REST/JSON · MessagePack · Avro ───────────────────────────────────────
		{
			Key: "required", Label: "required      ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{
			Key: "nullable", Label: "nullable      ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{
			Key: "validation", Label: "validation    ", Kind: KindMultiSelect,
			Options: []string{
				"required", "min_length", "max_length", "min_value", "max_value",
				"email", "url", "regex", "uuid", "enum", "phone", "pattern", "custom",
			},
		},
		// ── Default value (Avro, Thrift, FlatBuffers, Cap'n Proto, REST/JSON) ───
		{Key: "default", Label: "default       ", Kind: KindText},
		// ── Protobuf-specific ────────────────────────────────────────────────────
		{Key: "field_number", Label: "field_number  ", Kind: KindText},
		{
			Key: "proto_modifier", Label: "proto_modifier", Kind: KindSelect,
			Options: []string{"optional", "repeated", "oneof"}, Value: "optional",
		},
		{Key: "json_name", Label: "json_name     ", Kind: KindText},
		// ── Thrift / Cap'n Proto ─────────────────────────────────────────────────
		{Key: "field_id", Label: "field_id      ", Kind: KindText},
		// ── Thrift-specific ──────────────────────────────────────────────────────
		{
			Key: "thrift_mod", Label: "thrift_mod    ", Kind: KindSelect,
			Options: []string{"required", "optional", "default"}, Value: "optional",
		},
		// ── FlatBuffers-specific ─────────────────────────────────────────────────
		{
			Key: "deprecated", Label: "deprecated    ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{Key: "notes", Label: "notes         ", Kind: KindText},
	}
}

// refreshDTOFieldTypeOptions updates the type field options in a field form to match
// the given protocol, preserving the current value when possible.
func refreshDTOFieldTypeOptions(form []Field, protocol string) []Field {
	opts := typeOptionsForDTOProtocol(protocol)
	for i := range form {
		if form[i].Key != "type" {
			continue
		}
		cur := form[i].DisplayValue()
		form[i].Options = opts
		form[i].SelIdx = 0
		form[i].Value = opts[0]
		for j, t := range opts {
			if t == cur {
				form[i].SelIdx = j
				form[i].Value = t
				break
			}
		}
		break
	}
	return form
}

// currentDTOProtocol returns the serialisation protocol selected in the DTO form.
func (ce ContractsEditor) currentDTOProtocol() string {
	proto := fieldGet(ce.dtoForm, "protocol")
	if proto == "" {
		return "REST/JSON"
	}
	return proto
}

// visibleDTOFieldFormFields returns only the field-form fields relevant to the
// current DTO protocol, hiding inapplicable options.
func (ce ContractsEditor) visibleDTOFieldFormFields() []Field {
	proto := ce.currentDTOProtocol()
	var visible []Field
	for _, f := range ce.dtoFieldForm {
		switch f.Key {
		case "required", "nullable":
			if proto != "REST/JSON" && proto != "MessagePack" && proto != "Avro" {
				continue
			}
		case "validation":
			if proto != "REST/JSON" && proto != "MessagePack" {
				continue
			}
		case "default":
			if proto == "Protobuf" {
				continue
			}
		case "field_number", "proto_modifier", "json_name":
			if proto != "Protobuf" {
				continue
			}
		case "field_id":
			if proto != "Thrift" && proto != "Cap'n Proto" {
				continue
			}
		case "thrift_mod":
			if proto != "Thrift" {
				continue
			}
		case "deprecated":
			if proto != "FlatBuffers" {
				continue
			}
		}
		visible = append(visible, f)
	}
	return visible
}

// dtoFieldFormFieldByKey returns a pointer to the field-form field with the given key.
func (ce *ContractsEditor) dtoFieldFormFieldByKey(key string) *Field {
	for i := range ce.dtoFieldForm {
		if ce.dtoFieldForm[i].Key == key {
			return &ce.dtoFieldForm[i]
		}
	}
	return nil
}

func defaultEndpointFormFields(serviceOptions, dtoOptions, roleOptions []string) []Field {
	// Ensure at least empty slice so KindSelect works
	if serviceOptions == nil {
		serviceOptions = []string{}
	}
	if dtoOptions == nil {
		dtoOptions = []string{}
	}
	if roleOptions == nil {
		roleOptions = []string{}
	}
	fields := []Field{
		{Key: "service_unit", Label: "service_unit  ", Kind: KindSelect,
			Options: serviceOptions,
			Value:   placeholderFor(serviceOptions, "(no services configured)"),
		},
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
		{Key: "auth_roles", Label: "auth_roles    ", Kind: KindMultiSelect,
			Options: roleOptions,
			Value:   placeholderFor(roleOptions, "(no roles configured)"),
		},
		{Key: "request_dto", Label: "request_dto   ", Kind: KindSelect,
			Options: dtoOptions,
			Value:   placeholderFor(dtoOptions, "(no DTOs configured)"),
		},
		{Key: "response_dto", Label: "response_dto  ", Kind: KindSelect,
			Options: dtoOptions,
			Value:   placeholderFor(dtoOptions, "(no DTOs configured)"),
		},
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

func defaultExternalAPIFormFields(dtoOptions []string) []Field {
	if dtoOptions == nil {
		dtoOptions = []string{}
	}
	return []Field{
		// ── Common ──────────────────────────────────────────────────────────────
		{Key: "provider", Label: "provider      ", Kind: KindText},
		{
			Key: "protocol", Label: "protocol      ", Kind: KindSelect,
			Options: []string{"REST", "GraphQL", "gRPC", "WebSocket", "Webhook", "SOAP"},
			Value:   "REST",
		},
		{
			Key: "auth_mechanism", Label: "auth_mechanism", Kind: KindSelect,
			Options: []string{"API Key", "OAuth2 Client Credentials", "OAuth2 PKCE", "Bearer Token", "Basic Auth", "mTLS", "None"},
			Value:   "API Key",
		},
		{
			Key: "failure_strategy", Label: "failure_strat ", Kind: KindSelect,
			Options: []string{"Circuit Breaker", "Retry with backoff", "Fallback value", "Timeout + fail", "None"},
			Value:   "Circuit Breaker",
		},
		{Key: "request_dto", Label: "request_dto   ", Kind: KindSelect,
			Options: dtoOptions,
			Value:   placeholderFor(dtoOptions, "(no DTOs configured)"),
		},
		{Key: "response_dto", Label: "response_dto  ", Kind: KindSelect,
			Options: dtoOptions,
			Value:   placeholderFor(dtoOptions, "(no DTOs configured)"),
		},

		// ── REST ────────────────────────────────────────────────────────────────
		{Key: "base_url", Label: "base_url      ", Kind: KindText},
		{
			Key: "http_method", Label: "http_method   ", Kind: KindSelect,
			Options: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "Any"},
			Value:   "GET",
		},
		{
			Key: "content_type", Label: "content_type  ", Kind: KindSelect,
			Options: []string{"JSON", "XML", "Form Data", "Multipart"},
			Value:   "JSON",
		},
		{Key: "rate_limit", Label: "rate_limit    ", Kind: KindText},
		{Key: "webhook_endpoint", Label: "webhook_path  ", Kind: KindText},

		// ── GraphQL ─────────────────────────────────────────────────────────────
		// base_url shared with REST
		{
			Key: "gql_operation", Label: "gql_operation ", Kind: KindSelect,
			Options: []string{"Query", "Mutation", "Subscription"},
			Value:   "Query",
		},
		// rate_limit shared with REST

		// ── gRPC ────────────────────────────────────────────────────────────────
		// base_url shared with REST (used as endpoint address)
		{
			Key: "grpc_stream_type", Label: "stream_type   ", Kind: KindSelect,
			Options: []string{"Unary", "Server streaming", "Client streaming", "Bidirectional"},
			Value:   "Unary",
		},
		{
			Key: "tls_mode", Label: "tls_mode      ", Kind: KindSelect,
			Options: []string{"TLS", "mTLS", "Insecure"},
			Value:   "TLS",
		},

		// ── WebSocket ───────────────────────────────────────────────────────────
		// base_url shared (ws:// / wss://)
		{Key: "ws_subprotocol", Label: "subprotocol   ", Kind: KindText},
		{
			Key: "message_format", Label: "message_format", Kind: KindSelect,
			Options: []string{"JSON", "MessagePack", "Binary", "Text"},
			Value:   "JSON",
		},

		// ── Webhook (inbound) ───────────────────────────────────────────────────
		// webhook_endpoint shared with REST (our inbound path)
		{Key: "hmac_header", Label: "hmac_header   ", Kind: KindText, Value: "X-Hub-Signature-256"},
		{
			Key: "retry_policy", Label: "retry_policy  ", Kind: KindSelect,
			Options: []string{"Retry 3x", "Retry 5x", "Immediate fail", "None"},
			Value:   "Retry 3x",
		},

		// ── SOAP ────────────────────────────────────────────────────────────────
		// base_url shared (WSDL URL)
		{
			Key: "soap_version", Label: "soap_version  ", Kind: KindSelect,
			Options: []string{"1.1", "1.2"},
			Value:   "1.1",
		},
	}
}

// visibleExtFormFields returns only the fields relevant to the currently
// selected protocol, hiding all others.
func (ce ContractsEditor) visibleExtFormFields() []Field {
	if len(ce.extForm) == 0 {
		return nil
	}
	proto := fieldGet(ce.extForm, "protocol")
	var visible []Field
	for _, f := range ce.extForm {
		switch f.Key {
		// REST-only
		case "http_method", "content_type":
			if proto != "REST" {
				continue
			}
		case "rate_limit":
			if proto != "REST" && proto != "GraphQL" {
				continue
			}
		case "webhook_endpoint":
			if proto != "REST" && proto != "Webhook" {
				continue
			}
		// GraphQL-only
		case "gql_operation":
			if proto != "GraphQL" {
				continue
			}
		// gRPC-only
		case "grpc_stream_type", "tls_mode":
			if proto != "gRPC" {
				continue
			}
		// WebSocket-only
		case "ws_subprotocol", "message_format":
			if proto != "WebSocket" {
				continue
			}
		// Webhook-only
		case "hmac_header", "retry_policy":
			if proto != "Webhook" {
				continue
			}
		// SOAP-only
		case "soap_version":
			if proto != "SOAP" {
				continue
			}
		// base_url: shown for REST, GraphQL, gRPC, WebSocket, SOAP (not Webhook)
		case "base_url":
			if proto == "Webhook" {
				continue
			}
		}
		visible = append(visible, f)
	}
	return visible
}

// extFormFieldByKey returns a pointer to the ext form field with the given key.
func (ce *ContractsEditor) extFormFieldByKey(key string) *Field {
	for i := range ce.extForm {
		if ce.extForm[i].Key == key {
			return &ce.extForm[i]
		}
	}
	return nil
}

// updateExtDependentFields refreshes protocol-filtered DTO options and clamps
// extFormIdx to the visible field range after a protocol change.
func (ce *ContractsEditor) updateExtDependentFields() {
	ce.refreshExtDTOOptions()
	visible := ce.visibleExtFormFields()
	if len(visible) > 0 && ce.extFormIdx >= len(visible) {
		ce.extFormIdx = len(visible) - 1
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

// SetAuthRoles updates the auth role options used in endpoint forms.
func (ce *ContractsEditor) SetAuthRoles(roles []string) {
	ce.availableAuthRoles = roles
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

// visibleDTOFields returns only the DTO form fields relevant to the selected
// serialisation protocol, hiding the other protocol-specific fields.
func (ce ContractsEditor) visibleDTOFields() []Field {
	if len(ce.dtoForm) == 0 {
		return nil
	}
	proto := fieldGet(ce.dtoForm, "protocol")
	var visible []Field
	for _, f := range ce.dtoForm {
		switch f.Key {
		case "proto_package", "proto_syntax", "proto_options":
			if proto != "Protobuf" {
				continue
			}
		case "avro_namespace", "schema_registry":
			if proto != "Avro" {
				continue
			}
		case "thrift_namespace", "thrift_language":
			if proto != "Thrift" {
				continue
			}
		case "namespace":
			if proto != "FlatBuffers" && proto != "Cap'n Proto" {
				continue
			}
		}
		visible = append(visible, f)
	}
	return visible
}

// dtoFormFieldByKey returns a pointer to the DTO form field with the given key.
func (ce *ContractsEditor) dtoFormFieldByKey(key string) *Field {
	for i := range ce.dtoForm {
		if ce.dtoForm[i].Key == key {
			return &ce.dtoForm[i]
		}
	}
	return nil
}

// updateDTODependentFields clamps dtoFormIdx to the visible field range after a
// protocol change so the cursor never lands on a hidden field.
func (ce *ContractsEditor) updateDTODependentFields() {
	if ce.activeTab != contractsTabDTOs || ce.dtoSubView != ceViewForm {
		return
	}
	visible := ce.visibleDTOFields()
	if len(visible) > 0 && ce.dtoFormIdx >= len(visible) {
		ce.dtoFormIdx = len(visible) - 1
	}
}

// visibleEPFields returns only the endpoint form fields relevant to the
// currently selected protocol and auth setting.
func (ce ContractsEditor) visibleEPFields() []Field {
	if len(ce.epForm) == 0 {
		return nil
	}
	proto := fieldGet(ce.epForm, "protocol")
	authRequired := fieldGet(ce.epForm, "auth_required")
	var visible []Field
	for _, f := range ce.epForm {
		switch f.Key {
		case "auth_roles":
			if authRequired != "true" {
				continue
			}
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
			if f.PrepareCustomEntry() {
				ce.updateEPDependentFields()
				return ce.tryEnterInsert()
			}
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
		if f.Kind == KindSelect && f.PrepareCustomEntry() {
			ce.updateEPDependentFields()
			return ce.tryEnterInsert()
		}
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = ce.ddOptIdx
		}
		ce.ddOpen = false
	}
	// After any dropdown interaction, refresh dependent fields for DTO, EP, and ext forms.
	ce.updateDTODependentFields()
	ce.updateEPDependentFields()
	ce.updateExtDependentFields()
	return ce, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (ce ContractsEditor) Update(msg tea.Msg) (ContractsEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		ce.width = wsz.Width
		ce.formInput.Width = wsz.Width - 22
		return ce, nil
	}
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
			ce.internalMode = ceInsert
			ce.formInput.SetValue(f.TextInputValue())
			ce.formInput.Width = ce.width - 22
			ce.formInput.CursorEnd()
			return ce, ce.formInput.Focus()
		}
		ce.advanceField(1)
	}
	return ce, nil
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
		existing := make([]string, 0, len(ce.dtos)-1)
		for i, d := range ce.dtos {
			if i != ce.dtoIdx {
				existing = append(existing, d.Name)
			}
		}
		ce.dtoForm = setFieldValue(ce.dtoForm, "name", uniqueName("dto", existing))
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
			if d.Protocol != "" {
				ce.dtoForm = setFieldValue(ce.dtoForm, "protocol", d.Protocol)
			}
			ce.dtoForm = setFieldValue(ce.dtoForm, "proto_package", d.ProtoPackage)
			ce.dtoForm = setFieldValue(ce.dtoForm, "proto_syntax", d.ProtoSyntax)
			ce.dtoForm = setFieldValue(ce.dtoForm, "proto_options", d.ProtoOptions)
			ce.dtoForm = setFieldValue(ce.dtoForm, "avro_namespace", d.AvroNamespace)
			ce.dtoForm = setFieldValue(ce.dtoForm, "schema_registry", d.SchemaRegistry)
			ce.dtoForm = setFieldValue(ce.dtoForm, "thrift_namespace", d.ThriftNamespace)
			if d.ThriftLanguage != "" {
				ce.dtoForm = setFieldValue(ce.dtoForm, "thrift_language", d.ThriftLanguage)
			}
			ce.dtoForm = setFieldValue(ce.dtoForm, "namespace", d.Namespace)
			ce.dtoFormIdx = 0
			// Rebuild field items
			proto := d.Protocol
			if proto == "" {
				proto = "REST/JSON"
			}
			ce.dtoFieldItems = make([][]Field, len(d.Fields))
			for i, df := range d.Fields {
				f := defaultDTOFieldForm(proto)
				f = setFieldValue(f, "name", df.Name)
				f = setFieldValue(f, "type", df.Type)
				if df.Required {
					f = setFieldValue(f, "required", "true")
				}
				if df.Nullable {
					f = setFieldValue(f, "nullable", "true")
				}
				f = restoreMultiSelectValue(f, "validation", df.Validation)
				f = setFieldValue(f, "default", df.Default)
				f = setFieldValue(f, "field_number", df.FieldNumber)
				if df.ProtoModifier != "" {
					f = setFieldValue(f, "proto_modifier", df.ProtoModifier)
				}
				f = setFieldValue(f, "json_name", df.JsonName)
				f = setFieldValue(f, "field_id", df.FieldID)
				if df.ThriftModifier != "" {
					f = setFieldValue(f, "thrift_mod", df.ThriftModifier)
				}
				if df.Deprecated {
					f = setFieldValue(f, "deprecated", "true")
				}
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
	visible := ce.visibleDTOFields()
	switch key.String() {
	case "j", "down":
		if ce.dtoFormIdx < len(visible)-1 {
			ce.dtoFormIdx++
		}
	case "k", "up":
		if ce.dtoFormIdx > 0 {
			ce.dtoFormIdx--
		}
	case "enter", " ":
		f := ce.activeCEFieldPtr()
		if f == nil {
			break
		}
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
		f := ce.activeCEFieldPtr()
		if f != nil && f.Kind == KindSelect {
			f.CyclePrev()
			ce.updateDTODependentFields()
		}
	case "i", "a":
		f := ce.activeCEFieldPtr()
		if f != nil && f.CanEditAsText() {
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
				f := defaultDTOFieldForm(ce.currentDTOProtocol())
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
	d.Protocol = fieldGet(ce.dtoForm, "protocol")
	d.ProtoPackage = fieldGet(ce.dtoForm, "proto_package")
	d.ProtoSyntax = fieldGet(ce.dtoForm, "proto_syntax")
	d.ProtoOptions = fieldGet(ce.dtoForm, "proto_options")
	d.AvroNamespace = fieldGet(ce.dtoForm, "avro_namespace")
	d.SchemaRegistry = fieldGet(ce.dtoForm, "schema_registry")
	d.ThriftNamespace = fieldGet(ce.dtoForm, "thrift_namespace")
	d.ThriftLanguage = fieldGet(ce.dtoForm, "thrift_language")
	d.Namespace = fieldGet(ce.dtoForm, "namespace")

	d.Fields = make([]manifest.DTOField, len(ce.dtoFieldItems))
	for i, item := range ce.dtoFieldItems {
		d.Fields[i] = manifest.DTOField{
			Name:           fieldGet(item, "name"),
			Type:           fieldGet(item, "type"),
			Required:       fieldGet(item, "required") == "true",
			Nullable:       fieldGet(item, "nullable") == "true",
			Validation:     fieldGetMulti(item, "validation"),
			Default:        fieldGet(item, "default"),
			FieldNumber:    fieldGet(item, "field_number"),
			ProtoModifier:  fieldGet(item, "proto_modifier"),
			JsonName:       fieldGet(item, "json_name"),
			FieldID:        fieldGet(item, "field_id"),
			ThriftModifier: fieldGet(item, "thrift_mod"),
			Deprecated:     fieldGet(item, "deprecated") == "true",
			Notes:          fieldGet(item, "notes"),
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
		ce.dtoFieldItems = append(ce.dtoFieldItems, defaultDTOFieldForm(ce.currentDTOProtocol()))
		ce.dtoFieldIdx = len(ce.dtoFieldItems) - 1
		ce.dtoFieldForm = copyFields(ce.dtoFieldItems[ce.dtoFieldIdx])
		existing := make([]string, 0, len(ce.dtoFieldItems)-1)
		for i, f := range ce.dtoFieldItems {
			if i != ce.dtoFieldIdx {
				existing = append(existing, fieldGet(f, "name"))
			}
		}
		ce.dtoFieldForm = setFieldValue(ce.dtoFieldForm, "name", uniqueName("field", existing))
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
			ce.dtoFieldForm = refreshDTOFieldTypeOptions(ce.dtoFieldForm, ce.currentDTOProtocol())
			ce.dtoFieldFormIdx = 0
			ce.dtoSubView = ceViewSubForm
		}
	case "b", "esc":
		ce.dtoSubView = ceViewForm
	}
	return ce, nil
}

func (ce ContractsEditor) updateDTOFieldForm(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	visible := ce.visibleDTOFieldFormFields()
	switch key.String() {
	case "j", "down":
		if ce.dtoFieldFormIdx < len(visible)-1 {
			ce.dtoFieldFormIdx++
		}
	case "k", "up":
		if ce.dtoFieldFormIdx > 0 {
			ce.dtoFieldFormIdx--
		}
	case "enter", " ":
		f := ce.activeCEFieldPtr()
		if f == nil {
			break
		}
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
		f := ce.activeCEFieldPtr()
		if f != nil && f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		f := ce.activeCEFieldPtr()
		if f != nil && f.CanEditAsText() {
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
				ce.ddOpen = true
				if f.Kind == KindMultiSelect {
					ce.ddOptIdx = f.DDCursor
				} else {
					ce.ddOptIdx = f.SelIdx
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
				ce.ddOpen = true
				if f.Kind == KindMultiSelect {
					ce.ddOptIdx = f.DDCursor
				} else {
					ce.ddOptIdx = f.SelIdx
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
		lines = append(lines, renderFormFields(w, visible, ce.extFormIdx, ce.internalMode == ceInsert, ce.formInput, ce.ddOpen, ce.ddOptIdx)...)
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
		if ce.dtoSubView == ceViewList {
			dtoLines = appendViewport(dtoLines, 2, ce.dtoIdx, h-ceHeaderH)
		}
		lines = append(lines, dtoLines...)
	case contractsTabEndpoints:
		epLines := ce.viewEndpoints(w)
		if ce.epSubView == ceViewList {
			epLines = appendViewport(epLines, 2, ce.epIdx, h-ceHeaderH)
		}
		lines = append(lines, epLines...)
	case contractsTabVersioning:
		lines = append(lines, ce.viewVersioning(w)...)
	case contractsTabExternal:
		extLines := ce.viewExternal(w)
		if ce.extSubView == ceViewList {
			extLines = appendViewport(extLines, 2, ce.extIdx, h-ceHeaderH)
		}
		lines = append(lines, extLines...)
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
		lines = append(lines, renderFormFields(w, ce.visibleDTOFields(), ce.dtoFormIdx, ce.internalMode == ceInsert, ce.formInput, ce.ddOpen, ce.ddOptIdx)...)
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
		lines = append(lines, renderFormFields(w, ce.visibleDTOFieldFormFields(), ce.dtoFieldFormIdx, ce.internalMode == ceInsert, ce.formInput, ce.ddOpen, ce.ddOptIdx)...)
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
		lines = append(lines, renderFormFields(w, visible, ce.epFormIdx, ce.internalMode == ceInsert, ce.formInput, ce.ddOpen, ce.ddOptIdx)...)
		return lines
	}
	return nil
}

func (ce ContractsEditor) viewVersioning(w int) []string {
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  # API Versioning"), "")
	if !ce.versioningEnabled {
		lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		return lines
	}
	lines = append(lines, renderFormFields(w, ce.versioningFields, ce.verFormIdx, ce.internalMode == ceInsert, ce.formInput, ce.ddOpen, ce.ddOptIdx)...)
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

