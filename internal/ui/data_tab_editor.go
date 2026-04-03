package ui

import (
	"fmt"
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

// ── caching list+form types ───────────────────────────────────────────────────

type cachingView int

const (
	cachingViewList cachingView = iota
	cachingViewForm
)

// ── mode ─────────────────────────────────────────────────────────────────────

type dtMode int

const (
	dtNormal dtMode = iota
	dtInsert
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
	governanceFields []Field
	govFormIdx       int
	govEnabled       bool

	// Context from backend — used to filter migration tool options.
	backendLangs []string

	// Shared
	internalMode dtMode
	formInput    textinput.Model
	width        int

	// Dropdown state for multiselect fields in domain/caching/fs forms
	ddOpen   bool
	ddOptIdx int
}

func newDataTabEditor() DataTabEditor {
	return DataTabEditor{
		dbEditor:         newDBEditor(),
		dataEditor:       newDataEditor(),
		governanceFields: defaultGovernanceFields(),
		formInput:        newFormInput(),
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

// ── field definitions ─────────────────────────────────────────────────────────

func defaultDomainFormFields(dbOptions []string) []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "description", Label: "description   ", Kind: KindText},
		{
			Key: "databases", Label: "databases     ", Kind: KindMultiSelect,
			Options: dbOptions,
			Value:   placeholderFor(dbOptions, "(no databases configured)"),
		},
		{Key: "attr_names", Label: "attr_names    ", Kind: KindText,
			// Hint: type comma-separated attribute names to batch-create attributes
		},
	}
}

// typesForDBTech returns native data types for a given database technology.
func typesForDBTech(tech string) []string {
	switch tech {
	case "PostgreSQL":
		return []string{
			"varchar", "text", "char", "int", "bigint", "smallint",
			"serial", "bigserial", "boolean", "float", "double precision",
			"decimal", "numeric", "uuid", "timestamp", "timestamptz",
			"date", "time", "interval", "json", "jsonb", "bytea",
			"enum", "array", "inet", "tsvector", "xml",
		}
	case "MySQL":
		return []string{
			"varchar", "text", "char", "tinytext", "mediumtext", "longtext",
			"int", "bigint", "smallint", "tinyint", "mediumint",
			"float", "double", "decimal",
			"boolean", "date", "datetime", "timestamp", "time", "year",
			"json", "binary", "varbinary", "blob", "enum", "set",
		}
	case "SQLite":
		return []string{"TEXT", "INTEGER", "REAL", "NUMERIC", "BLOB", "NULL"}
	case "MongoDB":
		return []string{
			"String", "Int32", "Int64", "Double", "Decimal128",
			"Boolean", "Date", "ObjectId", "UUID",
			"Array", "Object", "Binary", "Null", "Timestamp", "Mixed",
		}
	case "DynamoDB":
		return []string{
			"String (S)", "Number (N)", "Binary (B)",
			"StringSet (SS)", "NumberSet (NS)", "BinarySet (BS)",
			"List (L)", "Map (M)", "Boolean (BOOL)", "Null (NULL)",
		}
	case "Cassandra":
		return []string{
			"text", "varchar", "ascii", "int", "bigint", "smallint", "tinyint", "varint",
			"float", "double", "decimal", "boolean",
			"date", "timestamp", "time", "uuid", "timeuuid",
			"blob", "list", "set", "map", "tuple", "frozen",
		}
	case "Redis", "Memcached":
		return []string{"String", "List", "Set", "Sorted Set", "Hash", "Stream"}
	case "ClickHouse":
		return []string{
			"UInt8", "UInt16", "UInt32", "UInt64",
			"Int8", "Int16", "Int32", "Int64",
			"Float32", "Float64", "Decimal",
			"String", "FixedString", "Date", "DateTime", "UUID",
			"Array", "Tuple", "Nullable", "Enum", "LowCardinality",
		}
	case "Elasticsearch":
		return []string{
			"text", "keyword", "long", "integer", "short", "byte",
			"double", "float", "boolean", "date", "binary", "ip",
			"object", "nested", "geo_point",
		}
	default:
		return []string{
			"String", "Int", "Float", "Boolean", "DateTime",
			"UUID", "JSON", "Binary", "Array", "Enum", "Ref",
		}
	}
}

// attrTypesForSources resolves attribute type options for a domain based on its
// selected database aliases. Returns (nil, true) when no databases are selected.
// When multiple databases are selected, types are merged (deduplicated, first-seen order).
func attrTypesForSources(selectedDBs string, sources []manifest.DBSourceDef) (types []string, noDB bool) {
	if selectedDBs == "" {
		return nil, true
	}
	seen := map[string]bool{}
	for _, alias := range strings.Split(selectedDBs, ", ") {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		for _, src := range sources {
			if src.Alias == alias {
				for _, t := range typesForDBTech(string(src.Type)) {
					if !seen[t] {
						seen[t] = true
						types = append(types, t)
					}
				}
			}
		}
	}
	if len(types) == 0 {
		return nil, true
	}
	return types, false
}

// refreshAttrTypeOptions updates the "type" field options in an attr form to match
// the current database selection, preserving the selected value when possible.
func refreshAttrTypeOptions(form []Field, types []string) []Field {
	for i := range form {
		if form[i].Key != "type" {
			continue
		}
		cur := form[i].DisplayValue()
		if len(types) == 0 {
			form[i].Options = nil
			form[i].Value = "(select a database first)"
			form[i].SelIdx = 0
		} else {
			form[i].Options = types
			form[i].SelIdx = 0
			form[i].Value = types[0]
			for j, t := range types {
				if t == cur {
					form[i].SelIdx = j
					form[i].Value = t
					break
				}
			}
		}
		break
	}
	return form
}

// currentDomainAttrTypes returns type options for the domain currently being edited,
// based on its selected databases. Returns nil when no databases are selected.
func (dt DataTabEditor) currentDomainAttrTypes() []string {
	dbs := fieldGetMulti(dt.domainForm, "databases")
	types, _ := attrTypesForSources(dbs, dt.dbEditor.Sources)
	return types
}

func defaultAttrFields(types []string) []Field {
	var typeOpts []string
	var typeVal string
	if len(types) == 0 {
		// No database selected — use empty options so DisplayValue returns the placeholder.
		typeVal = "(select a database first)"
	} else {
		typeOpts = types
		typeVal = types[0]
	}
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{
			Key: "type", Label: "type          ", Kind: KindSelect,
			Options: typeOpts,
			Value:   typeVal,
		},
		{
			Key: "constraints", Label: "constraints   ", Kind: KindMultiSelect,
			Options: []string{
				"required", "unique", "not_null", "min", "max",
				"min_length", "max_length", "email", "url", "regex",
				"positive", "future", "past", "enum",
			},
		},
		{Key: "default", Label: "default       ", Kind: KindText},
		{
			Key: "sensitive", Label: "sensitive     ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{
			Key: "validation", Label: "validation    ", Kind: KindMultiSelect,
			Options: []string{
				"email", "url", "regex", "min_length", "max_length",
				"min_value", "max_value", "phone", "uuid", "date_format", "enum", "custom",
			},
		},
		{
			Key: "indexed", Label: "indexed       ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{
			Key: "unique", Label: "unique        ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
	}
}

