package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-mvp/internal/manifest"
)

// ── sub-tabs ──────────────────────────────────────────────────────────────────

type feTabIdx int

const (
	feTabTech feTabIdx = iota
	feTabTheme
	feTabPages
	feTabNav
	feTabI18n
	feTabA11ySEO
)

var feTabLabels = []string{"TECHNOLOGIES", "THEMING", "PAGES", "NAVIGATION", "I18N", "A11Y/SEO"}

// ── mode ──────────────────────────────────────────────────────────────────────

type feMode int

const (
	feNormal feMode = iota
	feInsert
)

// ── framework options per language/platform ───────────────────────────────────

var frontendFrameworksByLang = map[string][]string{
	"TypeScript": {"React", "Vue", "Svelte", "Angular", "Solid", "Qwik", "HTMX"},
	"JavaScript": {"React", "Vue", "Svelte", "Angular", "Solid", "Qwik", "HTMX"},
	"Dart":       {"Flutter"},
	"Kotlin":     {"Jetpack Compose", "KMP (Compose Multiplatform)"},
	"Swift":      {"SwiftUI", "UIKit"},
}

var frontendMetaframeworks = []string{
	"Next.js", "Nuxt", "SvelteKit", "Remix", "Astro", "None",
}

// ── field definitions ─────────────────────────────────────────────────────────

