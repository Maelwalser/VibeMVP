package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── modes ─────────────────────────────────────────────────────────────────────


// ── arch options ──────────────────────────────────────────────────────────────

type archOption struct {
	value string
	label string
	desc  string
}

var beArchOptions = []archOption{
	{"monolith", "Monolith", "Single deployable unit — all features in one codebase"},
	{"modular-monolith", "Modular Monolith", "Clear domain boundaries, single deployment"},
	{"microservices", "Microservices", "Independent services communicating over a network"},
	{"event-driven", "Event-Driven", "Services communicate asynchronously via events"},
	{"hybrid", "Hybrid", "Mix of patterns — each service unit tagged with its own pattern"},
}

// ── sub-tab IDs per arch ──────────────────────────────────────────────────────

// backendSubTab enumerates the logical sub-tabs in the backend section.
type backendSubTab int

const (
	beTabEnv backendSubTab = iota
	beTabServices
	beTabComm
	beTabMessaging
	beTabAPIGW
	beTabJobs
	beTabSecurity
	beTabAuth
)

// subTabsForArch returns the ordered list of sub-tabs for the given arch value.
func subTabsForArch(arch string) []backendSubTab {
	switch arch {
	case "monolith":
		return []backendSubTab{beTabEnv, beTabServices, beTabJobs, beTabSecurity, beTabAuth}
	case "modular-monolith":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabJobs, beTabSecurity, beTabAuth}
	case "microservices":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabAPIGW, beTabJobs, beTabSecurity, beTabAuth}
	case "event-driven":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabMessaging, beTabJobs, beTabSecurity, beTabAuth}
	case "hybrid":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabMessaging, beTabAPIGW, beTabJobs, beTabSecurity, beTabAuth}
	default:
		return []backendSubTab{beTabEnv, beTabServices, beTabJobs, beTabSecurity, beTabAuth}
	}
}

func subTabLabel(t backendSubTab) string {
	switch t {
	case beTabEnv:
		return "CONFIG"
	case beTabServices:
		return "SERVICES"
	case beTabComm:
		return "COMM"
	case beTabMessaging:
		return "MESSAGING"
	case beTabAPIGW:
		return "API GW"
	case beTabJobs:
		return "JOBS"
	case beTabSecurity:
		return "SECURITY"
	case beTabAuth:
		return "AUTH"
	}
	return "?"
}

// ── beSubView / beAuthView types ──────────────────────────────────────────────

type beSubView int

const (
	beViewList beSubView = iota
	beViewForm
)

type beAuthView int

const (
	beAuthViewConfig   beAuthView = iota // flat config fields (strategy, provider, etc.)
	beAuthViewRoleList                   // roles list
	beAuthViewRoleForm                   // single role edit form
	beAuthViewPermList                   // permissions list
	beAuthViewPermForm                   // single permission edit form
)

// ── list + form sub-editor for services and comm links ───────────────────────

type beListView int

const (
	beListViewList beListView = iota
	beListViewForm
)

type beListEditor struct {
	items    [][]Field        // each item is a slice of fields
	itemView beListView
	itemIdx  int
	formIdx  int
	form     []Field
}

func newBeListEditor() beListEditor {
	return beListEditor{itemView: beListViewList}
}

// ── BackendEditor ─────────────────────────────────────────────────────────────

// BackendEditor manages the BACKEND section.
type BackendEditor struct {
	// Arch selection
	ArchIdx       int
	ArchConfirmed bool
	dropdownOpen  bool
	dropdownIdx   int

	// Sub-tab state
	activeTabIdx int // index into subTabsForArch(arch)
	activeField  int

	// Field stores
	EnvFields       []Field
	envEnabled      bool
	MessagingFields []Field
	APIGWFields     []Field
	apiGWEnabled    bool
	AuthFields      []Field
	authEnabled     bool

	// Security/WAF tab
	securityFields []Field
	secFormIdx     int
	secEnabled     bool

	// Jobs tab
	jobQueues   []manifest.JobQueueDef
	jobsSubView beSubView
	jobsIdx     int
	jobsForm    []Field
	jobsFormIdx int

	// Auth permissions + roles sub-editors
	authSubView     beAuthView
	authPerms       []manifest.PermissionDef
	authPermsIdx    int
	authPermForm    []Field
	authPermFormIdx int
	authRoles       []manifest.RoleDef
	authRolesIdx    int
	authRoleForm    []Field
	authRoleFormIdx int

	// List editors
	serviceEditor     beListEditor
	commEditor        beListEditor
	eventEditor       beListEditor // event catalog within messaging
	stackConfigEditor beListEditor // CONFIG tab stack configs (non-monolith)

	// Stack configs for non-monolith arches (serialized to manifest).
	StackConfigs []manifest.StackConfig

	// Internal mode
	internalMode Mode
	formInput    textinput.Model
	width        int

	// Dropdown state (shared across all sub-contexts; only one can be open)
	dd DropdownState

	// Services and comm links for manifest export
	Services  []manifest.ServiceDef
	CommLinks []manifest.CommLink
	Events    []manifest.EventDef

	// Cross-tab references (injected from model.go)
	DomainNames        []string
	availableDTOs      []string
	availableEndpoints []string
	cacheAliases       []string // IsCache DB aliases from the Data pillar
	dbSourceAliases    []string // All DB source aliases from the Data pillar (for health_deps)
	dtoProtocols       []string // unique DTO serialisation protocols from ContractsEditor
	environmentNames   []string                        // InfraPillar environment names for service env dropdowns
	environmentDefs    []manifest.ServerEnvironmentDef // InfraPillar full env defs for API GW tech filtering
	orchestrator       string                          // Primary orchestrator from InfraPillar for service discovery
	cloudProvider      string                          // Primary cloud provider from InfraPillar for messaging deployment options

	// Vim motion state
	countBuf string
	gBuf     bool
	cBuf     bool
}

