package data

import (
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── Caching, file storage, and governance field definitions ───────────────────

func defaultCachingFields() []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{
			Key: "layer", Label: "layer         ", Kind: core.KindSelect,
			Options: []string{
				"Application-level", "Dedicated cache",
				"CDN", "None",
			},
			Value: "None", SelIdx: 3,
		},
		{
			Key: "cache_db", Label: "cache db      ", Kind: core.KindSelect,
			Options: []string{},
		},
		{
			Key: "strategy", Label: "strategy      ", Kind: core.KindMultiSelect,
			Options: []string{"Cache-aside", "Read-through", "Write-through", "Write-behind", "CDN purge"},
		},
		{
			Key: "invalidation", Label: "invalidation  ", Kind: core.KindSelect,
			Options: []string{"TTL-based", "Event-driven", "Manual", "Hybrid"},
			Value:   "TTL-based",
		},
		{
			Key: "ttl", Label: "ttl           ", Kind: core.KindSelect,
			Options: []string{"30s", "1m", "5m", "15m", "1h", "24h", "Custom"},
			Value:   "5m", SelIdx: 2,
		},
		{
			Key: "entities", Label: "entities      ", Kind: core.KindMultiSelect,
			Options: []string{}, // populated dynamically from domain names
		},
	}
}

func cachingFormFromDef(def manifest.CachingConfig) []core.Field {
	f := defaultCachingFields()
	f = core.SetFieldValue(f, "name", def.Name)
	if def.Layer != "" {
		f = core.SetFieldValue(f, "layer", def.Layer)
	}
	if def.CacheDB != "" {
		f = core.SetFieldValue(f, "cache_db", def.CacheDB)
	}
	if def.Strategy != "" {
		f = core.SetFieldValue(f, "strategy", def.Strategy)
	}
	if def.Invalidation != "" {
		f = core.SetFieldValue(f, "invalidation", def.Invalidation)
	}
	if def.TTL != "" {
		f = core.SetFieldValue(f, "ttl", def.TTL)
	}
	if def.Entities != "" {
		f = core.RestoreMultiSelectValue(f, "entities", def.Entities)
	}
	return f
}

func cachingDefFromForm(fields []core.Field) manifest.CachingConfig {
	return manifest.CachingConfig{
		Name:         core.FieldGet(fields, "name"),
		Layer:        core.FieldGet(fields, "layer"),
		CacheDB:      core.FieldGet(fields, "cache_db"),
		Strategy:     core.FieldGet(fields, "strategy"),
		Invalidation: core.FieldGet(fields, "invalidation"),
		TTL:          core.FieldGet(fields, "ttl"),
		Entities:     core.FieldGetMulti(fields, "entities"),
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
	if core.StringSlicesEqual(dt.serviceNames, names) {
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
	if core.StringSlicesEqual(dt.backendLangs, langs) {
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
	if core.StringSlicesEqual(dt.availableDTOs, names) {
		return
	}
	dt.availableDTOs = names
}

// SetEnvironmentNames injects environment names from the infra tab so that
// database forms and file storage forms show an environment selector dropdown.
func (dt *DataTabEditor) SetEnvironmentNames(names []string) {
	dt.dbEditor.SetEnvironmentNames(names)
	if core.StringSlicesEqual(dt.environmentNames, names) {
		return
	}
	dt.environmentNames = names
	if len(dt.fsForm) == 0 {
		return
	}
	opts, defaultVal := core.NoneOrPlaceholder(names, "(no environments configured)")
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
func (dt DataTabEditor) govSelectedCategories(form []core.Field) []string {
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
func (dt DataTabEditor) isGovFieldDisabled(form []core.Field, idx int) bool {
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
func (dt DataTabEditor) withRefreshedGovOptions(form []core.Field) []core.Field {
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

	newForm := make([]core.Field, len(form))
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

// preserveSelectOption updates a core.KindSelect field's Options, keeping the
// current Value selected if still present; otherwise selects the last option.
func preserveSelectOption(f core.Field, opts []string) core.Field {
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

// refreshMultiSelectOptions updates a core.KindMultiSelect field's Options,
// re-mapping existing selections by value rather than by index.
func refreshMultiSelectOptions(f core.Field, opts []string, placeholder string) core.Field {
	oldOpts := f.Options
	f.Options = opts
	f.Value = core.PlaceholderFor(opts, placeholder)
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
