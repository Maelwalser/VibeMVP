package ui

import (
	"strings"

	"github.com/vibe-menu/internal/manifest"
)

// ── Caching, file storage, and governance field definitions ───────────────────

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
			Options: []string{"Cache-aside", "Read-through", "Write-through", "Write-behind", "CDN purge"},
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

// updateSearchTechOptions refreshes search_tech options in the active governance
// form when it is open (called after the DATABASES sub-tab changes).
func (dt *DataTabEditor) updateSearchTechOptions() {
	if dt.govSubView == govViewForm {
		dt.govForm = dt.withRefreshedGovOptions(dt.govForm)
	}
}

// SetServiceNames updates the backend service names used to populate the
// service selector in file storage forms.
func (dt *DataTabEditor) SetServiceNames(names []string) {
	if stringSlicesEqual(dt.serviceNames, names) {
		return
	}
	dt.serviceNames = names
}

// archivalStorageByProvider maps infra cloud_provider → valid archival storage options.
var archivalStorageByProvider = map[string][]string{
	"AWS":         {"S3 Glacier", "S3 Glacier Deep Archive", "None"},
	"GCP":         {"GCS Archive", "GCS Coldline", "None"},
	"Azure":       {"Azure Archive", "Azure Cool", "None"},
	"Cloudflare":  {"None"},
	"Hetzner":     {"On-premise", "None"},
	"Self-hosted": {"On-premise", "None"},
}

// archivalStorageOptionsFor returns the archival storage options for the given
// cloud provider, falling back to all options when unset or unrecognised.
func archivalStorageOptionsFor(provider string) []string {
	if opts, ok := archivalStorageByProvider[provider]; ok {
		return opts
	}
	return []string{"S3 Glacier", "GCS Archive", "Azure Archive", "On-premise", "None"}
}

// fsStorageByProvider maps infra cloud_provider → valid file storage technologies.
var fsStorageByProvider = map[string][]string{
	"AWS":         {"S3", "MinIO", "Local disk"},
	"GCP":         {"GCS", "MinIO", "Local disk"},
	"Azure":       {"Azure Blob", "MinIO", "Local disk"},
	"Cloudflare":  {"Cloudflare R2", "S3", "Local disk"},
	"Hetzner":     {"MinIO", "S3", "Local disk"},
	"Self-hosted": {"MinIO", "Local disk"},
}

// fsStorageOptionsFor returns the technology options for the given cloud provider,
// falling back to all options when the provider is unset or unrecognised.
func fsStorageOptionsFor(provider string) []string {
	if opts, ok := fsStorageByProvider[provider]; ok {
		return opts
	}
	return []string{"S3", "GCS", "Azure Blob", "MinIO", "Cloudflare R2", "Local disk"}
}

// SetCloudProvider updates the infra cloud provider and narrows the technology
// options in the active FS form (if open) and the archival storage options in
// the active governance form (if open), and any re-opened forms thereafter.
func (dt *DataTabEditor) SetCloudProvider(provider string) {
	if dt.cloudProvider == provider {
		return
	}
	dt.cloudProvider = provider

	// Refresh FS form technology options.
	if len(dt.fsForm) > 0 {
		newOpts := fsStorageOptionsFor(provider)
		for i := range dt.fsForm {
			if dt.fsForm[i].Key != "technology" {
				continue
			}
			current := dt.fsForm[i].DisplayValue()
			dt.fsForm[i].Options = newOpts
			found := false
			for j, opt := range newOpts {
				if opt == current {
					dt.fsForm[i].SelIdx = j
					dt.fsForm[i].Value = opt
					found = true
					break
				}
			}
			if !found {
				dt.fsForm[i].SelIdx = 0
				dt.fsForm[i].Value = newOpts[0]
			}
			break
		}
	}

	// Refresh governance form archival storage options.
	if dt.govSubView == govViewForm {
		dt.govForm = dt.withRefreshedGovOptions(dt.govForm)
	}
}