func newBackendEditor() BackendEditor {
	return BackendEditor{
		EnvFields:         defaultEnvFields(),
		MessagingFields:   defaultMessagingFields(),
		APIGWFields:       defaultAPIGWFields(),
		AuthFields:        defaultAuthFields(),
		securityFields:    defaultSecurityFields(),
		serviceEditor:     newBeListEditor(),
		commEditor:        newBeListEditor(),
		eventEditor:       newBeListEditor(),
		stackConfigEditor: newBeListEditor(),
		formInput:         newFormInput(),
		dropdownOpen:      true,
	}
}

// SetDomainNames injects domain names from the data tab for event domain dropdowns.
// SetDomainNames stores domain names from the Data pillar for dropdown population.
func (be *BackendEditor) SetDomainNames(names []string) {
	be.DomainNames = names
}

// SetCacheAliases stores the IsCache DB aliases from the Data pillar.
// Options are applied lazily when the rate_limit_backend dropdown is opened,
// not on every keypress, to avoid corrupting SelIdx during dropdown navigation.
func (be *BackendEditor) SetCacheAliases(aliases []string) {
	be.cacheAliases = aliases
}

// SetDBSourceAliases stores all DB source aliases from the Data pillar and
// refreshes the health_deps multiselect options in the monolith CONFIG tab and
// in every per-service form (non-monolith arches).
func (be *BackendEditor) SetDBSourceAliases(aliases []string) {
	be.dbSourceAliases = aliases
	// Monolith: global health_deps lives in EnvFields (CONFIG tab).
	be.applyHealthDepsOptionsToFields(be.EnvFields)
	// Non-monolith: per-service health_deps lives in service forms.
	be.applyHealthDepsOptionsToFields(be.serviceEditor.form)
	for _, item := range be.serviceEditor.items {
		be.applyHealthDepsOptionsToFields(item)
	}
}

// applyHealthDepsOptionsToFields refreshes the health_deps multiselect options
// in a field slice, preserving any currently selected aliases by name.
// Also handles lazy restoration when options haven't been set yet but Value
// holds a comma-separated list of names (written by serviceFieldsFromDef).
func (be *BackendEditor) applyHealthDepsOptionsToFields(fields []Field) {
	for i := range fields {
		if fields[i].Key != "health_deps" {
			continue
		}
		// Collect previously selected names before replacing options.
		var selectedNames []string
		if len(fields[i].Options) > 0 {
			for _, idx := range fields[i].SelectedIdxs {
				if idx < len(fields[i].Options) {
					selectedNames = append(selectedNames, fields[i].Options[idx])
				}
			}
		} else if fields[i].Value != "" {
			// Options not yet populated — Value holds comma-sep names from manifest restore.
			for _, name := range strings.Split(fields[i].Value, ", ") {
				if name != "" {
					selectedNames = append(selectedNames, name)
				}
			}
		}
		fields[i].Options = be.dbSourceAliases
		fields[i].Value = ""
		fields[i].SelectedIdxs = nil
		for _, name := range selectedNames {
			for j, opt := range fields[i].Options {
				if opt == name {
					fields[i].SelectedIdxs = append(fields[i].SelectedIdxs, j)
					break
				}
			}
		}
		break
	}
}

// SetDTONames injects DTO names from the contracts tab for job payload dropdowns.
func (be *BackendEditor) SetDTONames(names []string) {
	be.availableDTOs = names
}

// dtoProtocolToSerialization maps a DTO protocol name to the corresponding
// messaging serialization option, or "" if no mapping exists.
func dtoProtocolToSerialization(proto string) string {
	switch proto {
	case "Protobuf":
		return "Protobuf"
	case "Avro":
		return "Avro"
	case "MessagePack":
		return "MessagePack"
	case "REST/JSON":
		return "JSON"
	default:
		return ""
	}
}

