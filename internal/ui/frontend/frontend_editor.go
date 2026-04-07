package frontend

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── sub-tabs ──────────────────────────────────────────────────────────────────

type feTabIdx int

const (
	feTabTech feTabIdx = iota
	feTabTheme
	feTabPages
	feTabComponents
	feTabNav
	feTabI18n
	feTabA11ySEO
	feTabAssets
)

var feTabLabels = []string{"TECHNOLOGIES", "THEMING", "PAGES", "COMPONENTS", "NAVIGATION", "I18N", "A11Y/SEO", "ASSETS"}

// ── FrontendEditor ────────────────────────────────────────────────────────────

// FrontendEditor manages the FRONTEND main-tab.
type FrontendEditor struct {
	activeTab feTabIdx

	// TECHNOLOGIES
	techFields  []core.Field
	techFormIdx int
	techEnabled bool

	// THEMING
	themeFields  []core.Field
	themeFormIdx int
	themeEnabled bool

	// PAGES
	pages       []manifest.PageDef
	pageSubView core.SubView // core.ViewList / core.ViewForm
	pageIdx     int
	pageForm    []core.Field
	pageFormIdx int

	// NAVIGATION
	navFields  []core.Field
	navFormIdx int
	navEnabled bool

	// I18N
	i18nFields  []core.Field
	i18nFormIdx int
	i18nEnabled bool

	// A11Y/SEO
	a11yFields  []core.Field
	a11yFormIdx int
	a11yEnabled bool

	// ASSETS
	assets       []manifest.AssetDef
	assetSubView core.SubView // core.ViewList | core.ViewForm
	assetIdx     int
	assetForm    []core.Field
	assetFormIdx int

	// COMPONENTS (shared library — managed in COMPONENTS tab)
	components  []manifest.PageComponentDef
	compSubView core.SubView // core.ViewList | core.ViewForm within COMPONENTS tab
	compIdx     int
	compForm    []core.Field
	compFormIdx int

	// ACTIONS (drill-down from component form — press A)
	inCompAction    bool
	currentCompType string // component type of the comp being edited, for action form options
	compActions     []manifest.ComponentActionDef
	actionSubView   core.SubView
	actionIdx       int
	actionForm      []core.Field
	actionFormIdx   int

	// Cross-editor data
	availableAuthRoles   []string // from BackendEditor auth roles
	backendProtocols     []string // comm-link protocols (REST (HTTP), GraphQL, gRPC, tRPC, …)
	backendSvcFrameworks []string // service frameworks (tRPC, NestJS, …)
	availableEndpoints   []string // from ContractsEditor.EndpointNames()

	// Dropdown state for KindSelect/KindMultiSelect fields
	dd core.DropdownState

	// Shared
	internalMode core.Mode
	formInput    textinput.Model
	formTextArea textarea.Model
	inTextArea   bool
	width        int

	cBuf bool

	// Per-subtab undo stacks (structural add/delete only)
	pagesUndo  core.UndoStack[[]manifest.PageDef]
	assetsUndo core.UndoStack[[]manifest.AssetDef]
	compsUndo  core.UndoStack[[]manifest.PageComponentDef]
}

func NewEditor() FrontendEditor {
	return FrontendEditor{
		techFields:   defaultFETechFields(),
		themeFields:  defaultFEThemeFields(),
		navFields:    defaultNavFields(),
		i18nFields:   defaultI18nFields(),
		a11yFields:   defaultA11ySEOFields(),
		formInput:    core.NewFormInput(),
		formTextArea: core.NewFormTextArea(),
	}
}

// SetAuthRoles sets the available auth role options for page forms.
func (fe *FrontendEditor) SetAuthRoles(roles []string) {
	fe.availableAuthRoles = roles
}

