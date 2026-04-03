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
		return "ENV"
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
	serviceEditor beListEditor
	commEditor    beListEditor
	eventEditor   beListEditor // event catalog within messaging

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

	// Vim motion state
	countBuf string
	gBuf     bool
}

func newBackendEditor() BackendEditor {
	return BackendEditor{
		EnvFields:       defaultEnvFields(),
		MessagingFields: defaultMessagingFields(),
		APIGWFields:     defaultAPIGWFields(),
		AuthFields:      defaultAuthFields(),
		securityFields:  defaultSecurityFields(),
		serviceEditor:   newBeListEditor(),
		commEditor:      newBeListEditor(),
		eventEditor:     newBeListEditor(),
		formInput:       newFormInput(),
		dropdownOpen:    true,
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

// SetDTONames injects DTO names from the contracts tab for job payload dropdowns.
func (be *BackendEditor) SetDTONames(names []string) {
	be.availableDTOs = names
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
	if be.envEnabled {
		env = manifest.EnvConfig{
			ComputeEnv:    fieldGet(be.EnvFields, "compute_env"),
			CloudProvider: fieldGet(be.EnvFields, "cloud_provider"),
			Orchestrator:  fieldGet(be.EnvFields, "orchestrator"),
			Regions:       fieldGetMulti(be.EnvFields, "regions"),
			Stages:        fieldGet(be.EnvFields, "stages"),
		}
	}

	var auth manifest.AuthConfig
	if be.authEnabled {
		auth = manifest.AuthConfig{
			Strategy:     fieldGetMulti(be.AuthFields, "strategy"),
			Provider:     fieldGet(be.AuthFields, "provider"),
			AuthzModel:   fieldGet(be.AuthFields, "authz_model"),
			Permissions:  be.authPerms,
			Roles:        be.authRoles,
			TokenStorage: fieldGetMulti(be.AuthFields, "token_storage"),
			RefreshToken: fieldGet(be.AuthFields, "refresh_token"),
			MFA:          fieldGet(be.AuthFields, "mfa"),
		}
	}

	bp := manifest.BackendPillar{
		ArchPattern: manifest.ArchPattern(arch),
		Env:         env,
		Services:    be.Services,
		CommLinks:   be.CommLinks,
		Auth:        auth,
		JobQueues:   be.jobQueues,
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
		}
	}
	if be.envEnabled {
		bp.CORSStrategy = fieldGet(be.EnvFields, "cors_strategy")
		bp.CORSOrigins = fieldGet(be.EnvFields, "cors_origins")
		bp.SessionMgmt = fieldGet(be.EnvFields, "session_mgmt")
		bp.BackendLinter = fieldGet(be.EnvFields, "be_linter")
	}

	tabs := subTabsForArch(arch)
	for _, t := range tabs {
		if t == beTabMessaging {
			mc := manifest.MessagingConfig{
				BrokerTech:    fieldGet(be.MessagingFields, "broker_tech"),
				Deployment:    fieldGet(be.MessagingFields, "deployment"),
				Serialization: fieldGet(be.MessagingFields, "serialization"),
				Delivery:      fieldGet(be.MessagingFields, "delivery"),
			}
			bp.Messaging = &mc
		}
		if t == beTabAPIGW && be.apiGWEnabled {
			gw := manifest.APIGatewayConfig{
				Technology: fieldGet(be.APIGWFields, "technology"),
				Routing:    fieldGet(be.APIGWFields, "routing"),
				Features:   fieldGetMulti(be.APIGWFields, "features"),
				Endpoints:  fieldGetMulti(be.APIGWFields, "endpoints"),
			}
			bp.APIGateway = &gw
		}
	}

	// Legacy compat fields
	bp.ComputeEnv = manifest.ComputeEnv(env.ComputeEnv)
	bp.CloudProvider = env.CloudProvider
	if arch == "monolith" {
		bp.Language = fieldGet(be.EnvFields, "monolith_lang")
		bp.Framework = fieldGet(be.EnvFields, "monolith_fw")
	} else if len(be.Services) > 0 {
		bp.Language = be.Services[0].Language
		bp.Framework = be.Services[0].Framework
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

	// Env fields.
	if bp.Env.ComputeEnv != "" || bp.Env.CloudProvider != "" || bp.CORSStrategy != "" || bp.BackendLinter != "" {
		be.envEnabled = true
		be.EnvFields = setFieldValue(be.EnvFields, "compute_env", bp.Env.ComputeEnv)
		be.updateEnvOrchestratorOptions()
		be.EnvFields = setFieldValue(be.EnvFields, "cloud_provider", bp.Env.CloudProvider)
		be.EnvFields = setFieldValue(be.EnvFields, "orchestrator", bp.Env.Orchestrator)
		be.EnvFields = restoreMultiSelectValue(be.EnvFields, "regions", bp.Env.Regions)
		be.EnvFields = setFieldValue(be.EnvFields, "stages", bp.Env.Stages)
		be.EnvFields = setFieldValue(be.EnvFields, "cors_strategy", bp.CORSStrategy)
		be.EnvFields = setFieldValue(be.EnvFields, "cors_origins", bp.CORSOrigins)
		be.EnvFields = setFieldValue(be.EnvFields, "session_mgmt", bp.SessionMgmt)
		be.EnvFields = setFieldValue(be.EnvFields, "be_linter", bp.BackendLinter)
		if arch == "monolith" {
			be.EnvFields = setFieldValue(be.EnvFields, "monolith_lang", bp.Language)
			be.updateEnvMonolithOptions()
			be.EnvFields = setFieldValue(be.EnvFields, "monolith_fw", bp.Framework)
		}
	}

	// Auth fields.
	if bp.Auth.Strategy != "" || bp.Auth.Provider != "" {
		be.authEnabled = true
		be.AuthFields = restoreMultiSelectValue(be.AuthFields, "strategy", bp.Auth.Strategy)
		be.AuthFields = setFieldValue(be.AuthFields, "provider", bp.Auth.Provider)
		be.AuthFields = setFieldValue(be.AuthFields, "authz_model", bp.Auth.AuthzModel)
		be.authPerms = bp.Auth.Permissions
		be.authRoles = bp.Auth.Roles
		be.AuthFields = restoreMultiSelectValue(be.AuthFields, "token_storage", bp.Auth.TokenStorage)
		be.AuthFields = setFieldValue(be.AuthFields, "refresh_token", bp.Auth.RefreshToken)
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
	}

	// Messaging fields.
	if bp.Messaging != nil {
		be.MessagingFields = setFieldValue(be.MessagingFields, "broker_tech", bp.Messaging.BrokerTech)
		be.MessagingFields = setFieldValue(be.MessagingFields, "deployment", bp.Messaging.Deployment)
		be.MessagingFields = setFieldValue(be.MessagingFields, "serialization", bp.Messaging.Serialization)
		be.MessagingFields = setFieldValue(be.MessagingFields, "delivery", bp.Messaging.Delivery)
	}

	// API Gateway fields.
	if bp.APIGateway != nil {
		be.apiGWEnabled = true
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
		be.serviceEditor.items[i] = serviceFieldsFromDef(svc)
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

func (be BackendEditor) CloudProvider() string {
	return fieldGet(be.EnvFields, "cloud_provider")
}

// Orchestrator returns the selected orchestrator from the Env tab.
// Returns an empty string if the env section has not been configured.
func (be BackendEditor) Orchestrator() string {
	return fieldGet(be.EnvFields, "orchestrator")
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
// all backend services (and the monolith language when applicable).
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
	}
	for _, item := range be.serviceEditor.items {
		add(fieldGet(item, "language"))
	}
	return langs
}

// ArchPattern returns the currently selected architecture pattern value (e.g. "monolith", "microservices").
func (be BackendEditor) ArchPattern() string {
	return be.currentArch()
}

// ServiceFrameworks returns the unique set of frameworks used across all configured
// backend services (e.g. "tRPC", "NestJS"). For monolith arch the monolith framework
// is included instead of the service list.
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
	}
	for _, item := range be.serviceEditor.items {
		add(fieldGet(item, "framework"))
	}
	return fws
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