// SetMigrationContext updates the backend languages used to filter migration tool
// options. Refreshes the active governance form if it is open.
func (dt *DataTabEditor) SetMigrationContext(langs []string) {
	if stringSlicesEqual(dt.backendLangs, langs) {
		return
	}
	dt.backendLangs = langs
	if dt.govSubView == govViewForm {
		dt.govForm = dt.withRefreshedGovOptions(dt.govForm)
	}
}

// SetDTONames injects DTO names from the contracts tab so that the caching
// entities multiselect includes both domain names and DTO names.
func (dt *DataTabEditor) SetDTONames(names []string) {
	if stringSlicesEqual(dt.availableDTOs, names) {
		return
	}
	dt.availableDTOs = names
}

// SetEnvironmentNames injects environment names from the infra tab so that
// database forms and file storage forms show an environment selector dropdown.
func (dt *DataTabEditor) SetEnvironmentNames(names []string) {
	dt.dbEditor.SetEnvironmentNames(names)
	if stringSlicesEqual(dt.environmentNames, names) {
		return
	}
	dt.environmentNames = names
	if len(dt.fsForm) == 0 {
		return
	}
	opts, defaultVal := noneOrPlaceholder(names, "(no environments configured)")
	for i := range dt.fsForm {
		if dt.fsForm[i].Key != "environment" {
			continue
		}
		current := dt.fsForm[i].DisplayValue()
		dt.fsForm[i].Options = opts
		found := false
		for j, opt := range opts {
			if opt == current {
				dt.fsForm[i].SelIdx = j
				dt.fsForm[i].Value = opt
				found = true
				break
			}
		}
		if !found {
			dt.fsForm[i].SelIdx = 0
			dt.fsForm[i].Value = defaultVal
		}
		break
	}
}

// ── Governance form fields ─────────────────────────────────────────────────────

// govDbCategory classifies a database type string into a broad category used
// for governance option filtering.
func govDbCategory(dbType string, isCache bool) string {
	if isCache {
		return "cache"
	}
	switch dbType {
	case "PostgreSQL", "MySQL", "SQLite":
		return "relational"
	case "MongoDB", "DynamoDB":
		return "document"
	case "Redis", "Memcached":
		return "cache"
	case "ClickHouse":
		return "analytics"
	case "Elasticsearch":
		return "analytics"
	case "Cassandra":
		return "wide-column"
	default:
		return "relational"
	}
}

func allGovCategories(cats []string, cat string) bool {
	if len(cats) == 0 {
		return false
	}
	for _, c := range cats {
		if c != cat {
			return false
		}
	}
	return true
}

func govRetentionOptions(cats []string) []string {
	if allGovCategories(cats, "cache") {
		return []string{"1 hour", "24 hours", "7 days", "30 days", "Custom"}
	}
	if allGovCategories(cats, "analytics") {
		return []string{"7 days", "30 days", "90 days", "1 year", "3 years", "Indefinite", "Custom"}
	}
	return []string{"30 days", "90 days", "1 year", "3 years", "7 years", "Indefinite", "Custom"}
}

func govDeleteStrategyOptions(cats []string) []string {
	if allGovCategories(cats, "cache") {
		return []string{"TTL expiry", "Manual flush", "LRU eviction"}
	}
	if allGovCategories(cats, "analytics") {
		return []string{"Time-based drop", "Compaction", "Archival", "Manual purge"}
	}
	return []string{"Soft-delete", "Hard-delete", "Archival", "Soft + periodic purge"}
}

// backupStrategyByProvider maps infra cloud_provider → provider-specific backup options.
var backupStrategyByProvider = map[string][]string{
	"AWS":         {"AWS Backup", "RDS automated snapshots", "Manual snapshots", "None"},
	"GCP":         {"Cloud SQL backups", "Manual snapshots", "None"},
	"Self-hosted": {"pg_dump/mongodump cron", "Manual snapshots", "None"},
}