func defaultFETechFields() []Field {
	return []Field{
		{
			Key: "language", Label: "language      ", Kind: KindSelect,
			Options: []string{"TypeScript", "JavaScript", "Dart", "Kotlin", "Swift"},
			Value:   "TypeScript",
		},
		{
			Key: "platform", Label: "platform      ", Kind: KindSelect,
			Options: []string{
				"Web (SPA)", "Web (SSR/SSG)", "Mobile (cross-platform)",
				"Mobile (native)", "Desktop",
			},
			Value: "Web (SPA)",
		},
		{
			Key: "framework", Label: "framework     ", Kind: KindSelect,
			Options: frontendFrameworksByLang["TypeScript"],
			Value:   "React",
		},
		{
			Key: "meta_framework", Label: "meta_framework", Kind: KindSelect,
			Options: frontendMetaframeworks,
			Value:   "None", SelIdx: 5,
		},
		{
			Key: "pkg_manager", Label: "pkg_manager   ", Kind: KindSelect,
			Options: []string{"npm", "yarn", "pnpm", "bun"},
			Value:   "pnpm", SelIdx: 2,
		},
		{
			Key: "styling", Label: "styling       ", Kind: KindSelect,
			Options: []string{
				"Tailwind CSS", "CSS Modules", "Styled Components",
				"Sass/SCSS", "Vanilla CSS", "UnoCSS",
			},
			Value: "Tailwind CSS",
		},
		{
			Key: "component_lib", Label: "component_lib ", Kind: KindSelect,
			Options: []string{
				"shadcn/ui", "Radix", "Material UI", "Ant Design",
				"Headless UI", "DaisyUI", "None", "Custom",
			},
			Value: "shadcn/ui",
		},
		{
			Key: "state_mgmt", Label: "state_mgmt    ", Kind: KindSelect,
			Options: []string{
				"React Context", "Zustand", "Redux Toolkit", "Jotai",
				"Pinia", "Svelte stores", "Signals", "None",
			},
			Value: "Zustand", SelIdx: 1,
		},
		{
			Key: "data_fetching", Label: "data_fetching ", Kind: KindSelect,
			Options: []string{
				"TanStack Query", "SWR", "Apollo Client",
				"tRPC client", "RTK Query", "Native fetch",
			},
			Value: "TanStack Query",
		},
		{
			Key: "form_handling", Label: "form_handling ", Kind: KindSelect,
			Options: []string{"React Hook Form", "Formik", "Zod + native", "Vee-Validate", "None"},
			Value:   "React Hook Form",
		},
		{
			Key: "validation", Label: "validation    ", Kind: KindSelect,
			Options: []string{"Zod", "Yup", "Valibot", "Joi", "Class-validator", "None"},
			Value:   "Zod",
		},
		{
			Key: "pwa_support", Label: "PWA Support   ", Kind: KindSelect,
			Options: []string{"None", "Basic (manifest + service worker)", "Full offline", "Push notifications"},
			Value:   "None",
		},
		{
			Key: "realtime", Label: "Real-time     ", Kind: KindSelect,
			Options: []string{"WebSocket", "SSE", "Polling", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "image_opt", Label: "Image Optim.  ", Kind: KindSelect,
			Options: []string{"Next/Image (built-in)", "Cloudinary", "Imgix", "Sharp (self-hosted)", "CDN transform", "None"},
			Value:   "None", SelIdx: 5,
		},
		{
			Key: "auth_flow", Label: "Auth Flow     ", Kind: KindSelect,
			Options: []string{"Redirect (OAuth/OIDC)", "Modal login", "Magic link", "Passwordless", "Social only"},
			Value:   "Redirect (OAuth/OIDC)",
		},
		{
			Key: "error_boundary", Label: "Error Boundary", Kind: KindSelect,
			Options: []string{"React Error Boundary", "Global try-catch", "Framework default", "Custom"},
			Value:   "Framework default", SelIdx: 2,
		},
		{
			Key: "bundle_opt", Label: "Bundle Optim. ", Kind: KindSelect,
			Options: []string{"Code splitting (route-based)", "Dynamic imports", "Tree shaking only", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "fe_testing", Label: "FE Testing    ", Kind: KindSelect,
			Options: []string{"Vitest", "Jest", "Testing Library", "Storybook", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "fe_linter", Label: "Linter        ", Kind: KindSelect,
			Options: []string{"ESLint + Prettier", "Biome", "oxlint", "Stylelint", "Custom", "None"},
			Value:   "None", SelIdx: 5,
		},
	}
}

func defaultFEThemeFields() []Field {
	return []Field{
		{
			Key: "dark_mode", Label: "dark_mode     ", Kind: KindSelect,
			Options: []string{"None", "Toggle (user preference)", "System preference", "Dark only"},
			Value:   "System preference", SelIdx: 2,
		},
		{
			Key: "border_radius", Label: "border_radius ", Kind: KindSelect,
			Options: []string{"Sharp (0)", "Subtle (4px)", "Rounded (8px)", "Pill (999px)", "Custom"},
			Value:   "Rounded (8px)", SelIdx: 2,
		},
		{
			Key: "spacing", Label: "spacing       ", Kind: KindSelect,
			Options: []string{"Compact (4px base)", "Default (8px base)", "Spacious (12px base)"},
			Value:   "Default (8px base)", SelIdx: 1,
		},
		{
			Key: "elevation", Label: "elevation     ", Kind: KindSelect,
			Options: []string{"Shadows", "Borders", "Both", "Flat"},
			Value:   "Shadows",
		},
		{
			Key: "motion", Label: "motion        ", Kind: KindSelect,
			Options: []string{"None", "Subtle transitions", "Animated (spring/ease)"},
			Value:   "Subtle transitions", SelIdx: 1,
		},
		{
			Key: "vibe", Label: "vibe          ", Kind: KindSelect,
			Options: []string{
				"Professional", "Playful", "Minimal", "Bold",
				"Elegant", "Technical", "Creative", "Friendly", "Serious", "Modern",
			},
			Value: "Professional",
		},
		{Key: "colors", Label: "colors        ", Kind: KindText},
		{Key: "description", Label: "description   ", Kind: KindTextArea},
	}
}

func defaultPageFormFields(authRoleOptions, pageRouteOptions []string) []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "route", Label: "route         ", Kind: KindText},
		{
			Key: "auth_required", Label: "auth_required ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{
			Key: "layout", Label: "layout        ", Kind: KindSelect,
			Options: []string{"Default", "Sidebar", "Full-width", "Blank", "Custom (specify)"},
			Value:   "Default",
		},
		{Key: "description", Label: "description   ", Kind: KindText},
		{Key: "core_actions", Label: "core_actions  ", Kind: KindText},
		{
			Key: "loading", Label: "loading       ", Kind: KindSelect,
			Options: []string{"Skeleton", "Spinner", "Progressive", "Instant (SSR/SSG)"},
			Value:   "Skeleton",
		},
		{
			Key: "error_handling", Label: "error_handling", Kind: KindSelect,
			Options: []string{"Inline", "Toast", "Error boundary / fallback page", "Retry"},
			Value:   "Toast", SelIdx: 1,
		},
		{
			Key: "auth_roles", Label: "auth_roles    ", Kind: KindMultiSelect,
			Options: authRoleOptions,
		},
		{
			Key: "linked_pages", Label: "linked_pages  ", Kind: KindMultiSelect,
			Options: pageRouteOptions,
		},
	}
}

