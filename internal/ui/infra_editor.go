package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// envView is the sub-view state for the Environments list+form.
type envView int

const (
	envViewList envView = iota
	envViewForm
)

// ── InfraEditor ───────────────────────────────────────────────────────────────

// InfraEditor manages the INFRASTRUCTURE main-tab.
type InfraEditor struct {
	activeTab infraTabIdx

	networkingFields []Field
	netFormIdx       int
	netEnabled       bool

	cicdFields  []Field
	cicdFormIdx int
	cicdEnabled bool

	obsFields  []Field
	obsFormIdx int
	obsEnabled bool

	// Environments list+form (replaces the old flat env-topology form).
	envs       []manifest.ServerEnvironmentDef
	envIdx     int
	envView    envView
	envForm    []Field
	envFormIdx int

	internalMode Mode
	formInput    textinput.Model
	width        int

	// Vim motion state
	nav  VimNav
	cBuf bool

	dd DropdownState

	// backendLanguages mirrors the languages from the backend services/monolith
	// so that container_runtime options reflect what is actually being built.
	backendLanguages string // joined with "," for cheap equality checks

	// cloudProvider caches the last provider used to narrow networking/cicd/obs
	// option lists, to avoid redundant field-slice rebuilds.
	cloudProvider string

	// Per-subtab undo stack (structural add/delete only)
	envsUndo UndoStack[[]manifest.ServerEnvironmentDef]
}

func (ie InfraEditor) activeTabEnabled() bool {
	switch ie.activeTab {
	case infraTabNetworking:
		return ie.netEnabled
	case infraTabCICD:
		return ie.cicdEnabled
	case infraTabObservability:
		return ie.obsEnabled
	case infraTabEnvironments:
		return true // environments list is always accessible
	}
	return false
}

func (ie *InfraEditor) enableActiveTab() {
	switch ie.activeTab {
	case infraTabNetworking:
		ie.netEnabled = true
		ie.netFormIdx = 0
	case infraTabCICD:
		ie.cicdEnabled = true
		ie.cicdFormIdx = 0
	case infraTabObservability:
		ie.obsEnabled = true
		ie.obsFormIdx = 0
	}
}

func (ie *InfraEditor) disableActiveTab() {
	switch ie.activeTab {
	case infraTabNetworking:
		ie.netEnabled = false
		ie.networkingFields = defaultNetworkingFields()
		ie.netFormIdx = 0
	case infraTabCICD:
		ie.cicdEnabled = false
		ie.cicdFields = defaultInfraCICDFields()
		ie.cicdFormIdx = 0
	case infraTabObservability:
		ie.obsEnabled = false
		ie.obsFields = defaultObservabilityFields()
		ie.obsFormIdx = 0
	}
}

func newInfraEditor() InfraEditor {
	return InfraEditor{
		networkingFields: defaultNetworkingFields(),
		cicdFields:       defaultInfraCICDFields(),
		obsFields:        defaultObservabilityFields(),
		formInput:        newFormInput(),
	}
}

// EnvironmentNames returns the names of all configured server environments.
// Used by backend and data editors to populate environment selector dropdowns.
func (ie InfraEditor) EnvironmentNames() []string {
	names := make([]string, 0, len(ie.envs))
	for _, e := range ie.envs {
		if e.Name != "" {
			names = append(names, e.Name)
		}
	}
	return names
}

// EnvironmentDefs returns lightweight records of each configured environment's
// name, orchestrator, and cloud provider. Used by the API Gateway editor to
// filter technology options based on the selected environment.
func (ie InfraEditor) EnvironmentDefs() []manifest.ServerEnvironmentDef {
	out := make([]manifest.ServerEnvironmentDef, 0, len(ie.envs))
	for _, e := range ie.envs {
		if e.Name != "" {
			out = append(out, manifest.ServerEnvironmentDef{
				Name:          e.Name,
				Orchestrator:  e.Orchestrator,
				CloudProvider: e.CloudProvider,
			})
		}
	}
	return out
}

// PrimaryOrchestrator returns the orchestrator of the first configured environment,
// or an empty string when no environments have been defined yet.
func (ie InfraEditor) PrimaryOrchestrator() string {
	for _, e := range ie.envs {
		if e.Orchestrator != "" {
			return e.Orchestrator
		}
	}
	return ""
}

