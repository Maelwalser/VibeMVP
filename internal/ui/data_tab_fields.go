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

// SetEnvironmentNames injects environment names from the infra tab so that
// database forms show an environment selector dropdown.
func (dt *DataTabEditor) SetEnvironmentNames(names []string) {
	dt.dbEditor.SetEnvironmentNames(names)
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

