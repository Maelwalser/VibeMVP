package backend

import (
	"strings"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── language / framework / linter tables ─────────────────────────────────────

var backendFrameworksByLang = map[string][]string{
	"Go":              {"Fiber", "Gin", "Echo", "Chi", "net/http (stdlib)", "Connect"},
	"TypeScript/Node": {"Express", "Fastify", "NestJS", "Hono", "tRPC", "Elysia (Bun)"},
	"Python":          {"FastAPI", "Django", "Flask", "Litestar", "Starlette"},
	"Java":            {"Spring Boot", "Quarkus", "Micronaut", "Jakarta EE"},
	"Kotlin":          {"Ktor", "Spring Boot (Kotlin)", "http4k"},
	"C#/.NET":         {"ASP.NET Core", "Minimal APIs", "Carter"},
	"Rust":            {"Axum", "Actix-web", "Rocket", "Warp"},
	"Ruby":            {"Rails", "Sinatra", "Hanami", "Roda"},
	"PHP":             {"Laravel", "Symfony", "Slim", "Laminas"},
	"Elixir":          {"Phoenix", "Plug", "Bandit"},
	"Other":           {"Other"},
}

var backendLanguages = []string{
	"Go", "TypeScript/Node", "Python", "Java", "Kotlin",
	"C#/.NET", "Rust", "Ruby", "PHP", "Elixir", "Other",
}

// errorFormatByProtocol maps a technology/protocol name to the error formats
// that are idiomatic for it. Used to narrow the error_format dropdown based on
// the technologies selected for a service.
var errorFormatByProtocol = map[string][]string{
	"REST":      {"RFC 7807 (Problem Details)", "Custom JSON envelope"},
	"GraphQL":   {"GraphQL spec errors", "Custom extensions"},
	"gRPC":      {"gRPC status codes", "google.rpc.Status"},
	"WebSocket": {"Custom JSON envelope"},
	"SSE":       {"Custom JSON envelope"},
	"tRPC":      {"tRPC error format"},
}

// errorFormatOptsForTechs derives the appropriate error_format option list from
// the selected technologies. Options from each matching protocol are unioned in
// encounter order; "Platform default" is always appended as the last option.
// If no protocol matches, the static default set is returned.
func errorFormatOptsForTechs(techs []string) []string {
	seen := map[string]bool{"Platform default": true}
	var opts []string
	for _, tech := range techs {
		if formats, ok := errorFormatByProtocol[tech]; ok {
			for _, f := range formats {
				if !seen[f] {
					seen[f] = true
					opts = append(opts, f)
				}
			}
		}
	}
	if len(opts) == 0 {
		return []string{"RFC 7807 (Problem Details)", "Custom JSON envelope", "Platform default"}
	}
	return append(opts, "Platform default")
}

// ── field definitions ─────────────────────────────────────────────────────────
// Default field slices and manifest serialization helpers for BackendEditor.

func defaultStackConfigFields() []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{
			Key: "language", Label: "language      ", Kind: core.KindSelect,
			Options: backendLanguages, Value: "Go",
		},
		{
			Key: "language_version", Label: "lang version  ", Kind: core.KindSelect,
			Options: core.LangVersions["Go"], Value: core.LangVersions["Go"][0],
		},
		{
			Key: "framework", Label: "framework     ", Kind: core.KindSelect,
			Options: backendFrameworksByLang["Go"], Value: "Fiber",
		},
		{
			Key: "framework_version", Label: "fw version    ", Kind: core.KindSelect,
			Options: core.CompatibleFrameworkVersions("Go", core.LangVersions["Go"][0], "Fiber"),
			Value:   core.CompatibleFrameworkVersions("Go", core.LangVersions["Go"][0], "Fiber")[0],
		},
	}
}

