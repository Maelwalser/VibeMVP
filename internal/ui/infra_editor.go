package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-mvp/internal/manifest"
)

// ── sub-tabs ──────────────────────────────────────────────────────────────────

type infraTabIdx int

const (
	infraTabNetworking infraTabIdx = iota
	infraTabCICD
	infraTabObservability
	infraTabEnvironments
)

var infraTabLabels = []string{"NETWORKING", "CI/CD", "OBSERVABILITY", "ENVIRONMENTS"}

// ── mode ──────────────────────────────────────────────────────────────────────

type infraMode int

const (
	infraNormal infraMode = iota
	infraInsert
)

// ── field definitions ─────────────────────────────────────────────────────────

func defaultNetworkingFields() []Field {
	return []Field{
		{
			Key: "dns_provider", Label: "dns_provider  ", Kind: KindSelect,
			Options: []string{"Cloudflare", "Route53", "Cloud DNS", "Other"},
			Value:   "Cloudflare",
		},
		{
			Key: "tls_ssl", Label: "tls_ssl       ", Kind: KindSelect,
			Options: []string{"Let's Encrypt", "Cloudflare", "ACM", "Manual", "None (dev)"},
			Value:   "Let's Encrypt",
		},
		{
			Key: "reverse_proxy", Label: "reverse_proxy ", Kind: KindSelect,
			Options: []string{"Nginx", "Caddy", "Traefik", "Cloudflare Tunnel", "Cloud LB"},
			Value:   "Caddy", SelIdx: 1,
		},
		{
			Key: "cdn", Label: "cdn           ", Kind: KindSelect,
			Options: []string{"Cloudflare", "CloudFront", "Fastly", "Vercel Edge", "None"},
			Value:   "None", SelIdx: 4,
		},
		{Key: "primary_domain", Label: "Primary Domain", Kind: KindText},
		{
			Key: "domain_strategy", Label: "Domain Strat. ", Kind: KindSelect,
			Options: []string{"Subdomain per service", "Path-based routing", "Single domain", "Custom"},
			Value:   "Single domain", SelIdx: 2,
		},
		{
			Key: "cors_infra", Label: "CORS Enforced ", Kind: KindSelect,
			Options: []string{"Reverse proxy (Nginx/Caddy)", "Application-level", "CDN/WAF", "Both"},
			Value:   "Application-level", SelIdx: 1,
		},
		{
			Key: "ssl_cert", Label: "SSL Cert Mgmt ", Kind: KindSelect,
			Options: []string{"Auto-renew (certbot/ACME)", "Managed (cloud provider)", "Manual", "Cloudflare proxy"},
			Value:   "Auto-renew (certbot/ACME)",
		},
	}
}

func defaultInfraCICDFields() []Field {
	return []Field{
		{
			Key: "platform", Label: "platform      ", Kind: KindSelect,
			Options: []string{
				"GitHub Actions", "GitLab CI", "Jenkins",
				"CircleCI", "ArgoCD", "Tekton",
			},
			Value: "GitHub Actions",
		},
		{
			Key: "registry", Label: "registry      ", Kind: KindSelect,
			Options: []string{"Docker Hub", "GHCR", "ECR", "GCR", "Self-hosted"},
			Value:   "GHCR", SelIdx: 1,
		},
		{
			Key: "deploy_strategy", Label: "deploy_strat  ", Kind: KindSelect,
			Options: []string{"Rolling", "Blue-green", "Canary", "Recreate"},
			Value:   "Rolling",
		},
		{
			Key: "iac_tool", Label: "iac_tool      ", Kind: KindSelect,
			Options: []string{"Terraform", "Pulumi", "CloudFormation", "Ansible", "None"},
			Value:   "Terraform",
		},
		{
			Key: "secrets_mgmt", Label: "secrets_mgmt  ", Kind: KindSelect,
			Options: []string{
				"GitHub Secrets", "HashiCorp Vault", "AWS Secrets Manager",
				"GCP Secret Manager", "None",
			},
			Value: "GitHub Secrets",
		},
		{
			Key: "container_runtime", Label: "Container     ", Kind: KindSelect,
			Options: []string{"Node Alpine", "Go scratch", "Python slim", "Distroless", "Ubuntu", "Custom"},
			Value:   "Go scratch", SelIdx: 1,
		},
		{
			Key: "backup_dr", Label: "Backup/DR     ", Kind: KindSelect,
			Options: []string{"Cross-region replication", "Daily snapshots", "Managed provider DR", "None"},
			Value:   "None", SelIdx: 3,
		},
	}
}