// PrimaryCloudProvider returns the cloud_provider of the first environment,
// for use by other editors that need to narrow their options by cloud provider.
func (ie InfraEditor) PrimaryCloudProvider() string {
	return ie.primaryCloudProvider()
}

// primaryCloudProvider returns the cloud_provider of the first environment for
// narrowing networking/cicd/obs option lists.
func (ie InfraEditor) primaryCloudProvider() string {
	for _, e := range ie.envs {
		if e.CloudProvider != "" {
			return e.CloudProvider
		}
	}
	return ""
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (ie InfraEditor) ToManifestInfraPillar() manifest.InfraPillar {
	var p manifest.InfraPillar
	if ie.netEnabled {
		p.Networking = &manifest.NetworkingConfig{
			DNSProvider:     noneToEmpty(fieldGet(ie.networkingFields, "dns_provider")),
			TLSSSL:          noneToEmpty(fieldGet(ie.networkingFields, "tls_ssl")),
			ReverseProxy:    noneToEmpty(fieldGet(ie.networkingFields, "reverse_proxy")),
			CDN:             noneToEmpty(fieldGet(ie.networkingFields, "cdn")),
			PrimaryDomain:   fieldGet(ie.networkingFields, "primary_domain"),
			DomainStrategy:  noneToEmpty(fieldGet(ie.networkingFields, "domain_strategy")),
			CORSEnforcement: noneToEmpty(fieldGet(ie.networkingFields, "cors_infra")),
			CORSStrategy:    noneToEmpty(fieldGet(ie.networkingFields, "cors_strategy")),
			CORSOrigins:     fieldGet(ie.networkingFields, "cors_origins"),
			SSLCertMgmt:     noneToEmpty(fieldGet(ie.networkingFields, "ssl_cert")),
		}
	}
	if ie.cicdEnabled {
		p.CICD = &manifest.CICDConfig{
			Platform:          noneToEmpty(fieldGet(ie.cicdFields, "platform")),
			ContainerRegistry: noneToEmpty(fieldGet(ie.cicdFields, "registry")),
			DeployStrategy:    noneToEmpty(fieldGet(ie.cicdFields, "deploy_strategy")),
			IaCTool:           noneToEmpty(fieldGet(ie.cicdFields, "iac_tool")),
			SecretsMgmt:       noneToEmpty(fieldGet(ie.cicdFields, "secrets_mgmt")),
			ContainerRuntime:  noneToEmpty(fieldGet(ie.cicdFields, "container_runtime")),
			BackupDR:          noneToEmpty(fieldGet(ie.cicdFields, "backup_dr")),
		}
	}
	if ie.obsEnabled {
		p.Observability = &manifest.ObservabilityConfig{
			Logging:       noneToEmpty(fieldGet(ie.obsFields, "logging")),
			Metrics:       noneToEmpty(fieldGet(ie.obsFields, "metrics")),
			Tracing:       noneToEmpty(fieldGet(ie.obsFields, "tracing")),
			ErrorTracking: noneToEmpty(fieldGet(ie.obsFields, "error_tracking")),
			HealthChecks:  fieldGet(ie.obsFields, "health_checks") == "true",
			Alerting:      noneToEmpty(fieldGet(ie.obsFields, "alerting")),
			LogRetention:  fieldGet(ie.obsFields, "log_retention"),
		}
	}
	p.Environments = ie.envs
	return p
}

// FromInfraPillar populates the editor from a saved manifest InfraPillar,
// reversing the ToManifestInfraPillar() operation.
func (ie InfraEditor) FromInfraPillar(ip manifest.InfraPillar) InfraEditor {
	if ip.Networking != nil && (ip.Networking.DNSProvider != "" || ip.Networking.ReverseProxy != "" || ip.Networking.CDN != "") {
		ie.netEnabled = true
		ie.networkingFields = setFieldValue(ie.networkingFields, "dns_provider", ip.Networking.DNSProvider)
		ie.networkingFields = setFieldValue(ie.networkingFields, "tls_ssl", ip.Networking.TLSSSL)
		ie.networkingFields = setFieldValue(ie.networkingFields, "reverse_proxy", ip.Networking.ReverseProxy)
		ie.networkingFields = setFieldValue(ie.networkingFields, "cdn", ip.Networking.CDN)
		ie.networkingFields = setFieldValue(ie.networkingFields, "primary_domain", ip.Networking.PrimaryDomain)
		ie.networkingFields = setFieldValue(ie.networkingFields, "domain_strategy", ip.Networking.DomainStrategy)
		ie.networkingFields = setFieldValue(ie.networkingFields, "cors_infra", ip.Networking.CORSEnforcement)
		ie.networkingFields = setFieldValue(ie.networkingFields, "cors_strategy", ip.Networking.CORSStrategy)
		ie.networkingFields = setFieldValue(ie.networkingFields, "cors_origins", ip.Networking.CORSOrigins)
		ie.networkingFields = setFieldValue(ie.networkingFields, "ssl_cert", ip.Networking.SSLCertMgmt)
	}

	if ip.CICD != nil && ip.CICD.Platform != "" {
		ie.cicdEnabled = true
		ie.cicdFields = setFieldValue(ie.cicdFields, "platform", ip.CICD.Platform)
		ie.cicdFields = setFieldValue(ie.cicdFields, "registry", ip.CICD.ContainerRegistry)
		ie.cicdFields = setFieldValue(ie.cicdFields, "deploy_strategy", ip.CICD.DeployStrategy)
		ie.cicdFields = setFieldValue(ie.cicdFields, "iac_tool", ip.CICD.IaCTool)
		ie.cicdFields = setFieldValue(ie.cicdFields, "secrets_mgmt", ip.CICD.SecretsMgmt)
		ie.cicdFields = setFieldValue(ie.cicdFields, "container_runtime", ip.CICD.ContainerRuntime)
		ie.cicdFields = setFieldValue(ie.cicdFields, "backup_dr", ip.CICD.BackupDR)
	}

	if o := ip.Observability; o != nil && (o.Logging != "" || o.Metrics != "") {
		ie.obsEnabled = true
		ie.obsFields = setFieldValue(ie.obsFields, "logging", o.Logging)
		ie.obsFields = setFieldValue(ie.obsFields, "metrics", o.Metrics)
		// Narrow alerting/tracing options to those compatible with the saved metrics
		// backend before restoring their values, so SelIdx is computed from the
		// correct (narrowed) option list.
		ie.obsFields = applyMetricsToObsFields(ie.obsFields)
		ie.obsFields = setFieldValue(ie.obsFields, "tracing", o.Tracing)
		ie.obsFields = setFieldValue(ie.obsFields, "error_tracking", o.ErrorTracking)
		boolStr := func(b bool) string {
			if b {
				return "true"
			}
			return "false"
		}
		ie.obsFields = setFieldValue(ie.obsFields, "health_checks", boolStr(o.HealthChecks))
		ie.obsFields = setFieldValue(ie.obsFields, "alerting", o.Alerting)
		ie.obsFields = setFieldValue(ie.obsFields, "log_retention", o.LogRetention)
	}

	if len(ip.Environments) > 0 {
		ie.envs = ip.Environments
		ie.envIdx = 0
		ie.envView = envViewList
	}

	return ie
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (ie InfraEditor) Mode() Mode {
	if ie.internalMode == ModeInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (ie InfraEditor) HintLine() string {
	if ie.internalMode == ModeInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	if ie.dd.Open {
		return hintBar("j/k", "navigate", "Enter/Space", "select", "Esc", "cancel")
	}
	if ie.activeTab == infraTabEnvironments {
		switch ie.envView {
		case envViewForm:
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "H", "cycle back", "b/Esc", "save & back", "h/l", "sub-tab")
		default:
			return hintBar("j/k", "navigate", "a", "add env", "d", "delete", "u", "undo", "Enter", "edit", "h/l", "sub-tab")
		}
	}
	if !ie.activeTabEnabled() {
		return hintBar("a", "configure", "h/l", "sub-tab")
	}
	return hintBar("j/k", "navigate", "gg/G", "top/bottom", "[n]j/k", "jump n lines", "Enter/Space", "dropdown", "H", "cycle back", "D", "delete config", "a/i", "edit text", "h/l", "sub-tab")
}

// ── Update ────────────────────────────────────────────────────────────────────

func (ie InfraEditor) Update(msg tea.Msg) (InfraEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		ie.width = wsz.Width
		ie.formInput.Width = wsz.Width - 22
		return ie, nil
	}
	if ie.internalMode == ModeInsert {
		return ie.updateInsert(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return ie, nil
	}

	if ie.dd.Open && ie.activeTab != infraTabEnvironments {
		return ie.updateInfraDropdown(key)
	}

	switch key.String() {
	case "h", "left", "l", "right":
		ie.activeTab = infraTabIdx(NavigateTab(key.String(), int(ie.activeTab), len(infraTabLabels)))
		return ie, nil
	}

	// cc detection: clear field and enter insert mode
	if !ie.dd.Open {
		if key.String() == "c" {
			if ie.cBuf {
				ie.cBuf = false
				return ie.clearAndEnterInsert()
			}
			ie.cBuf = true
			return ie, nil
		}
		ie.cBuf = false
	}

	switch ie.activeTab {
	case infraTabNetworking:
		return ie.updateFields(key)
	case infraTabCICD:
		return ie.updateFields(key)
	case infraTabObservability:
		return ie.updateFields(key)
	case infraTabEnvironments:
		return ie.updateEnvTab(key)
	}
	return ie, nil
}

func (ie InfraEditor) updateFields(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	if !ie.activeTabEnabled() {
		if key.String() == "a" {
			ie.enableActiveTab()
		}
		return ie, nil
	}
	var fields []Field
	var idx int
	switch ie.activeTab {
	case infraTabNetworking:
		fields, idx = ie.networkingFields, ie.netFormIdx
	case infraTabCICD:
		fields, idx = ie.cicdFields, ie.cicdFormIdx
	case infraTabObservability:
		fields, idx = ie.obsFields, ie.obsFormIdx
	default:
		return ie, nil
	}
	n := len(fields)
	k := key.String()

	metricsChanged := false
	if newIdx, consumed := ie.nav.Handle(k, idx, n); consumed {
		idx = newIdx
	} else {
		ie.nav.Reset()
		switch k {
		case "enter", " ":
			if idx < n {
				f := &fields[idx]
				if f.Kind == KindSelect {
					ie.dd.Open = true
					ie.dd.OptIdx = f.SelIdx
				} else {
					switch ie.activeTab {
					case infraTabNetworking:
						ie.networkingFields = fields
						ie.netFormIdx = idx
					case infraTabCICD:
						ie.cicdFields = fields
						ie.cicdFormIdx = idx
					case infraTabObservability:
						ie.obsFields = fields
						ie.obsFormIdx = idx
					}
					return ie.tryEnterInsert()
				}
			}
		case "H", "shift+left":
			if idx < n {
				f := &fields[idx]
				if f.Kind == KindSelect {
					prev := f.Value
					f.CyclePrev()
					if ie.activeTab == infraTabObservability && f.Key == "metrics" && f.Value != prev {
						metricsChanged = true
					}
				}
			}
		case "D":
			ie.disableActiveTab()
			return ie, nil
		case "i", "a":
			switch ie.activeTab {
			case infraTabNetworking:
				ie.networkingFields = fields
				ie.netFormIdx = idx
			case infraTabCICD:
				ie.cicdFields = fields
				ie.cicdFormIdx = idx
			case infraTabObservability:
				ie.obsFields = fields
				ie.obsFormIdx = idx
			}
			return ie.tryEnterInsert()
		}
	}
	// Write back updated fields and index
	switch ie.activeTab {
	case infraTabNetworking:
		ie.networkingFields = fields
		ie.netFormIdx = idx
	case infraTabCICD:
		ie.cicdFields = fields
		ie.cicdFormIdx = idx
	case infraTabObservability:
		ie.obsFields = fields
		ie.obsFormIdx = idx
		if metricsChanged {
			ie.obsFields = applyMetricsToObsFields(ie.obsFields)
		}
	}
	return ie, nil
}

func (ie InfraEditor) updateInfraDropdown(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	var f *Field
	switch ie.activeTab {
	case infraTabNetworking:
		if ie.netFormIdx < len(ie.networkingFields) {
			f = &ie.networkingFields[ie.netFormIdx]
		}
	case infraTabCICD:
		if ie.cicdFormIdx < len(ie.cicdFields) {
			f = &ie.cicdFields[ie.cicdFormIdx]
		}
	case infraTabObservability:
		if ie.obsFormIdx < len(ie.obsFields) {
			f = &ie.obsFields[ie.obsFormIdx]
		}
	}
	if f == nil {
		ie.dd.Open = false
		return ie, nil
	}
	ie.dd.OptIdx = NavigateDropdown(key.String(), ie.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = ie.dd.OptIdx
		if ie.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[ie.dd.OptIdx]
		}
		ie.dd.Open = false
		if ie.activeTab == infraTabObservability && f.Key == "metrics" {
			ie.obsFields = applyMetricsToObsFields(ie.obsFields)
		}
		if f.PrepareCustomEntry() {
			return ie.tryEnterInsert()
		}
	case "esc", "b":
		ie.dd.Open = false
	}
	return ie, nil
}

func (ie InfraEditor) updateInsert(msg tea.Msg) (InfraEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			ie.saveInput()
			ie.internalMode = ModeNormal
			ie.formInput.Blur()
			return ie, nil
		case "tab":
			ie.saveInput()
			ie.advanceField(1)
			return ie.tryEnterInsert()
		case "shift+tab":
			ie.saveInput()
			ie.advanceField(-1)
			return ie.tryEnterInsert()
		}
	}
	var cmd tea.Cmd
	ie.formInput, cmd = ie.formInput.Update(msg)
	return ie, cmd
}