func govBackupStrategyOptions(cats []string, cloudProvider string) []string {
	if allGovCategories(cats, "cache") {
		return []string{"RDB snapshot", "AOF persistence", "None"}
	}
	if opts, ok := backupStrategyByProvider[cloudProvider]; ok {
		return opts
	}
	return []string{"Automated daily", "Point-in-time recovery", "Manual snapshots", "Managed provider DR", "None"}
}

// govSelectedCategories returns the DB categories for the databases selected
// in the governance form.
func (dt DataTabEditor) govSelectedCategories(form []Field) []string {
	var aliases []string
	for _, f := range form {
		if f.Key == "databases" {
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					aliases = append(aliases, f.Options[idx])
				}
			}
			break
		}
	}
	if len(aliases) == 0 {
		return nil
	}
	var cats []string
	for _, alias := range aliases {
		for _, src := range dt.dbEditor.Sources {
			if src.Alias == alias {
				cats = append(cats, govDbCategory(string(src.Type), src.IsCache))
				break
			}
		}
	}
	return cats
}

// isGovFieldDisabled returns true when a governance form field should be hidden
// based on the selected database categories.
func (dt DataTabEditor) isGovFieldDisabled(form []Field, idx int) bool {
	key := form[idx].Key
	if key != "migration_tool" && key != "archival_storage" {
		return false
	}
	cats := dt.govSelectedCategories(form)
	if len(cats) == 0 {
		return false
	}
	switch key {
	case "migration_tool":
		return allGovCategories(cats, "cache") || allGovCategories(cats, "analytics")
	case "archival_storage":
		return allGovCategories(cats, "cache")
	}
	return false
}

// withRefreshedGovOptions returns a copy of the governance form with all
// DB-aware field options recomputed from the currently selected databases.
func (dt DataTabEditor) withRefreshedGovOptions(form []Field) []Field {
	cats := dt.govSelectedCategories(form)
	retentionOpts := govRetentionOptions(cats)
	deleteOpts := govDeleteStrategyOptions(cats)
	backupOpts := govBackupStrategyOptions(cats, dt.cloudProvider)
	searchOpts := searchTechForSources(dt.dbEditor.Sources)

	migrationOpts := migrationToolsForLangs(dt.backendLangs)
	if allGovCategories(cats, "cache") || allGovCategories(cats, "analytics") {
		migrationOpts = []string{"N/A"}
	}

	archivalOpts := archivalStorageOptionsFor(dt.cloudProvider)

	// Also refresh the databases multiselect options from current DB aliases.
	dbAliases := dt.dbNames()

	newForm := make([]Field, len(form))
	copy(newForm, form)
	for i := range newForm {
		switch newForm[i].Key {
		case "databases":
			newForm[i] = refreshMultiSelectOptions(newForm[i], dbAliases, "(no databases configured)")
		case "retention_policy":
			newForm[i] = preserveSelectOption(newForm[i], retentionOpts)
		case "delete_strategy":
			newForm[i] = preserveSelectOption(newForm[i], deleteOpts)
		case "backup_strategy":
			newForm[i] = preserveSelectOption(newForm[i], backupOpts)
		case "migration_tool":
			newForm[i] = preserveSelectOption(newForm[i], migrationOpts)
		case "search_tech":
			newForm[i] = preserveSelectOption(newForm[i], searchOpts)
		case "archival_storage":
			newForm[i] = preserveSelectOption(newForm[i], archivalOpts)
		}
	}
	return newForm
}

// preserveSelectOption updates a KindSelect field's Options, keeping the
// current Value selected if still present; otherwise selects the last option.
func preserveSelectOption(f Field, opts []string) Field {
	cur := f.DisplayValue()
	f.Options = opts
	for j, opt := range opts {
		if opt == cur {
			f.SelIdx = j
			f.Value = opt
			return f
		}
	}
	if len(opts) > 0 {
		last := len(opts) - 1
		f.SelIdx = last
		f.Value = opts[last]
	}
	return f
}