// SetDTOProtocols injects the unique serialisation protocols used by DTOs in
// the Contracts pillar. When all DTOs share a single protocol that maps to a
// messaging serialization option (Protobuf, Avro, MessagePack), the messaging
// serialization field is updated to match. Mixed or unmappable protocols leave
// the current selection unchanged.
func (be *BackendEditor) SetDTOProtocols(protocols []string) {
	if stringSlicesEqual(be.dtoProtocols, protocols) {
		return
	}
	be.dtoProtocols = protocols

	// Determine a single dominant serialization suggestion.
	if len(protocols) != 1 {
		return // mixed or no DTOs — leave current selection
	}
	suggested := dtoProtocolToSerialization(protocols[0])
	if suggested == "" || suggested == "JSON" {
		return // no actionable mapping
	}

	for i := range be.MessagingFields {
		if be.MessagingFields[i].Key != "serialization" {
			continue
		}
		for j, opt := range be.MessagingFields[i].Options {
			if opt == suggested {
				be.MessagingFields[i].SelIdx = j
				be.MessagingFields[i].Value = suggested
				break
			}
		}
		break
	}
}

// SetEndpointNames injects endpoint names from the contracts tab for the API
// Gateway endpoints multiselect, preserving any existing selection by name.
func (be *BackendEditor) SetEndpointNames(names []string) {
	be.availableEndpoints = names
	for i := range be.APIGWFields {
		if be.APIGWFields[i].Key != "endpoints" {
			continue
		}
		// Collect currently selected names before replacing options.
		var selectedNames []string
		if len(be.APIGWFields[i].Options) > 0 {
			for _, idx := range be.APIGWFields[i].SelectedIdxs {
				if idx < len(be.APIGWFields[i].Options) {
					selectedNames = append(selectedNames, be.APIGWFields[i].Options[idx])
				}
			}
		} else if be.APIGWFields[i].Value != "" {
			selectedNames = strings.Split(be.APIGWFields[i].Value, ", ")
		}
		be.APIGWFields[i].Options = names
		be.APIGWFields[i].SelectedIdxs = nil
		for _, name := range selectedNames {
			for j, opt := range names {
				if opt == name {
					be.APIGWFields[i].SelectedIdxs = append(be.APIGWFields[i].SelectedIdxs, j)
					break
				}
			}
		}
		break
	}
}

// stackConfigNames returns the names of all defined stack configs.
func (be BackendEditor) stackConfigNames() []string {
	var names []string
	for _, item := range be.stackConfigEditor.items {
		if n := fieldGet(item, "name"); n != "" {
			names = append(names, n)
		}
	}
	return names
}

// langForConfig returns the language of the stack config with the given name,
// or "" if the name is empty, "(any)", or not found.
func (be BackendEditor) langForConfig(configName string) string {
	if configName == "" || configName == "(any)" {
		return ""
	}
	for _, item := range be.stackConfigEditor.items {
		if fieldGet(item, "name") == configName {
			return fieldGet(item, "language")
		}
	}
	return ""
}

// updateJobQueueTechOptions refreshes the technology options in the active jobs
// form based on the currently selected config_ref. Called after config_ref changes.
func (be *BackendEditor) updateJobQueueTechOptions() {
	lang := be.langForConfig(fieldGet(be.jobsForm, "config_ref"))
	var langs []string
	if lang != "" {
		langs = []string{lang}
	} else {
		langs = be.Languages()
	}
	opts, defaultVal := jobQueueTechOptions(langs)
	cur := fieldGet(be.jobsForm, "technology")
	for i := range be.jobsForm {
		if be.jobsForm[i].Key != "technology" {
			continue
		}
		be.jobsForm[i].Options = opts
		found := false
		for j, o := range opts {
			if o == cur {
				be.jobsForm[i].SelIdx = j
				be.jobsForm[i].Value = o
				found = true
				break
			}
		}
		if !found {
			be.jobsForm[i].SelIdx = 0
			be.jobsForm[i].Value = defaultVal
		}
		break
	}
}

// applyStackConfigNamesToServices updates the config_ref dropdown in all service
// forms to reflect the current set of stack config names. Called whenever stack
// configs are added, renamed, or deleted.
func (be *BackendEditor) applyStackConfigNamesToServices() {
	var names []string
	for _, item := range be.stackConfigEditor.items {
		if n := fieldGet(item, "name"); n != "" {
			names = append(names, n)
		}
	}
	opts, placeholder := noneOrPlaceholder(names, "(no configs defined)")
	applyOpts := func(fields []Field) {
		for i := range fields {
			if fields[i].Key != "config_ref" {
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
				fields[i].Value = placeholder
				fields[i].SelIdx = 0
			}
			break
		}
	}
	applyOpts(be.serviceEditor.form)
	for _, item := range be.serviceEditor.items {
		applyOpts(item)
	}
}

