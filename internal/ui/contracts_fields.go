package ui

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
			Options: OptionsOffOn, Value: "false",
		},
		{
			Key: "nullable", Label: "nullable      ", Kind: KindSelect,
			Options: OptionsOffOn, Value: "false",
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
			Options: OptionsOffOn, Value: "false",
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
			Options: OptionsOffOn, Value: "false",
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

// versioningByProtocol maps each versionable endpoint protocol to its valid
// versioning strategies.
var versioningByProtocol = map[string][]string{
	"REST":    {"URL path (/v1/)", "Header (Accept-Version)", "Query param", "None"},
	"GraphQL": {"Schema evolution", "None"},
	"gRPC":    {"Package versioning", "None"},
}

// versioningStrategyFieldKey returns the field key used to store the versioning
// strategy for a given protocol.
func versioningStrategyFieldKey(proto string) string {
	return "strategy_" + proto
}

// versioningStrategyLabel returns the label for a per-protocol strategy field.
func versioningStrategyLabel(proto string) string {
	switch proto {
	case "REST":
		return "REST strategy  "
	case "GraphQL":
		return "GraphQL strat  "
	case "gRPC":
		return "gRPC strategy  "
	default:
		return proto + " strategy "
	}
}

// defaultVersioningTailFields returns the non-strategy versioning fields
// (current version + deprecation policy) with their defaults.
func defaultVersioningTailFields() []Field {
	return []Field{
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

// defaultVersioningFields returns an initial versioning field list assuming REST
// as the only protocol. rebuildVersioningFields() replaces this with the
// correct per-protocol set once endpoints are known.
func defaultVersioningFields() []Field {
	restOpts := versioningByProtocol["REST"]
	stratField := Field{
		Key: versioningStrategyFieldKey("REST"), Label: versioningStrategyLabel("REST"),
		Kind: KindSelect, Options: restOpts, Value: restOpts[0],
	}
	return append([]Field{stratField}, defaultVersioningTailFields()...)
}

// activeEndpointProtocols returns the distinct versionable protocols (REST,
// GraphQL, gRPC) that have at least one endpoint defined, in a stable order.
func (ce ContractsEditor) activeEndpointProtocols() []string {
	seen := make(map[string]bool)
	order := []string{"REST", "GraphQL", "gRPC"}
	for _, ep := range ce.endpoints {
		switch ep.Protocol {
		case "REST", "GraphQL", "gRPC":
			seen[ep.Protocol] = true
		}
	}
	var result []string
	for _, p := range order {
		if seen[p] {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{"REST"} // sensible default when no endpoints exist
	}
	return result
}

// rebuildVersioningFields rebuilds the versioning field slice to show one
// strategy selector per active endpoint protocol, followed by the shared fields
// (current_version, deprecation). Existing values are preserved by key.
func (ce *ContractsEditor) rebuildVersioningFields() {
	// Snapshot current values so we can restore them after rebuild.
	saved := make(map[string]string, len(ce.versioningFields))
	for _, f := range ce.versioningFields {
		saved[f.Key] = f.DisplayValue()
	}

	protos := ce.activeEndpointProtocols()
	var fields []Field
	for _, proto := range protos {
		opts := versioningByProtocol[proto]
		key := versioningStrategyFieldKey(proto)
		cur, hasSaved := saved[key]
		f := Field{
			Key: key, Label: versioningStrategyLabel(proto),
			Kind: KindSelect, Options: opts, Value: opts[0],
		}
		if hasSaved {
			for j, opt := range opts {
				if opt == cur {
					f.SelIdx = j
					f.Value = opt
					break
				}
			}
		}
		fields = append(fields, f)
	}

	tail := defaultVersioningTailFields()
	// Preserve current values for tail fields.
	for i := range tail {
		if v, ok := saved[tail[i].Key]; ok && v != "" {
			tail[i].Value = v
		}
	}
	ce.versioningFields = append(fields, tail...)

	// Clamp cursor.
	if ce.verFormIdx >= len(ce.versioningFields) {
		ce.verFormIdx = len(ce.versioningFields) - 1
	}
}

func defaultExternalAPIFormFields() []Field {
	return []Field{
		// ── Common ──────────────────────────────────────────────────────────────
		{Key: "provider", Label: "provider      ", Kind: KindText},
		{Key: "responsibility", Label: "responsibility", Kind: KindText},
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

		// ── REST / GraphQL / gRPC / WebSocket / SOAP ────────────────────────────
		{Key: "base_url", Label: "base_url      ", Kind: KindText},
		{Key: "rate_limit", Label: "rate_limit    ", Kind: KindText},
		{Key: "webhook_endpoint", Label: "webhook_path  ", Kind: KindText},

		// ── gRPC ────────────────────────────────────────────────────────────────
		{
			Key: "tls_mode", Label: "tls_mode      ", Kind: KindSelect,
			Options: []string{"TLS", "mTLS", "Insecure"},
			Value:   "TLS",
		},

		// ── WebSocket ───────────────────────────────────────────────────────────
		{Key: "ws_subprotocol", Label: "subprotocol   ", Kind: KindText},
		{
			Key: "message_format", Label: "message_format", Kind: KindSelect,
			Options: []string{"JSON", "MessagePack", "Binary", "Text"},
			Value:   "JSON",
		},

		// ── Webhook (inbound) ───────────────────────────────────────────────────
		{Key: "hmac_header", Label: "hmac_header   ", Kind: KindText, Value: "X-Hub-Signature-256"},
		{
			Key: "retry_policy", Label: "retry_policy  ", Kind: KindSelect,
			Options: []string{"Retry 3x", "Retry 5x", "Immediate fail", "None"},
			Value:   "Retry 3x",
		},

		// ── SOAP ────────────────────────────────────────────────────────────────
		{
			Key: "soap_version", Label: "soap_version  ", Kind: KindSelect,
			Options: []string{"1.1", "1.2"},
			Value:   "1.1",
		},
	}
}

// defaultExtInteractionFormFields returns the form fields for a single
// interaction/call on an external API. Protocol-specific fields are filtered
// by visibleExtIntFormFields before rendering.
func defaultExtInteractionFormFields(dtoOptions []string) []Field {
	if dtoOptions == nil {
		dtoOptions = []string{}
	}
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "path", Label: "path          ", Kind: KindText},
		{Key: "request_dto", Label: "request_dto   ", Kind: KindSelect,
			Options: dtoOptions,
			Value:   placeholderFor(dtoOptions, "(no DTOs configured)"),
		},
		{Key: "response_dto", Label: "response_dto  ", Kind: KindSelect,
			Options: dtoOptions,
			Value:   placeholderFor(dtoOptions, "(no DTOs configured)"),
		},
		// ── REST ────────────────────────────────────────────────────────────────
		{
			Key: "http_method", Label: "http_method   ", Kind: KindSelect,
			Options: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			Value:   "GET",
		},
		// ── GraphQL ─────────────────────────────────────────────────────────────
		{
			Key: "gql_operation", Label: "gql_operation ", Kind: KindSelect,
			Options: []string{"Query", "Mutation", "Subscription"},
			Value:   "Query",
		},
		// ── gRPC ────────────────────────────────────────────────────────────────
		{
			Key: "grpc_stream_type", Label: "stream_type   ", Kind: KindSelect,
			Options: []string{"Unary", "Server streaming", "Client streaming", "Bidirectional"},
			Value:   "Unary",
		},
		// ── WebSocket ───────────────────────────────────────────────────────────
		{
			Key: "ws_direction", Label: "ws_direction  ", Kind: KindSelect,
			Options: []string{"Send", "Receive", "Bidirectional"},
			Value:   "Send",
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
		case "rate_limit":
			if proto != "REST" && proto != "GraphQL" {
				continue
			}
		case "webhook_endpoint":
			if proto != "REST" && proto != "Webhook" {
				continue
			}
		// gRPC-only
		case "tls_mode":
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

// visibleExtIntFormFields returns only the interaction form fields relevant to
// the parent external API's protocol.
func (ce ContractsEditor) visibleExtIntFormFields() []Field {
	if len(ce.extIntForm) == 0 {
		return nil
	}
	proto := ""
	if ce.extIdx < len(ce.externalAPIs) {
		proto = ce.externalAPIs[ce.extIdx].Protocol
	}
	if proto == "" {
		proto = "REST"
	}
	var visible []Field
	for _, f := range ce.extIntForm {
		switch f.Key {
		case "path":
			if proto == "Webhook" {
				continue
			}
		case "http_method":
			if proto != "REST" {
				continue
			}
		case "gql_operation":
			if proto != "GraphQL" {
				continue
			}
		case "grpc_stream_type":
			if proto != "gRPC" {
				continue
			}
		case "ws_direction":
			if proto != "WebSocket" {
				continue
			}
		}
		visible = append(visible, f)
	}
	return visible
}

// extIntFormFieldByKey returns a pointer to the interaction form field with the given key.
func (ce *ContractsEditor) extIntFormFieldByKey(key string) *Field {
	for i := range ce.extIntForm {
		if ce.extIntForm[i].Key == key {
			return &ce.extIntForm[i]
		}
	}
	return nil
}

// refreshExtIntDTOOptions updates the request_dto and response_dto option lists
// in the interaction form to match the parent API's protocol.
func (ce *ContractsEditor) refreshExtIntDTOOptions() {
	if ce.extIdx >= len(ce.externalAPIs) {
		return
	}
	proto := ce.externalAPIs[ce.extIdx].Protocol
	if proto == "" {
		proto = "REST"
	}
	opts := ce.dtoNamesForProtocol(proto)
	placeholder := placeholderFor(opts, "(no matching DTOs)")
	for i := range ce.extIntForm {
		key := ce.extIntForm[i].Key
		if key != "request_dto" && key != "response_dto" {
			continue
		}
		f := &ce.extIntForm[i]
		prev := f.Value
		f.Options = opts
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

// extFormFieldByKey returns a pointer to the ext form field with the given key.
func (ce *ContractsEditor) extFormFieldByKey(key string) *Field {
	for i := range ce.extForm {
		if ce.extForm[i].Key == key {
			return &ce.extForm[i]
		}
	}
	return nil
}

// failureStrategyByProtocol maps each external API protocol to its valid
// failure strategies. Circuit breaker is only relevant for synchronous
// protocols; async protocols (Webhook) use retry/DLQ semantics instead.
var failureStrategyByProtocol = map[string][]string{
	"REST":      {"Circuit Breaker", "Retry with backoff", "Fallback value", "Timeout + fail", "None"},
	"GraphQL":   {"Circuit Breaker", "Retry with backoff", "Fallback value", "Timeout + fail", "None"},
	"gRPC":      {"Circuit Breaker", "Retry with backoff", "Timeout + fail", "None"},
	"WebSocket": {"Reconnect with backoff", "Fallback value", "None"},
	"Webhook":   {"Retry with backoff", "DLQ", "None"},
	"SOAP":      {"Circuit Breaker", "Retry with backoff", "Timeout + fail", "None"},
}

// updateExtDependentFields filters failure_strategy options by protocol and
// clamps extFormIdx to the visible field range after a protocol change.
func (ce *ContractsEditor) updateExtDependentFields() {
	protocol := fieldGet(ce.extForm, "protocol")
	if opts, ok := failureStrategyByProtocol[protocol]; ok {
		for i := range ce.extForm {
			if ce.extForm[i].Key == "failure_strategy" {
				ce.extForm[i].Options = opts
				// Reset to first valid option if current value is no longer valid.
				current := ce.extForm[i].Value
				valid := false
				for _, o := range opts {
					if o == current {
						valid = true
						break
					}
				}
				if !valid {
					ce.extForm[i].Value = opts[0]
					ce.extForm[i].SelIdx = 0
				}
				break
			}
		}
	}
	visible := ce.visibleExtFormFields()
	if len(visible) > 0 && ce.extFormIdx >= len(visible) {
		ce.extFormIdx = len(visible) - 1
	}
}

// ── DTO and Endpoint field visibility ────────────────────────────────────────

// protocolsForService returns the endpoint protocols supported by a service
// based on its technology selections in the Backend pillar.
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
		case "pagination":
			if proto == "WebSocket message" || proto == "gRPC" || proto == "Event" {
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
