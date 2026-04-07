package data

import (
	"fmt"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── DB form fields ────────────────────────────────────────────────────────────

// defaultDBForm returns a blank database source form.
func defaultDBForm() []core.Field {
	return []core.Field{
		{Key: "alias", Label: "alias         ", Kind: core.KindText},
		{Key: "type", Label: "type          ", Kind: core.KindSelect,
			Options: []string{
				"PostgreSQL", "MySQL", "SQLite",
				"MongoDB", "DynamoDB",
				"Cassandra",
				"Redis", "Memcached",
				"ClickHouse", "Elasticsearch", "other",
			},
			Value: "PostgreSQL",
		},
		{Key: "version", Label: "version       ", Kind: core.KindText},
		{Key: "namespace", Label: "namespace     ", Kind: core.KindText},
		{Key: "is_cache", Label: "is_cache      ", Kind: core.KindSelect,
			Options: []string{"no", "yes"}, Value: "no",
		},
		// Security / network integrity (conditionally shown by type)
		{Key: "ssl_mode", Label: "  ssl_mode    ", Kind: core.KindSelect,
			Options: []string{"require", "disable", "verify-ca", "verify-full"},
			Value:   "require",
		},
		{Key: "consistency", Label: "  consistency ", Kind: core.KindSelect,
			Options: []string{"strong", "eventual", "LOCAL_QUORUM", "ONE", "QUORUM", "ALL", "LOCAL_ONE"},
			Value:   "strong",
		},
		// Availability topology (conditionally shown by type)
		{Key: "replication", Label: "  replication ", Kind: core.KindSelect,
			Options: []string{"single-node", "primary-replica", "multi-region"},
			Value:   "single-node",
		},
		// Connection pooling
		{Key: "pool_min", Label: "  pool_min    ", Kind: core.KindText},
		{Key: "pool_max", Label: "  pool_max    ", Kind: core.KindText},
		// environment is a core.KindSelect populated dynamically from InfraPillar.Environments.
		{
			Key: "environment", Label: "environment   ", Kind: core.KindSelect,
			Options: []string{"(no environments configured)"},
			Value:   "(no environments configured)",
		},
		{Key: "notes", Label: "notes         ", Kind: core.KindText},
	}
}

// isDBFormFieldDisabled returns true when a field is gated by the current db type.
func isDBFormFieldDisabled(form []core.Field, idx int) bool {
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

func nextDBFormIdx(form []core.Field, cur int) int {
	return core.NextFormIdx(form, cur, isDBFormFieldDisabled)
}
func prevDBFormIdx(form []core.Field, cur int) int {
	return core.PrevFormIdx(form, cur, isDBFormFieldDisabled)
}

func dbFormFromSourceWithEnvs(src manifest.DBSourceDef, envNames []string) []core.Field {
	f := dbFormFromSource(src)
	applyEnvNamesToDBForm(f, envNames)
	if src.Environment != "" {
		f = core.SetFieldValue(f, "environment", src.Environment)
	}
	return f
}

func applyEnvNamesToDBForm(fields []core.Field, envNames []string) {
	opts, val := core.NoneOrPlaceholder(envNames, "(no environments configured)")
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

func dbFormFromSource(src manifest.DBSourceDef) []core.Field {
	f := defaultDBForm()
	setVal := func(key, val string) {
		for i := range f {
			if f[i].Key != key {
				continue
			}
			f[i].Value = val
			if f[i].Kind == core.KindSelect {
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

func dbFormToSource(form []core.Field) manifest.DBSourceDef {
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
		_, _ = fmt.Sscanf(v, "%d", &n)
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