func defaultI18nFields() []Field {
	return []Field{
		{
			Key: "enabled", Label: "enabled       ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{Key: "default_locale", Label: "default_locale", Kind: KindText, Value: "en"},
		{Key: "supported_locales", Label: "locales       ", Kind: KindText, Value: "en"},
		{
			Key: "translation_strategy", Label: "i18n_library  ", Kind: KindSelect,
			Options: []string{"i18next", "next-intl", "react-i18next", "LinguiJS", "vue-i18n", "Custom", "None"},
			Value:   "None", SelIdx: 6,
		},
		{
			Key: "timezone_handling", Label: "timezone      ", Kind: KindSelect,
			Options: []string{"UTC always", "User preference", "Auto-detect", "Manual"},
			Value:   "UTC always",
		},
	}
}

func defaultA11ySEOFields() []Field {
	return []Field{
		{
			Key: "wcag_level", Label: "wcag_level    ", Kind: KindSelect,
			Options: []string{"A", "AA", "AAA", "None"},
			Value:   "AA", SelIdx: 1,
		},
		{
			Key: "seo_render_strategy", Label: "seo_rendering ", Kind: KindSelect,
			Options: []string{"SSR", "SSG", "ISR", "Prerender", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "sitemap", Label: "sitemap       ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{
			Key: "meta_tag_injection", Label: "meta_tags     ", Kind: KindSelect,
			Options: []string{"Manual", "Automatic (react-helmet)", "Framework-native", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "analytics", Label: "analytics     ", Kind: KindSelect,
			Options: []string{"PostHog", "Google Analytics 4", "Plausible", "Mixpanel", "Segment", "Custom", "None"},
			Value:   "None", SelIdx: 6,
		},
		{
			Key: "telemetry", Label: "frontend_rum  ", Kind: KindSelect,
			Options: []string{"Sentry", "Datadog RUM", "LogRocket", "New Relic Browser", "Custom", "None"},
			Value:   "None", SelIdx: 5,
		},
	}
}

func defaultNavFields() []Field {
	return []Field{
		{
			Key: "nav_type", Label: "nav_type      ", Kind: KindSelect,
			Options: []string{
				"Top bar", "Sidebar", "Bottom tabs (mobile)",
				"Hamburger menu", "Combined",
			},
			Value: "Top bar",
		},
		{
			Key: "breadcrumbs", Label: "breadcrumbs   ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{
			Key: "auth_aware", Label: "auth_aware    ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "true", SelIdx: 1,
		},
	}
}

// ── FrontendEditor ────────────────────────────────────────────────────────────

// FrontendEditor manages the FRONTEND main-tab.
type FrontendEditor struct {
	activeTab feTabIdx

	// TECHNOLOGIES
	techFields  []Field
	techFormIdx int

	// THEMING
	themeFields  []Field
	themeFormIdx int

	// PAGES
	pages       []manifest.PageDef
	pageSubView ceSubView // reuse ceSubView: ceViewList / ceViewForm
	pageIdx     int
	pageForm    []Field
	pageFormIdx int

	// NAVIGATION
	navFields  []Field
	navFormIdx int

	// I18N
	i18nFields  []Field
	i18nFormIdx int

	// A11Y/SEO
	a11yFields  []Field
	a11yFormIdx int

	// Cross-editor data
	availableAuthRoles []string // from BackendEditor auth roles

	// Dropdown state for KindSelect/KindMultiSelect fields
	ddOpen   bool
	ddOptIdx int

	// Shared
	internalMode feMode
	formInput    textinput.Model
	width        int
}

func newFrontendEditor() FrontendEditor {
	return FrontendEditor{
		techFields:  defaultFETechFields(),
		themeFields: defaultFEThemeFields(),
		navFields:   defaultNavFields(),
		i18nFields:  defaultI18nFields(),
		a11yFields:  defaultA11ySEOFields(),
		formInput:   newFormInput(),
	}
}

// SetAuthRoles sets the available auth role options for page forms.
func (fe *FrontendEditor) SetAuthRoles(roles []string) {
	fe.availableAuthRoles = roles
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

// ── ToManifest ────────────────────────────────────────────────────────────────

func (fe FrontendEditor) ToManifestFrontendPillar() manifest.FrontendPillar {
	return manifest.FrontendPillar{
		Tech: manifest.FrontendTechConfig{
			Language:           fieldGet(fe.techFields, "language"),
			Platform:           fieldGet(fe.techFields, "platform"),
			Framework:          fieldGet(fe.techFields, "framework"),
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
			FrontendTesting:    fieldGet(fe.techFields, "fe_testing"),
			FrontendLinter:     fieldGet(fe.techFields, "fe_linter"),
		},
		Theme: manifest.FrontendTheme{
			DarkMode:     fieldGet(fe.themeFields, "dark_mode"),
			BorderRadius: fieldGet(fe.themeFields, "border_radius"),
			Spacing:      fieldGet(fe.themeFields, "spacing"),
			Elevation:    fieldGet(fe.themeFields, "elevation"),
			Motion:       fieldGet(fe.themeFields, "motion"),
			Vibe:         fieldGet(fe.themeFields, "vibe"),
			Colors:       fieldGet(fe.themeFields, "colors"),
			Description:  fieldGet(fe.themeFields, "description"),
		},
		Pages: fe.pages,
		Navigation: manifest.NavigationConfig{
			NavType:     fieldGet(fe.navFields, "nav_type"),
			Breadcrumbs: fieldGet(fe.navFields, "breadcrumbs") == "true",
			AuthAware:   fieldGet(fe.navFields, "auth_aware") == "true",
		},
		I18n: manifest.I18nConfig{
			Enabled:             fieldGet(fe.i18nFields, "enabled"),
			DefaultLocale:       fieldGet(fe.i18nFields, "default_locale"),
			SupportedLocales:    fieldGet(fe.i18nFields, "supported_locales"),
			TranslationStrategy: fieldGet(fe.i18nFields, "translation_strategy"),
			TimezoneHandling:    fieldGet(fe.i18nFields, "timezone_handling"),
		},
		A11ySEO: manifest.A11ySEOConfig{
			WCAGLevel:         fieldGet(fe.a11yFields, "wcag_level"),
			SEORenderStrategy: fieldGet(fe.a11yFields, "seo_render_strategy"),
			Sitemap:           fieldGet(fe.a11yFields, "sitemap"),
			MetaTagInjection:  fieldGet(fe.a11yFields, "meta_tag_injection"),
			Analytics:         fieldGet(fe.a11yFields, "analytics"),
			Telemetry:         fieldGet(fe.a11yFields, "telemetry"),
		},
		// Legacy compatibility
		Rendering: manifest.RenderingMode(fieldGet(fe.techFields, "platform")),
		Framework: fieldGet(fe.techFields, "framework"),
		Styling:   fieldGet(fe.techFields, "styling"),
	}
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (fe FrontendEditor) Mode() Mode {
	if fe.internalMode == feInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (fe FrontendEditor) HintLine() string {
	if fe.internalMode == feInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch fe.activeTab {
	case feTabTech, feTabTheme, feTabNav, feTabI18n, feTabA11ySEO:
		return hintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "i", "edit text", "h/l", "sub-tab")
	case feTabPages:
		if fe.pageSubView == ceViewList {
			return hintBar("j/k", "navigate", "a", "add page", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	}
	return ""
}

// ── Update ────────────────────────────────────────────────────────────────────

func (fe FrontendEditor) Update(msg tea.Msg) (FrontendEditor, tea.Cmd) {
	if fe.internalMode == feInsert {
		return fe.updateInsert(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return fe, nil
	}

	// Sub-tab switching always available in normal mode
	switch key.String() {
	case "h", "left":
		if fe.activeTab > 0 {
			fe.activeTab--
		}
		return fe, nil
	case "l", "right":
		if int(fe.activeTab) < len(feTabLabels)-1 {
			fe.activeTab++
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
	case feTabNav:
		return fe.updateNav(key)
	case feTabI18n:
		return fe.updateI18n(key)
	case feTabA11ySEO:
		return fe.updateA11ySEO(key)
	}
	return fe, nil
}

func (fe *FrontendEditor) resetIdx() {
	fe.techFormIdx = 0
	fe.themeFormIdx = 0
	fe.navFormIdx = 0
}

func (fe FrontendEditor) updateInsert(msg tea.Msg) (FrontendEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			fe.saveInput()
			fe.internalMode = feNormal
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
		n := len(fe.techFields)
		if n > 0 {
			fe.techFormIdx = (fe.techFormIdx + delta + n) % n
		}
	case feTabTheme:
		n := len(fe.themeFields)
		if n > 0 {
			fe.themeFormIdx = (fe.themeFormIdx + delta + n) % n
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
	}
}

func (fe *FrontendEditor) saveInput() {
	val := fe.formInput.Value()
	switch fe.activeTab {
	case feTabTech:
		if fe.techFormIdx < len(fe.techFields) {
			k := fe.techFields[fe.techFormIdx].Kind
			if k == KindText || k == KindTextArea {
				fe.techFields[fe.techFormIdx].Value = val
			}
		}
	case feTabTheme:
		if fe.themeFormIdx < len(fe.themeFields) {
			k := fe.themeFields[fe.themeFormIdx].Kind
			if k == KindText || k == KindTextArea {
				fe.themeFields[fe.themeFormIdx].Value = val
			}
		}
	case feTabPages:
		if fe.pageSubView == ceViewForm && fe.pageFormIdx < len(fe.pageForm) {
			k := fe.pageForm[fe.pageFormIdx].Kind
			if k == KindText || k == KindTextArea {
				fe.pageForm[fe.pageFormIdx].Value = val
			}
		}
	case feTabNav:
		if fe.navFormIdx < len(fe.navFields) {
			k := fe.navFields[fe.navFormIdx].Kind
			if k == KindText || k == KindTextArea {
				fe.navFields[fe.navFormIdx].Value = val
			}
		}
	case feTabI18n:
		if fe.i18nFormIdx < len(fe.i18nFields) {
			k := fe.i18nFields[fe.i18nFormIdx].Kind
			if k == KindText || k == KindTextArea {
				fe.i18nFields[fe.i18nFormIdx].Value = val
			}
		}
	case feTabA11ySEO:
		if fe.a11yFormIdx < len(fe.a11yFields) {
			k := fe.a11yFields[fe.a11yFormIdx].Kind
			if k == KindText || k == KindTextArea {
				fe.a11yFields[fe.a11yFormIdx].Value = val
			}
		}
	}
}

func (fe FrontendEditor) tryEnterInsert() (FrontendEditor, tea.Cmd) {
	var f *Field
	switch fe.activeTab {
	case feTabTech:
		if fe.techFormIdx < len(fe.techFields) {
			f = &fe.techFields[fe.techFormIdx]
		}
	case feTabTheme:
		if fe.themeFormIdx < len(fe.themeFields) {
			f = &fe.themeFields[fe.themeFormIdx]
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
	}
	if f == nil || (f.Kind != KindText && f.Kind != KindTextArea) {
		return fe, nil
	}
	fe.internalMode = feInsert
	fe.formInput.SetValue(f.Value)
	fe.formInput.Width = fe.width - 22
	fe.formInput.CursorEnd()
	return fe, fe.formInput.Focus()
}

func (fe FrontendEditor) updateTech(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.ddOpen {
		return fe.updateTechDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.techFormIdx < len(fe.techFields)-1 {
			fe.techFormIdx++
		}
	case "k", "up":
		if fe.techFormIdx > 0 {
			fe.techFormIdx--
		}
	case "enter", " ":
		f := &fe.techFields[fe.techFormIdx]
		if f.Kind == KindSelect {
			fe.ddOpen = true
			fe.ddOptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.techFields[fe.techFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
			if f.Key == "language" {
				fe.updateFEFrameworkOptions()
			}
		}
	case "i":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateTechDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.techFormIdx >= len(fe.techFields) {
		fe.ddOpen = false
		return fe, nil
	}
	f := &fe.techFields[fe.techFormIdx]
	switch key.String() {
	case "j", "down":
		if fe.ddOptIdx < len(f.Options)-1 {
			fe.ddOptIdx++
		}
	case "k", "up":
		if fe.ddOptIdx > 0 {
			fe.ddOptIdx--
		}
	case " ", "enter":
		f.SelIdx = fe.ddOptIdx
		if fe.ddOptIdx < len(f.Options) {
			f.Value = f.Options[fe.ddOptIdx]
		}
		fe.ddOpen = false
		if f.Key == "language" {
			fe.updateFEFrameworkOptions()
		}
	case "esc", "b":
		fe.ddOpen = false
	}
	return fe, nil
}

func (fe *FrontendEditor) updateFEFrameworkOptions() {
	lang := fieldGet(fe.techFields, "language")
	opts, ok := frontendFrameworksByLang[lang]
	if !ok {
		opts = []string{"React", "Vue", "Svelte"}
	}
	for i := range fe.techFields {
		if fe.techFields[i].Key == "framework" {
			fe.techFields[i].Options = opts
			fe.techFields[i].SelIdx = 0
			fe.techFields[i].Value = opts[0]
			break
		}
	}
}

func (fe FrontendEditor) updateTheme(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.ddOpen {
		return fe.updateThemeDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.themeFormIdx < len(fe.themeFields)-1 {
			fe.themeFormIdx++
		}
	case "k", "up":
		if fe.themeFormIdx > 0 {
			fe.themeFormIdx--
		}
	case "enter", " ":
		f := &fe.themeFields[fe.themeFormIdx]
		if f.Kind == KindSelect {
			fe.ddOpen = true
			fe.ddOptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.themeFields[fe.themeFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateThemeDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.themeFormIdx >= len(fe.themeFields) {
		fe.ddOpen = false
		return fe, nil
	}
	f := &fe.themeFields[fe.themeFormIdx]
	switch key.String() {
	case "j", "down":
		if fe.ddOptIdx < len(f.Options)-1 {
			fe.ddOptIdx++
		}
	case "k", "up":
		if fe.ddOptIdx > 0 {
			fe.ddOptIdx--
		}
	case " ", "enter":
		f.SelIdx = fe.ddOptIdx
		if fe.ddOptIdx < len(f.Options) {
			f.Value = f.Options[fe.ddOptIdx]
		}
		fe.ddOpen = false
	case "esc", "b":
		fe.ddOpen = false
	}
	return fe, nil
}

func (fe FrontendEditor) updatePages(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.pageSubView == ceViewList {
		return fe.updatePageList(key)
	}
	return fe.updatePageForm(key)
}

func (fe FrontendEditor) updatePageList(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	n := len(fe.pages)
	switch key.String() {
	case "j", "down":
		if n > 0 && fe.pageIdx < n-1 {
			fe.pageIdx++
		}
	case "k", "up":
		if fe.pageIdx > 0 {
			fe.pageIdx--
		}
	case "a":
		fe.pages = append(fe.pages, manifest.PageDef{})
		fe.pageIdx = len(fe.pages) - 1
		fe.pageForm = defaultPageFormFields(fe.availableAuthRoles, fe.pageRoutes())
		fe.pageFormIdx = 0
		fe.pageSubView = ceViewForm
		return fe.tryEnterInsert()
	case "d":
		if n > 0 {
			fe.pages = append(fe.pages[:fe.pageIdx], fe.pages[fe.pageIdx+1:]...)
			if fe.pageIdx > 0 && fe.pageIdx >= len(fe.pages) {
				fe.pageIdx = len(fe.pages) - 1
			}
		}
	case "enter":
		if n > 0 {
			p := fe.pages[fe.pageIdx]
			// Exclude current page's route from linked_pages options
			otherRoutes := make([]string, 0, len(fe.pages))
			for i, pg := range fe.pages {
				if i != fe.pageIdx && pg.Route != "" {
					otherRoutes = append(otherRoutes, pg.Route)
				}
			}
			fe.pageForm = defaultPageFormFields(fe.availableAuthRoles, otherRoutes)
			fe.pageForm = setFieldValue(fe.pageForm, "name", p.Name)
			fe.pageForm = setFieldValue(fe.pageForm, "route", p.Route)
			fe.pageForm = setFieldValue(fe.pageForm, "auth_required", p.AuthRequired)
			if p.Layout != "" {
				fe.pageForm = setFieldValue(fe.pageForm, "layout", p.Layout)
			}
			fe.pageForm = setFieldValue(fe.pageForm, "description", p.Description)
			fe.pageForm = setFieldValue(fe.pageForm, "core_actions", p.CoreActions)
			if p.Loading != "" {
				fe.pageForm = setFieldValue(fe.pageForm, "loading", p.Loading)
			}
			if p.ErrorHandling != "" {
				fe.pageForm = setFieldValue(fe.pageForm, "error_handling", p.ErrorHandling)
			}
			// Restore multiselect for auth_roles
			if p.AuthRoles != "" {
				for i := range fe.pageForm {
					if fe.pageForm[i].Key == "auth_roles" {
						for _, sel := range strings.Split(p.AuthRoles, ", ") {
							for j, opt := range fe.pageForm[i].Options {
								if opt == strings.TrimSpace(sel) {
									fe.pageForm[i].SelectedIdxs = append(fe.pageForm[i].SelectedIdxs, j)
								}
							}
						}
						break
					}
				}
			}
			// Restore multiselect for linked_pages
			if p.LinkedPages != "" {
				for i := range fe.pageForm {
					if fe.pageForm[i].Key == "linked_pages" {
						for _, sel := range strings.Split(p.LinkedPages, ", ") {
							for j, opt := range fe.pageForm[i].Options {
								if opt == strings.TrimSpace(sel) {
									fe.pageForm[i].SelectedIdxs = append(fe.pageForm[i].SelectedIdxs, j)
								}
							}
						}
						break
					}
				}
			}
			fe.pageFormIdx = 0
			fe.pageSubView = ceViewForm
		}
	}
	return fe, nil
}

func (fe FrontendEditor) updatePageForm(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	// Handle dropdown if open
	if fe.ddOpen {
		return fe.updatePageFormDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.pageFormIdx < len(fe.pageForm)-1 {
			fe.pageFormIdx++
		}
	case "k", "up":
		if fe.pageFormIdx > 0 {
			fe.pageFormIdx--
		}
	case "enter", " ":
		f := &fe.pageForm[fe.pageFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			fe.ddOpen = true
			if f.Kind == KindSelect {
				fe.ddOptIdx = f.SelIdx
			} else {
				fe.ddOptIdx = f.DDCursor
			}
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.pageForm[fe.pageFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if fe.pageForm[fe.pageFormIdx].Kind == KindText {
			return fe.tryEnterInsert()
		}
	case "b", "esc":
		fe.savePageForm()
		fe.pageSubView = ceViewList
	}
	return fe, nil
}

func (fe FrontendEditor) updatePageFormDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.pageFormIdx >= len(fe.pageForm) {
		fe.ddOpen = false
		return fe, nil
	}
	f := &fe.pageForm[fe.pageFormIdx]
	switch key.String() {
	case "j", "down":
		if fe.ddOptIdx < len(f.Options)-1 {
			fe.ddOptIdx++
		}
	case "k", "up":
		if fe.ddOptIdx > 0 {
			fe.ddOptIdx--
		}
	case " ":
		if f.Kind == KindMultiSelect {
			f.ToggleMultiSelect(fe.ddOptIdx)
			f.DDCursor = fe.ddOptIdx
		} else if f.Kind == KindSelect {
			f.SelIdx = fe.ddOptIdx
			if fe.ddOptIdx < len(f.Options) {
				f.Value = f.Options[fe.ddOptIdx]
			}
			fe.ddOpen = false
		}
	case "enter":
		if f.Kind == KindMultiSelect {
			f.DDCursor = fe.ddOptIdx
		} else if f.Kind == KindSelect {
			f.SelIdx = fe.ddOptIdx
			if fe.ddOptIdx < len(f.Options) {
				f.Value = f.Options[fe.ddOptIdx]
			}
		}
		fe.ddOpen = false
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = fe.ddOptIdx
		}
		fe.ddOpen = false
	}
	return fe, nil
}

func (fe *FrontendEditor) savePageForm() {
	if fe.pageIdx >= len(fe.pages) {
		return
	}
	p := &fe.pages[fe.pageIdx]
	p.Name = fieldGet(fe.pageForm, "name")
	p.Route = fieldGet(fe.pageForm, "route")
	p.AuthRequired = fieldGet(fe.pageForm, "auth_required")
	p.Layout = fieldGet(fe.pageForm, "layout")
	p.Description = fieldGet(fe.pageForm, "description")
	p.CoreActions = fieldGet(fe.pageForm, "core_actions")
	p.Loading = fieldGet(fe.pageForm, "loading")
	p.ErrorHandling = fieldGet(fe.pageForm, "error_handling")
	p.AuthRoles = fieldGetMulti(fe.pageForm, "auth_roles")
	p.LinkedPages = fieldGetMulti(fe.pageForm, "linked_pages")
}

func (fe FrontendEditor) updateNav(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.ddOpen {
		return fe.updateNavDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.navFormIdx < len(fe.navFields)-1 {
			fe.navFormIdx++
		}
	case "k", "up":
		if fe.navFormIdx > 0 {
			fe.navFormIdx--
		}
	case "enter", " ":
		f := &fe.navFields[fe.navFormIdx]
		if f.Kind == KindSelect {
			fe.ddOpen = true
			fe.ddOptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.navFields[fe.navFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateNavDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.navFormIdx >= len(fe.navFields) {
		fe.ddOpen = false
		return fe, nil
	}
	f := &fe.navFields[fe.navFormIdx]
	switch key.String() {
	case "j", "down":
		if fe.ddOptIdx < len(f.Options)-1 {
			fe.ddOptIdx++
		}
	case "k", "up":
		if fe.ddOptIdx > 0 {
			fe.ddOptIdx--
		}
	case " ", "enter":
		f.SelIdx = fe.ddOptIdx
		if fe.ddOptIdx < len(f.Options) {
			f.Value = f.Options[fe.ddOptIdx]
		}
		fe.ddOpen = false
	case "esc", "b":
		fe.ddOpen = false
	}
	return fe, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (fe FrontendEditor) View(w, h int) string {
	fe.width = w
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Frontend — technologies, theming, pages, and navigation"),
		"",
		renderSubTabBar(feTabLabels, int(fe.activeTab)),
		"",
	)

	switch fe.activeTab {
	case feTabTech:
		lines = append(lines, renderFormFieldsWithDropdown(w, fe.techFields, fe.techFormIdx, fe.internalMode == feInsert, fe.formInput, fe.ddOpen, fe.ddOptIdx)...)
	case feTabTheme:
		lines = append(lines, renderFormFieldsWithDropdown(w, fe.themeFields, fe.themeFormIdx, fe.internalMode == feInsert, fe.formInput, fe.ddOpen, fe.ddOptIdx)...)
	case feTabPages:
		lines = append(lines, fe.viewPages(w)...)
	case feTabNav:
		lines = append(lines, renderFormFieldsWithDropdown(w, fe.navFields, fe.navFormIdx, fe.internalMode == feInsert, fe.formInput, fe.ddOpen, fe.ddOptIdx)...)
	case feTabI18n:
		lines = append(lines, renderFormFieldsWithDropdown(w, fe.i18nFields, fe.i18nFormIdx, fe.internalMode == feInsert, fe.formInput, fe.ddOpen, fe.ddOptIdx)...)
	case feTabA11ySEO:
		lines = append(lines, renderFormFieldsWithDropdown(w, fe.a11yFields, fe.a11yFormIdx, fe.internalMode == feInsert, fe.formInput, fe.ddOpen, fe.ddOptIdx)...)
	}

	return fillTildes(lines, h)
}

func (fe FrontendEditor) updateI18n(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.ddOpen {
		return fe.updateI18nDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.i18nFormIdx < len(fe.i18nFields)-1 {
			fe.i18nFormIdx++
		}
	case "k", "up":
		if fe.i18nFormIdx > 0 {
			fe.i18nFormIdx--
		}
	case "enter", " ":
		f := &fe.i18nFields[fe.i18nFormIdx]
		if f.Kind == KindSelect {
			fe.ddOpen = true
			fe.ddOptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.i18nFields[fe.i18nFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateI18nDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.i18nFormIdx >= len(fe.i18nFields) {
		fe.ddOpen = false
		return fe, nil
	}
	f := &fe.i18nFields[fe.i18nFormIdx]
	switch key.String() {
	case "j", "down":
		if fe.ddOptIdx < len(f.Options)-1 {
			fe.ddOptIdx++
		}
	case "k", "up":
		if fe.ddOptIdx > 0 {
			fe.ddOptIdx--
		}
	case " ", "enter":
		f.SelIdx = fe.ddOptIdx
		if fe.ddOptIdx < len(f.Options) {
			f.Value = f.Options[fe.ddOptIdx]
		}
		fe.ddOpen = false
	case "esc", "b":
		fe.ddOpen = false
	}
	return fe, nil
}

func (fe FrontendEditor) updateA11ySEO(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.ddOpen {
		return fe.updateA11ySEODropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.a11yFormIdx < len(fe.a11yFields)-1 {
			fe.a11yFormIdx++
		}
	case "k", "up":
		if fe.a11yFormIdx > 0 {
			fe.a11yFormIdx--
		}
	case "enter", " ":
		f := &fe.a11yFields[fe.a11yFormIdx]
		if f.Kind == KindSelect {
			fe.ddOpen = true
			fe.ddOptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.a11yFields[fe.a11yFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateA11ySEODropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.a11yFormIdx >= len(fe.a11yFields) {
		fe.ddOpen = false
		return fe, nil
	}
	f := &fe.a11yFields[fe.a11yFormIdx]
	switch key.String() {
	case "j", "down":
		if fe.ddOptIdx < len(f.Options)-1 {
			fe.ddOptIdx++
		}
	case "k", "up":
		if fe.ddOptIdx > 0 {
			fe.ddOptIdx--
		}
	case " ", "enter":
		f.SelIdx = fe.ddOptIdx
		if fe.ddOptIdx < len(f.Options) {
			f.Value = f.Options[fe.ddOptIdx]
		}
		fe.ddOpen = false
	case "esc", "b":
		fe.ddOpen = false
	}
	return fe, nil
}

func (fe FrontendEditor) viewPages(w int) []string {
	switch fe.pageSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Pages — a: add  d: delete  Enter: edit"), "")
		if len(fe.pages) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no pages yet — press 'a' to add)"))
		} else {
			for i, p := range fe.pages {
				name := p.Name
				if name == "" {
					name = fmt.Sprintf("(page #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == fe.pageIdx, "  ▶ ", name, p.Route))
			}
		}
		return lines

	case ceViewForm:
		name := fieldGet(fe.pageForm, "name")
		if name == "" {
			name = "(new page)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFieldsWithDropdown(w, fe.pageForm, fe.pageFormIdx, fe.internalMode == feInsert, fe.formInput, fe.ddOpen, fe.ddOptIdx)...)
		return lines
	}
	return nil
}