// refreshMultiSelectOptions updates a KindMultiSelect field's Options,
// re-mapping existing selections by value rather than by index.
func refreshMultiSelectOptions(f Field, opts []string, placeholder string) Field {
	oldOpts := f.Options
	f.Options = opts
	f.Value = placeholderFor(opts, placeholder)
	newSelected := make([]int, 0, len(f.SelectedIdxs))
	for _, oldIdx := range f.SelectedIdxs {
		if oldIdx >= len(oldOpts) {
			continue
		}
		oldVal := oldOpts[oldIdx]
		for j, newOpt := range opts {
			if newOpt == oldVal {
				newSelected = append(newSelected, j)
				break
			}
		}
	}
	f.SelectedIdxs = newSelected
	return f
}

func defaultGovFormFields(dbAliases []string, cloudProvider string) []Field {
	dbPlaceholder := placeholderFor(dbAliases, "(no databases configured)")
	archivalOpts := archivalStorageOptionsFor(cloudProvider)
	backupOpts := govBackupStrategyOptions(nil, cloudProvider)
	return []Field{
		{Key: "name", Label: "policy name   ", Kind: KindText},
		{
			Key: "databases", Label: "applies to    ", Kind: KindMultiSelect,
			Options: dbAliases,
			Value:   dbPlaceholder,
		},
		{
			Key: "retention_policy", Label: "retention     ", Kind: KindSelect,
			Options: []string{"30 days", "90 days", "1 year", "3 years", "7 years", "Indefinite", "Custom"},
			Value:   "Indefinite", SelIdx: 5,
		},
		{
			Key: "delete_strategy", Label: "delete strat. ", Kind: KindSelect,
			Options: []string{"Soft-delete", "Hard-delete", "Archival", "Soft + periodic purge"},
			Value:   "Soft-delete",
		},
		{
			Key: "pii_encryption", Label: "pii encryption", Kind: KindSelect,
			Options: []string{"Field-level AES-256", "Full database encryption", "Application-level", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "compliance_frameworks", Label: "compliance    ", Kind: KindMultiSelect,
			Options: []string{"GDPR", "HIPAA", "SOC2 Type II", "PCI-DSS", "ISO-27001", "CCPA", "PIPEDA"},
		},
		{
			Key: "data_residency", Label: "data residency", Kind: KindSelect,
			Options: []string{"US", "EU", "APAC", "US + EU", "Global", "Custom"},
			Value:   "Global", SelIdx: 4,
		},
		{
			Key: "archival_storage", Label: "archival      ", Kind: KindSelect,
			Options: archivalOpts,
			Value:   archivalOpts[len(archivalOpts)-1],
			SelIdx:  len(archivalOpts) - 1,
		},
		{
			Key: "migration_tool", Label: "migration     ", Kind: KindSelect,
			Options: allMigrationTools,
			Value:   "None", SelIdx: len(allMigrationTools) - 1,
		},
		{
			Key: "backup_strategy", Label: "backup strat. ", Kind: KindSelect,
			Options: backupOpts,
			Value:   "None",
			SelIdx:  len(backupOpts) - 1,
		},
		{
			Key: "search_tech", Label: "search tech   ", Kind: KindSelect,
			Options: []string{"Elasticsearch", "Meilisearch", "Algolia", "Typesense", "None"},
			Value:   "None", SelIdx: 4,
		},
	}
}

func govFormFromDef(def manifest.DataGovernanceConfig, dbAliases []string, cloudProvider string) []Field {
	f := defaultGovFormFields(dbAliases, cloudProvider)
	f = setFieldValue(f, "name", def.Name)
	if len(def.Databases) > 0 {
		f = restoreMultiSelectValue(f, "databases", strings.Join(def.Databases, ", "))
	}
	if def.RetentionPolicy != "" {
		f = setFieldValue(f, "retention_policy", def.RetentionPolicy)
	}
	if def.DeleteStrategy != "" {
		f = setFieldValue(f, "delete_strategy", def.DeleteStrategy)
	}
	if def.PIIEncryption != "" {
		f = setFieldValue(f, "pii_encryption", def.PIIEncryption)
	}
	if def.ComplianceFrameworks != "" {
		f = restoreMultiSelectValue(f, "compliance_frameworks", def.ComplianceFrameworks)
	}
	if def.DataResidency != "" {
		f = setFieldValue(f, "data_residency", def.DataResidency)
	}
	if def.ArchivalStorage != "" {
		f = setFieldValue(f, "archival_storage", def.ArchivalStorage)
	}
	if def.MigrationTool != "" {
		f = setFieldValue(f, "migration_tool", def.MigrationTool)
	}
	if def.BackupStrategy != "" {
		f = setFieldValue(f, "backup_strategy", def.BackupStrategy)
	}
	if def.SearchTech != "" {
		f = setFieldValue(f, "search_tech", def.SearchTech)
	}
	return f
}

func govDefFromForm(fields []Field) manifest.DataGovernanceConfig {
	var dbs []string
	for _, f := range fields {
		if f.Key == "databases" {
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					dbs = append(dbs, f.Options[idx])
				}
			}
			break
		}
	}
	return manifest.DataGovernanceConfig{
		Name:                 fieldGet(fields, "name"),
		Databases:            dbs,
		RetentionPolicy:      fieldGet(fields, "retention_policy"),
		DeleteStrategy:       fieldGet(fields, "delete_strategy"),
		PIIEncryption:        fieldGet(fields, "pii_encryption"),
		ComplianceFrameworks: fieldGetMulti(fields, "compliance_frameworks"),
		DataResidency:        fieldGet(fields, "data_residency"),
		ArchivalStorage:      fieldGet(fields, "archival_storage"),
		MigrationTool:        fieldGet(fields, "migration_tool"),
		BackupStrategy:       fieldGet(fields, "backup_strategy"),
		SearchTech:           fieldGet(fields, "search_tech"),
	}
}