func defaultServiceFields() []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{Key: "responsibility", Label: "responsibility", Kind: core.KindText},
		// config_ref: picks a StackConfig from the CONFIG tab (non-monolith arches only).
		{
			Key:     "config_ref",
			Label:   "stack config  ",
			Kind:    core.KindSelect,
			Options: []string{"(no configs defined)"},
			Value:   "(no configs defined)",
		},
		{
			Key: "language", Label: "language      ", Kind: core.KindSelect,
			Options: backendLanguages,
			Value:   "Go",
		},
		{
			Key: "language_version", Label: "lang version  ", Kind: core.KindSelect,
			Options: core.LangVersions["Go"],
			Value:   core.LangVersions["Go"][0],
		},
		{
			Key: "framework", Label: "framework     ", Kind: core.KindSelect,
			Options: backendFrameworksByLang["Go"],
			Value:   "Fiber",
		},
		{
			Key: "framework_version", Label: "fw version    ", Kind: core.KindSelect,
			Options: core.CompatibleFrameworkVersions("Go", core.LangVersions["Go"][0], "Fiber"),
			Value:   core.CompatibleFrameworkVersions("Go", core.LangVersions["Go"][0], "Fiber")[0],
		},
		{
			Key: "technologies", Label: "technologies  ", Kind: core.KindMultiSelect,
			Options: []string{"WebSocket", "gRPC", "REST", "GraphQL", "SSE", "tRPC", "MQTT", "Kafka consumer"},
		},
		{
			Key: "pattern_tag", Label: "pattern_tag   ", Kind: core.KindSelect,
			Options: []string{
				"Monolith part", "Modular module", "Microservice",
				"Event processor", "Serverless function",
			},
			Value: "Microservice",
		},
		{Key: "health_deps", Label: "Health Deps   ", Kind: core.KindMultiSelect},
		{
			Key: "error_format", Label: "Error Format  ", Kind: core.KindSelect,
			Options: []string{"RFC 7807 (Problem Details)", "Custom JSON envelope", "Platform default"},
			Value:   "Platform default", SelIdx: 2,
		},
		{
			Key: "service_discovery", Label: "Svc Discovery ", Kind: core.KindSelect,
			Options: []string{"DNS-based", "Consul", "Kubernetes DNS", "Eureka", "Static config", "None"},
			Value:   "None", SelIdx: 5,
		},
		// environment is a core.KindSelect populated dynamically from InfraPillar.Environments.
		{
			Key: "environment", Label: "environment   ", Kind: core.KindSelect,
			Options: []string{"(no environments configured)"},
			Value:   "(no environments configured)",
		},
	}
}

func serviceFieldsFromDef(s manifest.ServiceDef) []core.Field {
	f := defaultServiceFields()
	f = core.SetFieldValue(f, "name", s.Name)
	f = core.SetFieldValue(f, "responsibility", s.Responsibility)
	if s.ConfigRef != "" {
		f = core.SetFieldValue(f, "config_ref", s.ConfigRef)
	}
	if s.Language != "" {
		f = core.SetFieldValue(f, "language", s.Language)
		// Update language_version options for this language.
		if vers, ok := core.LangVersions[s.Language]; ok {
			for i := range f {
				if f[i].Key == "language_version" {
					f[i].Options = vers
					f[i].SelIdx = 0
					f[i].Value = vers[0]
					if s.LanguageVersion != "" {
						f = core.SetFieldValue(f, "language_version", s.LanguageVersion)
					}
					break
				}
			}
		}
		// Update framework options based on language.
		if opts, ok := backendFrameworksByLang[s.Language]; ok {
			for i := range f {
				if f[i].Key == "framework" {
					f[i].Options = opts
					f[i].SelIdx = 0
					f[i].Value = opts[0]
					if s.Framework != "" {
						f = core.SetFieldValue(f, "framework", s.Framework)
					}
					break
				}
			}
		}
		// Update framework_version options based on language + language_version + framework.
		lang := s.Language
		langVer := s.LanguageVersion
		fw := s.Framework
		if fw == "" {
			if opts, ok := backendFrameworksByLang[lang]; ok && len(opts) > 0 {
				fw = opts[0]
			}
		}
		fwVers := core.CompatibleFrameworkVersions(lang, langVer, fw)
		for i := range f {
			if f[i].Key == "framework_version" {
				f[i].Options = fwVers
				f[i].SelIdx = 0
				f[i].Value = fwVers[0]
				if s.FrameworkVersion != "" {
					f = core.SetFieldValue(f, "framework_version", s.FrameworkVersion)
				}
				break
			}
		}
	}
	if s.PatternTag != "" {
		f = core.SetFieldValue(f, "pattern_tag", s.PatternTag)
	}
	if s.Environment != "" {
		f = core.SetFieldValue(f, "environment", s.Environment)
	}
	// Restore technologies selections and narrow error_format options accordingly.
	if len(s.Technologies) > 0 {
		for i := range f {
			if f[i].Key != "technologies" {
				continue
			}
			f[i].SelectedIdxs = nil
			for _, tech := range s.Technologies {
				for j, opt := range f[i].Options {
					if opt == tech {
						f[i].SelectedIdxs = append(f[i].SelectedIdxs, j)
						break
					}
				}
			}
			break
		}
		errOpts := errorFormatOptsForTechs(s.Technologies)
		for i := range f {
			if f[i].Key == "error_format" {
				f[i].Options = errOpts
				break
			}
		}
	}
	// Restore error_format value (must happen after options are set).
	if s.ErrorFormat != "" {
		f = core.SetFieldValue(f, "error_format", s.ErrorFormat)
	}
	// Restore health_deps selections — options are populated dynamically later
	// via SetDBSourceAliases; store names in Value for lazy restoration.
	if len(s.HealthDeps) > 0 {
		for i := range f {
			if f[i].Key == "health_deps" {
				f[i].Value = strings.Join(s.HealthDeps, ", ")
				break
			}
		}
	}
	return f
}

