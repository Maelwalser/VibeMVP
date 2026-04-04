package ui

import "strings"

// ── Meta-tag injection options per framework ─────────────────────────────────

var metaTagByFramework = map[string][]string{
	"React":   {"Manual", "react-helmet", "Framework-native", "None"},
	"Vue":     {"Manual", "@vueuse/head", "Framework-native", "None"},
	"Svelte":  {"Manual", "svelte:head", "Framework-native", "None"},
	"Angular": {"Manual", "Framework-native", "None"},
	"Solid":   {"Manual", "@solidjs/meta", "Framework-native", "None"},
	"Preact":  {"Manual", "react-helmet", "Framework-native", "None"},
	"Lit":     {"Manual", "Framework-native", "None"},
}

// refreshMetaTagOptions rebuilds the Options (and clamps SelIdx/Value) for the
// meta_tag_injection field inside the supplied a11y field slice.
func refreshMetaTagOptions(fields []Field, framework string) []Field {
	opts, ok := metaTagByFramework[framework]
	if !ok {
		opts = []string{"Manual", "Framework-native", "None"}
	}
	updated := make([]Field, len(fields))
	copy(updated, fields)
	for i, f := range updated {
		if f.Key != "meta_tag_injection" {
			continue
		}
		f.Options = opts
		found := false
		for j, o := range opts {
			if o == f.Value {
				f.SelIdx = j
				found = true
				break
			}
		}
		if !found {
			f.SelIdx = len(opts) - 1
			f.Value = opts[len(opts)-1]
		}
		updated[i] = f
		break
	}
	return updated
}

// ── SEO render strategy options per meta-framework ───────────────────────────

var seoRenderByMetaFramework = map[string][]string{
	"Next.js":   {"SSR", "SSG", "ISR", "Prerender", "None"},
	"Nuxt":      {"SSR", "SSG", "ISR", "None"},
	"SvelteKit": {"SSR", "SSG", "Prerender", "None"},
	"Remix":     {"SSR", "None"},
	"Astro":     {"SSG", "SSR", "None"},
	"None":      {"Prerender", "None"},
}

// seoRenderOptions returns the valid seo_render_strategy options given the
// current platform and meta-framework selections.
// Mobile and Desktop platforms do not support server rendering strategies.
func seoRenderOptions(platform, metaFramework string) []string {
	lower := strings.ToLower(platform)
	if strings.Contains(lower, "mobile") || strings.Contains(lower, "desktop") {
		return []string{"None"}
	}
	if opts, ok := seoRenderByMetaFramework[metaFramework]; ok {
		return opts
	}
	// Web platform with unrecognised meta-framework: omit ISR (Next.js-only)
	return []string{"SSR", "SSG", "Prerender", "None"}
}

// refreshSEORenderOptions rebuilds the Options (and clamps SelIdx/Value) for the
// seo_render_strategy field inside the supplied a11y field slice.
func refreshSEORenderOptions(fields []Field, platform, metaFramework string) []Field {
	opts := seoRenderOptions(platform, metaFramework)
	updated := make([]Field, len(fields))
	copy(updated, fields)
	for i, f := range updated {
		if f.Key != "seo_render_strategy" {
			continue
		}
		f.Options = opts
		// Keep current value if still valid; otherwise fall back to last option.
		found := false
		for j, o := range opts {
			if o == f.Value {
				f.SelIdx = j
				found = true
				break
			}
		}
		if !found {
			f.SelIdx = len(opts) - 1
			f.Value = opts[len(opts)-1]
		}
		updated[i] = f
		break
	}
	return updated
}

// ── framework options per language/platform ───────────────────────────────────