func (ie *InfraEditor) advanceField(delta int) {
	switch ie.activeTab {
	case infraTabNetworking:
		n := len(ie.networkingFields)
		if n > 0 {
			ie.netFormIdx = (ie.netFormIdx + delta + n) % n
		}
	case infraTabCICD:
		n := len(ie.cicdFields)
		if n > 0 {
			ie.cicdFormIdx = (ie.cicdFormIdx + delta + n) % n
		}
	case infraTabObservability:
		n := len(ie.obsFields)
		if n > 0 {
			ie.obsFormIdx = (ie.obsFormIdx + delta + n) % n
		}
	case infraTabEnvironments:
		n := len(ie.envForm)
		if n > 0 {
			ie.envFormIdx = (ie.envFormIdx + delta + n) % n
		}
	}
}

func (ie *InfraEditor) saveInput() {
	val := ie.formInput.Value()
	switch ie.activeTab {
	case infraTabNetworking:
		if ie.netFormIdx < len(ie.networkingFields) && ie.networkingFields[ie.netFormIdx].CanEditAsText() {
			ie.networkingFields[ie.netFormIdx].SaveTextInput(val)
		}
	case infraTabCICD:
		if ie.cicdFormIdx < len(ie.cicdFields) && ie.cicdFields[ie.cicdFormIdx].CanEditAsText() {
			ie.cicdFields[ie.cicdFormIdx].SaveTextInput(val)
		}
	case infraTabObservability:
		if ie.obsFormIdx < len(ie.obsFields) && ie.obsFields[ie.obsFormIdx].CanEditAsText() {
			ie.obsFields[ie.obsFormIdx].SaveTextInput(val)
		}
	case infraTabEnvironments:
		if ie.envFormIdx < len(ie.envForm) && ie.envForm[ie.envFormIdx].CanEditAsText() {
			ie.envForm[ie.envFormIdx].SaveTextInput(val)
		}
	}
}

