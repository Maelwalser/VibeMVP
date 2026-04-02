package manifest

// ── Frontend tab types ────────────────────────────────────────────────────────

// FrontendTechConfig describes the technology stack choices for the frontend.
type FrontendTechConfig struct {
	Language           string `json:"language"`
	Platform           string `json:"platform"`
	Framework          string `json:"framework"`
	MetaFramework      string `json:"meta_framework,omitempty"`
	PackageManager     string `json:"package_manager"`
	Styling            string `json:"styling"`
	ComponentLib       string `json:"component_lib,omitempty"`
	StateManagement    string `json:"state_management,omitempty"`
	DataFetching       string `json:"data_fetching,omitempty"`
	FormHandling       string `json:"form_handling,omitempty"`
	Validation         string `json:"validation,omitempty"`
	PWASupport         string `json:"pwa_support,omitempty"`
	RealtimeStrategy   string `json:"realtime_strategy,omitempty"`
	ImageOptimization  string `json:"image_optimization,omitempty"`
	AuthFlowType       string `json:"auth_flow_type,omitempty"`
	ErrorBoundary      string `json:"error_boundary,omitempty"`
	BundleOptimization string `json:"bundle_optimization,omitempty"`
	FrontendTesting    string `json:"frontend_testing,omitempty"`
	FrontendLinter     string `json:"frontend_linter,omitempty"`
}

// FrontendTheme describes the visual theme settings.
type FrontendTheme struct {
	DarkMode     string `json:"dark_mode"`
	BorderRadius string `json:"border_radius"`
	Spacing      string `json:"spacing"`
	Elevation    string `json:"elevation"`
	Motion       string `json:"motion"`
	Vibe         string `json:"vibe,omitempty"`
	Colors       string `json:"colors,omitempty"`
	Description  string `json:"description,omitempty"`
}

// PageDef describes a frontend page.
type PageDef struct {
	Name          string `json:"name"`
	Route         string `json:"route"`
	AuthRequired  string `json:"auth_required"`
	Layout        string `json:"layout"`
	Description   string `json:"description,omitempty"`
	CoreActions   string `json:"core_actions,omitempty"`
	Loading       string `json:"loading"`
	ErrorHandling string `json:"error_handling"`
	AuthRoles     string `json:"auth_roles,omitempty"`
	LinkedPages   string `json:"linked_pages,omitempty"`
}

// NavigationConfig describes frontend navigation settings.
type NavigationConfig struct {
	NavType     string `json:"nav_type"`
	Breadcrumbs bool   `json:"breadcrumbs"`
	AuthAware   bool   `json:"auth_aware"`
}

// I18nConfig describes internationalization and localization settings.
type I18nConfig struct {
	Enabled             string `json:"enabled,omitempty"`
	DefaultLocale       string `json:"default_locale,omitempty"`
	SupportedLocales    string `json:"supported_locales,omitempty"`
	TranslationStrategy string `json:"translation_strategy,omitempty"`
	TimezoneHandling    string `json:"timezone_handling,omitempty"`
}

// A11ySEOConfig describes accessibility and SEO settings.
type A11ySEOConfig struct {
	WCAGLevel         string `json:"wcag_level,omitempty"`
	SEORenderStrategy string `json:"seo_render_strategy,omitempty"`
	Sitemap           string `json:"sitemap,omitempty"`
	MetaTagInjection  string `json:"meta_tag_injection,omitempty"`
	Analytics         string `json:"analytics,omitempty"`
	Telemetry         string `json:"telemetry,omitempty"`
}

// AssetUsage classifies whether an asset is used directly in the project or
// only serves as design inspiration.
type AssetUsage string

const (
	AssetUsageProject     AssetUsage = "project"
	AssetUsageInspiration AssetUsage = "inspiration"
)

// AssetDef describes a single frontend asset entry.
type AssetDef struct {
	Name        string     `json:"name"`
	Path        string     `json:"path"`
	AssetType   string     `json:"asset_type"`          // image, icon, font, video, mockup, moodboard
	Format      string     `json:"format"`              // png, jpg, svg, gif, mp4, pdf, figma, sketch, other
	Usage       AssetUsage `json:"usage"`               // project | inspiration
	Description string     `json:"description,omitempty"`
}

// FrontendPillar covers the full frontend configuration.
type FrontendPillar struct {
	Tech       FrontendTechConfig `json:"tech"`
	Theme      FrontendTheme      `json:"theme"`
	Pages      []PageDef          `json:"pages,omitempty"`
	Assets     []AssetDef         `json:"assets,omitempty"`
	Navigation NavigationConfig   `json:"navigation"`
	I18n       I18nConfig         `json:"i18n,omitempty"`
	A11ySEO    A11ySEOConfig      `json:"a11y_seo,omitempty"`

	// Legacy fields preserved for backward compatibility.
	Rendering     RenderingMode `json:"rendering,omitempty"`
	Framework     string        `json:"framework,omitempty"`
	ServerState   string        `json:"server_state,omitempty"`
	ClientState   string        `json:"client_state,omitempty"`
	Styling       string        `json:"styling,omitempty"`
	BrowserMatrix string        `json:"browser_matrix,omitempty"`
}