var frontendFrameworksByLang = map[string][]string{
	"TypeScript": {"React", "Vue", "Svelte", "Angular", "Solid", "Qwik", "HTMX"},
	"JavaScript": {"React", "Vue", "Svelte", "Angular", "Solid", "Qwik", "HTMX"},
	"Dart":       {"Flutter"},
	"Kotlin":     {"Jetpack Compose", "KMP (Compose Multiplatform)"},
	"Swift":      {"SwiftUI", "UIKit"},
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
			Key: "language_version", Label: "lang version  ", Kind: KindSelect,
			Options: langVersions["TypeScript"],
			Value:   langVersions["TypeScript"][0],
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
			Key: "framework_version", Label: "fw version    ", Kind: KindSelect,
			Options: compatibleFrameworkVersions("TypeScript", langVersions["TypeScript"][0], "React"),
			Value:   compatibleFrameworkVersions("TypeScript", langVersions["TypeScript"][0], "React")[0],
		},
		{
			Key: "meta_framework", Label: "meta_framework", Kind: KindSelect,
			Options: frontendMetaframeworksByFramework["React"],
			Value:   "None", SelIdx: 3,
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
				"tRPC client", "gRPC-web client", "Connect client",
				"RTK Query", "Native fetch",
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
				"Custom",
			},
			Value: "Professional",
		},
		{
			Key: "font", Label: "font          ", Kind: KindSelect,
			Options: []string{
				"Inter", "Roboto", "Open Sans", "Lato", "Poppins", "Nunito",
				"Source Sans Pro", "Raleway", "Montserrat", "Playfair Display",
				"Merriweather", "Fira Code", "JetBrains Mono", "System default", "Custom",
			},
			Value: "Inter",
		},
		{
			Key: "colors", Label: "colors        ", Kind: KindMultiSelect,
			ColorSwatch: true,
			Options:     themeColorPalette,
		},
		{Key: "description", Label: "description   ", Kind: KindTextArea},
	}
}

func defaultPageFormFields(authRoleOptions, linkedPageOptions, assetNameOptions, componentNameOptions []string) []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "route", Label: "route         ", Kind: KindText},
		{
			Key: "purpose", Label: "purpose       ", Kind: KindSelect,
			Options: []string{
				"Landing/Marketing", "Dashboard/Overview", "List/Index",
				"Detail/View", "Create/Form", "Edit/Form",
				"Auth/Login", "Settings/Profile", "Error/404", "Admin", "Other",
			},
			Value: "Other", SelIdx: 10,
		},
		{
			Key: "auth_required", Label: "auth_required ", Kind: KindSelect,
			Options: OptionsOffOn, Value: "false",
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
			Value:   placeholderFor(authRoleOptions, "(no auth roles configured)"),
		},
		{
			Key: "linked_pages", Label: "linked_pages  ", Kind: KindMultiSelect,
			Options: linkedPageOptions,
			Value:   placeholderFor(linkedPageOptions, "(no other pages)"),
		},
		{
			Key: "assets", Label: "assets        ", Kind: KindMultiSelect,
			Options: assetNameOptions,
			Value:   placeholderFor(assetNameOptions, "(no assets configured)"),
		},
		{
			Key: "component_refs", Label: "components    ", Kind: KindMultiSelect,
			Options: componentNameOptions,
			Value:   placeholderFor(componentNameOptions, "(no components defined)"),
		},
	}
}

func defaultComponentFormFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{
			Key: "comp_type", Label: "comp_type     ", Kind: KindSelect,
			Options: []string{"Form", "Table", "Card", "List", "Chart", "Modal", "Button", "Navigation", "Custom"},
			Value:   "Form",
		},
		{Key: "description", Label: "description   ", Kind: KindText},
	}
}


func defaultI18nFields() []Field {
	return []Field{
		{
			Key: "enabled", Label: "enabled       ", Kind: KindSelect,
			Options: OptionsOffOn, Value: "false",
		},
		{
			Key: "default_locale", Label: "default_locale", Kind: KindSelect,
			Options: []string{
				"en", "en-US", "en-GB", "en-AU", "en-CA",
				"fr", "fr-FR", "fr-CA", "de", "de-DE", "de-AT",
				"es", "es-ES", "es-MX", "es-AR", "pt", "pt-BR", "pt-PT",
				"it", "nl", "nl-NL", "pl", "ru", "ja", "zh", "zh-CN", "zh-TW",
				"ko", "ar", "hi", "tr", "sv", "da", "fi", "nb", "cs", "hu",
				"ro", "vi", "th", "id", "ms", "uk", "he",
			},
			Value: "en",
		},
		{
			Key: "supported_locales", Label: "locales       ", Kind: KindMultiSelect,
			Options: []string{
				"en", "en-US", "en-GB", "en-AU", "en-CA",
				"fr", "fr-FR", "fr-CA", "de", "de-DE", "de-AT",
				"es", "es-ES", "es-MX", "es-AR", "pt", "pt-BR", "pt-PT",
				"it", "nl", "nl-NL", "pl", "ru", "ja", "zh", "zh-CN", "zh-TW",
				"ko", "ar", "hi", "tr", "sv", "da", "fi", "nb", "cs", "hu",
				"ro", "vi", "th", "id", "ms", "uk", "he",
			},
		},
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
			Options: OptionsOffOn, Value: "false",
		},
		{
			Key: "meta_tag_injection", Label: "meta_tags     ", Kind: KindSelect,
			Options: []string{"Manual", "Framework-native", "None"},
			Value:   "None", SelIdx: 2,
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
			Options: OptionsOffOn, Value: "false",
		},
		{
			Key: "auth_aware", Label: "auth_aware    ", Kind: KindSelect,
			Options: OptionsOffOn, Value: "true", SelIdx: 1,
		},
	}
}