func defaultEnvTopologyFields() []Field {
	return []Field{
		{
			Key: "stages", Label: "stages        ", Kind: KindSelect,
			Options: []string{
				"dev + prod", "dev + staging + prod",
				"dev + qa + staging + prod", "dev + staging + qa + preview + prod", "Custom",
			},
			Value: "dev + staging + prod", SelIdx: 1,
		},
		{
			Key: "promotion_pipeline", Label: "promotion     ", Kind: KindSelect,
			Options: []string{
				"Dev → Staging → Prod", "Dev → QA → Staging → Prod",
				"Dev → Prod (direct)", "Manual", "None",
			},
			Value: "Dev → Staging → Prod",
		},
		{
			Key: "secret_key_strategy", Label: "secret_keys   ", Kind: KindSelect,
			Options: []string{"Per-environment", "Shared base + overrides", "Fully shared", "None"},
			Value:   "Per-environment",
		},
		{
			Key: "migration_strategy", Label: "db_migrations ", Kind: KindSelect,
			Options: []string{
				"Auto on deploy", "Manual CI step", "Flyway", "Liquibase",
				"Atlas", "golang-migrate", "None",
			},
			Value: "Manual CI step", SelIdx: 1,
		},
		{
			Key: "db_seeding", Label: "db_seeding    ", Kind: KindSelect,
			Options: []string{"Automatic (fixtures)", "Manual", "None"},
			Value:   "None", SelIdx: 2,
		},
		{
			Key: "preview_envs", Label: "preview_envs  ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
	}
}

func defaultObservabilityFields() []Field {
	return []Field{
		{
			Key: "logging", Label: "logging       ", Kind: KindSelect,
			Options: []string{
				"Loki + Grafana", "ELK Stack", "CloudWatch",
				"Datadog", "Stdout/file",
			},
			Value: "Loki + Grafana",
		},
		{
			Key: "metrics", Label: "metrics       ", Kind: KindSelect,
			Options: []string{
				"Prometheus + Grafana", "Datadog", "CloudWatch", "New Relic", "None",
			},
			Value: "Prometheus + Grafana",
		},
		{
			Key: "tracing", Label: "tracing       ", Kind: KindSelect,
			Options: []string{
				"OpenTelemetry + Jaeger", "OpenTelemetry + Tempo", "Datadog APM", "None",
			},
			Value: "OpenTelemetry + Jaeger",
		},
		{
			Key: "error_tracking", Label: "error_tracking", Kind: KindSelect,
			Options: []string{"Sentry", "Datadog", "Rollbar", "Built-in", "None"},
			Value:   "Sentry",
		},
		{
			Key: "health_checks", Label: "health_checks ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "true", SelIdx: 1,
		},
		{
			Key: "alerting", Label: "alerting      ", Kind: KindSelect,
			Options: []string{
				"Grafana Alerting", "PagerDuty", "OpsGenie",
				"CloudWatch Alarms", "None",
			},
			Value: "Grafana Alerting",
		},
		{
			Key: "log_retention", Label: "Log Retention ", Kind: KindSelect,
			Options: []string{"7 days", "30 days", "90 days", "1 year", "Indefinite"},
			Value:   "30 days", SelIdx: 1,
		},
	}
}

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

	envTopoFields  []Field
	envTopoFormIdx int
	envEnabled     bool

	internalMode infraMode
	formInput    textinput.Model
	width        int

	// Vim motion state
	nav VimNav

	ddOpen   bool
	ddOptIdx int
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
		return ie.envEnabled
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
	case infraTabEnvironments:
		ie.envEnabled = true
		ie.envTopoFormIdx = 0
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
	case infraTabEnvironments:
		ie.envEnabled = false
		ie.envTopoFields = defaultEnvTopologyFields()
		ie.envTopoFormIdx = 0
	}
}

