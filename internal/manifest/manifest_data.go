package manifest

// ── Database source definitions ───────────────────────────────────────────────

// DBSourceDef describes a named database or cache source used in the project.
type DBSourceDef struct {
	Alias     string       `json:"alias"`
	Type      DatabaseType `json:"type,omitempty"`
	Version   string       `json:"version,omitempty"`
	Namespace string       `json:"namespace,omitempty"`
	IsCache   bool         `json:"is_cache,omitempty"`

	// Security / network integrity
	SSLMode     string `json:"ssl_mode,omitempty"`    // disable | require | verify-ca | verify-full
	Consistency string `json:"consistency,omitempty"` // strong | eventual | LOCAL_QUORUM | ONE | QUORUM | ALL | LOCAL_ONE

	// Connection pooling
	PoolMinSize int `json:"pool_min_size,omitempty"`
	PoolMaxSize int `json:"pool_max_size,omitempty"`

	// Availability topology
	Replication string `json:"replication,omitempty"` // single-node | primary-replica | multi-region

	// Deployment environment (references InfraPillar.Environments[*].Name)
	Environment string `json:"environment,omitempty"`

	Notes string `json:"notes,omitempty"`
}

// ── Column / Entity definitions ───────────────────────────────────────────────

type ForeignKey struct {
	RefEntity string        `json:"ref_entity"`
	RefColumn string        `json:"ref_column"`
	OnDelete  CascadeAction `json:"on_delete,omitempty"`
	OnUpdate  CascadeAction `json:"on_update,omitempty"`
}

type ColumnDef struct {
	Name       string      `json:"name"`
	Type       ColumnType  `json:"type,omitempty"`
	Length     string      `json:"length,omitempty"`
	Nullable   bool        `json:"nullable,omitempty"`
	PrimaryKey bool        `json:"primary_key,omitempty"`
	Unique     bool        `json:"unique,omitempty"`
	Default    string      `json:"default,omitempty"`
	Check      string      `json:"check,omitempty"`
	ForeignKey *ForeignKey `json:"foreign_key,omitempty"`
	Index      bool        `json:"index,omitempty"`
	IndexType  IndexType   `json:"index_type,omitempty"`
	Notes      string      `json:"notes,omitempty"`
}

type UniqueConstraint struct {
	Name    string   `json:"name,omitempty"`
	Columns []string `json:"columns"`
}

type EntityDef struct {
	Name        string `json:"name"`
	Database    string `json:"database,omitempty"`
	Description string `json:"description,omitempty"`

	Cached     bool   `json:"cached,omitempty"`
	CacheStore string `json:"cache_store,omitempty"`
	CacheTTL   string `json:"cache_ttl,omitempty"`

	Columns           []ColumnDef        `json:"columns,omitempty"`
	UniqueConstraints []UniqueConstraint `json:"unique_constraints,omitempty"`
	Notes             string             `json:"notes,omitempty"`
}

// ── Legacy global pillars ─────────────────────────────────────────────────────

type DomainPillar struct {
	Entities   []EntityDef `json:"entities,omitempty"`
	RBACMatrix string      `json:"rbac_matrix,omitempty"`
	Compliance string      `json:"compliance,omitempty"`
}

type TopologyPillar struct {
	ArchPattern   ArchPattern      `json:"arch_pattern,omitempty"`
	CommProtocol  CommProtocol     `json:"comm_protocol,omitempty"`
	Serialization SerializationFmt `json:"serialization,omitempty"`
	DomainNotes   string           `json:"domain_notes,omitempty"`
}

type GlobalNFRPillar struct {
	UptimeSLO      string `json:"uptime_slo,omitempty"`
	ConcurrentConn string `json:"concurrent_conn,omitempty"`
	RTO            string `json:"rto,omitempty"`
	RPO            string `json:"rpo,omitempty"`
	NFRNotes       string `json:"nfr_notes,omitempty"`
}

// ── Domain definitions ────────────────────────────────────────────────────────

// DomainDef is a bounded-context domain (not a DB entity/column).
type DomainDef struct {
	Name          string               `json:"name"`
	Description   string               `json:"description,omitempty"`
	Databases     string               `json:"databases,omitempty"`
	Attributes    []DomainAttribute    `json:"attributes,omitempty"`
	Relationships []DomainRelationship `json:"relationships,omitempty"`
}

type DomainAttribute struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Constraints string `json:"constraints,omitempty"`
	Default     string `json:"default,omitempty"`
	Sensitive   bool   `json:"sensitive,omitempty"`
	Validation  string `json:"validation,omitempty"`
	Indexed     bool   `json:"indexed,omitempty"`
	Unique      bool   `json:"unique,omitempty"`
}

type DomainRelationship struct {
	RelatedDomain string `json:"related_domain"`
	RelType       string `json:"rel_type"`
	ForeignKey    string `json:"foreign_key,omitempty"`
	Cascade       string `json:"cascade,omitempty"`
}

// ── Data pillar supporting types ──────────────────────────────────────────────

// CachingConfig describes a single application-level caching strategy.
type CachingConfig struct {
	Name         string `json:"name,omitempty"`
	Layer        string `json:"layer"`
	CacheDB      string `json:"cache_db,omitempty"`
	Strategy     string `json:"strategy"`
	Invalidation string `json:"invalidation"`
	TTL          string `json:"ttl,omitempty"`
	Entities     string `json:"entities,omitempty"`
}

// FileStorageDef describes a file/object storage bucket.
type FileStorageDef struct {
	Technology    string `json:"technology"`
	Purpose       string `json:"purpose,omitempty"`
	UsedByService string `json:"used_by_service,omitempty"`
	Environment   string `json:"environment,omitempty"`
	Access        string `json:"access"`
	MaxSize       string `json:"max_size,omitempty"`
	Domains       string `json:"domains,omitempty"`
	TTLMinutes    string `json:"ttl_minutes,omitempty"`
	AllowedTypes  string `json:"allowed_types,omitempty"`
}

// DataGovernanceConfig describes data lifecycle, privacy, and compliance settings
// for a specific set of databases.
type DataGovernanceConfig struct {
	Name                 string   `json:"name,omitempty"`
	Databases            []string `json:"databases,omitempty"`
	RetentionPolicy      string   `json:"retention_policy,omitempty"`
	DeleteStrategy       string   `json:"delete_strategy,omitempty"`
	PIIEncryption        string   `json:"pii_encryption,omitempty"`
	ComplianceFrameworks string   `json:"compliance_frameworks,omitempty"`
	DataResidency        string   `json:"data_residency,omitempty"`
	ArchivalStorage      string   `json:"archival_storage,omitempty"`
	MigrationTool        string   `json:"migration_tool,omitempty"`
	BackupStrategy       string   `json:"backup_strategy,omitempty"`
	SearchTech           string   `json:"search_tech,omitempty"`
}

// DataPillar groups all data-related configuration.
type DataPillar struct {
	Databases    []DBSourceDef          `json:"databases,omitempty"`
	Domains      []DomainDef            `json:"domains,omitempty"`
	Entities     []EntityDef            `json:"entities,omitempty"` // legacy
	Cachings     []CachingConfig        `json:"cachings,omitempty"`
	FileStorages []FileStorageDef       `json:"file_storages,omitempty"`
	Governances  []DataGovernanceConfig `json:"governances,omitempty"`
}