// SetBackendProtocols updates the backend protocol/framework context used to
// filter the data_fetching options. Triggers a re-evaluation of dependent fields.
func (fe *FrontendEditor) SetBackendProtocols(protocols, svcFrameworks []string) {
	if core.StringSlicesEqual(fe.backendProtocols, protocols) &&
		core.StringSlicesEqual(fe.backendSvcFrameworks, svcFrameworks) {
		return
	}
	fe.backendProtocols = protocols
	fe.backendSvcFrameworks = svcFrameworks
	fe.updateFEDependentOptions()
}

// SetBackendAuthStrategy updates the auth_flow field options to match the backend auth strategy.
// JWT → redirect/magic-link flows; Session-based → modal login; API Key → not applicable.
func (fe *FrontendEditor) SetBackendAuthStrategy(strategies []string) {
	opts := authFlowOptionsFor(strategies)
	for i, f := range fe.techFields {
		if f.Key != "auth_flow" {
			continue
		}
		// Only update when the option set actually changes.
		if core.StringSlicesEqual(f.Options, opts) {
			return
		}
		fe.techFields[i].Options = opts
		fe.techFields[i].SelIdx = 0
		fe.techFields[i].Value = opts[0]
		return
	}
}

// authFlowOptionsFor derives the appropriate auth_flow options from the backend strategy list.
func authFlowOptionsFor(strategies []string) []string {
	hasJWT, hasSession, hasOAuth, hasAPIKey := false, false, false, false
	for _, s := range strategies {
		switch {
		case strings.Contains(s, "JWT"):
			hasJWT = true
		case strings.Contains(s, "Session"):
			hasSession = true
		case strings.Contains(s, "OAuth") || strings.Contains(s, "OIDC"):
			hasOAuth = true
		case strings.Contains(s, "API Key"):
			hasAPIKey = true
		}
	}

	// No strategy configured — return all options.
	if !hasJWT && !hasSession && !hasOAuth && !hasAPIKey && len(strategies) == 0 {
		return []string{"Redirect (OAuth/OIDC)", "Modal login", "Magic link", "Passwordless", "Social only"}
	}

	var opts []string
	if hasOAuth || hasJWT {
		opts = append(opts, "Redirect (OAuth/OIDC)", "Magic link", "Passwordless", "Social only")
	}
	if hasSession {
		opts = append(opts, "Modal login")
	}
	if hasAPIKey && !hasJWT && !hasSession && !hasOAuth {
		opts = append(opts, "Not applicable (API Key only)")
	}
	if len(opts) == 0 {
		// Fallback for unrecognised strategies (e.g. mTLS / None).
		opts = []string{"Redirect (OAuth/OIDC)", "Modal login", "Magic link", "Passwordless", "Social only"}
	}
	return opts
}

// SetAvailableEndpoints updates the endpoint name list for component forms.
func (fe *FrontendEditor) SetAvailableEndpoints(endpoints []string) {
	fe.availableEndpoints = endpoints
}

// Language returns the frontend language selected in the Tech sub-tab.
func (fe FrontendEditor) Language() string { return core.FieldGet(fe.techFields, "language") }

// Framework returns the frontend framework selected in the Tech sub-tab.
func (fe FrontendEditor) Framework() string { return core.FieldGet(fe.techFields, "framework") }

// Platform returns the frontend platform selected in the Tech sub-tab.
func (fe FrontendEditor) Platform() string { return core.FieldGet(fe.techFields, "platform") }

// componentNames returns the names of all components in the shared library.
func (fe FrontendEditor) componentNames() []string {
	names := make([]string, 0, len(fe.components))
	for _, c := range fe.components {
		if c.Name != "" {
			names = append(names, c.Name)
		}
	}
	return names
}

// formComponentNames returns names of components with type "Form".
func (fe FrontendEditor) formComponentNames() []string {
	var names []string
	for _, c := range fe.components {
		if c.ComponentType == "Form" && c.Name != "" {
			names = append(names, c.Name)
		}
	}
	return names
}

// modalComponentNames returns names of components with type "Modal".
func (fe FrontendEditor) modalComponentNames() []string {
	var names []string
	for _, c := range fe.components {
		if c.ComponentType == "Modal" && c.Name != "" {
			names = append(names, c.Name)
		}
	}
	return names
}