func newInfraEditor() InfraEditor {
	return InfraEditor{
		networkingFields: defaultNetworkingFields(),
		cicdFields:       defaultInfraCICDFields(),
		obsFields:        defaultObservabilityFields(),
		envTopoFields:    defaultEnvTopologyFields(),
		formInput:        newFormInput(),
	}
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (ie InfraEditor) ToManifestInfraPillar() manifest.InfraPillar {
	var p manifest.InfraPillar
	if ie.netEnabled {
		p.Networking = manifest.NetworkingConfig{
			DNSProvider:     fieldGet(ie.networkingFields, "dns_provider"),
			TLSSSL:          fieldGet(ie.networkingFields, "tls_ssl"),
			ReverseProxy:    fieldGet(ie.networkingFields, "reverse_proxy"),
			CDN:             fieldGet(ie.networkingFields, "cdn"),
			PrimaryDomain:   fieldGet(ie.networkingFields, "primary_domain"),
			DomainStrategy:  fieldGet(ie.networkingFields, "domain_strategy"),
			CORSEnforcement: fieldGet(ie.networkingFields, "cors_infra"),
			SSLCertMgmt:     fieldGet(ie.networkingFields, "ssl_cert"),
		}
	}
	if ie.cicdEnabled {
		p.CICD = manifest.CICDConfig{
			Platform:          fieldGet(ie.cicdFields, "platform"),
			ContainerRegistry: fieldGet(ie.cicdFields, "registry"),
			DeployStrategy:    fieldGet(ie.cicdFields, "deploy_strategy"),
			IaCTool:           fieldGet(ie.cicdFields, "iac_tool"),
			SecretsMgmt:       fieldGet(ie.cicdFields, "secrets_mgmt"),
			ContainerRuntime:  fieldGet(ie.cicdFields, "container_runtime"),
			BackupDR:          fieldGet(ie.cicdFields, "backup_dr"),
		}
	}
	if ie.obsEnabled {
		p.Observability = manifest.ObservabilityConfig{
			Logging:       fieldGet(ie.obsFields, "logging"),
			Metrics:       fieldGet(ie.obsFields, "metrics"),
			Tracing:       fieldGet(ie.obsFields, "tracing"),
			ErrorTracking: fieldGet(ie.obsFields, "error_tracking"),
			HealthChecks:  fieldGet(ie.obsFields, "health_checks") == "true",
			Alerting:      fieldGet(ie.obsFields, "alerting"),
			LogRetention:  fieldGet(ie.obsFields, "log_retention"),
		}
	}
	if ie.envEnabled {
		p.EnvTopology = manifest.EnvTopologyConfig{
			Stages:            fieldGet(ie.envTopoFields, "stages"),
			PromotionPipeline: fieldGet(ie.envTopoFields, "promotion_pipeline"),
			SecretKeyStrategy: fieldGet(ie.envTopoFields, "secret_key_strategy"),
			MigrationStrategy: fieldGet(ie.envTopoFields, "migration_strategy"),
			DBSeeding:         fieldGet(ie.envTopoFields, "db_seeding"),
			PreviewEnvs:       fieldGet(ie.envTopoFields, "preview_envs"),
		}
	}
	return p
}

// FromInfraPillar populates the editor from a saved manifest InfraPillar,
// reversing the ToManifestInfraPillar() operation.
func (ie InfraEditor) FromInfraPillar(ip manifest.InfraPillar) InfraEditor {
	n := ip.Networking
	if n.DNSProvider != "" || n.ReverseProxy != "" || n.CDN != "" {
		ie.netEnabled = true
		ie.networkingFields = setFieldValue(ie.networkingFields, "dns_provider", n.DNSProvider)
		ie.networkingFields = setFieldValue(ie.networkingFields, "tls_ssl", n.TLSSSL)
		ie.networkingFields = setFieldValue(ie.networkingFields, "reverse_proxy", n.ReverseProxy)
		ie.networkingFields = setFieldValue(ie.networkingFields, "cdn", n.CDN)
		ie.networkingFields = setFieldValue(ie.networkingFields, "primary_domain", n.PrimaryDomain)
		ie.networkingFields = setFieldValue(ie.networkingFields, "domain_strategy", n.DomainStrategy)
		ie.networkingFields = setFieldValue(ie.networkingFields, "cors_infra", n.CORSEnforcement)
		ie.networkingFields = setFieldValue(ie.networkingFields, "ssl_cert", n.SSLCertMgmt)
	}

	c := ip.CICD
	if c.Platform != "" {
		ie.cicdEnabled = true
		ie.cicdFields = setFieldValue(ie.cicdFields, "platform", c.Platform)
		ie.cicdFields = setFieldValue(ie.cicdFields, "registry", c.ContainerRegistry)
		ie.cicdFields = setFieldValue(ie.cicdFields, "deploy_strategy", c.DeployStrategy)
		ie.cicdFields = setFieldValue(ie.cicdFields, "iac_tool", c.IaCTool)
		ie.cicdFields = setFieldValue(ie.cicdFields, "secrets_mgmt", c.SecretsMgmt)
		ie.cicdFields = setFieldValue(ie.cicdFields, "container_runtime", c.ContainerRuntime)
		ie.cicdFields = setFieldValue(ie.cicdFields, "backup_dr", c.BackupDR)
	}

	o := ip.Observability
	if o.Logging != "" || o.Metrics != "" {
		ie.obsEnabled = true
		ie.obsFields = setFieldValue(ie.obsFields, "logging", o.Logging)
		ie.obsFields = setFieldValue(ie.obsFields, "metrics", o.Metrics)
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

	e := ip.EnvTopology
	if e.Stages != "" || e.PromotionPipeline != "" {
		ie.envEnabled = true
		ie.envTopoFields = setFieldValue(ie.envTopoFields, "stages", e.Stages)
		ie.envTopoFields = setFieldValue(ie.envTopoFields, "promotion_pipeline", e.PromotionPipeline)
		ie.envTopoFields = setFieldValue(ie.envTopoFields, "secret_key_strategy", e.SecretKeyStrategy)
		ie.envTopoFields = setFieldValue(ie.envTopoFields, "migration_strategy", e.MigrationStrategy)
		ie.envTopoFields = setFieldValue(ie.envTopoFields, "db_seeding", e.DBSeeding)
		ie.envTopoFields = setFieldValue(ie.envTopoFields, "preview_envs", e.PreviewEnvs)
	}

	return ie
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (ie InfraEditor) Mode() Mode {
	if ie.internalMode == infraInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (ie InfraEditor) HintLine() string {
	if ie.internalMode == infraInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	if ie.ddOpen {
		return hintBar("j/k", "navigate", "Enter/Space", "select", "Esc", "cancel")
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
	if ie.internalMode == infraInsert {
		return ie.updateInsert(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return ie, nil
	}

	if ie.ddOpen {
		return ie.updateInfraDropdown(key)
	}

	switch key.String() {
	case "h", "left", "l", "right":
		ie.activeTab = infraTabIdx(NavigateTab(key.String(), int(ie.activeTab), len(infraTabLabels)))
		return ie, nil
	}

	switch ie.activeTab {
	case infraTabNetworking:
		return ie.updateFields(key)
	case infraTabCICD:
		return ie.updateFields(key)
	case infraTabObservability:
		return ie.updateFields(key)
	case infraTabEnvironments:
		return ie.updateFields(key)
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
	case infraTabEnvironments:
		fields, idx = ie.envTopoFields, ie.envTopoFormIdx
	default:
		return ie, nil
	}
	n := len(fields)
	k := key.String()

	if newIdx, consumed := ie.nav.Handle(k, idx, n); consumed {
		idx = newIdx
	} else {
		ie.nav.Reset()
		switch k {
		case "enter", " ":
			if idx < n {
				f := &fields[idx]
				if f.Kind == KindSelect {
					ie.ddOpen = true
					ie.ddOptIdx = f.SelIdx
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
					case infraTabEnvironments:
						ie.envTopoFields = fields
						ie.envTopoFormIdx = idx
					}
					return ie.tryEnterInsert()
				}
			}
		case "H", "shift+left":
			if idx < n {
				f := &fields[idx]
				if f.Kind == KindSelect {
					f.CyclePrev()
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
			case infraTabEnvironments:
				ie.envTopoFields = fields
				ie.envTopoFormIdx = idx
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
	case infraTabEnvironments:
		ie.envTopoFields = fields
		ie.envTopoFormIdx = idx
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
	case infraTabEnvironments:
		if ie.envTopoFormIdx < len(ie.envTopoFields) {
			f = &ie.envTopoFields[ie.envTopoFormIdx]
		}
	}
	if f == nil {
		ie.ddOpen = false
		return ie, nil
	}
	switch key.String() {
	case "j", "down":
		if ie.ddOptIdx < len(f.Options)-1 {
			ie.ddOptIdx++
		}
	case "k", "up":
		if ie.ddOptIdx > 0 {
			ie.ddOptIdx--
		}
	case " ", "enter":
		f.SelIdx = ie.ddOptIdx
		if ie.ddOptIdx < len(f.Options) {
			f.Value = f.Options[ie.ddOptIdx]
		}
		ie.ddOpen = false
	case "esc", "b":
		ie.ddOpen = false
	}
	return ie, nil
}

func (ie InfraEditor) updateInsert(msg tea.Msg) (InfraEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			ie.saveInput()
			ie.internalMode = infraNormal
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
		n := len(ie.envTopoFields)
		if n > 0 {
			ie.envTopoFormIdx = (ie.envTopoFormIdx + delta + n) % n
		}
	}
}

func (ie *InfraEditor) saveInput() {
	val := ie.formInput.Value()
	switch ie.activeTab {
	case infraTabNetworking:
		if ie.netFormIdx < len(ie.networkingFields) && ie.networkingFields[ie.netFormIdx].Kind == KindText {
			ie.networkingFields[ie.netFormIdx].Value = val
		}
	case infraTabCICD:
		if ie.cicdFormIdx < len(ie.cicdFields) && ie.cicdFields[ie.cicdFormIdx].Kind == KindText {
			ie.cicdFields[ie.cicdFormIdx].Value = val
		}
	case infraTabObservability:
		if ie.obsFormIdx < len(ie.obsFields) && ie.obsFields[ie.obsFormIdx].Kind == KindText {
			ie.obsFields[ie.obsFormIdx].Value = val
		}
	case infraTabEnvironments:
		if ie.envTopoFormIdx < len(ie.envTopoFields) && ie.envTopoFields[ie.envTopoFormIdx].Kind == KindText {
			ie.envTopoFields[ie.envTopoFormIdx].Value = val
		}
	}
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
		n = len(ie.envTopoFields)
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
			if ie.envTopoFormIdx < len(ie.envTopoFields) {
				f = &ie.envTopoFields[ie.envTopoFormIdx]
			}
		}
		if f == nil {
			break
		}
		if f.Kind == KindText || f.Kind == KindTextArea {
			ie.internalMode = infraInsert
			ie.formInput.SetValue(f.Value)
			ie.formInput.Width = ie.width - 22
			ie.formInput.CursorEnd()
			return ie, ie.formInput.Focus()
		}
		ie.advanceField(1)
	}
	return ie, nil
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

	switch ie.activeTab {
	case infraTabNetworking:
		if ie.netEnabled {
			lines = append(lines, renderFormFields(w, ie.networkingFields, ie.netFormIdx, ie.internalMode == infraInsert, ie.formInput, ie.ddOpen, ie.ddOptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case infraTabCICD:
		if ie.cicdEnabled {
			lines = append(lines, renderFormFields(w, ie.cicdFields, ie.cicdFormIdx, ie.internalMode == infraInsert, ie.formInput, ie.ddOpen, ie.ddOptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case infraTabObservability:
		if ie.obsEnabled {
			lines = append(lines, renderFormFields(w, ie.obsFields, ie.obsFormIdx, ie.internalMode == infraInsert, ie.formInput, ie.ddOpen, ie.ddOptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case infraTabEnvironments:
		if ie.envEnabled {
			lines = append(lines, renderFormFields(w, ie.envTopoFields, ie.envTopoFormIdx, ie.internalMode == infraInsert, ie.formInput, ie.ddOpen, ie.ddOptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	}

	return fillTildes(lines, h)
}
