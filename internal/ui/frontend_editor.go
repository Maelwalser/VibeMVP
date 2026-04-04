package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
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
	techFields  []Field
	techFormIdx int
	techEnabled bool

	// THEMING
	themeFields  []Field
	themeFormIdx int
	themeEnabled bool

	// PAGES
	pages       []manifest.PageDef
	pageSubView ceSubView // reuse ceSubView: ceViewList / ceViewForm
	pageIdx     int
	pageForm    []Field
	pageFormIdx int

	// NAVIGATION
	navFields  []Field
	navFormIdx int
	navEnabled bool

	// I18N
	i18nFields  []Field
	i18nFormIdx int
	i18nEnabled bool

	// A11Y/SEO
	a11yFields  []Field
	a11yFormIdx int
	a11yEnabled bool

	// ASSETS
	assets       []manifest.AssetDef
	assetSubView ceSubView // ceViewList | ceViewForm
	assetIdx     int
	assetForm    []Field
	assetFormIdx int

	// COMPONENTS (shared library — managed in COMPONENTS tab)
	components  []manifest.PageComponentDef
	compSubView ceSubView // ceViewList | ceViewForm within COMPONENTS tab
	compIdx     int
	compForm    []Field
	compFormIdx int

	// ACTIONS (drill-down from component form — press A)
	inCompAction    bool
	currentCompType string // component type of the comp being edited, for action form options
	compActions     []manifest.ComponentActionDef
	actionSubView   ceSubView
	actionIdx       int
	actionForm      []Field
	actionFormIdx   int

	// Cross-editor data
	availableAuthRoles   []string // from BackendEditor auth roles
	backendProtocols     []string // comm-link protocols (REST (HTTP), GraphQL, gRPC, tRPC, …)
	backendSvcFrameworks []string // service frameworks (tRPC, NestJS, …)
	availableEndpoints   []string // from ContractsEditor.EndpointNames()

	// Dropdown state for KindSelect/KindMultiSelect fields
	dd DropdownState

	// Shared
	internalMode Mode
	formInput    textinput.Model
	formTextArea textarea.Model
	inTextArea   bool
	width        int
}

func newFrontendEditor() FrontendEditor {
	return FrontendEditor{
		techFields:   defaultFETechFields(),
		themeFields:  defaultFEThemeFields(),
		navFields:    defaultNavFields(),
		i18nFields:   defaultI18nFields(),
		a11yFields:   defaultA11ySEOFields(),
		formInput:    newFormInput(),
		formTextArea: newFormTextArea(),
	}
}

// SetAuthRoles sets the available auth role options for page forms.
func (fe *FrontendEditor) SetAuthRoles(roles []string) {
	fe.availableAuthRoles = roles
}

