package ui

import "strings"

// ── field definitions ─────────────────────────────────────────────────────────

// unitOptionsForLanguages returns unit-testing framework options relevant to
// the given set of backend languages. Falls back to all options when empty.
func unitOptionsForLanguages(langs []string) []string {
	if len(langs) == 0 {
		return []string{"Jest", "Vitest", "pytest", "Go testing", "JUnit", "xUnit", "Other"}
	}
	seen := make(map[string]bool)
	var opts []string
	add := func(o string) {
		if !seen[o] {
			seen[o] = true
			opts = append(opts, o)
		}
	}
	for _, lang := range langs {
		switch strings.ToLower(lang) {
		case "go", "golang":
			add("Go testing")
			add("Testify")
		case "typescript", "javascript", "ts", "js":
			add("Jest")
			add("Vitest")
		case "python":
			add("pytest")
			add("unittest")
		case "java":
			add("JUnit")
			add("TestNG")
		case "kotlin":
			add("JUnit")
			add("Kotest")
		case "c#", "csharp", "dotnet", ".net":
			add("xUnit")
			add("NUnit")
			add("MSTest")
		case "rust":
			add("cargo test")
		case "ruby":
			add("RSpec")
			add("minitest")
		case "php":
			add("PHPUnit")
			add("Pest")
		default:
			add("Jest")
			add("pytest")
			add("Go testing")
			add("JUnit")
		}
	}
	add("Other")
	return opts
}

// e2eOptionsForFrontend returns E2E framework options suitable for the given
// frontend language and framework.
func e2eOptionsForFrontend(frontendLang, frontendFramework string) []string {
	lang := strings.ToLower(frontendLang)
	fw := strings.ToLower(frontendFramework)
	switch {
	case lang == "dart" || fw == "flutter":
		return []string{"Flutter Driver", "Integration Test", "None"}
	case lang == "kotlin" || fw == "compose multiplatform" || fw == "jetpack compose":
		return []string{"Espresso", "UI Automator", "None"}
	case lang == "swift" || fw == "swiftui" || fw == "uikit":
		return []string{"XCUITest", "EarlGrey", "None"}
	case lang == "" && fw == "":
		return []string{"None"}
	default:
		// Web frameworks
		return []string{"Playwright", "Cypress", "Selenium", "None"}
	}
}

// loadOptionsForLanguages returns load-testing tools relevant to the backend langs.
func loadOptionsForLanguages(langs []string) []string {
	base := []string{"k6", "Artillery", "JMeter", "None"}
	for _, lang := range langs {
		if strings.ToLower(lang) == "python" {
			return []string{"k6", "Locust", "Artillery", "JMeter", "None"}
		}
	}
	return base
}

// apiOptionsForProtocols returns API testing tool options relevant to the given
// communication protocols. Falls back to REST tools when no protocols are configured.
func apiOptionsForProtocols(protocols []string) []string {
	var hasREST, hasGraphQL, hasGRPC bool
	for _, p := range protocols {
		switch p {
		case "REST (HTTP)", "REST":
			hasREST = true
		case "GraphQL":
			hasGraphQL = true
		case "gRPC":
			hasGRPC = true
		}
	}
	// No relevant protocols — default to REST tools.
	if !hasREST && !hasGraphQL && !hasGRPC {
		return []string{"Bruno", "Hurl", "Postman/Newman", "REST Client", "None"}
	}
	// Mixed (more than one protocol present).
	activeCount := 0
	for _, v := range []bool{hasREST, hasGraphQL, hasGRPC} {
		if v {
			activeCount++
		}
	}
	if activeCount > 1 {
		return []string{"Bruno", "Postman/Newman", "None"}
	}
	if hasGraphQL {
		return []string{"Bruno", "Postman/Newman", "GraphQL Playground", "None"}
	}
	if hasGRPC {
		return []string{"grpcurl", "Postman/Newman", "BloomRPC", "None"}
	}
	// REST only.
	return []string{"Bruno", "Hurl", "Postman/Newman", "REST Client", "None"}
}

// contractOptionsForArchPattern returns contract-testing tool options based on
// the selected backend architecture pattern.
func contractOptionsForArchPattern(archPattern string) []string {
	switch archPattern {
	case "microservices":
		return []string{"Pact", "Schemathesis", "Dredd", "None"}
	case "event-driven":
		return []string{"Pact", "AsyncAPI validator", "None"}
	case "hybrid":
		return []string{"Pact", "Schemathesis", "Dredd", "None"}
	default: // monolith, modular-monolith, or unset
		return []string{"None", "Schemathesis"}
	}
}

// feTestingOptionsForLang returns frontend unit/component testing framework
// options for the given frontend language. Used when populating the fe_testing
// field in the CrossCut > Testing sub-tab.
func feTestingOptionsForLang(lang string) []string {
	if opts, ok := feTestingByLanguage[lang]; ok {
		return opts
	}
	return []string{"Vitest", "Jest", "Testing Library", "Storybook", "None"}
}