func serviceDefFromFields(fields []core.Field) manifest.ServiceDef {
	// Read technologies multiselect
	var techs []string
	var healthDeps []string
	for _, f := range fields {
		switch f.Key {
		case "technologies":
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					techs = append(techs, f.Options[idx])
				}
			}
		case "health_deps":
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					healthDeps = append(healthDeps, f.Options[idx])
				}
			}
		}
	}
	env := core.FieldGet(fields, "environment")
	if env == "(no environments configured)" {
		env = ""
	}
	cfgRef := core.FieldGet(fields, "config_ref")
	if cfgRef == "(no configs defined)" {
		cfgRef = ""
	}
	return manifest.ServiceDef{
		Name:             core.FieldGet(fields, "name"),
		Responsibility:   core.FieldGet(fields, "responsibility"),
		ConfigRef:        cfgRef,
		Language:         core.FieldGet(fields, "language"),
		LanguageVersion:  core.FieldGet(fields, "language_version"),
		Framework:        core.FieldGet(fields, "framework"),
		FrameworkVersion: core.FieldGet(fields, "framework_version"),
		PatternTag:       core.FieldGet(fields, "pattern_tag"),
		Technologies:     techs,
		HealthDeps:       healthDeps,
		ErrorFormat:      core.FieldGet(fields, "error_format"),
		ServiceDiscovery: core.FieldGet(fields, "service_discovery"),
		Environment:      env,
	}
}

func defaultCommFields() []core.Field {
	return []core.Field{
		{Key: "from", Label: "from          ", Kind: core.KindText},
		{Key: "to", Label: "to            ", Kind: core.KindText},
		{
			Key: "direction", Label: "direction     ", Kind: core.KindSelect,
			Options: []string{
				"Unidirectional (→)", "Bidirectional (↔)", "Pub/Sub (fan-out)",
			},
			Value: "Unidirectional (→)",
		},
		{
			Key: "protocol", Label: "protocol      ", Kind: core.KindSelect,
			Options: []string{
				"REST (HTTP)", "gRPC", "GraphQL", "WebSocket",
				"Message Queue", "Event Bus", "Internal (in-process)",
			},
			Value: "REST (HTTP)",
		},
		{Key: "trigger", Label: "trigger       ", Kind: core.KindText},
		{
			Key: "sync_async", Label: "sync_async    ", Kind: core.KindSelect,
			Options: []string{"Synchronous", "Asynchronous", "Fire-and-forget"},
			Value:   "Synchronous",
		},
		{
			Key: "resilience", Label: "resilience    ", Kind: core.KindMultiSelect,
			Options: []string{"Circuit breaker", "Retry with backoff", "Timeout", "Bulkhead", "None"},
		},
		// Options populated dynamically via withDTONames(); Value stores
		// comma-sep names for lazy restoration before options are injected.
		{Key: "payload_dto", Label: "payload_dto   ", Kind: core.KindMultiSelect},
		// Only shown when direction is "Bidirectional (↔)".
		{Key: "response_dto", Label: "response_dto  ", Kind: core.KindMultiSelect},
	}
}