func defaultRelFields(domainOptions []string) []Field {
	return []Field{
		{
			Key: "related_domain", Label: "related_domain", Kind: KindSelect,
			Options: domainOptions,
			Value:   placeholderFor(domainOptions, "(no domains configured)"),
		},
		{
			Key: "rel_type", Label: "rel_type      ", Kind: KindSelect,
			Options: []string{"One-to-One", "One-to-Many", "Many-to-Many"},
			Value:   "One-to-Many",
		},
		{
			Key: "cascade", Label: "cascade       ", Kind: KindSelect,
			Options: []string{"CASCADE", "SET NULL", "RESTRICT", "NO ACTION", "SET DEFAULT"},
			Value:   "NO ACTION",
		},
	}
}

func defaultCachingFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{
			Key: "layer", Label: "layer         ", Kind: KindSelect,
			Options: []string{
				"Application-level", "Dedicated cache",
				"CDN", "None",
			},
			Value: "None", SelIdx: 3,
		},
		{
			Key: "cache_db", Label: "cache db      ", Kind: KindSelect,
			Options: []string{},
		},
		{
			Key: "strategy", Label: "strategy      ", Kind: KindMultiSelect,
			Options: []string{"Cache-aside", "Read-through", "Write-through", "Write-behind"},
		},
		{
			Key: "invalidation", Label: "invalidation  ", Kind: KindSelect,
			Options: []string{"TTL-based", "Event-driven", "Manual", "Hybrid"},
			Value:   "TTL-based",
		},
		{
			Key: "ttl", Label: "ttl           ", Kind: KindSelect,
			Options: []string{"30s", "1m", "5m", "15m", "1h", "24h", "Custom"},
			Value:   "5m", SelIdx: 2,
		},
		{
			Key: "entities", Label: "entities      ", Kind: KindMultiSelect,
			Options: []string{}, // populated dynamically from domain names
		},
	}
}

func cachingFormFromDef(def manifest.CachingConfig) []Field {
	f := defaultCachingFields()
	f = setFieldValue(f, "name", def.Name)
	if def.Layer != "" {
		f = setFieldValue(f, "layer", def.Layer)
	}
	if def.CacheDB != "" {
		f = setFieldValue(f, "cache_db", def.CacheDB)
	}
	if def.Strategy != "" {
		f = setFieldValue(f, "strategy", def.Strategy)
	}
	if def.Invalidation != "" {
		f = setFieldValue(f, "invalidation", def.Invalidation)
	}
	if def.TTL != "" {
		f = setFieldValue(f, "ttl", def.TTL)
	}
	if def.Entities != "" {
		f = restoreMultiSelectValue(f, "entities", def.Entities)
	}
	return f
}

func cachingDefFromForm(fields []Field) manifest.CachingConfig {
	return manifest.CachingConfig{
		Name:         fieldGet(fields, "name"),
		Layer:        fieldGet(fields, "layer"),
		CacheDB:      fieldGet(fields, "cache_db"),
		Strategy:     fieldGet(fields, "strategy"),
		Invalidation: fieldGet(fields, "invalidation"),
		TTL:          fieldGet(fields, "ttl"),
		Entities:     fieldGetMulti(fields, "entities"),
	}
}

// migrationToolsByLang lists compatible migration handlers per backend language.
var migrationToolsByLang = map[string][]string{
	"Go":              {"golang-migrate", "Atlas", "goose", "None"},
	"TypeScript/Node": {"Prisma Migrate", "TypeORM Migrations", "Knex.js Migrations", "db-migrate", "None"},
	"Python":          {"Alembic", "Django Migrations", "Flyway", "None"},
	"Java":            {"Flyway", "Liquibase", "None"},
	"Kotlin":          {"Flyway", "Liquibase", "Exposed Migrations", "None"},
	"C#/.NET":         {"EF Core Migrations", "Flyway", "Liquibase", "None"},
	"Ruby":            {"Active Record Migrations", "Sequel Migrations", "None"},
	"PHP":             {"Doctrine Migrations", "Phinx", "Laravel Migrations", "None"},
	"Rust":            {"SQLx Migrations", "Diesel Migrations", "refinery", "None"},
	"Elixir":          {"Ecto Migrations", "None"},
	"Other":           {"Flyway", "Liquibase", "Atlas", "golang-migrate", "Alembic", "Prisma Migrate", "None"},
}

// allMigrationTools is the fallback shown before any backend language is configured.
var allMigrationTools = []string{
	"golang-migrate", "Atlas", "Flyway", "Liquibase", "Prisma Migrate", "Alembic", "None",
}

// migrationToolsForLangs returns the union of compatible migration tools for the
// given languages, with "None" always last. Falls back to allMigrationTools when empty.
func migrationToolsForLangs(langs []string) []string {
	if len(langs) == 0 {
		return allMigrationTools
	}
	seen := make(map[string]bool)
	var result []string
	for _, lang := range langs {
		for _, tool := range migrationToolsByLang[lang] {
			if tool == "None" {
				continue
			}
			if !seen[tool] {
				seen[tool] = true
				result = append(result, tool)
			}
		}
	}
	result = append(result, "None")
	return result
}

// searchTechForSources returns the valid search technology options given the
// set of configured database technologies. "PostgreSQL FTS" is only included
// when PostgreSQL is present; "MongoDB Atlas Search" only when MongoDB is
// present; etc. "None" is always appended as the last option.
func searchTechForSources(sources []manifest.DBSourceDef) []string {
	seen := map[string]bool{}
	var opts []string
	add := func(tech string) {
		if !seen[tech] {
			seen[tech] = true
			opts = append(opts, tech)
		}
	}

	var searchTechByDB = map[string][]string{
		"PostgreSQL":    {"PostgreSQL FTS", "Elasticsearch", "Meilisearch", "Typesense", "Algolia"},
		"MySQL":         {"Elasticsearch", "Meilisearch", "Typesense", "Algolia"},
		"MongoDB":       {"MongoDB Atlas Search", "Elasticsearch", "Meilisearch", "Algolia"},
		"Elasticsearch": {"Elasticsearch"},
	}

	for _, src := range sources {
		techs, ok := searchTechByDB[string(src.Type)]
		if !ok {
			// unknown DB: offer generic options
			for _, t := range []string{"Elasticsearch", "Meilisearch", "Algolia", "Typesense"} {
				add(t)
			}
			continue
		}
		for _, t := range techs {
			add(t)
		}
	}
	if len(opts) == 0 {
		opts = []string{"Elasticsearch", "Meilisearch", "Algolia", "Typesense"}
	}
	opts = append(opts, "None")
	return opts
}

