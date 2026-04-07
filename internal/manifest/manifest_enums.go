package manifest

// ── Enum types ────────────────────────────────────────────────────────────────

type ArchPattern string

const (
	ArchMonolith        ArchPattern = "monolith"
	ArchModularMonolith ArchPattern = "modular-monolith"
	ArchMicroservices   ArchPattern = "microservices"
	ArchEventDriven     ArchPattern = "event-driven"
	ArchHybrid          ArchPattern = "hybrid"
)

type CommProtocol string

const (
	ProtoREST       CommProtocol = "REST"
	ProtoGraphQL    CommProtocol = "GraphQL"
	ProtoGRPC       CommProtocol = "gRPC"
	ProtoWebSockets CommProtocol = "WebSockets"
	ProtoMixed      CommProtocol = "mixed"
)

type SerializationFmt string

const (
	SerialJSON        SerializationFmt = "JSON"
	SerialProtobuf    SerializationFmt = "Protobuf"
	SerialMessagePack SerializationFmt = "MessagePack"
	SerialMixed       SerializationFmt = "mixed"
)

type ComputeEnv string

const (
	ComputeServerless    ComputeEnv = "serverless"
	ComputeContainerized ComputeEnv = "containerized"
	ComputeBareMetalVM   ComputeEnv = "bare-metal/VM"
)

type DatabaseType string

const (
	DBPostgres DatabaseType = "PostgreSQL"
	DBMySQL    DatabaseType = "MySQL"
	DBMongo    DatabaseType = "MongoDB"
	DBDynamo   DatabaseType = "DynamoDB"
	DBSQLite   DatabaseType = "SQLite"
	DBOther    DatabaseType = "other"
)

type CacheStore string

const (
	CacheRedis     CacheStore = "Redis"
	CacheMemcached CacheStore = "Memcached"
	CacheNone      CacheStore = "none"
)

type RenderingMode string

const (
	RenderSPA RenderingMode = "SPA"
	RenderSSR RenderingMode = "SSR"
	RenderSSG RenderingMode = "SSG"
	RenderISR RenderingMode = "ISR"
)

type ColumnType string

const (
	ColTypeText        ColumnType = "text"
	ColTypeVarchar     ColumnType = "varchar"
	ColTypeChar        ColumnType = "char"
	ColTypeInt         ColumnType = "int"
	ColTypeBigInt      ColumnType = "bigint"
	ColTypeSmallInt    ColumnType = "smallint"
	ColTypeSerial      ColumnType = "serial"
	ColTypeBigSerial   ColumnType = "bigserial"
	ColTypeBoolean     ColumnType = "boolean"
	ColTypeFloat       ColumnType = "float"
	ColTypeDouble      ColumnType = "double"
	ColTypeDecimal     ColumnType = "decimal"
	ColTypeJSON        ColumnType = "json"
	ColTypeJSONB       ColumnType = "jsonb"
	ColTypeUUID        ColumnType = "uuid"
	ColTypeTimestamp   ColumnType = "timestamp"
	ColTypeTimestampTZ ColumnType = "timestamptz"
	ColTypeDate        ColumnType = "date"
	ColTypeTime        ColumnType = "time"
	ColTypeBytea       ColumnType = "bytea"
	ColTypeEnum        ColumnType = "enum"
	ColTypeArray       ColumnType = "array"
	ColTypeOther       ColumnType = "other"
)

type CascadeAction string

const (
	CascadeNoAction   CascadeAction = "NO ACTION"
	CascadeRestrict   CascadeAction = "RESTRICT"
	CascadeCascade    CascadeAction = "CASCADE"
	CascadeSetNull    CascadeAction = "SET NULL"
	CascadeSetDefault CascadeAction = "SET DEFAULT"
)

type IndexType string

const (
	IndexBTree IndexType = "btree"
	IndexHash  IndexType = "hash"
	IndexGIN   IndexType = "gin"
	IndexGIST  IndexType = "gist"
	IndexBRIN  IndexType = "brin"
)

// SyncAsyncMode enumerates valid values for CommLink.SyncAsync.
type SyncAsyncMode = string

const (
	SyncAsyncSync  SyncAsyncMode = "sync"
	SyncAsyncAsync SyncAsyncMode = "async"
)

// PatternTag enumerates valid values for ServiceDef.PatternTag (hybrid arch only).
type PatternTag = string

const (
	PatternTagDomain   PatternTag = "domain"
	PatternTagInfra    PatternTag = "infra"
	PatternTagBFF      PatternTag = "bff"
	PatternTagGateway  PatternTag = "gateway"
	PatternTagWorker   PatternTag = "worker"
	PatternTagExternal PatternTag = "external"
)

// AuthStrategy enumerates valid values for AuthConfig.Strategy.
type AuthStrategy = string

const (
	AuthStrategyJWT     AuthStrategy = "JWT"
	AuthStrategySession AuthStrategy = "Session-based"
	AuthStrategySAML    AuthStrategy = "SAML"
	AuthStrategyOIDC    AuthStrategy = "OIDC"
	AuthStrategyAPIKey  AuthStrategy = "API Key"
	AuthStrategyNone    AuthStrategy = "None"
)