// themeColorPalette is the curated set of hex colors offered in the Theme → colors picker.
// Colors are grouped by hue family for easy visual scanning in the dropdown.
var themeColorPalette = []string{
	// Custom hex entry (always first)
	"Custom",
	// Neutrals
	"#ffffff", "#f8fafc", "#e2e8f0", "#94a3b8", "#64748b",
	"#334155", "#1e293b", "#0f172a", "#000000",
	// Blues
	"#bfdbfe", "#60a5fa", "#3b82f6", "#1d4ed8",
	// Greens
	"#6ee7b7", "#10b981", "#059669", "#047857",
	// Reds
	"#f87171", "#ef4444", "#b91c1c",
	// Ambers
	"#fbbf24", "#f59e0b", "#d97706",
	// Purples
	"#a78bfa", "#7c3aed", "#4c1d95",
	// Pinks
	"#f472b6", "#ec4899", "#be185d",
	// Cyans/Teals
	"#22d3ee", "#06b6d4", "#0e7490",
}

// ── compatibility maps ────────────────────────────────────────────────────────

var frontendMetaframeworksByFramework = map[string][]string{
	"React":                       {"Next.js", "Remix", "Astro", "None"},
	"Vue":                         {"Nuxt", "Astro", "None"},
	"Svelte":                      {"SvelteKit", "Astro", "None"},
	"Angular":                     {"Analog", "None"},
	"Solid":                       {"SolidStart", "Astro", "None"},
	"Qwik":                        {"Qwik City", "None"},
	"HTMX":                        {"None"},
	"Flutter":                     {"None"},
	"Jetpack Compose":             {"None"},
	"KMP (Compose Multiplatform)": {"None"},
	"SwiftUI":                     {"None"},
	"UIKit":                       {"None"},
}

var feComponentLibByFramework = map[string][]string{
	"React":                       {"shadcn/ui", "Radix", "Material UI", "Ant Design", "Headless UI", "DaisyUI", "None", "Custom"},
	"Vue":                         {"Material UI", "None", "Custom"},
	"Angular":                     {"Material UI", "None", "Custom"},
	"Svelte":                      {"None", "Custom"},
	"Solid":                       {"None", "Custom"},
	"Qwik":                        {"None", "Custom"},
	"HTMX":                        {"None", "Custom"},
	"Flutter":                     {"None", "Custom"},
	"Jetpack Compose":             {"None", "Custom"},
	"KMP (Compose Multiplatform)": {"None", "Custom"},
	"SwiftUI":                     {"None", "Custom"},
	"UIKit":                       {"None", "Custom"},
}

var feStateMgmtByFramework = map[string][]string{
	"React":                       {"React Context", "Zustand", "Redux Toolkit", "Jotai", "None"},
	"Vue":                         {"Pinia", "None"},
	"Svelte":                      {"Svelte stores", "None"},
	"Angular":                     {"Signals", "None"},
	"Solid":                       {"Signals", "None"},
	"Qwik":                        {"Signals", "None"},
	"HTMX":                        {"None"},
	"Flutter":                     {"None"},
	"Jetpack Compose":             {"None"},
	"KMP (Compose Multiplatform)": {"None"},
	"SwiftUI":                     {"None"},
	"UIKit":                       {"None"},
}