// pageRoutes returns routes of all existing pages (for linked_pages options).
func (fe FrontendEditor) pageRoutes() []string {
	routes := make([]string, 0, len(fe.pages))
	for _, p := range fe.pages {
		if p.Route != "" {
			routes = append(routes, p.Route)
		}
	}
	return routes
}

// assetNames returns names of all existing assets (for page assets options).
func (fe FrontendEditor) assetNames() []string {
	names := make([]string, 0, len(fe.assets))
	for _, a := range fe.assets {
		if a.Name != "" {
			names = append(names, a.Name)
		}
	}
	return names
}

// ── ToManifest ────────────────────────────────────────────────────────────────

// legacyRenderingMode maps a meta-framework value to the closest legacy RenderingMode.
func legacyRenderingMode(metaFW string) manifest.RenderingMode {
	switch strings.ToLower(metaFW) {
	case "next.js", "nuxt", "sveltekit", "remix":
		return manifest.RenderSSR
	case "gatsby", "astro":
		return manifest.RenderSSG
	default:
		return manifest.RenderSPA
	}
}

func (fe FrontendEditor) ToManifestFrontendPillar() manifest.FrontendPillar {
	assets := make([]manifest.AssetDef, len(fe.assets))
	copy(assets, fe.assets)
	components := make([]manifest.PageComponentDef, len(fe.components))
	copy(components, fe.components)
	p := manifest.FrontendPillar{
		Components: components,
		Pages:      fe.pages,
		Assets:     assets,
	}
	if fe.techEnabled {
		p.Tech = &manifest.FrontendTechConfig{
			Language:           core.NoneToEmpty(core.FieldGet(fe.techFields, "language")),
			LanguageVersion:    core.FieldGet(fe.techFields, "language_version"),
			Platform:           core.NoneToEmpty(core.FieldGet(fe.techFields, "platform")),
			Framework:          core.NoneToEmpty(core.FieldGet(fe.techFields, "framework")),
			FrameworkVersion:   core.FieldGet(fe.techFields, "framework_version"),
			MetaFramework:      core.NoneToEmpty(core.FieldGet(fe.techFields, "meta_framework")),
			PackageManager:     core.NoneToEmpty(core.FieldGet(fe.techFields, "pkg_manager")),
			Styling:            core.NoneToEmpty(core.FieldGet(fe.techFields, "styling")),
			ComponentLib:       core.NoneToEmpty(core.FieldGet(fe.techFields, "component_lib")),
			StateManagement:    core.NoneToEmpty(core.FieldGet(fe.techFields, "state_mgmt")),
			DataFetching:       core.NoneToEmpty(core.FieldGet(fe.techFields, "data_fetching")),
			FormHandling:       core.NoneToEmpty(core.FieldGet(fe.techFields, "form_handling")),
			Validation:         core.NoneToEmpty(core.FieldGet(fe.techFields, "validation")),
			PWASupport:         core.NoneToEmpty(core.FieldGet(fe.techFields, "pwa_support")),
			RealtimeStrategy:   core.NoneToEmpty(core.FieldGet(fe.techFields, "realtime")),
			ImageOptimization:  core.NoneToEmpty(core.FieldGet(fe.techFields, "image_opt")),
			AuthFlowType:       core.NoneToEmpty(core.FieldGet(fe.techFields, "auth_flow")),
			ErrorBoundary:      core.NoneToEmpty(core.FieldGet(fe.techFields, "error_boundary")),
			BundleOptimization: core.NoneToEmpty(core.FieldGet(fe.techFields, "bundle_opt")),
		}
		// Legacy compatibility — Rendering is SPA/SSR/SSG/ISR, not the platform value.
		// Map meta-framework choices to approximate rendering modes for old consumers.
		p.Rendering = legacyRenderingMode(core.NoneToEmpty(core.FieldGet(fe.techFields, "meta_framework")))
		p.Framework = core.NoneToEmpty(core.FieldGet(fe.techFields, "framework"))
		p.Styling = core.NoneToEmpty(core.FieldGet(fe.techFields, "styling"))
	}
	if fe.themeEnabled {
		p.Theme = &manifest.FrontendTheme{
			DarkMode:     core.NoneToEmpty(core.FieldGet(fe.themeFields, "dark_mode")),
			BorderRadius: core.NoneToEmpty(core.FieldGet(fe.themeFields, "border_radius")),
			Spacing:      core.NoneToEmpty(core.FieldGet(fe.themeFields, "spacing")),
			Elevation:    core.NoneToEmpty(core.FieldGet(fe.themeFields, "elevation")),
			Motion:       core.NoneToEmpty(core.FieldGet(fe.themeFields, "motion")),
			Vibe:         core.NoneToEmpty(core.FieldGet(fe.themeFields, "vibe")),
			Font:         core.FieldGet(fe.themeFields, "font"),
			Colors:       core.FieldGet(fe.themeFields, "colors"),
			Description:  core.FieldGet(fe.themeFields, "description"),
		}
	}
	if fe.navEnabled {
		p.Navigation = &manifest.NavigationConfig{
			NavType:     core.NoneToEmpty(core.FieldGet(fe.navFields, "nav_type")),
			Breadcrumbs: core.FieldGet(fe.navFields, "breadcrumbs") == "true",
			AuthAware:   core.FieldGet(fe.navFields, "auth_aware") == "true",
		}
	}
	if fe.i18nEnabled {
		p.I18n = &manifest.I18nConfig{
			Enabled:             core.NoneToEmpty(core.FieldGet(fe.i18nFields, "enabled")),
			DefaultLocale:       core.FieldGet(fe.i18nFields, "default_locale"),
			SupportedLocales:    core.FieldGetMulti(fe.i18nFields, "supported_locales"),
			TranslationStrategy: core.NoneToEmpty(core.FieldGet(fe.i18nFields, "translation_strategy")),
			TimezoneHandling:    core.NoneToEmpty(core.FieldGet(fe.i18nFields, "timezone_handling")),
		}
	}
	if fe.a11yEnabled {
		p.A11ySEO = &manifest.A11ySEOConfig{
			WCAGLevel:         core.NoneToEmpty(core.FieldGet(fe.a11yFields, "wcag_level")),
			SEORenderStrategy: core.NoneToEmpty(core.FieldGet(fe.a11yFields, "seo_render_strategy")),
			Sitemap:           core.NoneToEmpty(core.FieldGet(fe.a11yFields, "sitemap")),
			MetaTagInjection:  core.NoneToEmpty(core.FieldGet(fe.a11yFields, "meta_tag_injection")),
			Analytics:         core.NoneToEmpty(core.FieldGet(fe.a11yFields, "analytics")),
			Telemetry:         core.NoneToEmpty(core.FieldGet(fe.a11yFields, "telemetry")),
		}
	}
	return p
}