func commLinkFromFields(fields []core.Field) manifest.CommLink {
	var resilience []string
	var payloadDTOs []string
	var responseDTOs []string
	for _, f := range fields {
		switch f.Key {
		case "resilience":
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					resilience = append(resilience, f.Options[idx])
				}
			}
		case "payload_dto":
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					payloadDTOs = append(payloadDTOs, f.Options[idx])
				}
			}
		case "response_dto":
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					responseDTOs = append(responseDTOs, f.Options[idx])
				}
			}
		}
	}
	return manifest.CommLink{
		From:               core.FieldGet(fields, "from"),
		To:                 core.FieldGet(fields, "to"),
		Direction:          core.FieldGet(fields, "direction"),
		Protocol:           core.FieldGet(fields, "protocol"),
		Trigger:            core.FieldGet(fields, "trigger"),
		SyncAsync:          core.FieldGet(fields, "sync_async"),
		ResiliencePatterns: resilience,
		DTOs:               payloadDTOs,
		ResponseDTOs:       responseDTOs,
	}
}

func commFieldsFromLink(l manifest.CommLink) []core.Field {
	f := defaultCommFields()
	f = core.SetFieldValue(f, "from", l.From)
	f = core.SetFieldValue(f, "to", l.To)
	if l.Direction != "" {
		f = core.SetFieldValue(f, "direction", l.Direction)
	}
	if l.Protocol != "" {
		f = core.SetFieldValue(f, "protocol", l.Protocol)
	}
	f = core.SetFieldValue(f, "trigger", l.Trigger)
	if l.SyncAsync != "" {
		f = core.SetFieldValue(f, "sync_async", l.SyncAsync)
	}
	// Store selected DTO names in Value for lazy restoration via withDTONames().
	if len(l.DTOs) > 0 {
		for i := range f {
			if f[i].Key == "payload_dto" {
				f[i].Value = strings.Join(l.DTOs, ", ")
				break
			}
		}
	}
	if len(l.ResponseDTOs) > 0 {
		for i := range f {
			if f[i].Key == "response_dto" {
				f[i].Value = strings.Join(l.ResponseDTOs, ", ")
				break
			}
		}
	}
	return f
}

var jobQueueTechByLang = map[string][]string{
	"Go":              {"Asynq", "River", "Temporal", "Faktory", "Custom"},
	"TypeScript/Node": {"BullMQ", "Temporal", "Custom"},
	"Python":          {"Celery", "Temporal", "Custom"},
	"Ruby":            {"Sidekiq", "Temporal", "Custom"},
	"Java":            {"Temporal", "Custom"},
	"Kotlin":          {"Temporal", "Custom"},
	"C#/.NET":         {"Hangfire", "Temporal", "Custom"},
	"Rust":            {"Temporal", "Custom"},
	"PHP":             {"Laravel Queues", "Temporal", "Custom"},
	"Elixir":          {"Oban", "Temporal", "Custom"},
	"Other":           {"Temporal", "Custom"},
}

// jobQueueTechOptions returns filtered technology options based on languages.
// When no languages are configured, returns the full set.
func jobQueueTechOptions(langs []string) ([]string, string) {
	if len(langs) == 0 {
		return []string{"Temporal", "BullMQ", "Sidekiq", "Celery", "Faktory", "Asynq", "River", "Custom"}, "Temporal"
	}
	seen := make(map[string]bool)
	var opts []string
	for _, lang := range langs {
		for _, tech := range jobQueueTechByLang[lang] {
			if !seen[tech] {
				seen[tech] = true
				opts = append(opts, tech)
			}
		}
	}
	if len(opts) == 0 {
		return []string{"Temporal", "Custom"}, "Temporal"
	}
	return opts, opts[0]
}