// withRefreshedCachingEntities returns a copy of the DataTabEditor with the
// entities multiselect options in cachingForm updated to reflect current domain
// and DTO names.
func (dt DataTabEditor) withRefreshedCachingEntities() DataTabEditor {
	domOpts := dt.domainNames()
	opts := make([]string, 0, len(domOpts)+len(dt.availableDTOs))
	opts = append(opts, domOpts...)
	for _, name := range dt.availableDTOs {
		opts = append(opts, "dto:"+name)
	}
	newFields := make([]Field, len(dt.cachingForm))
	copy(newFields, dt.cachingForm)
	for i := range newFields {
		if newFields[i].Key == "entities" {
			// Preserve existing selections by re-mapping
			oldOpts := newFields[i].Options
			newFields[i].Options = opts
			newFields[i].Value = placeholderFor(opts, "(no domains or DTOs configured)")
			newSelected := make([]int, 0)
			for _, oldIdx := range newFields[i].SelectedIdxs {
				if oldIdx < len(oldOpts) {
					oldVal := oldOpts[oldIdx]
					for j, newOpt := range opts {
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
	return nextFormIdx(fields, cur, isCachingFieldDisabled)
}

func prevCachingFormIdx(fields []Field, cur int) int {
	return prevFormIdx(fields, cur, isCachingFieldDisabled)
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

// strategyOptionsForLayer returns the valid caching strategy options for a given layer.
// CDN caching only supports read-oriented and CDN-specific strategies;
// application-level and dedicated cache do not expose CDN purge.
func strategyOptionsForLayer(layer string) []string {
	switch layer {
	case "CDN":
		return []string{"Cache-aside", "Read-through", "CDN purge"}
	case "Application-level", "Dedicated cache":
		return []string{"Cache-aside", "Read-through", "Write-through", "Write-behind"}
	default:
		return []string{"Cache-aside", "Read-through", "Write-through", "Write-behind", "CDN purge"}
	}
}

// withRefreshedCachingStrategies returns a copy of the DataTabEditor with the
// strategy multiselect options filtered to those valid for the currently selected layer.
// Existing selections that are no longer valid are dropped.
func (dt DataTabEditor) withRefreshedCachingStrategies() DataTabEditor {
	layer := fieldGet(dt.cachingForm, "layer")
	validOpts := strategyOptionsForLayer(layer)

	newFields := make([]Field, len(dt.cachingForm))
	copy(newFields, dt.cachingForm)
	for i := range newFields {
		if newFields[i].Key != "strategy" {
			continue
		}
		oldOpts := newFields[i].Options
		newFields[i].Options = validOpts
		// Re-map selected indices, dropping any that are no longer valid
		newSelected := make([]int, 0, len(newFields[i].SelectedIdxs))
		for _, oldIdx := range newFields[i].SelectedIdxs {
			if oldIdx >= len(oldOpts) {
				continue
			}
			oldVal := oldOpts[oldIdx]
			for j, opt := range validOpts {
				if opt == oldVal {
					newSelected = append(newSelected, j)
					break
				}
			}
		}
		newFields[i].SelectedIdxs = newSelected
		// Clamp DDCursor
		if newFields[i].DDCursor >= len(validOpts) {
			newFields[i].DDCursor = 0
		}
		break
	}
	dt.cachingForm = newFields
	return dt
}

func defaultFSFormFields(domainOptions []string, cloudProvider string, environmentNames []string, serviceOptions []string) []Field {
	techOpts := fsStorageOptionsFor(cloudProvider)
	envOpts, envDefault := noneOrPlaceholder(environmentNames, "(no environments configured)")
	return []Field{
		{
			Key: "technology", Label: "technology    ", Kind: KindSelect,
			Options: techOpts,
			Value:   techOpts[0],
		},
		{Key: "purpose", Label: "purpose       ", Kind: KindText},
		{
			Key: "used_by_service", Label: "used_by       ", Kind: KindSelect,
			Options: append([]string{"(any / unspecified)"}, serviceOptions...),
			Value:   "(any / unspecified)",
		},
		{
			Key: "environment", Label: "environment   ", Kind: KindSelect,
			Options: envOpts,
			Value:   envDefault,
		},
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

func fsFormFromDef(def manifest.FileStorageDef, domainOptions []string, cloudProvider string, environmentNames []string, serviceOptions []string) []Field {
	f := defaultFSFormFields(domainOptions, cloudProvider, environmentNames, serviceOptions)
	f = setFieldValue(f, "technology", def.Technology)
	f = setFieldValue(f, "purpose", def.Purpose)
	if def.UsedByService != "" {
		f = setFieldValue(f, "used_by_service", def.UsedByService)
	}
	f = setFieldValue(f, "environment", def.Environment)
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
	usedBy := fieldGet(fields, "used_by_service")
	if usedBy == "(any / unspecified)" {
		usedBy = ""
	}
	return manifest.FileStorageDef{
		Technology:    fieldGet(fields, "technology"),
		Purpose:       fieldGet(fields, "purpose"),
		UsedByService: usedBy,
		Environment:   fieldGet(fields, "environment"),
		Access:        fieldGet(fields, "access"),
		MaxSize:       fieldGet(fields, "max_size"),
		Domains:       fieldGetMulti(fields, "domains"),
		TTLMinutes:    fieldGet(fields, "ttl_minutes"),
		AllowedTypes:  fieldGetMulti(fields, "allowed_types"),
	}
}