func (ie InfraEditor) clearAndEnterInsert() (InfraEditor, tea.Cmd) {
	ie, cmd := ie.tryEnterInsert()
	if ie.internalMode == ModeInsert {
		ie.formInput.SetValue("")
	}
	return ie, cmd
}

func (ie InfraEditor) tryEnterInsert() (InfraEditor, tea.Cmd) {
	n := 0
	switch ie.activeTab {
	case infraTabNetworking:
		n = len(ie.networkingFields)
	case infraTabCICD:
		n = len(ie.cicdFields)
	case infraTabObservability:
		n = len(ie.obsFields)
	case infraTabEnvironments:
		n = len(ie.envForm)
	}
	for range n {
		var f *Field
		switch ie.activeTab {
		case infraTabNetworking:
			if ie.netFormIdx < len(ie.networkingFields) {
				f = &ie.networkingFields[ie.netFormIdx]
			}
		case infraTabCICD:
			if ie.cicdFormIdx < len(ie.cicdFields) {
				f = &ie.cicdFields[ie.cicdFormIdx]
			}
		case infraTabObservability:
			if ie.obsFormIdx < len(ie.obsFields) {
				f = &ie.obsFields[ie.obsFormIdx]
			}
		case infraTabEnvironments:
			if ie.envFormIdx < len(ie.envForm) {
				f = &ie.envForm[ie.envFormIdx]
			}
		}
		if f == nil {
			break
		}
		if f.CanEditAsText() {
			ie.internalMode = ModeInsert
			ie.formInput.SetValue(f.TextInputValue())
			ie.formInput.Width = ie.width - 22
			ie.formInput.CursorEnd()
			return ie, ie.formInput.Focus()
		}
		ie.advanceField(1)
	}
	return ie, nil
}