// computeTestingFields builds testing Field definitions filtered to the given
// backend languages, protocols, arch pattern, and frontend tech. Existing values
// are preserved when the option is still available; otherwise the first option is selected.
func computeTestingFields(backendLangs, backendProtocols []string, backendArchPattern, frontendLang, frontendFramework string, existing []Field) []Field {
	unitOpts := unitOptionsForLanguages(backendLangs)
	e2eOpts := e2eOptionsForFrontend(frontendLang, frontendFramework)
	loadOpts := loadOptionsForLanguages(backendLangs)
	apiOpts := apiOptionsForProtocols(backendProtocols)
	contractOpts := contractOptionsForArchPattern(backendArchPattern)

	feTestOpts := feTestingOptionsForLang(frontendLang)

	template := []struct {
		key, label string
		opts       []string
	}{
		{"unit", "unit          ", unitOpts},
		{"integration", "integration   ", []string{"Testcontainers", "Docker Compose", "In-memory fakes", "None"}},
		{"e2e", "e2e           ", e2eOpts},
		{"fe_testing", "fe_testing    ", feTestOpts},
		{"api", "api           ", apiOpts},
		{"load", "load          ", loadOpts},
		{"contract", "contract      ", contractOpts},
	}

	// Build lookup of existing values.
	existingVals := make(map[string]string, len(existing))
	for _, f := range existing {
		existingVals[f.Key] = f.Value
	}

	fields := make([]Field, 0, len(template))
	for _, t := range template {
		selIdx := 0
		val := t.opts[0]
		// Preserve current value when still valid.
		if prev, ok := existingVals[t.key]; ok {
			for i, o := range t.opts {
				if o == prev {
					selIdx = i
					val = o
					break
				}
			}
		}
		// Default contract to "None".
		if t.key == "contract" && val == t.opts[0] && existingVals[t.key] == "" {
			for i, o := range t.opts {
				if o == "None" {
					selIdx = i
					val = o
					break
				}
			}
		}
		fields = append(fields, Field{
			Key:    t.key,
			Label:  t.label,
			Kind:   KindSelect,
			Options: t.opts,
			Value:  val,
			SelIdx: selIdx,
		})
	}
	return fields
}

func defaultTestingFields() []Field {
	return computeTestingFields(nil, nil, "", "", "", nil)
}

// linterOptionsForLanguages returns the deduplicated set of linter options for
// the given backend languages. Falls back to a comprehensive union when empty.
func linterOptionsForLanguages(langs []string) []string {
	if len(langs) == 0 {
		// Return a representative union as default.
		return []string{"golangci-lint", "ESLint", "Ruff", "Checkstyle", "Clippy", "None"}
	}
	seen := make(map[string]bool)
	var out []string
	add := func(o string) {
		if !seen[o] {
			seen[o] = true
			out = append(out, o)
		}
	}
	for _, lang := range langs {
		for _, opt := range backendLintersByLang[lang] {
			add(opt)
		}
	}
	if len(out) == 0 {
		return []string{"None"}
	}
	return out
}

func defaultStandardsFields() []Field {
	linterOpts := linterOptionsForLanguages(nil)
	return []Field{
		{
			Key: "dep_updates", Label: "Dep. Updates  ", Kind: KindSelect,
			Options: []string{"Dependabot", "Renovate", "Manual", "None"},
			Value:   "Dependabot",
		},
		{
			Key: "feature_flags", Label: "Feature Flags ", Kind: KindSelect,
			Options: []string{"LaunchDarkly", "Unleash", "Flagsmith", "Custom (env vars)", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key:     "be_linter",
			Label:   "Backend Linter",
			Kind:    KindSelect,
			Options: linterOpts,
			Value:   "None", SelIdx: len(linterOpts) - 1,
		},
	}
}

func defaultDocsFields() []Field {
	return []Field{
		{
			Key: "api_docs", Label: "api_docs      ", Kind: KindSelect,
			Options: []string{
				"OpenAPI/Swagger", "GraphQL Playground",
				"gRPC reflection", "None",
			},
			Value: "OpenAPI/Swagger",
		},
		{
			Key: "auto_generate", Label: "auto_generate ", Kind: KindSelect,
			Options: OptionsOffOn, Value: "true", SelIdx: 1,
		},
	}
}

// ── Runtime context update ────────────────────────────────────────────────────

// SetTestingContext re-evaluates the testing framework options based on the
// current backend languages and frontend tech. A no-op when inputs are unchanged.
func (cc *CrossCutEditor) SetTestingContext(backendLangs, backendProtocols []string, backendArchPattern, frontendLang, frontendFramework string) {
	if stringSlicesEqual(cc.backendLangs, backendLangs) &&
		stringSlicesEqual(cc.backendProtocols, backendProtocols) &&
		cc.backendArchPattern == backendArchPattern &&
		cc.frontendLang == frontendLang &&
		cc.frontendFramework == frontendFramework {
		return
	}
	cc.backendLangs = backendLangs
	cc.backendProtocols = backendProtocols
	cc.backendArchPattern = backendArchPattern
	cc.frontendLang = frontendLang
	cc.frontendFramework = frontendFramework
	cc.testingFields = computeTestingFields(backendLangs, backendProtocols, backendArchPattern, frontendLang, frontendFramework, cc.testingFields)
	cc.updateLinterOptions(backendLangs)
}

// updateLinterOptions narrows the be_linter select options in standardsFields
// to match the configured backend languages.
func (cc *CrossCutEditor) updateLinterOptions(langs []string) {
	opts := linterOptionsForLanguages(langs)
	for i := range cc.standardsFields {
		if cc.standardsFields[i].Key != "be_linter" {
			continue
		}
		cc.standardsFields[i].Options = opts
		// Keep current value when still valid; otherwise reset to None.
		found := false
		for j, o := range opts {
			if o == cc.standardsFields[i].Value {
				cc.standardsFields[i].SelIdx = j
				found = true
				break
			}
		}
		if !found && len(opts) > 0 {
			cc.standardsFields[i].Value = opts[len(opts)-1] // last option is always "None"
			cc.standardsFields[i].SelIdx = len(opts) - 1
		}
		break
	}
}