// feDataFetchingByFramework defines the maximum set of data-fetching options
// each framework supports. Protocol-based filtering narrows this further at
// runtime via dataFetchingForContext.
var feDataFetchingByFramework = map[string][]string{
	"React":                       {"TanStack Query", "SWR", "Apollo Client", "tRPC client", "gRPC-web client", "Connect client", "RTK Query", "Native fetch"},
	"Vue":                         {"TanStack Query", "Apollo Client", "gRPC-web client", "Connect client", "Native fetch"},
	"Svelte":                      {"TanStack Query", "SWR", "gRPC-web client", "Native fetch"},
	"Angular":                     {"Apollo Client", "gRPC-web client", "Connect client", "Native fetch"},
	"Solid":                       {"TanStack Query", "Native fetch"},
	"Qwik":                        {"Native fetch"},
	"HTMX":                        {"Native fetch"},
	"Flutter":                     {"Native fetch"},
	"Jetpack Compose":             {"Native fetch"},
	"KMP (Compose Multiplatform)": {"Native fetch"},
	"SwiftUI":                     {"Native fetch"},
	"UIKit":                       {"Native fetch"},
}

// dataFetchingForContext filters the framework's maximum data-fetching options
// down to those relevant given the backend protocols and service frameworks.
// When no backend context is configured every framework-supported option is shown.
func dataFetchingForContext(framework string, backendProtocols, backendSvcFrameworks []string) []string {
	allOpts, ok := feDataFetchingByFramework[framework]
	if !ok {
		return []string{"Native fetch"}
	}
	// No backend context yet — return the full framework list.
	if len(backendProtocols) == 0 && len(backendSvcFrameworks) == 0 {
		return allOpts
	}

	hasProtocol := func(needle string) bool {
		for _, p := range backendProtocols {
			if strings.EqualFold(p, needle) {
				return true
			}
		}
		return false
	}
	hasFramework := func(needle string) bool {
		for _, fw := range backendSvcFrameworks {
			if strings.EqualFold(fw, needle) {
				return true
			}
		}
		return false
	}

	// Determine which protocol-specific tools to include.
	wantTRPC := hasProtocol("trpc") || hasFramework("trpc")
	wantGraphQL := hasProtocol("graphql")
	wantREST := hasProtocol("rest (http)") || hasProtocol("rest")
	wantGRPC := hasProtocol("grpc")

	// If none of the above are detected, treat as REST (safe default).
	if !wantTRPC && !wantGraphQL && !wantREST && !wantGRPC {
		wantREST = true
	}

	// Build the allowed set from the framework's maximum list, preserving order.
	allowed := make(map[string]bool)
	if wantREST {
		allowed["TanStack Query"] = true
		allowed["SWR"] = true
		allowed["RTK Query"] = true
	}
	if wantGraphQL {
		allowed["Apollo Client"] = true
	}
	if wantTRPC {
		allowed["tRPC client"] = true
	}
	if wantGRPC {
		allowed["gRPC-web client"] = true
		allowed["Connect client"] = true
	}
	// "Native fetch" is always available.
	allowed["Native fetch"] = true

	var filtered []string
	for _, opt := range allOpts {
		if allowed[opt] {
			filtered = append(filtered, opt)
		}
	}
	if len(filtered) == 0 {
		return []string{"Native fetch"}
	}
	return filtered
}

var feFormHandlingByFramework = map[string][]string{
	"React":                       {"React Hook Form", "Formik", "Zod + native", "None"},
	"Vue":                         {"Vee-Validate", "Zod + native", "None"},
	"Svelte":                      {"Zod + native", "None"},
	"Angular":                     {"Zod + native", "None"},
	"Solid":                       {"Zod + native", "None"},
	"Qwik":                        {"Zod + native", "None"},
	"HTMX":                        {"None"},
	"Flutter":                     {"None"},
	"Jetpack Compose":             {"None"},
	"KMP (Compose Multiplatform)": {"None"},
	"SwiftUI":                     {"None"},
	"UIKit":                       {"None"},
}

var feStylingByLanguage = map[string][]string{
	"TypeScript": {"Tailwind CSS", "CSS Modules", "Styled Components", "Sass/SCSS", "Vanilla CSS", "UnoCSS"},
	"JavaScript": {"Tailwind CSS", "CSS Modules", "Styled Components", "Sass/SCSS", "Vanilla CSS", "UnoCSS"},
	"Dart":       {"None", "Custom"},
	"Kotlin":     {"None", "Custom"},
	"Swift":      {"None", "Custom"},
}