// FromFrontendPillar populates the editor from a saved manifest FrontendPillar,
// reversing the ToManifestFrontendPillar() operation.
func (fe FrontendEditor) FromFrontendPillar(fp manifest.FrontendPillar) FrontendEditor {
	if fp.Tech != nil && (fp.Tech.Language != "" || fp.Tech.Framework != "" || fp.Tech.Platform != "") {
		fe.techEnabled = true
		fe.techFields = core.SetFieldValue(fe.techFields, "language", fp.Tech.Language)
		fe.techFields = core.SetFieldValue(fe.techFields, "language_version", fp.Tech.LanguageVersion)
		fe.techFields = core.SetFieldValue(fe.techFields, "platform", fp.Tech.Platform)
		fe.techFields = core.SetFieldValue(fe.techFields, "framework", fp.Tech.Framework)
		fe.techFields = core.SetFieldValue(fe.techFields, "framework_version", fp.Tech.FrameworkVersion)
		fe.techFields = core.SetFieldValue(fe.techFields, "meta_framework", fp.Tech.MetaFramework)
		fe.techFields = core.SetFieldValue(fe.techFields, "pkg_manager", fp.Tech.PackageManager)
		fe.techFields = core.SetFieldValue(fe.techFields, "styling", fp.Tech.Styling)
		fe.techFields = core.SetFieldValue(fe.techFields, "component_lib", fp.Tech.ComponentLib)
		fe.techFields = core.SetFieldValue(fe.techFields, "state_mgmt", fp.Tech.StateManagement)
		fe.techFields = core.SetFieldValue(fe.techFields, "data_fetching", fp.Tech.DataFetching)
		fe.techFields = core.SetFieldValue(fe.techFields, "form_handling", fp.Tech.FormHandling)
		fe.techFields = core.SetFieldValue(fe.techFields, "validation", fp.Tech.Validation)
		fe.techFields = core.SetFieldValue(fe.techFields, "pwa_support", fp.Tech.PWASupport)
		fe.techFields = core.SetFieldValue(fe.techFields, "realtime", fp.Tech.RealtimeStrategy)
		fe.techFields = core.SetFieldValue(fe.techFields, "image_opt", fp.Tech.ImageOptimization)
		fe.techFields = core.SetFieldValue(fe.techFields, "auth_flow", fp.Tech.AuthFlowType)
		fe.techFields = core.SetFieldValue(fe.techFields, "error_boundary", fp.Tech.ErrorBoundary)
		fe.techFields = core.SetFieldValue(fe.techFields, "bundle_opt", fp.Tech.BundleOptimization)
	}

	if fp.Theme != nil && (fp.Theme.DarkMode != "" || fp.Theme.BorderRadius != "") {
		fe.themeEnabled = true
		fe.themeFields = core.SetFieldValue(fe.themeFields, "dark_mode", fp.Theme.DarkMode)
		fe.themeFields = core.SetFieldValue(fe.themeFields, "border_radius", fp.Theme.BorderRadius)
		fe.themeFields = core.SetFieldValue(fe.themeFields, "spacing", fp.Theme.Spacing)
		fe.themeFields = core.SetFieldValue(fe.themeFields, "elevation", fp.Theme.Elevation)
		fe.themeFields = core.SetFieldValue(fe.themeFields, "motion", fp.Theme.Motion)
		fe.themeFields = core.SetFieldValue(fe.themeFields, "vibe", fp.Theme.Vibe)
		fe.themeFields = core.SetFieldValue(fe.themeFields, "font", fp.Theme.Font)
		fe.themeFields = core.RestoreMultiSelectValue(fe.themeFields, "colors", fp.Theme.Colors)
		fe.themeFields = core.SetFieldValue(fe.themeFields, "description", fp.Theme.Description)
	}

	// Collections stored directly; per-item forms rebuilt lazily on navigation.
	fe.components = fp.Components
	fe.pages = fp.Pages
	fe.assets = fp.Assets

	if fp.Navigation != nil && fp.Navigation.NavType != "" {
		fe.navEnabled = true
		fe.navFields = core.SetFieldValue(fe.navFields, "nav_type", fp.Navigation.NavType)
		boolStr := func(b bool) string {
			if b {
				return "true"
			}
			return "false"
		}
		fe.navFields = core.SetFieldValue(fe.navFields, "breadcrumbs", boolStr(fp.Navigation.Breadcrumbs))
		fe.navFields = core.SetFieldValue(fe.navFields, "auth_aware", boolStr(fp.Navigation.AuthAware))
	}

	if fp.I18n != nil && (fp.I18n.Enabled != "" || fp.I18n.DefaultLocale != "") {
		fe.i18nEnabled = true
		fe.i18nFields = core.SetFieldValue(fe.i18nFields, "enabled", fp.I18n.Enabled)
		fe.i18nFields = core.SetFieldValue(fe.i18nFields, "default_locale", fp.I18n.DefaultLocale)
		fe.i18nFields = core.RestoreMultiSelectValue(fe.i18nFields, "supported_locales", fp.I18n.SupportedLocales)
		fe.i18nFields = core.SetFieldValue(fe.i18nFields, "translation_strategy", fp.I18n.TranslationStrategy)
		fe.i18nFields = core.SetFieldValue(fe.i18nFields, "timezone_handling", fp.I18n.TimezoneHandling)
	}

	if fp.A11ySEO != nil && (fp.A11ySEO.WCAGLevel != "" || fp.A11ySEO.SEORenderStrategy != "") {
		fe.a11yEnabled = true
		fe.a11yFields = core.SetFieldValue(fe.a11yFields, "wcag_level", fp.A11ySEO.WCAGLevel)
		fe.a11yFields = core.SetFieldValue(fe.a11yFields, "seo_render_strategy", fp.A11ySEO.SEORenderStrategy)
		fe.a11yFields = core.SetFieldValue(fe.a11yFields, "sitemap", fp.A11ySEO.Sitemap)
		fe.a11yFields = core.SetFieldValue(fe.a11yFields, "meta_tag_injection", fp.A11ySEO.MetaTagInjection)
		fe.a11yFields = core.SetFieldValue(fe.a11yFields, "analytics", fp.A11ySEO.Analytics)
		fe.a11yFields = core.SetFieldValue(fe.a11yFields, "telemetry", fp.A11ySEO.Telemetry)
	}

	return fe
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (fe FrontendEditor) Mode() core.Mode {
	if fe.internalMode == core.ModeInsert {
		return core.ModeInsert
	}
	return core.ModeNormal
}

func (fe FrontendEditor) HintLine() string {
	if fe.internalMode == core.ModeInsert {
		return core.StyleInsertMode.Render(" -- INSERT -- ") +
			core.StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch fe.activeTab {
	case feTabTech:
		if !fe.techEnabled {
			return core.HintBar("a", "configure", "h/l", "sub-tab")
		}
		return core.HintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit", "h/l", "sub-tab")
	case feTabTheme:
		if !fe.themeEnabled {
			return core.HintBar("a", "configure", "h/l", "sub-tab")
		}
		return core.HintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit", "h/l", "sub-tab")
	case feTabNav:
		if !fe.navEnabled {
			return core.HintBar("a", "configure", "h/l", "sub-tab")
		}
		return core.HintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit", "h/l", "sub-tab")
	case feTabI18n:
		if !fe.i18nEnabled {
			return core.HintBar("a", "configure", "h/l", "sub-tab")
		}
		return core.HintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit", "h/l", "sub-tab")
	case feTabA11ySEO:
		if !fe.a11yEnabled {
			return core.HintBar("a", "configure", "h/l", "sub-tab")
		}
		return core.HintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit", "h/l", "sub-tab")
	case feTabComponents:
		if fe.inCompAction {
			if fe.actionSubView == core.ViewList {
				return core.HintBar("j/k", "navigate", "a", "add action", "d", "delete", "Enter", "edit", "b/Esc", "back")
			}
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
		if fe.compSubView == core.ViewForm {
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "A", "actions", "b/Esc", "back")
		}
		return core.HintBar("j/k", "navigate", "a", "add", "d", "delete", "u", "undo", "Enter", "edit", "h/l", "sub-tab")
	case feTabPages:
		if fe.pageSubView == core.ViewList {
			return core.HintBar("j/k", "navigate", "a", "add page", "d", "delete", "u", "undo", "Enter", "edit", "h/l", "sub-tab")
		}
		return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	case feTabAssets:
		if fe.assetSubView == core.ViewList {
			return core.HintBar("j/k", "navigate", "a", "add asset", "d", "delete", "u", "undo", "Enter", "edit", "h/l", "sub-tab")
		}
		return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	}
	return ""
}