// updateSearchTechOptions recomputes the search_tech governance field options
// from the current database sources, preserving the selection when still valid.
func (dt *DataTabEditor) updateSearchTechOptions() {
	opts := searchTechForSources(dt.dbEditor.Sources)
	for i := range dt.governanceFields {
		if dt.governanceFields[i].Key != "search_tech" {
			continue
		}
		current := dt.governanceFields[i].DisplayValue()
		dt.governanceFields[i].Options = opts
		found := false
		for j, opt := range opts {
			if opt == current {
				dt.governanceFields[i].SelIdx = j
				dt.governanceFields[i].Value = opt
				found = true
				break
			}
		}
		if !found {
			last := len(opts) - 1
			dt.governanceFields[i].SelIdx = last
			dt.governanceFields[i].Value = opts[last]
		}
		break
	}
}

// SetMigrationContext updates the backend languages used to filter migration tool
// options. Recomputes the governance field options immediately, preserving the
// current selection when it remains valid.
func (dt *DataTabEditor) SetMigrationContext(langs []string) {
	if stringSlicesEqual(dt.backendLangs, langs) {
		return
	}
	dt.backendLangs = langs
	opts := migrationToolsForLangs(langs)
	for i := range dt.governanceFields {
		if dt.governanceFields[i].Key != "migration_tool" {
			continue
		}
		current := dt.governanceFields[i].DisplayValue()
		dt.governanceFields[i].Options = opts
		found := false
		for j, opt := range opts {
			if opt == current {
				dt.governanceFields[i].SelIdx = j
				dt.governanceFields[i].Value = opt
				found = true
				break
			}
		}
		if !found {
			last := len(opts) - 1
			dt.governanceFields[i].SelIdx = last
			dt.governanceFields[i].Value = opts[last]
		}
		break
	}
}