// SetEnvironmentNames injects environment names from the infra tab so that
// the monolith env tab, service forms, and messaging broker config can show
// an environment selector dropdown.
func (be *BackendEditor) SetEnvironmentNames(names []string) {
	be.environmentNames = names
	// Refresh the monolith shared environment dropdown in the env tab.
	be.applyEnvNamesToServiceFields(be.EnvFields)
	// Refresh environment dropdowns in the active form and all stored items.
	be.applyEnvNamesToServiceFields(be.serviceEditor.form)
	for _, item := range be.serviceEditor.items {
		be.applyEnvNamesToServiceFields(item)
	}
	// Refresh environment dropdown in the messaging broker config.
	be.applyEnvNamesToServiceFields(be.MessagingFields)
	// Refresh environment dropdown in the API gateway config.
	be.applyEnvNamesToServiceFields(be.APIGWFields)
}

// SetEnvironmentDefs injects full environment definitions so the API Gateway
// technology options can be filtered by the selected environment's orchestrator
// and cloud provider.
func (be *BackendEditor) SetEnvironmentDefs(defs []manifest.ServerEnvironmentDef) {
	be.environmentDefs = defs
	be.updateAPIGWTechOptions()
}

// updateAPIGWTechOptions re-filters the API gateway technology options based
// on the currently selected environment's orchestrator and cloud provider.
func (be *BackendEditor) updateAPIGWTechOptions() {
	envVal := fieldGet(be.APIGWFields, "environment")
	var orch, cloud string
	for _, d := range be.environmentDefs {
		if d.Name == envVal {
			orch = d.Orchestrator
			cloud = d.CloudProvider
			break
		}
	}
	opts := apiGWTechOptionsForEnv(orch, cloud)
	cur := fieldGet(be.APIGWFields, "technology")
	for i := range be.APIGWFields {
		if be.APIGWFields[i].Key != "technology" {
			continue
		}
		be.APIGWFields[i].Options = opts
		// Keep current value when still valid; otherwise reset to first option.
		valid := false
		for j, o := range opts {
			if o == cur {
				be.APIGWFields[i].SelIdx = j
				valid = true
				break
			}
		}
		if !valid && len(opts) > 0 {
			be.APIGWFields[i].SelIdx = 0
			be.APIGWFields[i].Value = opts[0]
		}
		break
	}
}

// SetMessagingCloudProvider injects the primary cloud provider from infra so
// that the messaging deployment dropdown shows cloud-specific managed options.
func (be *BackendEditor) SetMessagingCloudProvider(cp string) {
	if be.cloudProvider == cp {
		return
	}
	be.cloudProvider = cp
	be.refreshMessagingDeploymentOptions()
}

// SetOrchestrator injects the primary orchestrator from infra for narrowing
// service discovery options. A no-op when unchanged.
func (be *BackendEditor) SetOrchestrator(orch string) {
	if be.orchestrator == orch {
		return
	}
	be.orchestrator = orch
	be.updateServiceDiscoveryOptions()
}

