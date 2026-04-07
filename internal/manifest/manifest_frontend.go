package manifest

// ── Frontend tab types ────────────────────────────────────────────────────────

// FrontendTechConfig describes the technology stack choices for the frontend.
type FrontendTechConfig struct {
	Language           string `json:"language,omitempty"`
	LanguageVersion    string `json:"language_version,omitempty"`
	Platform           string `json:"platform,omitempty"`
	Framework          string `json:"framework,omitempty"`
	FrameworkVersion   string `json:"framework_version,omitempty"`
	MetaFramework      string `json:"meta_framework,omitempty"`
	PackageManager     string `json:"package_manager,omitempty"`
	Styling            string `json:"styling,omitempty"`
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
}

// FrontendTheme describes the visual theme settings.
type FrontendTheme struct {
	DarkMode     string `json:"dark_mode,omitempty"`
	BorderRadius string `json:"border_radius,omitempty"`
	Spacing      string `json:"spacing,omitempty"`
	Elevation    string `json:"elevation,omitempty"`
	Motion       string `json:"motion,omitempty"`
	Vibe         string `json:"vibe,omitempty"`
	Font         string `json:"font,omitempty"`
	Colors       string `json:"colors,omitempty"`
	Description  string `json:"description,omitempty"`
}

// ComponentActionDef describes a user interaction action wired to a component.
type ComponentActionDef struct {
	Trigger    string `json:"trigger"`
	ActionType string `json:"action_type"`
	// API request fields (Fetch Data, Submit Form, Download, Upload, Delete, Refresh, Export)
	Endpoint      string `json:"endpoint,omitempty"`
	HttpMethod    string `json:"http_method,omitempty"`
	RequestBody   string `json:"request_body,omitempty"`
	SuccessAction string `json:"success_action,omitempty"`
	ErrorAction   string `json:"error_action,omitempty"`
	// Component targets
	FormTarget  string `json:"form_target,omitempty"`  // Submit Form, Reset Form
	ModalTarget string `json:"modal_target,omitempty"` // Open Modal, Close Modal
	// Navigation
	TargetPage string `json:"target_page,omitempty"`
	// Toast
	ToastMessage string `json:"toast_message,omitempty"`
	ToastType    string `json:"toast_type,omitempty"`
	// Delete confirmation
	ConfirmDialog string `json:"confirm_dialog,omitempty"`
	// State management
	StateKey   string `json:"state_key,omitempty"`
	StateValue string `json:"state_value,omitempty"`
	// Custom
	CustomHandler string `json:"custom_handler,omitempty"`
	Description   string `json:"description,omitempty"`
}

// PageComponentDef describes a UI component within a page.
type PageComponentDef struct {
	Name               string               `json:"name"`
	ComponentType      string               `json:"component_type"`
	ConnectedEndpoints string               `json:"connected_endpoints,omitempty"`
	Actions            []ComponentActionDef `json:"actions,omitempty"`
	Description        string               `json:"description,omitempty"`
}

// PageDef describes a frontend page.
type PageDef struct {
	Name          string `json:"name"`
	Route         string `json:"route,omitempty"`
	AuthRequired  string `json:"auth_required,omitempty"`
	Layout        string `json:"layout,omitempty"`
	Purpose       string `json:"purpose,omitempty"`
	Description   string `json:"description,omitempty"`
	CoreActions   string `json:"core_actions,omitempty"`
	Loading       string `json:"loading,omitempty"`
	ErrorHandling string `json:"error_handling,omitempty"`
	AuthRoles     string `json:"auth_roles,omitempty"`
	LinkedPages   string `json:"linked_pages,omitempty"`   // comma-separated routes of related pages
	Assets        string `json:"assets,omitempty"`         // comma-separated asset names used on this page
	ComponentRefs string `json:"component_refs,omitempty"` // comma-sep names from the component library
}

// NavigationConfig describes frontend navigation settings.
type NavigationConfig struct {
	NavType     string `json:"nav_type,omitempty"`
	Breadcrumbs bool   `json:"breadcrumbs,omitempty"`
	AuthAware   bool   `json:"auth_aware,omitempty"`
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
	AssetType   string     `json:"asset_type"`      // image, icon, font, video, mockup, moodboard
	Format      string     `json:"format"`          // png, jpg, svg, gif, mp4, pdf, figma, sketch, other
	Usage       AssetUsage `json:"usage"`           // project | inspiration
	Pages       string     `json:"pages,omitempty"` // comma-separated page routes this asset is used on
	Description string     `json:"description,omitempty"`
}

// FrontendPillar covers the full frontend configuration.
type FrontendPillar struct {
	Tech       *FrontendTechConfig `json:"tech,omitempty"`
	Theme      *FrontendTheme      `json:"theme,omitempty"`
	Components []PageComponentDef  `json:"components,omitempty"` // shared component library
	Pages      []PageDef           `json:"pages,omitempty"`
	Assets     []AssetDef          `json:"assets,omitempty"`
	Navigation *NavigationConfig   `json:"navigation,omitempty"`
	I18n       *I18nConfig         `json:"i18n,omitempty"`
	A11ySEO    *A11ySEOConfig      `json:"a11y_seo,omitempty"`

	// Legacy fields preserved for backward compatibility.
	Rendering     RenderingMode `json:"rendering,omitempty"`
	Framework     string        `json:"framework,omitempty"`
	ServerState   string        `json:"server_state,omitempty"`
	ClientState   string        `json:"client_state,omitempty"`
	Styling       string        `json:"styling,omitempty"`
	BrowserMatrix string        `json:"browser_matrix,omitempty"`
}