var feValidationByLanguage = map[string][]string{
	"TypeScript": {"Zod", "Yup", "Valibot", "Joi", "Class-validator", "None"},
	"JavaScript": {"Zod", "Yup", "Valibot", "Joi", "None"},
	"Dart":       {"None"},
	"Kotlin":     {"None"},
	"Swift":      {"None"},
}

var fePkgManagerByLanguage = map[string][]string{
	"TypeScript": {"npm", "yarn", "pnpm", "bun"},
	"JavaScript": {"npm", "yarn", "pnpm", "bun"},
	"Dart":       {"pub"},
	"Kotlin":     {"Gradle"},
	"Swift":      {"SwiftPM"},
}

var feErrorBoundaryByFramework = map[string][]string{
	"React":                       {"React Error Boundary", "Global try-catch", "Framework default", "Custom"},
	"Vue":                         {"Global try-catch", "Framework default", "Custom"},
	"Angular":                     {"Global try-catch", "Framework default", "Custom"},
	"Svelte":                      {"Global try-catch", "Framework default", "Custom"},
	"Solid":                       {"Global try-catch", "Framework default", "Custom"},
	"Qwik":                        {"Global try-catch", "Framework default", "Custom"},
	"HTMX":                        {"Global try-catch", "Custom"},
	"Flutter":                     {"Framework default", "Custom"},
	"Jetpack Compose":             {"Framework default", "Custom"},
	"KMP (Compose Multiplatform)": {"Framework default", "Custom"},
	"SwiftUI":                     {"Framework default", "Custom"},
	"UIKit":                       {"Framework default", "Custom"},
}

var feTestingByLanguage = map[string][]string{
	"TypeScript": {"Vitest", "Jest", "Testing Library", "Storybook", "None"},
	"JavaScript": {"Vitest", "Jest", "Testing Library", "Storybook", "None"},
	"Dart":       {"None"},
	"Kotlin":     {"None"},
	"Swift":      {"None"},
}

var feLinterByLanguage = map[string][]string{
	"TypeScript": {"ESLint + Prettier", "Biome", "oxlint", "Stylelint", "Custom", "None"},
	"JavaScript": {"ESLint + Prettier", "Biome", "oxlint", "Stylelint", "Custom", "None"},
	"Dart":       {"Custom", "None"},
	"Kotlin":     {"Custom", "None"},
	"Swift":      {"Custom", "None"},
}

var fePwaSupportByPlatform = map[string][]string{
	"Web (SPA)":               {"None", "Basic (manifest + service worker)", "Full offline", "Push notifications"},
	"Web (SSR/SSG)":           {"None", "Basic (manifest + service worker)", "Full offline", "Push notifications"},
	"Mobile (cross-platform)": {"None"},
	"Mobile (native)":         {"None"},
	"Desktop":                 {"None"},
}

var feBundleOptByLanguage = map[string][]string{
	"TypeScript": {"Code splitting (route-based)", "Dynamic imports", "Tree shaking only", "None"},
	"JavaScript": {"Code splitting (route-based)", "Dynamic imports", "Tree shaking only", "None"},
	"Dart":       {"None"},
	"Kotlin":     {"None"},
	"Swift":      {"None"},
}

var i18nLibByFramework = map[string][]string{
	"React":                       {"react-i18next", "next-intl", "LinguiJS", "i18next", "Custom", "None"},
	"Vue":                         {"vue-i18n", "i18next", "Custom", "None"},
	"Svelte":                      {"svelte-i18n", "i18next", "Custom", "None"},
	"Angular":                     {"@angular/localize", "ngx-translate", "Custom", "None"},
	"Solid":                       {"i18next", "Custom", "None"},
	"Qwik":                        {"i18next", "Custom", "None"},
	"HTMX":                        {"i18next", "Custom", "None"},
	"Flutter":                     {"flutter_localizations", "Custom", "None"},
	"Jetpack Compose":             {"Android Localization", "Custom", "None"},
	"KMP (Compose Multiplatform)": {"Lyricist", "Custom", "None"},
	"SwiftUI":                     {"Swift Localization", "Custom", "None"},
	"UIKit":                       {"Swift Localization", "Custom", "None"},
}

var feImageOptByPlatform = map[string][]string{
	"Web (SPA)":               {"Next/Image (built-in)", "Cloudinary", "Imgix", "Sharp (self-hosted)", "CDN transform", "None"},
	"Web (SSR/SSG)":           {"Next/Image (built-in)", "Cloudinary", "Imgix", "Sharp (self-hosted)", "CDN transform", "None"},
	"Mobile (cross-platform)": {"None"},
	"Mobile (native)":         {"None"},
	"Desktop":                 {"None"},
}

