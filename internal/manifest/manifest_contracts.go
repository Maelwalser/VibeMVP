package manifest

// ── Contracts tab types ───────────────────────────────────────────────────────

// DTOField describes a single field within a DTO.
type DTOField struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Required   bool   `json:"required"`
	Nullable   bool   `json:"nullable"`
	Validation string `json:"validation,omitempty"`
	Default    string `json:"default,omitempty"`
	Notes      string `json:"notes,omitempty"`

	// Protobuf-specific
	FieldNumber   string `json:"field_number,omitempty"`
	ProtoModifier string `json:"proto_modifier,omitempty"`
	JsonName      string `json:"json_name,omitempty"`

	// Thrift / Cap'n Proto
	FieldID string `json:"field_id,omitempty"`

	// Thrift-specific
	ThriftModifier string `json:"thrift_modifier,omitempty"`

	// FlatBuffers-specific
	Deprecated bool `json:"deprecated,omitempty"`
}

// DTODef describes a Data Transfer Object.
type DTODef struct {
	Name          string     `json:"name"`
	Category      string     `json:"category"`
	SourceDomains string     `json:"source_domains,omitempty"`
	Description   string     `json:"description,omitempty"`
	Protocol      string     `json:"protocol,omitempty"`
	Fields        []DTOField `json:"fields,omitempty"`

	// Protobuf-specific
	ProtoPackage string `json:"proto_package,omitempty"`
	ProtoSyntax  string `json:"proto_syntax,omitempty"`
	ProtoOptions string `json:"proto_options,omitempty"`

	// Avro-specific
	AvroNamespace  string `json:"avro_namespace,omitempty"`
	SchemaRegistry string `json:"schema_registry,omitempty"`

	// Thrift-specific
	ThriftNamespace string `json:"thrift_namespace,omitempty"`
	ThriftLanguage  string `json:"thrift_language,omitempty"`

	// FlatBuffers / Cap'n Proto
	Namespace string `json:"namespace,omitempty"`
}

// EndpointDef describes an API endpoint or operation.
type EndpointDef struct {
	ServiceUnit        string `json:"service_unit"`
	NamePath           string `json:"name_path"`
	Protocol           string `json:"protocol"`
	AuthRequired       string `json:"auth_required"`
	AuthRoles          string `json:"auth_roles,omitempty"`
	RequestDTO         string `json:"request_dto,omitempty"`
	ResponseDTO        string `json:"response_dto,omitempty"`
	HTTPMethod         string `json:"http_method,omitempty"`
	Description        string `json:"description,omitempty"`
	GraphQLOpType      string `json:"graphql_op_type,omitempty"`
	GRPCStreamType     string `json:"grpc_stream_type,omitempty"`
	WSDirection        string `json:"ws_direction,omitempty"`
	PaginationStrategy string `json:"pagination_strategy,omitempty"`
	RateLimit          string `json:"rate_limit,omitempty"`
}

// APIVersioning describes how the API handles versioning.
type APIVersioning struct {
	// PerProtocolStrategies holds a versioning strategy for each endpoint
	// protocol in use (e.g. "REST" → "URL path (/v1/)", "gRPC" → "Package versioning").
	PerProtocolStrategies map[string]string `json:"per_protocol_strategies,omitempty"`
	CurrentVersion        string            `json:"current_version,omitempty"`
	DeprecationPolicy     string            `json:"deprecation_policy,omitempty"`
}

// ExternalAPIInteraction describes a single call/operation to an external API.
type ExternalAPIInteraction struct {
	Name           string `json:"name,omitempty"`
	Path           string `json:"path,omitempty"`
	RequestDTO     string `json:"request_dto,omitempty"`
	ResponseDTO    string `json:"response_dto,omitempty"`
	HTTPMethod     string `json:"http_method,omitempty"`      // REST
	GQLOperation   string `json:"gql_operation,omitempty"`    // GraphQL
	GRPCStreamType string `json:"grpc_stream_type,omitempty"` // gRPC
	WSDirection    string `json:"ws_direction,omitempty"`     // WebSocket
}

// ExternalAPIDef describes a third-party API that the system consumes.
type ExternalAPIDef struct {
	Provider        string                   `json:"provider"`
	CalledByService string                   `json:"called_by_service,omitempty"`
	Responsibility  string                   `json:"responsibility,omitempty"`
	Protocol        string                   `json:"protocol,omitempty"`
	AuthMechanism   string                   `json:"auth_mechanism"`
	FailureStrategy string                   `json:"failure_strategy"`
	Interactions    []ExternalAPIInteraction `json:"interactions,omitempty"`

	// REST / general HTTP
	BaseURL         string `json:"base_url,omitempty"`
	RateLimit       string `json:"rate_limit,omitempty"`
	WebhookEndpoint string `json:"webhook_endpoint,omitempty"`

	// gRPC
	TLSMode string `json:"tls_mode,omitempty"`

	// WebSocket
	WSSubprotocol string `json:"ws_subprotocol,omitempty"`
	MessageFormat string `json:"message_format,omitempty"`

	// Webhook (inbound)
	HMACHeader  string `json:"hmac_header,omitempty"`
	RetryPolicy string `json:"retry_policy,omitempty"`

	// SOAP
	SOAPVersion string `json:"soap_version,omitempty"`
}

// ContractsPillar groups all contract-related configuration.
type ContractsPillar struct {
	DTOs         []DTODef         `json:"dtos,omitempty"`
	Endpoints    []EndpointDef    `json:"endpoints,omitempty"`
	Versioning   *APIVersioning   `json:"versioning,omitempty"`
	ExternalAPIs []ExternalAPIDef `json:"external_apis,omitempty"`
}

// OrZero returns a dereferenced APIVersioning, or a zero value if nil.
func (v *APIVersioning) OrZero() APIVersioning {
	if v == nil {
		return APIVersioning{}
	}
	return *v
}