// visibleNetworkingFields hides cors_origins unless cors_strategy is "Strict allowlist".
func (ie InfraEditor) visibleNetworkingFields() []Field {
	corsStrategy := fieldGet(ie.networkingFields, "cors_strategy")
	var out []Field
	for _, f := range ie.networkingFields {
		if f.Key == "cors_origins" && corsStrategy != "Strict allowlist" {
			continue
		}
		out = append(out, f)
	}
	return out
}

// ── View ──────────────────────────────────────────────────────────────────────

func (ie InfraEditor) View(w, h int) string {
	ie.width = w
	ie.formInput.Width = w - 22
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Infrastructure — networking, CI/CD, and observability"),
		"",
		renderSubTabBar(infraTabLabels, int(ie.activeTab), w),
		"",
	)

	const infraHeaderH = 4
	switch ie.activeTab {
	case infraTabNetworking:
		if ie.netEnabled {
			fl := renderFormFields(w, ie.visibleNetworkingFields(), ie.netFormIdx, ie.internalMode == ModeInsert, ie.formInput, ie.dd.Open, ie.dd.OptIdx)
			lines = append(lines, appendViewport(fl, 0, ie.netFormIdx, h-infraHeaderH)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case infraTabCICD:
		if ie.cicdEnabled {
			fl := renderFormFields(w, ie.cicdFields, ie.cicdFormIdx, ie.internalMode == ModeInsert, ie.formInput, ie.dd.Open, ie.dd.OptIdx)
			lines = append(lines, appendViewport(fl, 0, ie.cicdFormIdx, h-infraHeaderH)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case infraTabObservability:
		if ie.obsEnabled {
			fl := renderFormFields(w, ie.obsFields, ie.obsFormIdx, ie.internalMode == ModeInsert, ie.formInput, ie.dd.Open, ie.dd.OptIdx)
			lines = append(lines, appendViewport(fl, 0, ie.obsFormIdx, h-infraHeaderH)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case infraTabEnvironments:
		lines = append(lines, ie.viewEnvTab(w)...)
	}

	return fillTildes(lines, h)
}

// ── Environments list+form ────────────────────────────────────────────────────

func (ie InfraEditor) updateEnvTab(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	switch ie.envView {
	case envViewList:
		return ie.updateEnvList(key)
	case envViewForm:
		return ie.updateEnvForm(key)
	}
	return ie, nil
}

func (ie InfraEditor) updateEnvList(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	n := len(ie.envs)
	switch key.String() {
	case "j", "down":
		if n > 0 && ie.envIdx < n-1 {
			ie.envIdx++
		}
	case "k", "up":
		if ie.envIdx > 0 {
			ie.envIdx--
		}
	case "u":
		if snap, ok := ie.envsUndo.Pop(); ok {
			ie.envs = snap
			if ie.envIdx >= len(ie.envs) && ie.envIdx > 0 {
				ie.envIdx = len(ie.envs) - 1
			}
		}
	case "a":
		ie.envsUndo.Push(copySlice(ie.envs))
		existing := make([]string, 0, len(ie.envs))
		for _, e := range ie.envs {
			existing = append(existing, e.Name)
		}
		newDef := manifest.ServerEnvironmentDef{
			Name:          uniqueName("environment", existing),
			ComputeEnv:    "Containers (Docker)",
			CloudProvider: "AWS",
			Orchestrator:  "Docker Compose",
		}
		ie.envs = append(ie.envs, newDef)
		ie.envIdx = len(ie.envs) - 1
		ie.envForm = serverEnvFormFromDef(ie.envs[ie.envIdx])
		ie.envFormIdx = 0
		ie.envView = envViewForm
		// apply compute_env narrowing to orchestrator options
		ie.applyEnvOrchestratorOptions()
		// propagate first env's cloud_provider to infra networking/cicd/obs
		ie.SetCloudProvider(ie.primaryCloudProvider())
	case "d":
		if n > 0 {
			ie.envsUndo.Push(copySlice(ie.envs))
			ie.envs = append(ie.envs[:ie.envIdx], ie.envs[ie.envIdx+1:]...)
			if ie.envIdx > 0 && ie.envIdx >= len(ie.envs) {
				ie.envIdx = len(ie.envs) - 1
			}
		}
	case "enter", "l", "right":
		if n > 0 {
			ie.envForm = serverEnvFormFromDef(ie.envs[ie.envIdx])
			ie.envFormIdx = 0
			ie.envView = envViewForm
			ie.applyEnvOrchestratorOptions()
		}
	}
	return ie, nil
}

func (ie InfraEditor) updateEnvForm(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	if ie.dd.Open {
		return ie.updateEnvFormDropdown(key)
	}
	n := len(ie.envForm)
	switch key.String() {
	case "j", "down":
		if n > 0 {
			ie.envFormIdx = (ie.envFormIdx + 1) % n
		}
	case "k", "up":
		if n > 0 {
			ie.envFormIdx = (ie.envFormIdx - 1 + n) % n
		}
	case "enter", " ":
		if ie.envFormIdx < n {
			f := &ie.envForm[ie.envFormIdx]
			if f.Kind == KindSelect || f.Kind == KindMultiSelect {
				ie.dd.Open = true
				ie.dd.OptIdx = f.SelIdx
			} else {
				return ie.tryEnterInsert()
			}
		}
	case "H", "shift+left":
		if ie.envFormIdx < n {
			f := &ie.envForm[ie.envFormIdx]
			if f.Kind == KindSelect {
				f.CyclePrev()
				ie.onEnvFormFieldChanged(f.Key)
			}
		}
	case "i", "a":
		if ie.envFormIdx < n && ie.envForm[ie.envFormIdx].CanEditAsText() {
			return ie.tryEnterInsert()
		}
	case "b", "esc":
		ie.saveEnvForm()
		ie.envView = envViewList
	}
	ie.saveEnvForm()
	return ie, nil
}

func (ie InfraEditor) updateEnvFormDropdown(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	if ie.envFormIdx >= len(ie.envForm) {
		ie.dd.Open = false
		return ie, nil
	}
	f := &ie.envForm[ie.envFormIdx]
	ie.dd.OptIdx = NavigateDropdown(key.String(), ie.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == KindMultiSelect {
			f.ToggleMultiSelect(ie.dd.OptIdx)
			f.DDCursor = ie.dd.OptIdx
		} else {
			f.SelIdx = ie.dd.OptIdx
			if ie.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[ie.dd.OptIdx]
			}
			ie.dd.Open = false
			ie.onEnvFormFieldChanged(f.Key)
			if f.PrepareCustomEntry() {
				return ie.tryEnterInsert()
			}
		}
	case "enter":
		if f.Kind == KindMultiSelect {
			f.ToggleMultiSelect(ie.dd.OptIdx)
			f.DDCursor = ie.dd.OptIdx
		} else {
			f.SelIdx = ie.dd.OptIdx
			if ie.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[ie.dd.OptIdx]
			}
			ie.dd.Open = false
			ie.onEnvFormFieldChanged(f.Key)
			if f.PrepareCustomEntry() {
				return ie.tryEnterInsert()
			}
		}
	case "esc", "b":
		ie.dd.Open = false
	}
	ie.saveEnvForm()
	return ie, nil
}

// onEnvFormFieldChanged reacts to a field value change inside the env form.
func (ie *InfraEditor) onEnvFormFieldChanged(key string) {
	switch key {
	case "compute_env":
		ie.applyEnvOrchestratorOptions()
	case "cloud_provider":
		ie.SetCloudProvider(ie.primaryCloudProvider())
	}
}

// applyEnvOrchestratorOptions narrows orchestrator options in envForm
// based on the current compute_env selection.
func (ie *InfraEditor) applyEnvOrchestratorOptions() {
	computeEnv := fieldGet(ie.envForm, "compute_env")
	opts := narrowOrchestratorOptions(computeEnv)
	for i := range ie.envForm {
		if ie.envForm[i].Key != "orchestrator" {
			continue
		}
		ie.envForm[i].Options = opts
		found := false
		for j, o := range opts {
			if o == ie.envForm[i].Value {
				ie.envForm[i].SelIdx = j
				found = true
				break
			}
		}
		if !found && len(opts) > 0 {
			ie.envForm[i].Value = opts[0]
			ie.envForm[i].SelIdx = 0
		}
		break
	}
}

// saveEnvForm writes the current envForm back to ie.envs[ie.envIdx].
func (ie *InfraEditor) saveEnvForm() {
	if ie.envIdx >= len(ie.envs) {
		return
	}
	ie.envs[ie.envIdx] = serverEnvDefFromForm(ie.envForm)
	// Keep networking/cicd/obs options in sync with primary env's settings.
	ie.SetCloudProvider(ie.primaryCloudProvider())
	ie.applyOrchestratorToCICD(ie.PrimaryOrchestrator())
	ie.applyOrchestratorToNetworking(ie.PrimaryOrchestrator())
}

func (ie InfraEditor) viewEnvTab(w int) []string {
	switch ie.envView {
	case envViewForm:
		return ie.viewEnvForm(w)
	default:
		return ie.viewEnvList(w)
	}
}

func (ie InfraEditor) viewEnvList(w int) []string {
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  server environments — define deployment targets (dev, staging, prod)"))
	lines = append(lines, "")
	if len(ie.envs) == 0 {
		lines = append(lines, StyleSectionDesc.Render("  (no environments — press 'a' to add one)"))
		return lines
	}
	for i, e := range ie.envs {
		cursor := "  "
		style := StyleFieldKey
		if i == ie.envIdx {
			cursor = StyleCursor.Render("▶ ")
			style = StyleFieldKeyActive
		}
		summary := e.Name
		if e.CloudProvider != "" {
			summary += "  " + StyleHelpDesc.Render(e.CloudProvider)
		}
		if e.ComputeEnv != "" {
			summary += "  " + StyleHelpDesc.Render(e.ComputeEnv)
		}
		if e.Orchestrator != "" {
			summary += "  " + StyleHelpDesc.Render(e.Orchestrator)
		}
		lines = append(lines, cursor+style.Render(summary))
	}
	return lines
}

func (ie InfraEditor) viewEnvForm(w int) []string {
	if ie.envIdx >= len(ie.envs) {
		return nil
	}
	header := StyleSectionDesc.Render("  editing environment: " + ie.envs[ie.envIdx].Name)
	lines := []string{header, ""}
	lines = append(lines, renderFormFields(w, ie.envForm, ie.envFormIdx, ie.internalMode == ModeInsert, ie.formInput, ie.dd.Open, ie.dd.OptIdx)...)
	return lines
}

// CurrentField returns the currently highlighted form field for the description panel.
// Returns nil when in list view or when no field can be resolved.
func (ie *InfraEditor) CurrentField() *Field {
	switch ie.activeTab {
	case infraTabNetworking:
		if ie.netEnabled && ie.netFormIdx >= 0 && ie.netFormIdx < len(ie.networkingFields) {
			return &ie.networkingFields[ie.netFormIdx]
		}
	case infraTabCICD:
		if ie.cicdEnabled && ie.cicdFormIdx >= 0 && ie.cicdFormIdx < len(ie.cicdFields) {
			return &ie.cicdFields[ie.cicdFormIdx]
		}
	case infraTabObservability:
		if ie.obsEnabled && ie.obsFormIdx >= 0 && ie.obsFormIdx < len(ie.obsFields) {
			return &ie.obsFields[ie.obsFormIdx]
		}
	case infraTabEnvironments:
		if ie.envView == envViewForm && ie.envFormIdx >= 0 && ie.envFormIdx < len(ie.envForm) {
			return &ie.envForm[ie.envFormIdx]
		}
	}
	return nil
}