// webOnlyTechFields is the set of tech field keys that are only meaningful for
// web platforms (SPA / SSR/SSG). They are hidden for mobile and desktop targets.
var webOnlyTechFields = map[string]bool{
	"meta_framework": true,
	"styling":        true,
	"component_lib":  true,
	"pwa_support":    true,
	"image_opt":      true,
	"bundle_opt":     true,
}

// visibleTechFields returns the subset of techFields that are relevant to the
// currently selected language, framework, and platform.
//
// Two rules are applied in order:
//  1. Web-only fields (styling, component_lib, pwa_support, image_opt,
//     bundle_opt, meta_framework) are hidden for mobile/desktop platforms.
//  2. Any remaining field whose options list has been narrowed to exactly
//     ["None"] by updateFEDependentOptions is hidden — it carries no
//     information and would confuse the realization agent.
func (fe FrontendEditor) visibleTechFields() []Field {
	platform := fieldGet(fe.techFields, "platform")
	isWeb := platform == "Web (SPA)" || platform == "Web (SSR/SSG)"
	var visible []Field
	for _, f := range fe.techFields {
		if webOnlyTechFields[f.Key] && !isWeb {
			continue
		}
		if len(f.Options) == 1 && f.Options[0] == "None" {
			continue
		}
		visible = append(visible, f)
	}
	return visible
}

// techFieldByKey returns a pointer to the tech field with the given key in the
// authoritative techFields slice (not the visible projection).
func (fe *FrontendEditor) techFieldByKey(key string) *Field {
	for i := range fe.techFields {
		if fe.techFields[i].Key == key {
			return &fe.techFields[i]
		}
	}
	return nil
}

// ── Runtime field population ──────────────────────────────────────────────────

// setTechFieldOptions updates a tech field's options, preserving the current
// value when it is still valid, or resetting to the first option otherwise.
func (fe *FrontendEditor) setTechFieldOptions(key string, opts []string) {
	for i := range fe.techFields {
		if fe.techFields[i].Key != key {
			continue
		}
		current := fe.techFields[i].Value
		fe.techFields[i].Options = opts
		found := false
		for j, opt := range opts {
			if opt == current {
				fe.techFields[i].SelIdx = j
				found = true
				break
			}
		}
		if !found && len(opts) > 0 {
			fe.techFields[i].SelIdx = 0
			fe.techFields[i].Value = opts[0]
		}
		return
	}
}

// setI18nFieldOptions updates an i18n field's options, preserving the current
// value when it is still valid, or resetting to the first option otherwise.
func (fe *FrontendEditor) setI18nFieldOptions(key string, opts []string) {
	for i := range fe.i18nFields {
		if fe.i18nFields[i].Key != key {
			continue
		}
		current := fe.i18nFields[i].Value
		fe.i18nFields[i].Options = opts
		found := false
		for j, opt := range opts {
			if opt == current {
				fe.i18nFields[i].SelIdx = j
				found = true
				break
			}
		}
		if !found && len(opts) > 0 {
			fe.i18nFields[i].SelIdx = 0
			fe.i18nFields[i].Value = opts[0]
		}
		return
	}
}