func defaultJobQueueFormFields(services, dtos, langs, configNames []string) []core.Field {
	workerOpts, workerVal := core.NoneOrPlaceholder(services, "(no services configured)")
	payloadOpts, payloadVal := core.NoneOrPlaceholder(dtos, "(no DTOs configured)")
	techOpts, techVal := jobQueueTechOptions(langs)

	var cfgOpts []string
	var cfgVal string
	if len(configNames) > 0 {
		cfgOpts = append([]string{"(any)"}, configNames...)
		cfgVal = "(any)"
	} else {
		cfgOpts = []string{"(no configs defined)"}
		cfgVal = "(no configs defined)"
	}

	fields := []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{Key: "description", Label: "description   ", Kind: core.KindText},
		{
			Key:     "config_ref",
			Label:   "stack config  ",
			Kind:    core.KindSelect,
			Options: cfgOpts,
			Value:   cfgVal,
		},
	}

	fields = append(fields, []core.Field{
		{
			Key: "technology", Label: "technology    ", Kind: core.KindSelect,
			Options: techOpts,
			Value:   techVal,
		},
		{Key: "concurrency", Label: "concurrency   ", Kind: core.KindText, Value: "10"},
		{Key: "max_retries", Label: "max_retries   ", Kind: core.KindText, Value: "3"},
		{
			Key: "retry_policy", Label: "retry_policy  ", Kind: core.KindSelect,
			Options: []string{"Exponential backoff", "Fixed interval", "Linear backoff", "None"},
			Value:   "Exponential backoff",
		},
		{
			Key: "dlq", Label: "dead_letter_q ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
		{
			Key: "worker_service", Label: "worker_service", Kind: core.KindSelect,
			Options: workerOpts, Value: workerVal,
		},
		{
			Key: "payload_dto", Label: "payload_dto   ", Kind: core.KindSelect,
			Options: payloadOpts, Value: payloadVal,
		},
	}...)
	return fields
}

// ── Runtime field population ──────────────────────────────────────────────────

// withServiceNames returns a copy of fields where from/to are upgraded to
// core.KindSelect dropdowns populated with the current service names.
func (be BackendEditor) withServiceNames(fields []core.Field) []core.Field {
	names := be.ServiceNames()
	out := core.CopyFields(fields)
	for i := range out {
		if out[i].Key != "from" && out[i].Key != "to" {
			continue
		}
		out[i].Kind = core.KindSelect
		out[i].Options = names
		out[i].Value = core.PlaceholderFor(names, "(no services configured)")
		out[i].SelIdx = 0
		for j, n := range names {
			if n == out[i].Value {
				out[i].SelIdx = j
				break
			}
		}
		if len(names) > 0 && out[i].Value == "" {
			out[i].Value = names[0]
		}
	}
	return out
}

// withDTONames returns a copy of fields where payload_dto and response_dto
// options are populated with the currently available DTO names, restoring any
// prior selections by matching option names.
func (be BackendEditor) withDTONames(fields []core.Field) []core.Field {
	out := core.CopyFields(fields)
	for i := range out {
		if out[i].Key != "payload_dto" && out[i].Key != "response_dto" {
			continue
		}
		// Collect previously selected DTO names before replacing options.
		var selectedNames []string
		if len(out[i].Options) > 0 {
			for _, idx := range out[i].SelectedIdxs {
				if idx < len(out[i].Options) {
					selectedNames = append(selectedNames, out[i].Options[idx])
				}
			}
		} else if out[i].Value != "" {
			// Options not yet set — Value stores comma-sep names from commFieldsFromLink.
			selectedNames = strings.Split(out[i].Value, ", ")
		}
		out[i].Options = be.availableDTOs
		out[i].SelectedIdxs = nil
		for _, name := range selectedNames {
			for j, opt := range out[i].Options {
				if opt == name {
					out[i].SelectedIdxs = append(out[i].SelectedIdxs, j)
					break
				}
			}
		}
	}
	return out
}