func defaultGovernanceFields() []Field {
	return []Field{
		{
			Key: "migration_tool", Label: "Migration     ", Kind: KindSelect,
			Options: allMigrationTools,
			Value:   "None", SelIdx: 6,
		},
		{
			Key: "backup_strategy", Label: "Backup Strat. ", Kind: KindSelect,
			Options: []string{"Automated daily", "Point-in-time recovery", "Manual snapshots", "Managed provider", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "search_tech", Label: "Search Tech   ", Kind: KindSelect,
			Options: []string{"Elasticsearch", "Meilisearch", "Algolia", "Typesense", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "retention_policy", Label: "retention     ", Kind: KindSelect,
			Options: []string{"30 days", "90 days", "1 year", "3 years", "7 years", "Indefinite", "Custom"},
			Value:   "Indefinite", SelIdx: 5,
		},
		{
			Key: "delete_strategy", Label: "delete_strat  ", Kind: KindSelect,
			Options: []string{"Soft-delete", "Hard-delete", "Archival", "Soft + periodic purge"},
			Value:   "Soft-delete",
		},
		{
			Key: "pii_encryption", Label: "pii_encryption", Kind: KindSelect,
			Options: []string{"Field-level AES-256", "Full database encryption", "Application-level", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "compliance_frameworks", Label: "compliance    ", Kind: KindMultiSelect,
			Options: []string{"GDPR", "HIPAA", "SOC2 Type II", "PCI-DSS", "ISO-27001", "CCPA", "PIPEDA"},
		},
		{
			Key: "data_residency", Label: "data_residency", Kind: KindSelect,
			Options: []string{"US", "EU", "APAC", "US + EU", "Global", "Custom"},
			Value:   "Global", SelIdx: 4,
		},
		{
			Key: "archival_storage", Label: "archival      ", Kind: KindSelect,
			Options: []string{"S3 Glacier", "GCS Archive", "Azure Archive", "On-premise", "None"},
			Value:   "None", SelIdx: 4,
		},
	}
}

// withRefreshedCachingEntities returns a copy of the DataTabEditor with the
// entities multiselect options in cachingForm updated to reflect current domain names.
func (dt DataTabEditor) withRefreshedCachingEntities() DataTabEditor {
	domOpts := dt.domainNames()
	newFields := make([]Field, len(dt.cachingForm))
	copy(newFields, dt.cachingForm)
	for i := range newFields {
		if newFields[i].Key == "entities" {
			// Preserve existing selections by re-mapping
			oldOpts := newFields[i].Options
			newFields[i].Options = domOpts
			newFields[i].Value = placeholderFor(domOpts, "(no domains configured)")
			newSelected := make([]int, 0)
			for _, oldIdx := range newFields[i].SelectedIdxs {
				if oldIdx < len(oldOpts) {
					oldVal := oldOpts[oldIdx]
					for j, newOpt := range domOpts {
						if newOpt == oldVal {
							newSelected = append(newSelected, j)
							break
						}
					}
				}
			}
			newFields[i].SelectedIdxs = newSelected
			break
		}
	}
	dt.cachingForm = newFields
	return dt
}

// isCachingFieldDisabled returns true when cache_db should be hidden because
// the selected layer is not "Dedicated cache".
func isCachingFieldDisabled(fields []Field, idx int) bool {
	if fields[idx].Key != "cache_db" {
		return false
	}
	for _, f := range fields {
		if f.Key == "layer" {
			return f.DisplayValue() != "Dedicated cache"
		}
	}
	return true
}

func nextCachingFormIdx(fields []Field, cur int) int {
	n := len(fields)
	next := (cur + 1) % n
	for next != cur && isCachingFieldDisabled(fields, next) {
		next = (next + 1) % n
	}
	return next
}

func prevCachingFormIdx(fields []Field, cur int) int {
	n := len(fields)
	prev := (cur - 1 + n) % n
	for prev != cur && isCachingFieldDisabled(fields, prev) {
		prev = (prev - 1 + n) % n
	}
	return prev
}

// cachingVisibleFields returns only the fields that should be rendered.
func cachingVisibleFields(fields []Field) []Field {
	out := make([]Field, 0, len(fields))
	for i, f := range fields {
		if !isCachingFieldDisabled(fields, i) {
			out = append(out, f)
		}
	}
	return out
}

// cachingVisibleIdx maps a full-list index to its position in the visible list.
func cachingVisibleIdx(fields []Field, fullIdx int) int {
	vis := 0
	for i := range fullIdx {
		if !isCachingFieldDisabled(fields, i) {
			vis++
		}
	}
	return vis
}

// withRefreshedCachingDBs returns a copy of the DataTabEditor with the cache_db
// select options in cachingForm populated from database sources that have IsCache == true.
func (dt DataTabEditor) withRefreshedCachingDBs() DataTabEditor {
	var aliases []string
	for _, src := range dt.dbEditor.Sources {
		if src.IsCache {
			aliases = append(aliases, src.Alias)
		}
	}
	newFields := make([]Field, len(dt.cachingForm))
	copy(newFields, dt.cachingForm)
	for i := range newFields {
		if newFields[i].Key == "cache_db" {
			cur := newFields[i].Value
			newFields[i].Options = aliases
			newFields[i].Value = placeholderFor(aliases, "(no cache DBs configured)")
			newFields[i].SelIdx = 0
			for j, a := range aliases {
				if a == cur {
					newFields[i].SelIdx = j
					newFields[i].Value = a
					break
				}
			}
			if len(aliases) > 0 && newFields[i].Value == "" {
				newFields[i].Value = aliases[0]
			}
			break
		}
	}
	dt.cachingForm = newFields
	return dt
}

func defaultFSFormFields(domainOptions []string) []Field {
	return []Field{
		{
			Key: "technology", Label: "technology    ", Kind: KindSelect,
			Options: []string{"S3", "GCS", "Azure Blob", "MinIO", "Cloudflare R2", "Local disk"},
			Value:   "S3",
		},
		{Key: "purpose", Label: "purpose       ", Kind: KindText},
		{
			Key: "access", Label: "access        ", Kind: KindSelect,
			Options: []string{"Public (CDN-fronted)", "Private (signed URLs)", "Internal only"},
			Value:   "Private (signed URLs)", SelIdx: 1,
		},
		{
			Key: "max_size", Label: "max_size      ", Kind: KindSelect,
			Options: []string{"1 MB", "5 MB", "10 MB", "25 MB", "50 MB", "100 MB", "500 MB", "1 GB", "Unlimited"},
			Value:   "10 MB", SelIdx: 2,
		},
		{
			Key: "domains", Label: "domains       ", Kind: KindMultiSelect,
			Options: domainOptions,
			Value:   placeholderFor(domainOptions, "(no domains configured)"),
		},
		{
			Key: "ttl_minutes", Label: "ttl_minutes   ", Kind: KindSelect,
			Options: []string{"30", "60", "1440", "10080", "Custom"},
			Value:   "1440", SelIdx: 2,
		},
		{
			Key: "allowed_types", Label: "allowed_types ", Kind: KindMultiSelect,
			Options: []string{"image/*", "application/pdf", "video/*", "audio/*", "text/*", "application/json"},
		},
	}
}

func fsFormFromDef(def manifest.FileStorageDef, domainOptions []string) []Field {
	f := defaultFSFormFields(domainOptions)
	f = setFieldValue(f, "technology", def.Technology)
	f = setFieldValue(f, "purpose", def.Purpose)
	if def.Access != "" {
		f = setFieldValue(f, "access", def.Access)
	}
	f = setFieldValue(f, "max_size", def.MaxSize)
	f = setFieldValue(f, "ttl_minutes", def.TTLMinutes)
	f = setFieldValue(f, "allowed_types", def.AllowedTypes)
	// Restore multi-select for domains
	if def.Domains != "" {
		for i := range f {
			if f[i].Key == "domains" {
				for _, sel := range strings.Split(def.Domains, ", ") {
					for j, opt := range f[i].Options {
						if opt == sel {
							f[i].SelectedIdxs = append(f[i].SelectedIdxs, j)
						}
					}
				}
				break
			}
		}
	}
	return f
}

func fsDefFromForm(fields []Field) manifest.FileStorageDef {
	return manifest.FileStorageDef{
		Technology:   fieldGet(fields, "technology"),
		Purpose:      fieldGet(fields, "purpose"),
		Access:       fieldGet(fields, "access"),
		MaxSize:      fieldGet(fields, "max_size"),
		Domains:      fieldGetMulti(fields, "domains"),
		TTLMinutes:   fieldGet(fields, "ttl_minutes"),
		AllowedTypes: fieldGetMulti(fields, "allowed_types"),
	}
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (dt DataTabEditor) ToManifestDataPillar() manifest.DataPillar {
	p := manifest.DataPillar{
		Databases:    dt.dbEditor.Sources,
		Domains:      dt.domains,
		Entities:     dt.dataEditor.Entities,
		FileStorages: dt.fileStorages,
	}
	p.Cachings = dt.cachings
	if dt.govEnabled {
		p.Governance = manifest.DataGovernanceConfig{
			RetentionPolicy:      fieldGet(dt.governanceFields, "retention_policy"),
			DeleteStrategy:       fieldGet(dt.governanceFields, "delete_strategy"),
			PIIEncryption:        fieldGet(dt.governanceFields, "pii_encryption"),
			ComplianceFrameworks: fieldGetMulti(dt.governanceFields, "compliance_frameworks"),
			DataResidency:        fieldGet(dt.governanceFields, "data_residency"),
			ArchivalStorage:      fieldGet(dt.governanceFields, "archival_storage"),
		}
		p.MigrationTool = fieldGet(dt.governanceFields, "migration_tool")
		p.BackupStrategy = fieldGet(dt.governanceFields, "backup_strategy")
		p.SearchTech = fieldGet(dt.governanceFields, "search_tech")
	}
	return p
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

	// Governance fields.
	dt.updateSearchTechOptions()
	if dp.Governance.RetentionPolicy != "" || dp.Governance.DeleteStrategy != "" || dp.MigrationTool != "" {
		dt.govEnabled = true
		dt.governanceFields = setFieldValue(dt.governanceFields, "retention_policy", dp.Governance.RetentionPolicy)
		dt.governanceFields = setFieldValue(dt.governanceFields, "delete_strategy", dp.Governance.DeleteStrategy)
		dt.governanceFields = setFieldValue(dt.governanceFields, "pii_encryption", dp.Governance.PIIEncryption)
		dt.governanceFields = restoreMultiSelectValue(dt.governanceFields, "compliance_frameworks", dp.Governance.ComplianceFrameworks)
		dt.governanceFields = setFieldValue(dt.governanceFields, "data_residency", dp.Governance.DataResidency)
		dt.governanceFields = setFieldValue(dt.governanceFields, "archival_storage", dp.Governance.ArchivalStorage)
		dt.governanceFields = setFieldValue(dt.governanceFields, "migration_tool", dp.MigrationTool)
		dt.governanceFields = setFieldValue(dt.governanceFields, "backup_strategy", dp.BackupStrategy)
		dt.governanceFields = setFieldValue(dt.governanceFields, "search_tech", dp.SearchTech)
	}

	return dt
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (dt DataTabEditor) Mode() Mode {
	switch dt.activeTab {
	case dataTabDatabases:
		return dt.dbEditor.Mode()
	case dataTabDomains:
		// data editor used for old entities, but this tab manages domains
		if dt.internalMode == dtInsert {
			return ModeInsert
		}
		return ModeNormal
	default:
		if dt.internalMode == dtInsert {
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
		if dt.internalMode == dtInsert {
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
		if dt.internalMode == dtInsert {
			return StyleInsertMode.Render(" -- INSERT -- ") +
				StyleHelpDesc.Render("  Esc: normal  Tab: next field")
		}
		if !dt.govEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit", "h/l", "sub-tab")
	}
	return ""
}

func (dt DataTabEditor) domainHintLine() string {
	if dt.internalMode == dtInsert {
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
	if dt.internalMode == dtInsert {
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
		if dt.govFormIdx < len(dt.governanceFields) {
			return &dt.governanceFields[dt.govFormIdx]
		}
	}
	return nil
}

func (dt DataTabEditor) updateDropdown(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	f := dt.activeDTFieldPtr()
	if f == nil {
		dt.ddOpen = false
		return dt, nil
	}
	switch key.String() {
	case "j", "down":
		if dt.ddOptIdx < len(f.Options)-1 {
			dt.ddOptIdx++
		}
	case "k", "up":
		if dt.ddOptIdx > 0 {
			dt.ddOptIdx--
		}
	case "g":
		dt.ddOptIdx = 0
	case "G":
		if len(f.Options) > 0 {
			dt.ddOptIdx = len(f.Options) - 1
		}
	case " ":
		if f.Kind == KindMultiSelect {
			f.ToggleMultiSelect(dt.ddOptIdx)
			f.DDCursor = dt.ddOptIdx
			// When a custom option is toggled ON, close the dropdown and let the user type.
			if dt.ddOptIdx < len(f.Options) && isCustomOption(f.Options[dt.ddOptIdx]) && f.IsMultiSelected(dt.ddOptIdx) {
				f.CustomText = ""
				dt.ddOpen = false
				return dt.tryEnterInsert()
			}
		} else if f.Kind == KindSelect {
			f.SelIdx = dt.ddOptIdx
			f.Value = f.Options[dt.ddOptIdx]
			dt.ddOpen = false
			if f.PrepareCustomEntry() {
				return dt.tryEnterInsert()
			}
		}
	case "enter":
		if f.Kind == KindMultiSelect {
			f.DDCursor = dt.ddOptIdx
			// Enter on a custom option: toggle it on (if not already) and enter insert mode.
			if dt.ddOptIdx < len(f.Options) && isCustomOption(f.Options[dt.ddOptIdx]) {
				if !f.IsMultiSelected(dt.ddOptIdx) {
					f.ToggleMultiSelect(dt.ddOptIdx)
				}
				f.CustomText = ""
				dt.ddOpen = false
				return dt.tryEnterInsert()
			}
		} else if f.Kind == KindSelect {
			f.SelIdx = dt.ddOptIdx
			f.Value = f.Options[dt.ddOptIdx]
		}
		dt.ddOpen = false
		if f.Kind == KindSelect && f.PrepareCustomEntry() {
			return dt.tryEnterInsert()
		}
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = dt.ddOptIdx
		}
		dt.ddOpen = false
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
	if dt.internalMode == dtInsert {
		return dt.updateInsert(msg)
	}

	// Dropdown mode for multiselect fields
	if dt.ddOpen && ok {
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
			dt.internalMode = dtNormal
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
		n := len(dt.governanceFields)
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
		if dt.govFormIdx < len(dt.governanceFields) && dt.governanceFields[dt.govFormIdx].CanEditAsText() {
			dt.governanceFields[dt.govFormIdx].SaveTextInput(val)
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
		n = len(dt.governanceFields)
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
			if dt.govFormIdx < len(dt.governanceFields) {
				f = &dt.governanceFields[dt.govFormIdx]
			}
		}
		if f == nil {
			break
		}
		if f.CanEditAsText() {
			dt.internalMode = dtInsert
			dt.formInput.SetValue(f.TextInputValue())
			dt.formInput.Width = dt.width - 22
			dt.formInput.CursorEnd()
			return dt, dt.formInput.Focus()
		}
		dt.advanceFormField(1)
	}
	return dt, nil
}

// ── Domain update ─────────────────────────────────────────────────────────────

func (dt DataTabEditor) updateDomains(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch dt.domainSubView {
	case domainViewList:
		return dt.updateDomainList(key)
	case domainViewForm:
		return dt.updateDomainForm(key)
	case domainViewAttrs:
		return dt.updateAttrList(key)
	case domainViewAttrForm:
		return dt.updateAttrForm(key)
	case domainViewRels:
		return dt.updateRelList(key)
	case domainViewRelForm:
		return dt.updateRelForm(key)
	}
	return dt, nil
}

func (dt DataTabEditor) updateDomainList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.domains)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.domainIdx < n-1 {
			dt.domainIdx++
		}
	case "k", "up":
		if dt.domainIdx > 0 {
			dt.domainIdx--
		}
	case "a":
		dt.domains = append(dt.domains, manifest.DomainDef{})
		dt.domainIdx = len(dt.domains) - 1
		dt.domainForm = defaultDomainFormFields(dt.dbNames())
		existing := make([]string, 0, len(dt.domains)-1)
		for i, d := range dt.domains {
			if i != dt.domainIdx {
				existing = append(existing, d.Name)
			}
		}
		dt.domainForm = setFieldValue(dt.domainForm, "name", uniqueName("domain", existing))
		dt.domainFormIdx = 0
		dt.attrItems = nil
		dt.relItems = nil
		dt.domainSubView = domainViewForm
		return dt.tryEnterInsert()
	case "d":
		if n > 0 {
			dt.domains = append(dt.domains[:dt.domainIdx], dt.domains[dt.domainIdx+1:]...)
			if dt.domainIdx > 0 && dt.domainIdx >= len(dt.domains) {
				dt.domainIdx = len(dt.domains) - 1
			}
		}
	case "enter":
		if n > 0 {
			d := dt.domains[dt.domainIdx]
			dbOpts := dt.dbNames()
			dt.domainForm = defaultDomainFormFields(dbOpts)
			dt.domainForm = setFieldValue(dt.domainForm, "name", d.Name)
			dt.domainForm = setFieldValue(dt.domainForm, "description", d.Description)
			// Restore multiselect for databases
			if d.Databases != "" {
				for i := range dt.domainForm {
					if dt.domainForm[i].Key == "databases" {
						for _, sel := range strings.Split(d.Databases, ", ") {
							for j, opt := range dt.domainForm[i].Options {
								if opt == sel {
									dt.domainForm[i].SelectedIdxs = append(dt.domainForm[i].SelectedIdxs, j)
								}
							}
						}
						break
					}
				}
			}
			dt.domainFormIdx = 0
			// Rebuild attr items
			attrTypes, _ := attrTypesForSources(d.Databases, dt.dbEditor.Sources)
			dt.attrItems = make([][]Field, len(d.Attributes))
			for i, attr := range d.Attributes {
				f := defaultAttrFields(attrTypes)
				f = setFieldValue(f, "name", attr.Name)
				f = setFieldValue(f, "type", attr.Type)
				f = restoreMultiSelectValue(f, "constraints", attr.Constraints)
				f = setFieldValue(f, "default", attr.Default)
				if attr.Sensitive {
					f = setFieldValue(f, "sensitive", "true")
				}
				f = restoreMultiSelectValue(f, "validation", attr.Validation)
				dt.attrItems[i] = f
			}
			// Rebuild rel items
			domOpts := dt.domainNames()
			dt.relItems = make([][]Field, len(d.Relationships))
			for i, rel := range d.Relationships {
				f := defaultRelFields(domOpts)
				if rel.RelatedDomain != "" {
					f = setFieldValue(f, "related_domain", rel.RelatedDomain)
				}
				if rel.RelType != "" {
					f = setFieldValue(f, "rel_type", rel.RelType)
				}
				if rel.Cascade != "" {
					f = setFieldValue(f, "cascade", rel.Cascade)
				}
				dt.relItems[i] = f
			}
			dt.domainSubView = domainViewForm
		}
	}
	return dt, nil
}

func (dt DataTabEditor) updateDomainForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.domainFormIdx < len(dt.domainForm)-1 {
			dt.domainFormIdx++
		}
	case "k", "up":
		if dt.domainFormIdx > 0 {
			dt.domainFormIdx--
		}
	case "enter", " ":
		f := &dt.domainForm[dt.domainFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			dt.ddOpen = true
			if f.Kind == KindSelect {
				dt.ddOptIdx = f.SelIdx
			} else {
				dt.ddOptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.domainForm[dt.domainFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.domainForm[dt.domainFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "A":
		dt.saveDomainForm()
		dt.attrIdx = 0
		dt.domainSubView = domainViewAttrs
	case "R":
		dt.saveDomainForm()
		dt.relIdx = 0
		dt.domainSubView = domainViewRels
	case "b", "esc":
		dt.saveDomainForm()
		dt.domainSubView = domainViewList
	}
	return dt, nil
}

func (dt *DataTabEditor) saveDomainForm() {
	if dt.domainIdx >= len(dt.domains) {
		return
	}
	d := &dt.domains[dt.domainIdx]
	d.Name = fieldGet(dt.domainForm, "name")
	d.Description = fieldGet(dt.domainForm, "description")
	d.Databases = fieldGetMulti(dt.domainForm, "databases")

	// Parse attr_names: comma-separated names create new attributes (if typed)
	dt.processAttrNames()

	// Save attrs
	d.Attributes = make([]manifest.DomainAttribute, len(dt.attrItems))
	for i, item := range dt.attrItems {
		d.Attributes[i] = manifest.DomainAttribute{
			Name:        fieldGet(item, "name"),
			Type:        fieldGet(item, "type"),
			Constraints: fieldGet(item, "constraints"),
			Default:     fieldGet(item, "default"),
			Sensitive:   fieldGet(item, "sensitive") == "true",
			Validation:  fieldGet(item, "validation"),
			Indexed:     fieldGet(item, "indexed") == "true",
			Unique:      fieldGet(item, "unique") == "true",
		}
	}

	// Save rels (no FK field — auto-inferred from rel_type)
	d.Relationships = make([]manifest.DomainRelationship, len(dt.relItems))
	for i, item := range dt.relItems {
		relType := fieldGet(item, "rel_type")
		relDomain := fieldGet(item, "related_domain")
		// Auto-generate FK name
		fk := ""
		if relDomain != "" {
			switch relType {
			case "One-to-Many":
				fk = strings.ToLower(relDomain) + "_id"
			case "One-to-One":
				fk = strings.ToLower(relDomain) + "_id"
			case "Many-to-Many":
				fk = "" // junction table; no single FK
			}
		}
		d.Relationships[i] = manifest.DomainRelationship{
			RelatedDomain: relDomain,
			RelType:       relType,
			ForeignKey:    fk,
			Cascade:       fieldGet(item, "cascade"),
		}
	}
}

// saveDomainAttrItemsOnly saves attrItems back to the current domain's Attributes
// without touching name/description/databases/attr_names fields.
func (dt *DataTabEditor) saveDomainAttrItemsOnly() {
	if dt.domainIdx >= len(dt.domains) {
		return
	}
	d := &dt.domains[dt.domainIdx]
	d.Attributes = make([]manifest.DomainAttribute, len(dt.attrItems))
	for i, item := range dt.attrItems {
		d.Attributes[i] = manifest.DomainAttribute{
			Name:        fieldGet(item, "name"),
			Type:        fieldGet(item, "type"),
			Constraints: fieldGet(item, "constraints"),
			Default:     fieldGet(item, "default"),
			Sensitive:   fieldGet(item, "sensitive") == "true",
			Validation:  fieldGet(item, "validation"),
			Indexed:     fieldGet(item, "indexed") == "true",
			Unique:      fieldGet(item, "unique") == "true",
		}
	}
}

// processAttrNames extracts comma-separated names from the attr_names field,
// adds any missing attributes to attrItems, and clears the field.
func (dt *DataTabEditor) processAttrNames() {
	attrNamesRaw := fieldGet(dt.domainForm, "attr_names")
	if attrNamesRaw == "" {
		return
	}
	for _, p := range strings.Split(attrNamesRaw, ",") {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		found := false
		for _, item := range dt.attrItems {
			if fieldGet(item, "name") == name {
				found = true
				break
			}
		}
		if !found {
			f := defaultAttrFields(dt.currentDomainAttrTypes())
			f = setFieldValue(f, "name", name)
			dt.attrItems = append(dt.attrItems, f)
		}
	}
	for i := range dt.domainForm {
		if dt.domainForm[i].Key == "attr_names" {
			dt.domainForm[i].Value = ""
			break
		}
	}
}

func (dt DataTabEditor) updateAttrList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.attrItems)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.attrIdx < n-1 {
			dt.attrIdx++
		}
	case "k", "up":
		if dt.attrIdx > 0 {
			dt.attrIdx--
		}
	case "a":
		dt.attrItems = append(dt.attrItems, defaultAttrFields(dt.currentDomainAttrTypes()))
		dt.saveDomainAttrItemsOnly()
		dt.attrIdx = len(dt.attrItems) - 1
		dt.attrForm = copyFields(dt.attrItems[dt.attrIdx])
		existing := make([]string, 0, len(dt.attrItems)-1)
		for i, item := range dt.attrItems {
			if i != dt.attrIdx {
				existing = append(existing, fieldGet(item, "name"))
			}
		}
		dt.attrForm = setFieldValue(dt.attrForm, "name", uniqueName("attribute", existing))
		dt.attrFormIdx = 0
		dt.domainSubView = domainViewAttrForm
		return dt.tryEnterInsert()
	case "d":
		if n > 0 {
			dt.attrItems = append(dt.attrItems[:dt.attrIdx], dt.attrItems[dt.attrIdx+1:]...)
			if dt.attrIdx > 0 && dt.attrIdx >= len(dt.attrItems) {
				dt.attrIdx = len(dt.attrItems) - 1
			}
			dt.saveDomainAttrItemsOnly()
		}
	case "enter":
		if n > 0 {
			dt.attrForm = copyFields(dt.attrItems[dt.attrIdx])
			dt.attrForm = refreshAttrTypeOptions(dt.attrForm, dt.currentDomainAttrTypes())
			dt.attrFormIdx = 0
			dt.domainSubView = domainViewAttrForm
		}
	case "b", "esc":
		dt.saveDomainAttrItemsOnly()
		dt.domainSubView = domainViewForm
	}
	return dt, nil
}

func (dt DataTabEditor) updateAttrForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.attrFormIdx < len(dt.attrForm)-1 {
			dt.attrFormIdx++
		}
	case "k", "up":
		if dt.attrFormIdx > 0 {
			dt.attrFormIdx--
		}
	case "enter", " ":
		f := &dt.attrForm[dt.attrFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			dt.ddOpen = true
			if f.Kind == KindSelect {
				dt.ddOptIdx = f.SelIdx
			} else {
				dt.ddOptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.attrForm[dt.attrFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.attrForm[dt.attrFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.attrIdx < len(dt.attrItems) {
			dt.attrItems[dt.attrIdx] = copyFields(dt.attrForm)
		}
		dt.saveDomainAttrItemsOnly()
		dt.domainSubView = domainViewAttrs
	}
	return dt, nil
}

func (dt DataTabEditor) updateRelList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.relItems)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.relIdx < n-1 {
			dt.relIdx++
		}
	case "k", "up":
		if dt.relIdx > 0 {
			dt.relIdx--
		}
	case "a":
		dt.relItems = append(dt.relItems, defaultRelFields(dt.domainNames()))
		dt.relIdx = len(dt.relItems) - 1
		dt.relForm = copyFields(dt.relItems[dt.relIdx])
		dt.relFormIdx = 0
		dt.domainSubView = domainViewRelForm
	case "d":
		if n > 0 {
			dt.relItems = append(dt.relItems[:dt.relIdx], dt.relItems[dt.relIdx+1:]...)
			if dt.relIdx > 0 && dt.relIdx >= len(dt.relItems) {
				dt.relIdx = len(dt.relItems) - 1
			}
		}
	case "enter":
		if n > 0 {
			dt.relForm = copyFields(dt.relItems[dt.relIdx])
			dt.relFormIdx = 0
			dt.domainSubView = domainViewRelForm
		}
	case "b", "esc":
		dt.domainSubView = domainViewForm
	}
	return dt, nil
}

func (dt DataTabEditor) updateRelForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.relFormIdx < len(dt.relForm)-1 {
			dt.relFormIdx++
		}
	case "k", "up":
		if dt.relFormIdx > 0 {
			dt.relFormIdx--
		}
	case "enter", " ":
		f := &dt.relForm[dt.relFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			dt.ddOpen = true
			if f.Kind == KindSelect {
				dt.ddOptIdx = f.SelIdx
			} else {
				dt.ddOptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.relForm[dt.relFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.relForm[dt.relFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.relIdx < len(dt.relItems) {
			dt.relItems[dt.relIdx] = copyFields(dt.relForm)
		}
		dt.domainSubView = domainViewRels
	}
	return dt, nil
}

// ── Caching update ────────────────────────────────────────────────────────────

func (dt DataTabEditor) updateCaching(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch dt.cachingSubView {
	case cachingViewList:
		return dt.updateCachingList(key)
	case cachingViewForm:
		return dt.updateCachingForm(key)
	}
	return dt, nil
}

func (dt DataTabEditor) updateCachingList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.cachings)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.cachingIdx < n-1 {
			dt.cachingIdx++
		}
	case "k", "up":
		if dt.cachingIdx > 0 {
			dt.cachingIdx--
		}
	case "a":
		dt.cachings = append(dt.cachings, manifest.CachingConfig{})
		dt.cachingIdx = len(dt.cachings) - 1
		dt.cachingForm = defaultCachingFields()
		existing := make([]string, 0, len(dt.cachings)-1)
		for i, c := range dt.cachings {
			if i != dt.cachingIdx {
				existing = append(existing, c.Name)
			}
		}
		dt.cachingForm = setFieldValue(dt.cachingForm, "name", uniqueName("caching", existing))
		dt.cachingFormIdx = 0
		dt.cachingSubView = cachingViewForm
	case "d":
		if n > 0 {
			dt.cachings = append(dt.cachings[:dt.cachingIdx], dt.cachings[dt.cachingIdx+1:]...)
			if dt.cachingIdx > 0 && dt.cachingIdx >= len(dt.cachings) {
				dt.cachingIdx = len(dt.cachings) - 1
			}
		}
	case "enter":
		if n > 0 {
			dt.cachingForm = cachingFormFromDef(dt.cachings[dt.cachingIdx])
			dt.cachingFormIdx = 0
			dt.cachingSubView = cachingViewForm
		}
	}
	return dt, nil
}

func (dt DataTabEditor) updateCachingForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	// Refresh dynamic options
	dt = dt.withRefreshedCachingEntities()
	dt = dt.withRefreshedCachingDBs()
	switch key.String() {
	case "j", "down":
		dt.cachingFormIdx = nextCachingFormIdx(dt.cachingForm, dt.cachingFormIdx)
	case "k", "up":
		dt.cachingFormIdx = prevCachingFormIdx(dt.cachingForm, dt.cachingFormIdx)
	case "enter", " ":
		f := &dt.cachingForm[dt.cachingFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			dt.ddOpen = true
			if f.Kind == KindSelect {
				dt.ddOptIdx = f.SelIdx
			} else {
				dt.ddOptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.cachingForm[dt.cachingFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.cachingForm[dt.cachingFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.cachingIdx < len(dt.cachings) {
			dt.cachings[dt.cachingIdx] = cachingDefFromForm(dt.cachingForm)
		}
		dt.cachingSubView = cachingViewList
	}
	return dt, nil
}

// ── Governance update ─────────────────────────────────────────────────────────

func (dt DataTabEditor) updateGovernance(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	if !dt.govEnabled {
		if key.String() == "a" {
			dt.govEnabled = true
			dt.govFormIdx = 0
		}
		return dt, nil
	}
	switch key.String() {
	case "j", "down":
		if dt.govFormIdx < len(dt.governanceFields)-1 {
			dt.govFormIdx++
		}
	case "k", "up":
		if dt.govFormIdx > 0 {
			dt.govFormIdx--
		}
	case "enter", " ":
		f := &dt.governanceFields[dt.govFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			dt.ddOpen = true
			if f.Kind == KindSelect {
				dt.ddOptIdx = f.SelIdx
			} else {
				dt.ddOptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.governanceFields[dt.govFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "D":
		dt.govEnabled = false
		dt.governanceFields = defaultGovernanceFields()
		dt.govFormIdx = 0
	case "i", "a":
		if dt.governanceFields[dt.govFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	}
	return dt, nil
}

// ── File storage update ───────────────────────────────────────────────────────

func (dt DataTabEditor) updateFileStorage(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch dt.fsSubView {
	case fsViewList:
		return dt.updateFSList(key)
	case fsViewForm:
		return dt.updateFSForm(key)
	}
	return dt, nil
}

func (dt DataTabEditor) updateFSList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.fileStorages)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.fsIdx < n-1 {
			dt.fsIdx++
		}
	case "k", "up":
		if dt.fsIdx > 0 {
			dt.fsIdx--
		}
	case "a":
		dt.fileStorages = append(dt.fileStorages, manifest.FileStorageDef{})
		dt.fsIdx = len(dt.fileStorages) - 1
		dt.fsForm = defaultFSFormFields(dt.domainNames())
		existing := make([]string, 0, len(dt.fileStorages)-1)
		for i, fs := range dt.fileStorages {
			if i != dt.fsIdx {
				existing = append(existing, fs.Purpose)
			}
		}
		dt.fsForm = setFieldValue(dt.fsForm, "purpose", uniqueName("storage", existing))
		dt.fsFormIdx = 0
		dt.fsSubView = fsViewForm
	case "d":
		if n > 0 {
			dt.fileStorages = append(dt.fileStorages[:dt.fsIdx], dt.fileStorages[dt.fsIdx+1:]...)
			if dt.fsIdx > 0 && dt.fsIdx >= len(dt.fileStorages) {
				dt.fsIdx = len(dt.fileStorages) - 1
			}
		}
	case "enter":
		if n > 0 {
			dt.fsForm = fsFormFromDef(dt.fileStorages[dt.fsIdx], dt.domainNames())
			dt.fsFormIdx = 0
			dt.fsSubView = fsViewForm
		}
	}
	return dt, nil
}

func (dt DataTabEditor) updateFSForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.fsFormIdx < len(dt.fsForm)-1 {
			dt.fsFormIdx++
		}
	case "k", "up":
		if dt.fsFormIdx > 0 {
			dt.fsFormIdx--
		}
	case "enter", " ":
		f := &dt.fsForm[dt.fsFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			dt.ddOpen = true
			if f.Kind == KindSelect {
				dt.ddOptIdx = f.SelIdx
			} else {
				dt.ddOptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.fsForm[dt.fsFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.fsForm[dt.fsFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.fsIdx < len(dt.fileStorages) {
			dt.fileStorages[dt.fsIdx] = fsDefFromForm(dt.fsForm)
		}
		dt.fsSubView = fsViewList
	}
	return dt, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

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
	}

	lines = append(lines, contentLines...)
	return fillTildes(lines, h)
}

func (dt DataTabEditor) viewDomains(w int) []string {
	switch dt.domainSubView {
	case domainViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Domains — a: add  d: delete  Enter: edit"), "")
		if len(dt.domains) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no domains yet — press 'a' to add one)"))
		} else {
			for i, d := range dt.domains {
				desc := d.Description
				if len(desc) > 40 {
					desc = desc[:39] + "…"
				}
				lines = append(lines, renderListItem(w, i == dt.domainIdx, "  ▶ ", d.Name, desc))
			}
		}
		return lines

	case domainViewForm:
		var lines []string
		name := fieldGet(dt.domainForm, "name")
		if name == "" {
			name = "(new domain)"
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFields(w, dt.domainForm, dt.domainFormIdx, dt.internalMode == dtInsert, dt.formInput, dt.ddOpen, dt.ddOptIdx)...)
		lines = append(lines, "", StyleSectionDesc.Render("  A: edit attributes  R: edit relationships"))
		attrCount := len(dt.attrItems)
		relCount := len(dt.relItems)
		lines = append(lines, StyleSectionDesc.Render(fmt.Sprintf("  %d attribute(s)  %d relationship(s)", attrCount, relCount)))
		return lines

	case domainViewAttrs:
		var lines []string
		if dt.domainIdx < len(dt.domains) {
			lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(dt.domains[dt.domainIdx].Name)+StyleSectionDesc.Render(" › Attributes"), "")
		}
		if len(dt.attrItems) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no attributes — press 'a' to add)"))
		} else {
			for i, item := range dt.attrItems {
				attrName := fieldGet(item, "name")
				attrType := fieldGet(item, "type")
				if attrName == "" {
					attrName = fmt.Sprintf("(attr #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == dt.attrIdx, "  ▶ ", attrName, attrType))
			}
		}
		return lines

	case domainViewAttrForm:
		var lines []string
		attrName := fieldGet(dt.attrForm, "name")
		if attrName == "" {
			attrName = "(new attribute)"
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(attrName), "")
		lines = append(lines, renderFormFields(w, dt.attrForm, dt.attrFormIdx, dt.internalMode == dtInsert, dt.formInput, dt.ddOpen, dt.ddOptIdx)...)
		return lines

	case domainViewRels:
		var lines []string
		if dt.domainIdx < len(dt.domains) {
			lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(dt.domains[dt.domainIdx].Name)+StyleSectionDesc.Render(" › Relationships"), "")
		}
		if len(dt.relItems) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no relationships — press 'a' to add)"))
		} else {
			for i, item := range dt.relItems {
				relName := fieldGet(item, "related_domain")
				relType := fieldGet(item, "rel_type")
				if relName == "" {
					relName = fmt.Sprintf("(rel #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == dt.relIdx, "  ▶ ", relName, relType))
			}
		}
		return lines

	case domainViewRelForm:
		var lines []string
		relName := fieldGet(dt.relForm, "related_domain")
		if relName == "" {
			relName = "(new relationship)"
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(relName), "")
		lines = append(lines, renderFormFields(w, dt.relForm, dt.relFormIdx, dt.internalMode == dtInsert, dt.formInput, dt.ddOpen, dt.ddOptIdx)...)
		return lines
	}
	return nil
}

func (dt DataTabEditor) viewCaching(w int) []string {
	var lines []string
	switch dt.cachingSubView {
	case cachingViewList:
		lines = append(lines, StyleSectionDesc.Render("  # Caching Strategies — a: add  d: delete  Enter: edit"), "")
		if len(dt.cachings) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no caching strategies yet — press 'a' to add)"))
		} else {
			for i, c := range dt.cachings {
				name := c.Name
				if name == "" {
					name = fmt.Sprintf("(strategy #%d)", i+1)
				}
				detail := c.Layer
				if c.Strategy != "" {
					detail += " / " + c.Strategy
				}
				lines = append(lines, renderListItem(w, i == dt.cachingIdx, "  ▶ ", name, detail))
			}
		}
	case cachingViewForm:
		dt = dt.withRefreshedCachingEntities()
		dt = dt.withRefreshedCachingDBs()
		name := fieldGet(dt.cachingForm, "name")
		if name == "" {
			name = "(new strategy)"
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		visible := cachingVisibleFields(dt.cachingForm)
		visIdx := cachingVisibleIdx(dt.cachingForm, dt.cachingFormIdx)
		lines = append(lines, renderFormFields(w, visible, visIdx, dt.internalMode == dtInsert, dt.formInput, dt.ddOpen, dt.ddOptIdx)...)
	}
	return lines
}

func (dt DataTabEditor) viewFileStorage(w int) []string {
	switch dt.fsSubView {
	case fsViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # File / Object Storage — a: add  d: delete  Enter: edit"), "")
		if len(dt.fileStorages) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no storage buckets yet — press 'a' to add)"))
		} else {
			for i, fs := range dt.fileStorages {
				tech := fs.Technology
				if tech == "" {
					tech = "?"
				}
				name := fs.Purpose
				if name == "" {
					name = fmt.Sprintf("(storage #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == dt.fsIdx, "  ▶ ", name, tech+" / "+fs.Access))
			}
		}
		return lines

	case fsViewForm:
		var lines []string
		tech := fieldGet(dt.fsForm, "technology")
		if tech == "" {
			tech = "(new storage)"
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(tech), "")
		lines = append(lines, renderFormFields(w, dt.fsForm, dt.fsFormIdx, dt.internalMode == dtInsert, dt.formInput, dt.ddOpen, dt.ddOptIdx)...)
		return lines
	}
	return nil
}

func (dt DataTabEditor) viewGovernance(w int) []string {
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  # Data Governance & Privacy"), "")
	if !dt.govEnabled {
		lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		return lines
	}
	lines = append(lines, renderFormFields(w, dt.governanceFields, dt.govFormIdx, dt.internalMode == dtInsert, dt.formInput, dt.ddOpen, dt.ddOptIdx)...)
	return lines
}

// Expose db sources for syncing into the DataEditor.
func (dt DataTabEditor) DBSources() []manifest.DBSourceDef {
	return dt.dbEditor.Sources
}