// applyEnvNamesToServiceFields sets the environment field options in a field slice.
func (be *BackendEditor) applyEnvNamesToServiceFields(fields []Field) {
	opts, val := noneOrPlaceholder(be.environmentNames, "(no environments configured)")
	for i := range fields {
		if fields[i].Key != "environment" {
			continue
		}
		fields[i].Options = opts
		// Keep current value when still valid.
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

// ── helpers ───────────────────────────────────────────────────────────────────

func (be BackendEditor) currentArch() string {
	if be.ArchIdx >= 0 && be.ArchIdx < len(beArchOptions) {
		return beArchOptions[be.ArchIdx].value
	}
	return beArchOptions[0].value
}

func (be BackendEditor) activeTabs() []backendSubTab {
	return subTabsForArch(be.currentArch())
}

func (be BackendEditor) activeTab() backendSubTab {
	tabs := be.activeTabs()
	if be.activeTabIdx >= 0 && be.activeTabIdx < len(tabs) {
		return tabs[be.activeTabIdx]
	}
	return beTabEnv
}

func (be BackendEditor) tabLabels() []string {
	tabs := be.activeTabs()
	labels := make([]string, len(tabs))
	for i, t := range tabs {
		labels[i] = subTabLabel(t)
	}
	return labels
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (be BackendEditor) ToManifest() manifest.BackendPillar {
	arch := be.currentArch()

	var env manifest.EnvConfig

	var auth manifest.AuthConfig
	if be.authEnabled {
		svcUnit := fieldGet(be.AuthFields, "service_unit")
		if svcUnit == "None (external)" || svcUnit == "None" || svcUnit == "(no services configured)" {
			svcUnit = ""
		}
		auth = manifest.AuthConfig{
			Strategy:     fieldGetMulti(be.AuthFields, "strategy"),
			Provider:     fieldGet(be.AuthFields, "provider"),
			ServiceUnit:  svcUnit,
			AuthzModel:   fieldGet(be.AuthFields, "authz_model"),
			SessionMgmt:  fieldGet(be.AuthFields, "session_mgmt"),
			Permissions:  be.authPerms,
			Roles:        be.authRoles,
			TokenStorage: fieldGetMulti(be.AuthFields, "token_storage"),
			RefreshToken: fieldGet(be.AuthFields, "refresh_token"),
			MFA:          fieldGet(be.AuthFields, "mfa"),
		}
	}

	// Derive stack configs from the list editor items.
	var stackConfigs []manifest.StackConfig
	for _, item := range be.stackConfigEditor.items {
		sc := manifest.StackConfig{
			Name:             fieldGet(item, "name"),
			Language:         fieldGet(item, "language"),
			LanguageVersion:  fieldGet(item, "language_version"),
			Framework:        fieldGet(item, "framework"),
			FrameworkVersion: fieldGet(item, "framework_version"),
		}
		if sc.Name != "" {
			stackConfigs = append(stackConfigs, sc)
		}
	}

	bp := manifest.BackendPillar{
		ArchPattern:  manifest.ArchPattern(arch),
		Env:          env,
		StackConfigs: stackConfigs,
		Services:     be.Services,
		CommLinks:    be.CommLinks,
		Auth:         auth,
		JobQueues:    be.jobQueues,
	}
	if be.secEnabled {
		bp.WAF = manifest.WAFConfig{
			Provider:          fieldGet(be.securityFields, "waf_provider"),
			Ruleset:           fieldGet(be.securityFields, "waf_ruleset"),
			CAPTCHA:           fieldGet(be.securityFields, "captcha"),
			BotProtection:     fieldGet(be.securityFields, "bot_protection"),
			RateLimitStrategy: fieldGet(be.securityFields, "rate_limit_strategy"),
			RateLimitBackend:  fieldGet(be.securityFields, "rate_limit_backend"),
			DDoSProtection:    fieldGet(be.securityFields, "ddos_protection"),
			InternalMTLS:      fieldGet(be.securityFields, "internal_mtls") == "Enabled",
		}
	}

	tabs := subTabsForArch(arch)
	for _, t := range tabs {
		if t == beTabMessaging {
			env := fieldGet(be.MessagingFields, "environment")
			if env == "(no environments configured)" {
				env = ""
			}
			mc := manifest.MessagingConfig{
				BrokerTech:    fieldGet(be.MessagingFields, "broker_tech"),
				Deployment:    fieldGet(be.MessagingFields, "deployment"),
				Serialization: fieldGet(be.MessagingFields, "serialization"),
				Delivery:      fieldGet(be.MessagingFields, "delivery"),
				Environment:   env,
			}
			bp.Messaging = &mc
			bp.Events = be.Events
		}
		if t == beTabAPIGW && be.apiGWEnabled {
			gw := manifest.APIGatewayConfig{
				Technology:  fieldGet(be.APIGWFields, "technology"),
				Routing:     fieldGet(be.APIGWFields, "routing"),
				Features:    fieldGetMulti(be.APIGWFields, "features"),
				Endpoints:   fieldGetMulti(be.APIGWFields, "endpoints"),
				Environment: fieldGet(be.APIGWFields, "environment"),
			}
			if gw.Environment == "(no environments configured)" {
				gw.Environment = ""
			}
			bp.APIGateway = &gw
		}
	}

	// Legacy compat fields (compute/cloud now live in InfraPillar.Environments)
	if arch == "monolith" {
		bp.Language = fieldGet(be.EnvFields, "monolith_lang")
		bp.LanguageVersion = fieldGet(be.EnvFields, "monolith_lang_ver")
		bp.Framework = fieldGet(be.EnvFields, "monolith_fw")
		bp.FrameworkVersion = fieldGet(be.EnvFields, "monolith_fw_ver")
		envVal := fieldGet(be.EnvFields, "environment")
		if envVal != "(no environments configured)" {
			bp.MonolithEnvironment = envVal
		}
		// Global health dependencies for monolith live in EnvConfig.
		env.HealthDeps = fieldGetSelectedSlice(be.EnvFields, "health_deps")
		bp.Env = env
	} else if len(be.Services) > 0 {
		bp.Language = be.Services[0].Language
		bp.LanguageVersion = be.Services[0].LanguageVersion
		bp.Framework = be.Services[0].Framework
		bp.FrameworkVersion = be.Services[0].FrameworkVersion
	}
	return bp
}

// FromBackendPillar populates the editor from a saved manifest BackendPillar,
// reversing the ToManifest() operation.
func (be BackendEditor) FromBackendPillar(bp manifest.BackendPillar) BackendEditor {
	// Restore arch selection.
	arch := string(bp.ArchPattern)
	for i, opt := range beArchOptions {
		if opt.value == arch {
			be.ArchIdx = i
			be.dropdownIdx = i
			break
		}
	}
	if arch != "" {
		be.ArchConfirmed = true
		be.dropdownOpen = false
	}

	// Env fields (monolith language/framework/environment only).
	if bp.Language != "" {
		be.envEnabled = true
		if arch == "monolith" {
			be.EnvFields = setFieldValue(be.EnvFields, "monolith_lang", bp.Language)
			be.updateEnvMonolithOptions()
			be.EnvFields = setFieldValue(be.EnvFields, "monolith_lang_ver", bp.LanguageVersion)
			be.EnvFields = setFieldValue(be.EnvFields, "monolith_fw", bp.Framework)
			be.updateEnvMonolithVersionOptions()
			be.EnvFields = setFieldValue(be.EnvFields, "monolith_fw_ver", bp.FrameworkVersion)
			if bp.MonolithEnvironment != "" {
				be.EnvFields = setFieldValue(be.EnvFields, "environment", bp.MonolithEnvironment)
			}
			// Restore global health deps — options populated lazily via SetDBSourceAliases.
			if len(bp.Env.HealthDeps) > 0 {
				for i := range be.EnvFields {
					if be.EnvFields[i].Key == "health_deps" {
						be.EnvFields[i].Value = strings.Join(bp.Env.HealthDeps, ", ")
						break
					}
				}
			}
		}
	}

	// Stack configs for non-monolith arches.
	if len(bp.StackConfigs) > 0 {
		be.envEnabled = true
		be.stackConfigEditor.items = make([][]Field, len(bp.StackConfigs))
		for i, sc := range bp.StackConfigs {
			f := defaultStackConfigFields()
			f = setFieldValue(f, "name", sc.Name)
			if sc.Language != "" {
				f = setFieldValue(f, "language", sc.Language)
				if vers, ok := langVersions[sc.Language]; ok {
					for j := range f {
						if f[j].Key == "language_version" {
							f[j].Options = vers
							f[j].SelIdx = 0
							f[j].Value = vers[0]
							break
						}
					}
					if sc.LanguageVersion != "" {
						f = setFieldValue(f, "language_version", sc.LanguageVersion)
					}
				}
				if opts, ok := backendFrameworksByLang[sc.Language]; ok {
					for j := range f {
						if f[j].Key == "framework" {
							f[j].Options = opts
							f[j].SelIdx = 0
							f[j].Value = opts[0]
							break
						}
					}
					if sc.Framework != "" {
						f = setFieldValue(f, "framework", sc.Framework)
					}
				}
				fw := sc.Framework
				if fw == "" {
					if opts, ok := backendFrameworksByLang[sc.Language]; ok && len(opts) > 0 {
						fw = opts[0]
					}
				}
				fwVers := compatibleFrameworkVersions(sc.Language, sc.LanguageVersion, fw)
				for j := range f {
					if f[j].Key == "framework_version" {
						f[j].Options = fwVers
						f[j].SelIdx = 0
						f[j].Value = fwVers[0]
						if sc.FrameworkVersion != "" {
							f = setFieldValue(f, "framework_version", sc.FrameworkVersion)
						}
						break
					}
				}
			}
			be.stackConfigEditor.items[i] = f
		}
		be.StackConfigs = bp.StackConfigs
		be.applyStackConfigNamesToServices()
	}

	// Auth fields.
	if bp.Auth.Strategy != "" || bp.Auth.Provider != "" {
		be.authEnabled = true
		be.AuthFields = restoreMultiSelectValue(be.AuthFields, "strategy", bp.Auth.Strategy)
		be.AuthFields = setFieldValue(be.AuthFields, "provider", bp.Auth.Provider)
		if bp.Auth.ServiceUnit != "" {
			// Restore service_unit; options will be repopulated lazily on first open.
			for i := range be.AuthFields {
				if be.AuthFields[i].Key == "service_unit" {
					be.AuthFields[i].Options = []string{bp.Auth.ServiceUnit}
					be.AuthFields[i].Value = bp.Auth.ServiceUnit
					be.AuthFields[i].SelIdx = 0
					break
				}
			}
		}
		be.AuthFields = setFieldValue(be.AuthFields, "authz_model", bp.Auth.AuthzModel)
		be.authPerms = bp.Auth.Permissions
		be.authRoles = bp.Auth.Roles
		be.AuthFields = restoreMultiSelectValue(be.AuthFields, "token_storage", bp.Auth.TokenStorage)
		be.AuthFields = setFieldValue(be.AuthFields, "session_mgmt", bp.Auth.SessionMgmt)
		be.AuthFields = setFieldValue(be.AuthFields, "refresh_token", bp.Auth.RefreshToken)
		// Recompute dynamic options after restoring strategy and provider.
		be.updateAuthTokenStorageOptions()
		be.updateAuthMFAOptions()
		be.AuthFields = setFieldValue(be.AuthFields, "mfa", bp.Auth.MFA)
	}

	// Security / WAF fields.
	if bp.WAF.Provider != "" || bp.WAF.Ruleset != "" {
		be.secEnabled = true
		be.securityFields = setFieldValue(be.securityFields, "waf_provider", bp.WAF.Provider)
		be.securityFields = setFieldValue(be.securityFields, "waf_ruleset", bp.WAF.Ruleset)
		be.securityFields = setFieldValue(be.securityFields, "captcha", bp.WAF.CAPTCHA)
		be.securityFields = setFieldValue(be.securityFields, "bot_protection", bp.WAF.BotProtection)
		be.securityFields = setFieldValue(be.securityFields, "rate_limit_strategy", bp.WAF.RateLimitStrategy)
		be.securityFields = setFieldValue(be.securityFields, "rate_limit_backend", bp.WAF.RateLimitBackend)
		be.securityFields = setFieldValue(be.securityFields, "ddos_protection", bp.WAF.DDoSProtection)
		mtlsVal := "Disabled"
		if bp.WAF.InternalMTLS {
			mtlsVal = "Enabled"
		}
		be.securityFields = setFieldValue(be.securityFields, "internal_mtls", mtlsVal)
	}

	// Messaging fields.
	if bp.Messaging != nil {
		be.MessagingFields = setFieldValue(be.MessagingFields, "broker_tech", bp.Messaging.BrokerTech)
		be.MessagingFields = setFieldValue(be.MessagingFields, "deployment", bp.Messaging.Deployment)
		be.MessagingFields = setFieldValue(be.MessagingFields, "serialization", bp.Messaging.Serialization)
		be.MessagingFields = setFieldValue(be.MessagingFields, "delivery", bp.Messaging.Delivery)
		if bp.Messaging.Environment != "" {
			be.MessagingFields = setFieldValue(be.MessagingFields, "environment", bp.Messaging.Environment)
		}
	}

	// Event catalog.
	be.Events = bp.Events
	be.eventEditor.items = make([][]Field, len(bp.Events))
	for i, evt := range bp.Events {
		f := defaultEventFields()
		f = setFieldValue(f, "name", evt.Name)
		f = setFieldValue(f, "publisher_service", evt.PublisherService)
		f = setFieldValue(f, "consumer_service", evt.ConsumerService)
		f = setFieldValue(f, "dto", evt.DTO)
		f = setFieldValue(f, "description", evt.Description)
		be.eventEditor.items[i] = f
	}

	// API Gateway fields.
	if bp.APIGateway != nil {
		be.apiGWEnabled = true
		if bp.APIGateway.Environment != "" {
			be.APIGWFields = setFieldValue(be.APIGWFields, "environment", bp.APIGateway.Environment)
		}
		be.APIGWFields = setFieldValue(be.APIGWFields, "technology", bp.APIGateway.Technology)
		be.APIGWFields = setFieldValue(be.APIGWFields, "routing", bp.APIGateway.Routing)
		be.APIGWFields = restoreMultiSelectValue(be.APIGWFields, "features", bp.APIGateway.Features)
		// Endpoint options are injected lazily via SetEndpointNames; store names
		// in Value so they can be restored once options become available.
		for i := range be.APIGWFields {
			if be.APIGWFields[i].Key == "endpoints" {
				be.APIGWFields[i].Value = bp.APIGateway.Endpoints
				break
			}
		}
	}

	// Collections — stored directly; per-item forms are rebuilt lazily on navigation.
	be.Services = bp.Services
	be.serviceEditor.items = make([][]Field, len(bp.Services))
	for i, svc := range bp.Services {
		fields := serviceFieldsFromDef(svc)
		// Monolith: health deps are global (CONFIG tab), not per-service.
		if arch == "monolith" {
			fields = withoutField(fields, "health_deps")
		}
		be.serviceEditor.items[i] = fields
	}
	// Apply orchestrator-based service discovery options now that items are populated.
	be.updateServiceDiscoveryOptions()

	be.CommLinks = bp.CommLinks
	be.commEditor.items = make([][]Field, len(bp.CommLinks))
	for i, link := range bp.CommLinks {
		be.commEditor.items[i] = commFieldsFromLink(link)
	}

	be.jobQueues = bp.JobQueues

	return be
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (be BackendEditor) Mode() Mode {
	if be.internalMode == ModeInsert {
		return ModeInsert
	}
	return ModeNormal
}

// ── Update ────────────────────────────────────────────────────────────────────

func (be BackendEditor) Update(msg tea.Msg) (BackendEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		be.width = wsz.Width
		be.formInput.Width = wsz.Width - 22
		return be, nil
	}
	if !be.ArchConfirmed {
		return be.updateArchSelect(msg)
	}
	if be.internalMode == ModeInsert {
		return be.updateInsert(msg)
	}
	if be.dd.Open {
		key, ok := msg.(tea.KeyMsg)
		if ok {
			return be.updateDropdown(key)
		}
		return be, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "c" {
			if be.cBuf {
				be.cBuf = false
				return be.clearAndEnterInsert()
			}
			be.cBuf = true
			return be, nil
		}
		be.cBuf = false
	}
	return be.updateNormal(msg)
}

func (be BackendEditor) updateArchSelect(msg tea.Msg) (BackendEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return be, nil
	}
	if be.dropdownOpen {
		switch key.String() {
		case "j", "down":
			if be.dropdownIdx < len(beArchOptions)-1 {
				be.dropdownIdx++
			}
		case "k", "up":
			if be.dropdownIdx > 0 {
				be.dropdownIdx--
			}
		case "g":
			be.dropdownIdx = 0
		case "G":
			be.dropdownIdx = len(beArchOptions) - 1
		case "enter", " ":
			be.ArchIdx = be.dropdownIdx
			be.dropdownOpen = false
			be.ArchConfirmed = true
			be.activeTabIdx = 0
			be.activeField = 0
		case "esc":
			be.dropdownOpen = false
		}
		return be, nil
	}
	switch key.String() {
	case "enter", " ":
		be.dropdownOpen = true
		be.dropdownIdx = be.ArchIdx
	}
	return be, nil
}

// Orchestrator returns the primary orchestrator injected from the infra tab.
func (be BackendEditor) Orchestrator() string {
	return be.orchestrator
}

// WAFRateLimitStrategy returns the configured WAF rate-limit strategy so that
// the Contracts editor can set a sensible default for endpoint rate_limit.
func (be BackendEditor) WAFRateLimitStrategy() string {
	return fieldGet(be.securityFields, "rate_limit_strategy")
}

// AuthRoleOptions returns role names for use in frontend page forms.
// Returns only explicitly configured roles; empty slice means none configured.
func (be BackendEditor) AuthRoleOptions() []string {
	names := make([]string, 0, len(be.authRoles))
	for _, r := range be.authRoles {
		if r.Name != "" {
			names = append(names, r.Name)
		}
	}
	return names
}

// ServiceNames returns the names of all created service units for cross-reference.
func (be BackendEditor) ServiceNames() []string {
	var names []string
	for _, item := range be.serviceEditor.items {
		name := fieldGet(item, "name")
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// ServiceDefs returns full service definitions for technology-based protocol filtering.
func (be BackendEditor) ServiceDefs() []manifest.ServiceDef {
	defs := make([]manifest.ServiceDef, 0, len(be.serviceEditor.items))
	for _, item := range be.serviceEditor.items {
		defs = append(defs, serviceDefFromFields(item))
	}
	return defs
}

// Languages returns the unique set of programming languages configured across
// all stack configs (non-monolith) or the monolith language.
func (be BackendEditor) Languages() []string {
	seen := make(map[string]bool)
	var langs []string
	add := func(l string) {
		if l != "" && !seen[l] {
			seen[l] = true
			langs = append(langs, l)
		}
	}
	if be.currentArch() == "monolith" {
		add(fieldGet(be.EnvFields, "monolith_lang"))
	} else {
		for _, item := range be.stackConfigEditor.items {
			add(fieldGet(item, "language"))
		}
	}
	return langs
}

// ArchPattern returns the currently selected architecture pattern value (e.g. "monolith", "microservices").
func (be BackendEditor) ArchPattern() string {
	return be.currentArch()
}

// ServiceFrameworks returns the unique set of frameworks used across all configured
// stack configs (non-monolith) or the monolith framework.
func (be BackendEditor) ServiceFrameworks() []string {
	seen := make(map[string]bool)
	var fws []string
	add := func(fw string) {
		if fw != "" && !seen[fw] {
			seen[fw] = true
			fws = append(fws, fw)
		}
	}
	if be.currentArch() == "monolith" {
		add(fieldGet(be.EnvFields, "monolith_fw"))
	} else {
		for _, item := range be.stackConfigEditor.items {
			add(fieldGet(item, "framework"))
		}
	}
	return fws
}

// AuthStrategy returns the selected backend auth strategies for cross-editor use.
func (be BackendEditor) AuthStrategy() []string {
	raw := fieldGetMulti(be.AuthFields, "strategy")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ", ")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// CommProtocols returns the unique set of protocols used across all communication links.
func (be BackendEditor) CommProtocols() []string {
	seen := make(map[string]bool)
	var protos []string
	for _, link := range be.CommLinks {
		p := link.Protocol
		if p != "" && !seen[p] {
			seen[p] = true
			protos = append(protos, p)
		}
	}
	return protos
}

