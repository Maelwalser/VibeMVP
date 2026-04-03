package ui

import (
	"fmt"

	"github.com/vibe-menu/internal/manifest"
)

// ── DB form fields ────────────────────────────────────────────────────────────

// defaultDBForm returns a blank database source form.
func defaultDBForm() []Field {
	return []Field{
		{Key: "alias", Label: "alias         ", Kind: KindText},
		{Key: "type", Label: "type          ", Kind: KindSelect,
			Options: []string{
				"PostgreSQL", "MySQL", "SQLite",
				"MongoDB", "DynamoDB",
				"Cassandra",
				"Redis", "Memcached",
				"ClickHouse", "Elasticsearch", "other",
			},
			Value: "PostgreSQL",
		},
		{Key: "version", Label: "version       ", Kind: KindText},
		{Key: "namespace", Label: "namespace     ", Kind: KindText},
		{Key: "is_cache", Label: "is_cache      ", Kind: KindSelect,
			Options: []string{"no", "yes"}, Value: "no",
		},
		// Security / network integrity (conditionally shown by type)
		{Key: "ssl_mode", Label: "  ssl_mode    ", Kind: KindSelect,
			Options: []string{"require", "disable", "verify-ca", "verify-full"},
			Value:   "require",
		},
		{Key: "consistency", Label: "  consistency ", Kind: KindSelect,
			Options: []string{"strong", "eventual", "LOCAL_QUORUM", "ONE", "QUORUM", "ALL", "LOCAL_ONE"},
			Value:   "strong",
		},
		// Availability topology (conditionally shown by type)
		{Key: "replication", Label: "  replication ", Kind: KindSelect,
			Options: []string{"single-node", "primary-replica", "multi-region"},
			Value:   "single-node",
		},
		// Connection pooling
		{Key: "pool_min", Label: "  pool_min    ", Kind: KindText},
		{Key: "pool_max", Label: "  pool_max    ", Kind: KindText},
		// environment is a KindSelect populated dynamically from InfraPillar.Environments.
		{
			Key: "environment", Label: "environment   ", Kind: KindSelect,
			Options: []string{"(no environments configured)"},
			Value:   "(no environments configured)",
		},
		{Key: "notes", Label: "notes         ", Kind: KindText},
	}
}

// isDBFormFieldDisabled returns true when a field is gated by the current db type.
func isDBFormFieldDisabled(form []Field, idx int) bool {
	key := form[idx].Key
	var dbType string
	for _, f := range form {
		if f.Key == "type" {
			dbType = f.DisplayValue()
			break
		}
	}
	switch key {
	case "ssl_mode":
		// Only relational databases support explicit SSL mode configuration
		return dbType != "PostgreSQL" && dbType != "MySQL"
	case "consistency":
		// Distributed DBs with tunable consistency
		return dbType != "Cassandra" && dbType != "MongoDB" && dbType != "DynamoDB"
	case "replication":
		// Cache stores and SQLite don't have meaningful replication topology options
		return dbType == "Redis" || dbType == "Memcached" || dbType == "SQLite"
	case "pool_min", "pool_max":
		// Connection pooling doesn't apply to cache stores
		return dbType == "Redis" || dbType == "Memcached"
	}
	return false
}

func nextDBFormIdx(form []Field, cur int) int { return nextFormIdx(form, cur, isDBFormFieldDisabled) }
func prevDBFormIdx(form []Field, cur int) int { return prevFormIdx(form, cur, isDBFormFieldDisabled) }

func dbFormFromSourceWithEnvs(src manifest.DBSourceDef, envNames []string) []Field {
	f := dbFormFromSource(src)
	applyEnvNamesToDBForm(f, envNames)
	if src.Environment != "" {
		f = setFieldValue(f, "environment", src.Environment)
	}
	return f
}

func applyEnvNamesToDBForm(fields []Field, envNames []string) {
	opts, val := noneOrPlaceholder(envNames, "(no environments configured)")
	for i := range fields {
		if fields[i].Key != "environment" {
			continue
		}
		fields[i].Options = opts
		found := false
		for j, o := range opts {
			if o == fields[i].Value {
				fields[i].SelIdx = j
				found = true
				break
			}
		}
		if !found {
			fields[i].Value = val
			fields[i].SelIdx = 0
		}
		break
	}
}

func dbFormFromSource(src manifest.DBSourceDef) []Field {
	f := defaultDBForm()
	setVal := func(key, val string) {
		for i := range f {
			if f[i].Key != key {
				continue
			}
			f[i].Value = val
			if f[i].Kind == KindSelect {
				for j, opt := range f[i].Options {
					if opt == val {
						f[i].SelIdx = j
						break
					}
				}
			}
			return
		}
	}
	setVal("alias", src.Alias)
	if src.Type != "" {
		setVal("type", string(src.Type))
	}
	setVal("version", src.Version)
	setVal("namespace", src.Namespace)
	if src.IsCache {
		setVal("is_cache", "yes")
	}
	setVal("ssl_mode", src.SSLMode)
	setVal("consistency", src.Consistency)
	setVal("replication", src.Replication)
	if src.PoolMinSize > 0 {
		setVal("pool_min", fmt.Sprintf("%d", src.PoolMinSize))
	}
	if src.PoolMaxSize > 0 {
		setVal("pool_max", fmt.Sprintf("%d", src.PoolMaxSize))
	}
	setVal("notes", src.Notes)
	return f
}

func dbFormToSource(form []Field) manifest.DBSourceDef {
	get := func(key string) string {
		for _, f := range form {
			if f.Key == key {
				return f.DisplayValue()
			}
		}
		return ""
	}
	getInt := func(key string) int {
		v := get(key)
		if v == "" {
			return 0
		}
		n := 0
		fmt.Sscanf(v, "%d", &n)
		return n
	}
	env := get("environment")
	if env == "(no environments configured)" {
		env = ""
	}
	src := manifest.DBSourceDef{
		Alias:       get("alias"),
		Type:        manifest.DatabaseType(get("type")),
		Version:     get("version"),
		Namespace:   get("namespace"),
		IsCache:     get("is_cache") == "yes",
		SSLMode:     get("ssl_mode"),
		Consistency: get("consistency"),
		Replication: get("replication"),
		PoolMinSize: getInt("pool_min"),
		PoolMaxSize: getInt("pool_max"),
		Environment: env,
		Notes:       get("notes"),
	}
	return src
}