// SetBackendProtocols updates the backend protocol/framework context used to
// filter the data_fetching options. Triggers a re-evaluation of dependent fields.
func (fe *FrontendEditor) SetBackendProtocols(protocols, svcFrameworks []string) {
	if stringSlicesEqual(fe.backendProtocols, protocols) &&
		stringSlicesEqual(fe.backendSvcFrameworks, svcFrameworks) {
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
		if stringSlicesEqual(f.Options, opts) {
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
func (fe FrontendEditor) Language() string { return fieldGet(fe.techFields, "language") }

// Framework returns the frontend framework selected in the Tech sub-tab.
func (fe FrontendEditor) Framework() string { return fieldGet(fe.techFields, "framework") }

// Platform returns the frontend platform selected in the Tech sub-tab.
func (fe FrontendEditor) Platform() string { return fieldGet(fe.techFields, "platform") }

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
		p.Tech = manifest.FrontendTechConfig{
			Language:           fieldGet(fe.techFields, "language"),
			LanguageVersion:    fieldGet(fe.techFields, "language_version"),
			Platform:           fieldGet(fe.techFields, "platform"),
			Framework:          fieldGet(fe.techFields, "framework"),
			FrameworkVersion:   fieldGet(fe.techFields, "framework_version"),
			MetaFramework:      fieldGet(fe.techFields, "meta_framework"),
			PackageManager:     fieldGet(fe.techFields, "pkg_manager"),
			Styling:            fieldGet(fe.techFields, "styling"),
			ComponentLib:       fieldGet(fe.techFields, "component_lib"),
			StateManagement:    fieldGet(fe.techFields, "state_mgmt"),
			DataFetching:       fieldGet(fe.techFields, "data_fetching"),
			FormHandling:       fieldGet(fe.techFields, "form_handling"),
			Validation:         fieldGet(fe.techFields, "validation"),
			PWASupport:         fieldGet(fe.techFields, "pwa_support"),
			RealtimeStrategy:   fieldGet(fe.techFields, "realtime"),
			ImageOptimization:  fieldGet(fe.techFields, "image_opt"),
			AuthFlowType:       fieldGet(fe.techFields, "auth_flow"),
			ErrorBoundary:      fieldGet(fe.techFields, "error_boundary"),
			BundleOptimization: fieldGet(fe.techFields, "bundle_opt"),
		}
		// Legacy compatibility
		p.Rendering = manifest.RenderingMode(fieldGet(fe.techFields, "platform"))
		p.Framework = fieldGet(fe.techFields, "framework")
		p.Styling = fieldGet(fe.techFields, "styling")
	}
	if fe.themeEnabled {
		p.Theme = manifest.FrontendTheme{
			DarkMode:     fieldGet(fe.themeFields, "dark_mode"),
			BorderRadius: fieldGet(fe.themeFields, "border_radius"),
			Spacing:      fieldGet(fe.themeFields, "spacing"),
			Elevation:    fieldGet(fe.themeFields, "elevation"),
			Motion:       fieldGet(fe.themeFields, "motion"),
			Vibe:         fieldGet(fe.themeFields, "vibe"),
			Font:         fieldGet(fe.themeFields, "font"),
			Colors:       fieldGet(fe.themeFields, "colors"),
			Description:  fieldGet(fe.themeFields, "description"),
		}
	}
	if fe.navEnabled {
		p.Navigation = manifest.NavigationConfig{
			NavType:     fieldGet(fe.navFields, "nav_type"),
			Breadcrumbs: fieldGet(fe.navFields, "breadcrumbs") == "true",
			AuthAware:   fieldGet(fe.navFields, "auth_aware") == "true",
		}
	}
	if fe.i18nEnabled {
		p.I18n = manifest.I18nConfig{
			Enabled:             fieldGet(fe.i18nFields, "enabled"),
			DefaultLocale:       fieldGet(fe.i18nFields, "default_locale"),
			SupportedLocales:    fieldGetMulti(fe.i18nFields, "supported_locales"),
			TranslationStrategy: fieldGet(fe.i18nFields, "translation_strategy"),
			TimezoneHandling:    fieldGet(fe.i18nFields, "timezone_handling"),
		}
	}
	if fe.a11yEnabled {
		p.A11ySEO = manifest.A11ySEOConfig{
			WCAGLevel:         fieldGet(fe.a11yFields, "wcag_level"),
			SEORenderStrategy: fieldGet(fe.a11yFields, "seo_render_strategy"),
			Sitemap:           fieldGet(fe.a11yFields, "sitemap"),
			MetaTagInjection:  fieldGet(fe.a11yFields, "meta_tag_injection"),
			Analytics:         fieldGet(fe.a11yFields, "analytics"),
			Telemetry:         fieldGet(fe.a11yFields, "telemetry"),
		}
	}
	return p
}

// FromFrontendPillar populates the editor from a saved manifest FrontendPillar,
// reversing the ToManifestFrontendPillar() operation.
func (fe FrontendEditor) FromFrontendPillar(fp manifest.FrontendPillar) FrontendEditor {
	t := fp.Tech
	if t.Language != "" || t.Framework != "" || t.Platform != "" {
		fe.techEnabled = true
		fe.techFields = setFieldValue(fe.techFields, "language", t.Language)
		fe.techFields = setFieldValue(fe.techFields, "language_version", t.LanguageVersion)
		fe.techFields = setFieldValue(fe.techFields, "platform", t.Platform)
		fe.techFields = setFieldValue(fe.techFields, "framework", t.Framework)
		fe.techFields = setFieldValue(fe.techFields, "framework_version", t.FrameworkVersion)
		fe.techFields = setFieldValue(fe.techFields, "meta_framework", t.MetaFramework)
		fe.techFields = setFieldValue(fe.techFields, "pkg_manager", t.PackageManager)
		fe.techFields = setFieldValue(fe.techFields, "styling", t.Styling)
		fe.techFields = setFieldValue(fe.techFields, "component_lib", t.ComponentLib)
		fe.techFields = setFieldValue(fe.techFields, "state_mgmt", t.StateManagement)
		fe.techFields = setFieldValue(fe.techFields, "data_fetching", t.DataFetching)
		fe.techFields = setFieldValue(fe.techFields, "form_handling", t.FormHandling)
		fe.techFields = setFieldValue(fe.techFields, "validation", t.Validation)
		fe.techFields = setFieldValue(fe.techFields, "pwa_support", t.PWASupport)
		fe.techFields = setFieldValue(fe.techFields, "realtime", t.RealtimeStrategy)
		fe.techFields = setFieldValue(fe.techFields, "image_opt", t.ImageOptimization)
		fe.techFields = setFieldValue(fe.techFields, "auth_flow", t.AuthFlowType)
		fe.techFields = setFieldValue(fe.techFields, "error_boundary", t.ErrorBoundary)
		fe.techFields = setFieldValue(fe.techFields, "bundle_opt", t.BundleOptimization)
	}

	th := fp.Theme
	if th.DarkMode != "" || th.BorderRadius != "" {
		fe.themeEnabled = true
		fe.themeFields = setFieldValue(fe.themeFields, "dark_mode", th.DarkMode)
		fe.themeFields = setFieldValue(fe.themeFields, "border_radius", th.BorderRadius)
		fe.themeFields = setFieldValue(fe.themeFields, "spacing", th.Spacing)
		fe.themeFields = setFieldValue(fe.themeFields, "elevation", th.Elevation)
		fe.themeFields = setFieldValue(fe.themeFields, "motion", th.Motion)
		fe.themeFields = setFieldValue(fe.themeFields, "vibe", th.Vibe)
		fe.themeFields = setFieldValue(fe.themeFields, "font", th.Font)
		fe.themeFields = restoreMultiSelectValue(fe.themeFields, "colors", th.Colors)
		fe.themeFields = setFieldValue(fe.themeFields, "description", th.Description)
	}

	// Collections stored directly; per-item forms rebuilt lazily on navigation.
	fe.components = fp.Components
	fe.pages = fp.Pages
	fe.assets = fp.Assets

	n := fp.Navigation
	if n.NavType != "" {
		fe.navEnabled = true
		fe.navFields = setFieldValue(fe.navFields, "nav_type", n.NavType)
		boolStr := func(b bool) string {
			if b {
				return "true"
			}
			return "false"
		}
		fe.navFields = setFieldValue(fe.navFields, "breadcrumbs", boolStr(n.Breadcrumbs))
		fe.navFields = setFieldValue(fe.navFields, "auth_aware", boolStr(n.AuthAware))
	}

	i := fp.I18n
	if i.Enabled != "" || i.DefaultLocale != "" {
		fe.i18nEnabled = true
		fe.i18nFields = setFieldValue(fe.i18nFields, "enabled", i.Enabled)
		fe.i18nFields = setFieldValue(fe.i18nFields, "default_locale", i.DefaultLocale)
		fe.i18nFields = restoreMultiSelectValue(fe.i18nFields, "supported_locales", i.SupportedLocales)
		fe.i18nFields = setFieldValue(fe.i18nFields, "translation_strategy", i.TranslationStrategy)
		fe.i18nFields = setFieldValue(fe.i18nFields, "timezone_handling", i.TimezoneHandling)
	}

	a := fp.A11ySEO
	if a.WCAGLevel != "" || a.SEORenderStrategy != "" {
		fe.a11yEnabled = true
		fe.a11yFields = setFieldValue(fe.a11yFields, "wcag_level", a.WCAGLevel)
		fe.a11yFields = setFieldValue(fe.a11yFields, "seo_render_strategy", a.SEORenderStrategy)
		fe.a11yFields = setFieldValue(fe.a11yFields, "sitemap", a.Sitemap)
		fe.a11yFields = setFieldValue(fe.a11yFields, "meta_tag_injection", a.MetaTagInjection)
		fe.a11yFields = setFieldValue(fe.a11yFields, "analytics", a.Analytics)
		fe.a11yFields = setFieldValue(fe.a11yFields, "telemetry", a.Telemetry)
	}

	return fe
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (fe FrontendEditor) Mode() Mode {
	if fe.internalMode == ModeInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (fe FrontendEditor) HintLine() string {
	if fe.internalMode == ModeInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch fe.activeTab {
	case feTabTech:
		if !fe.techEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit", "h/l", "sub-tab")
	case feTabTheme:
		if !fe.themeEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit", "h/l", "sub-tab")
	case feTabNav:
		if !fe.navEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit", "h/l", "sub-tab")
	case feTabI18n:
		if !fe.i18nEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit", "h/l", "sub-tab")
	case feTabA11ySEO:
		if !fe.a11yEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "a/i", "edit", "h/l", "sub-tab")
	case feTabComponents:
		if fe.inCompAction {
			if fe.actionSubView == ceViewList {
				return hintBar("j/k", "navigate", "a", "add action", "d", "delete", "Enter", "edit", "b/Esc", "back")
			}
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
		if fe.compSubView == ceViewForm {
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "A", "actions", "b/Esc", "back")
		}
		return hintBar("j/k", "navigate", "a", "add", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
	case feTabPages:
		if fe.pageSubView == ceViewList {
			return hintBar("j/k", "navigate", "a", "add page", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	case feTabAssets:
		if fe.assetSubView == ceViewList {
			return hintBar("j/k", "navigate", "a", "add asset", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	}
	return ""
}

// ── Update ────────────────────────────────────────────────────────────────────

func (fe FrontendEditor) Update(msg tea.Msg) (FrontendEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		fe.width = wsz.Width
		fe.formInput.Width = wsz.Width - 22
		return fe, nil
	}
	if fe.internalMode == ModeInsert {
		return fe.updateInsert(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return fe, nil
	}

	// Sub-tab switching — blocked while inside a component or action form.
	switch key.String() {
	case "h", "left", "l", "right":
		inCompForm := fe.activeTab == feTabComponents && (fe.compSubView == ceViewForm || fe.inCompAction)
		if !inCompForm {
			// Auto-save any open form before switching tabs.
			switch fe.activeTab {
			case feTabPages:
				if fe.pageSubView == ceViewForm {
					fe.savePageForm()
				}
			case feTabAssets:
				if fe.assetSubView == ceViewForm {
					fe.saveAssetForm()
				}
			}
			fe.activeTab = feTabIdx(NavigateTab(key.String(), int(fe.activeTab), len(feTabLabels)))
		}
		return fe, nil
	}

	switch fe.activeTab {
	case feTabTech:
		return fe.updateTech(key)
	case feTabTheme:
		return fe.updateTheme(key)
	case feTabPages:
		return fe.updatePages(key)
	case feTabComponents:
		return fe.updateComponents(key)
	case feTabNav:
		return fe.updateNav(key)
	case feTabI18n:
		return fe.updateI18n(key)
	case feTabA11ySEO:
		return fe.updateA11ySEO(key)
	case feTabAssets:
		return fe.updateAssets(key)
	}
	return fe, nil
}

func (fe *FrontendEditor) resetIdx() {
	fe.techFormIdx = 0
	fe.themeFormIdx = 0
	fe.navFormIdx = 0
}

func (fe FrontendEditor) updateInsert(msg tea.Msg) (FrontendEditor, tea.Cmd) {
	if fe.inTextArea {
		key, ok := msg.(tea.KeyMsg)
		if ok && key.String() == "esc" {
			fe.saveInput()
			fe.internalMode = ModeNormal
			fe.inTextArea = false
			fe.formTextArea.Blur()
			return fe, nil
		}
		var cmd tea.Cmd
		fe.formTextArea, cmd = fe.formTextArea.Update(msg)
		return fe, cmd
	}
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc", "enter":
			fe.saveInput()
			fe.internalMode = ModeNormal
			fe.formInput.Blur()
			return fe, nil
		case "tab":
			fe.saveInput()
			fe.advanceField(1)
			return fe.tryEnterInsert()
		case "shift+tab":
			fe.saveInput()
			fe.advanceField(-1)
			return fe.tryEnterInsert()
		}
	}
	var cmd tea.Cmd
	fe.formInput, cmd = fe.formInput.Update(msg)
	return fe, cmd
}

func (fe *FrontendEditor) advanceField(delta int) {
	switch fe.activeTab {
	case feTabTech:
		n := len(fe.visibleTechFields())
		if n > 0 {
			fe.techFormIdx = (fe.techFormIdx + delta + n) % n
		}
	case feTabTheme:
		n := len(fe.themeFields)
		if n > 0 {
			fe.themeFormIdx = (fe.themeFormIdx + delta + n) % n
		}
	case feTabComponents:
		if fe.inCompAction && fe.actionSubView == ceViewForm {
			if delta > 0 {
				fe.actionFormIdx = nextActionFormIdx(fe.actionForm, fe.actionFormIdx)
			} else {
				fe.actionFormIdx = prevActionFormIdx(fe.actionForm, fe.actionFormIdx)
			}
		} else if fe.compSubView == ceViewForm {
			n := len(fe.compForm)
			if n > 0 {
				fe.compFormIdx = (fe.compFormIdx + delta + n) % n
			}
		}
	case feTabPages:
		if fe.pageSubView == ceViewForm {
			n := len(fe.pageForm)
			if n > 0 {
				fe.pageFormIdx = (fe.pageFormIdx + delta + n) % n
			}
		}
	case feTabNav:
		n := len(fe.navFields)
		if n > 0 {
			fe.navFormIdx = (fe.navFormIdx + delta + n) % n
		}
	case feTabI18n:
		n := len(fe.i18nFields)
		if n > 0 {
			fe.i18nFormIdx = (fe.i18nFormIdx + delta + n) % n
		}
	case feTabA11ySEO:
		n := len(fe.a11yFields)
		if n > 0 {
			fe.a11yFormIdx = (fe.a11yFormIdx + delta + n) % n
		}
	case feTabAssets:
		if fe.assetSubView == ceViewForm {
			n := len(fe.assetForm)
			if n > 0 {
				fe.assetFormIdx = (fe.assetFormIdx + delta + n) % n
			}
		}
	}
}

func (fe *FrontendEditor) saveInput() {
	if fe.inTextArea {
		val := fe.formTextArea.Value()
		if fe.activeTab == feTabTheme && fe.themeFormIdx < len(fe.themeFields) {
			fe.themeFields[fe.themeFormIdx].Value = val
		}
		return
	}
	val := fe.formInput.Value()
	switch fe.activeTab {
	case feTabTech:
		visible := fe.visibleTechFields()
		if fe.techFormIdx < len(visible) {
			f := fe.techFieldByKey(visible[fe.techFormIdx].Key)
			if f != nil && f.CanEditAsText() {
				f.SaveTextInput(val)
			}
		}
	case feTabTheme:
		if fe.themeFormIdx < len(fe.themeFields) {
			f := &fe.themeFields[fe.themeFormIdx]
			if f.CanEditAsText() {
				if f.ColorSwatch {
					hex := strings.TrimSpace(val)
					if !strings.HasPrefix(hex, "#") && len(hex) > 0 {
						hex = "#" + hex // auto-prepend # if omitted
					}
					if !f.AddCustomHexColor(hex) {
						f.DeselectCustom() // invalid input: undo Custom selection
					}
				} else {
					f.SaveTextInput(val)
				}
			}
		}
	case feTabComponents:
		if fe.inCompAction && fe.actionSubView == ceViewForm && fe.actionFormIdx < len(fe.actionForm) && fe.actionForm[fe.actionFormIdx].CanEditAsText() {
			fe.actionForm[fe.actionFormIdx].SaveTextInput(val)
			fe.saveActionForm()
			fe.saveActionsToComp()
		} else if fe.compSubView == ceViewForm && fe.compFormIdx < len(fe.compForm) && fe.compForm[fe.compFormIdx].CanEditAsText() {
			fe.compForm[fe.compFormIdx].SaveTextInput(val)
			fe.saveCompForm()
		}
	case feTabPages:
		if fe.pageSubView == ceViewForm && fe.pageFormIdx < len(fe.pageForm) && fe.pageForm[fe.pageFormIdx].CanEditAsText() {
			fe.pageForm[fe.pageFormIdx].SaveTextInput(val)
			fe.savePageForm()
		}
	case feTabNav:
		if fe.navFormIdx < len(fe.navFields) && fe.navFields[fe.navFormIdx].CanEditAsText() {
			fe.navFields[fe.navFormIdx].SaveTextInput(val)
		}
	case feTabI18n:
		if fe.i18nFormIdx < len(fe.i18nFields) && fe.i18nFields[fe.i18nFormIdx].CanEditAsText() {
			fe.i18nFields[fe.i18nFormIdx].SaveTextInput(val)
		}
	case feTabA11ySEO:
		if fe.a11yFormIdx < len(fe.a11yFields) && fe.a11yFields[fe.a11yFormIdx].CanEditAsText() {
			fe.a11yFields[fe.a11yFormIdx].SaveTextInput(val)
		}
	case feTabAssets:
		if fe.assetSubView == ceViewForm && fe.assetFormIdx < len(fe.assetForm) && fe.assetForm[fe.assetFormIdx].CanEditAsText() {
			fe.assetForm[fe.assetFormIdx].SaveTextInput(val)
			fe.saveAssetForm()
		}
	}
}

func (fe FrontendEditor) tryEnterInsert() (FrontendEditor, tea.Cmd) {
	n := 0
	switch fe.activeTab {
	case feTabTech:
		n = len(fe.visibleTechFields())
	case feTabTheme:
		n = len(fe.themeFields)
	case feTabComponents:
		if fe.inCompAction && fe.actionSubView == ceViewForm {
			n = len(fe.actionForm)
		} else if fe.compSubView == ceViewForm {
			n = len(fe.compForm)
		}
	case feTabPages:
		if fe.pageSubView == ceViewForm {
			n = len(fe.pageForm)
		}
	case feTabNav:
		n = len(fe.navFields)
	case feTabI18n:
		n = len(fe.i18nFields)
	case feTabA11ySEO:
		n = len(fe.a11yFields)
	case feTabAssets:
		if fe.assetSubView == ceViewForm {
			n = len(fe.assetForm)
		}
	}
	for range n {
		var f *Field
		switch fe.activeTab {
		case feTabTech:
			visible := fe.visibleTechFields()
			if fe.techFormIdx < len(visible) {
				f = fe.techFieldByKey(visible[fe.techFormIdx].Key)
			}
		case feTabTheme:
			if fe.themeFormIdx < len(fe.themeFields) {
				f = &fe.themeFields[fe.themeFormIdx]
			}
		case feTabComponents:
			if fe.inCompAction && fe.actionSubView == ceViewForm && fe.actionFormIdx < len(fe.actionForm) {
				f = &fe.actionForm[fe.actionFormIdx]
			} else if fe.compSubView == ceViewForm && fe.compFormIdx < len(fe.compForm) {
				f = &fe.compForm[fe.compFormIdx]
			}
		case feTabPages:
			if fe.pageSubView == ceViewForm && fe.pageFormIdx < len(fe.pageForm) {
				f = &fe.pageForm[fe.pageFormIdx]
			}
		case feTabNav:
			if fe.navFormIdx < len(fe.navFields) {
				f = &fe.navFields[fe.navFormIdx]
			}
		case feTabI18n:
			if fe.i18nFormIdx < len(fe.i18nFields) {
				f = &fe.i18nFields[fe.i18nFormIdx]
			}
		case feTabA11ySEO:
			if fe.a11yFormIdx < len(fe.a11yFields) {
				f = &fe.a11yFields[fe.a11yFormIdx]
			}
		case feTabAssets:
			if fe.assetSubView == ceViewForm && fe.assetFormIdx < len(fe.assetForm) {
				f = &fe.assetForm[fe.assetFormIdx]
			}
		}
		if f == nil {
			break
		}
		if f.CanEditAsText() {
			fe.internalMode = ModeInsert
			if f.Kind == KindTextArea {
				fe.inTextArea = true
				fe.formTextArea.SetValue(f.Value)
				fe.formTextArea.SetWidth(fe.width - 4)
				return fe, fe.formTextArea.Focus()
			}
			fe.formInput.SetValue(f.TextInputValue())
			fe.formInput.Width = fe.width - 22
			fe.formInput.CursorEnd()
			return fe, fe.formInput.Focus()
		}
		fe.advanceField(1)
	}
	return fe, nil
}