// updateFEDependentOptions refreshes all tech fields whose valid options depend
// on the currently selected language, platform, or framework.
func (fe *FrontendEditor) updateFEDependentOptions() {
	lang := fieldGet(fe.techFields, "language")
	platform := fieldGet(fe.techFields, "platform")

	// language_version ← language
	if vers, ok := langVersions[lang]; ok {
		fe.setTechFieldOptions("language_version", vers)
	} else {
		fe.setTechFieldOptions("language_version", []string{"latest"})
	}

	// framework ← language
	if opts, ok := frontendFrameworksByLang[lang]; ok {
		fe.setTechFieldOptions("framework", opts)
	} else {
		fe.setTechFieldOptions("framework", []string{"React", "Vue", "Svelte"})
	}

	framework := fieldGet(fe.techFields, "framework")
	langVer := fieldGet(fe.techFields, "language_version")

	// framework_version ← language + language_version + framework
	fe.setTechFieldOptions("framework_version", compatibleFrameworkVersions(lang, langVer, framework))

	// meta_framework ← framework
	if opts, ok := frontendMetaframeworksByFramework[framework]; ok {
		fe.setTechFieldOptions("meta_framework", opts)
	} else {
		fe.setTechFieldOptions("meta_framework", []string{"None"})
	}

	// meta_tag_injection ← framework
	fe.a11yFields = refreshMetaTagOptions(fe.a11yFields, framework)

	// pkg_manager ← language
	if opts, ok := fePkgManagerByLanguage[lang]; ok {
		fe.setTechFieldOptions("pkg_manager", opts)
	}

	// styling ← language
	if opts, ok := feStylingByLanguage[lang]; ok {
		fe.setTechFieldOptions("styling", opts)
	}

	// component_lib ← framework
	if opts, ok := feComponentLibByFramework[framework]; ok {
		fe.setTechFieldOptions("component_lib", opts)
	} else {
		fe.setTechFieldOptions("component_lib", []string{"None", "Custom"})
	}

	// state_mgmt ← framework
	if opts, ok := feStateMgmtByFramework[framework]; ok {
		fe.setTechFieldOptions("state_mgmt", opts)
	} else {
		fe.setTechFieldOptions("state_mgmt", []string{"None"})
	}

	// data_fetching ← framework + backend protocols/frameworks
	fe.setTechFieldOptions("data_fetching", dataFetchingForContext(framework, fe.backendProtocols, fe.backendSvcFrameworks))

	// form_handling ← framework
	if opts, ok := feFormHandlingByFramework[framework]; ok {
		fe.setTechFieldOptions("form_handling", opts)
	} else {
		fe.setTechFieldOptions("form_handling", []string{"None"})
	}

	// validation ← language
	if opts, ok := feValidationByLanguage[lang]; ok {
		fe.setTechFieldOptions("validation", opts)
	} else {
		fe.setTechFieldOptions("validation", []string{"None"})
	}

	// pwa_support ← platform
	if opts, ok := fePwaSupportByPlatform[platform]; ok {
		fe.setTechFieldOptions("pwa_support", opts)
	} else {
		fe.setTechFieldOptions("pwa_support", []string{"None"})
	}

	// image_opt ← platform
	if opts, ok := feImageOptByPlatform[platform]; ok {
		fe.setTechFieldOptions("image_opt", opts)
	} else {
		fe.setTechFieldOptions("image_opt", []string{"None"})
	}

	// error_boundary ← framework
	if opts, ok := feErrorBoundaryByFramework[framework]; ok {
		fe.setTechFieldOptions("error_boundary", opts)
	} else {
		fe.setTechFieldOptions("error_boundary", []string{"Framework default", "Custom"})
	}

	// bundle_opt ← language
	if opts, ok := feBundleOptByLanguage[lang]; ok {
		fe.setTechFieldOptions("bundle_opt", opts)
	} else {
		fe.setTechFieldOptions("bundle_opt", []string{"None"})
	}

	// realtime ← backend comm-link protocols / frameworks
	hasWS, hasSSE := false, false
	for _, p := range fe.backendProtocols {
		if strings.Contains(p, "WebSocket") {
			hasWS = true
		}
		if strings.Contains(p, "SSE") {
			hasSSE = true
		}
	}
	for _, fw := range fe.backendSvcFrameworks {
		// tRPC supports subscriptions via WebSocket
		if strings.EqualFold(fw, "tRPC") {
			hasWS = true
		}
	}
	if hasWS {
		fe.setTechFieldOptions("realtime", []string{"WebSocket", "SSE", "Polling", "None"})
	} else if hasSSE {
		fe.setTechFieldOptions("realtime", []string{"SSE", "WebSocket", "Polling", "None"})
	}
	// else: keep static options with "None" default

	// translation_strategy ← framework
	if opts, ok := i18nLibByFramework[framework]; ok {
		fe.setI18nFieldOptions("translation_strategy", opts)
	} else {
		fe.setI18nFieldOptions("translation_strategy", []string{"i18next", "Custom", "None"})
	}

	// Clamp techFormIdx to visible field count so a platform switch from web to
	// mobile/desktop never leaves the cursor on a now-hidden field index.
	if n := len(fe.visibleTechFields()); fe.techFormIdx >= n && n > 0 {
		fe.techFormIdx = n - 1
	}

}

