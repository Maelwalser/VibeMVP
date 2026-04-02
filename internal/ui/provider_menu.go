package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-mvp/internal/manifest"
)

// pmFocus tracks which column owns keyboard input.
type pmFocus int

const (
	pmFocusProviders pmFocus = iota
	pmFocusModels
	pmFocusAuth
	pmFocusCredential // API key / OAuth token input step
)

// ProviderSelection holds a confirmed provider/model/auth/credential quad.
type ProviderSelection struct {
	Provider   string
	Model      string
	Version    string
	Auth       string
	Credential string // API key or OAuth token
}

// IsSet reports whether this selection is fully configured.
func (p ProviderSelection) IsSet() bool {
	return p.Provider != "" && p.Model != "" && p.Auth != "" && p.Credential != ""
}

// Short returns a compact display string like "Claude · Sonnet 4.5".
func (p ProviderSelection) Short() string {
	if !p.IsSet() {
		return ""
	}
	model := p.Model
	if p.Version != "" {
		model += " " + p.Version
	}
	return p.Provider + " · " + model
}

// modelTier represents one tier of a provider (e.g. "Sonnet") with its
// concrete version strings (e.g. "4.5", "4.0", "3.5").
type modelTier struct {
	name     string
	versions []string
}

// providerEntry defines an AI provider, its model tiers, and auth methods.
type providerEntry struct {
	label       string
	models      []modelTier
	authMethods []string
}

// ProviderMenu is the centered modal for configuring multiple providers.
// Each provider (Claude, Gemini, etc.) can be independently configured with
// its own model tier, auth method, and credential. The resulting registry is
// used by the realize tab for per-section model assignment.
type ProviderMenu struct {
	// Confirmed provider configurations, keyed by provider label.
	configured map[string]ProviderSelection

	// Provider/model/auth columns
	providers       []providerEntry
	cursor          int     // hovered row in provider list
	modelCursor     int     // hovered row in model list
	authCursor      int     // hovered row in auth list
	focus           pmFocus // column that owns input
	dropdownOpen    bool    // version dropdown visible
	versionCursor   int     // hovered row inside the dropdown
	selectedProv    int     // provider currently being edited (-1 = none)
	selectedModel   int     // -1 = none confirmed in current edit
	selectedVersion int     // -1 = none confirmed in current edit
	selectedAuth    int     // -1 = none confirmed in current edit

	// Credential input step
	credInput textinput.Model
}

func newProviderMenu() ProviderMenu {
	ci := textinput.New()
	ci.Prompt = ""
	ci.Width = pmBoxW - 10
	ci.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg))
	ci.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))

	pm := ProviderMenu{
		configured: make(map[string]ProviderSelection),
		credInput:  ci,
		providers: []providerEntry{
			{
				label: "Claude",
				models: []modelTier{
					{name: "Haiku", versions: []string{"3.5", "3.0"}},
					{name: "Sonnet", versions: []string{"4.5", "4.0", "3.5"}},
					{name: "Opus", versions: []string{"4.0", "3.0"}},
				},
				authMethods: []string{"API Key", "OAuth"},
			},
			{
				label: "ChatGPT",
				models: []modelTier{
					{name: "Mini", versions: []string{"o3-mini", "4o-mini"}},
					{name: "4o", versions: []string{"4o", "4o-2024"}},
					{name: "o1", versions: []string{"o1", "o1-preview"}},
				},
				authMethods: []string{"API Key", "OAuth"},
			},
			{
				label: "Gemini",
				models: []modelTier{
					{name: "Flash", versions: []string{"2.0", "1.5"}},
					{name: "Pro", versions: []string{"2.0", "1.5"}},
					{name: "Ultra", versions: []string{"1.0"}},
				},
				authMethods: []string{"API Key", "OAuth"},
			},
			{
				label: "Mistral",
				models: []modelTier{
					{name: "Nemo", versions: []string{"latest"}},
					{name: "Small", versions: []string{"3.1", "3.0"}},
					{name: "Large", versions: []string{"2.1", "2.0"}},
				},
				authMethods: []string{"API Key"},
			},
			{
				label: "Llama",
				models: []modelTier{
					{name: "8B", versions: []string{"3.2", "3.1"}},
					{name: "70B", versions: []string{"3.3", "3.1"}},
					{name: "405B", versions: []string{"3.1"}},
				},
				authMethods: []string{"API Key"},
			},
			{
				label: "Custom",
				models: []modelTier{
					{name: "Custom", versions: []string{"endpoint"}},
				},
				authMethods: []string{"API Key"},
			},
		},
		selectedProv:    -1,
		selectedModel:   -1,
		selectedVersion: -1,
		selectedAuth:    -1,
	}
	return pm
}

// GetConfiguredProviders returns all confirmed provider selections.
func (p ProviderMenu) GetConfiguredProviders() map[string]ProviderSelection {
	return p.configured
}

// ToManifestConfiguredProviders converts all confirmed configurations to manifest types.
func (p ProviderMenu) ToManifestConfiguredProviders() manifest.ProviderAssignments {
	if len(p.configured) == 0 {
		return nil
	}
	result := make(manifest.ProviderAssignments, len(p.configured))
	for label, sel := range p.configured {
		if sel.IsSet() {
			result[label] = manifest.ProviderAssignment{
				Provider:   sel.Provider,
				Model:      sel.Model,
				Version:    sel.Version,
				Auth:       sel.Auth,
				Credential: sel.Credential,
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// loadStateForProvider restores model/auth/version cursor positions from the
// existing configuration for the given provider label (if any).
func (p ProviderMenu) loadStateForProvider(label string) ProviderMenu {
	sel, ok := p.configured[label]
	if !ok || !sel.IsSet() {
		p.selectedModel = -1
		p.selectedVersion = -1
		p.selectedAuth = -1
		p.modelCursor = 0
		p.authCursor = 0
		p.credInput.SetValue("")
		return p
	}

	// Restore model + version cursors.
	if p.selectedProv >= 0 {
		models := p.providers[p.selectedProv].models
		for i, tier := range models {
			if tier.name == sel.Model {
				p.modelCursor = i
				p.selectedModel = i
				for j, v := range tier.versions {
					if v == sel.Version {
						p.versionCursor = j
						p.selectedVersion = j
						break
					}
				}
				break
			}
		}
	}

	// Restore auth cursor.
	if p.selectedProv >= 0 {
		auths := p.providers[p.selectedProv].authMethods
		for i, a := range auths {
			if a == sel.Auth {
				p.authCursor = i
				p.selectedAuth = i
				break
			}
		}
	}

	p.credInput.SetValue(sel.Credential)
	return p
}

// confirmCurrentSelection saves the current editing state for the active provider.
func (p ProviderMenu) confirmCurrentSelection() ProviderMenu {
	if p.selectedProv < 0 || p.selectedModel < 0 || p.selectedVersion < 0 || p.selectedAuth < 0 {
		return p
	}

	prov := p.providers[p.selectedProv]
	tier := prov.models[p.selectedModel]

	sel := ProviderSelection{
		Provider:   prov.label,
		Model:      tier.name,
		Version:    tier.versions[p.selectedVersion],
		Auth:       prov.authMethods[p.selectedAuth],
		Credential: p.credInput.Value(),
	}

	// Copy map (immutable pattern).
	newConfigured := make(map[string]ProviderSelection, len(p.configured)+1)
	for k, v := range p.configured {
		newConfigured[k] = v
	}
	newConfigured[prov.label] = sel
	p.configured = newConfigured
	return p
}

// clearCurrentProvider removes the configuration for the currently hovered provider.
func (p ProviderMenu) clearCurrentProvider() ProviderMenu {
	provLabel := p.providers[p.cursor].label
	newConfigured := make(map[string]ProviderSelection, len(p.configured))
	for k, v := range p.configured {
		if k != provLabel {
			newConfigured[k] = v
		}
	}
	p.configured = newConfigured
	// Reset edit state
	p.selectedProv = -1
	p.selectedModel = -1
	p.selectedVersion = -1
	p.selectedAuth = -1
	p.modelCursor = 0
	p.authCursor = 0
	p.credInput.SetValue("")
	return p
}

// oauthURL returns a browser URL for the given provider's authentication page.
func oauthURL(provider string) string {
	switch provider {
	case "Claude":
		return "https://console.anthropic.com/settings/keys"
	case "ChatGPT":
		return "https://platform.openai.com/api-keys"
	case "Gemini":
		return "https://aistudio.google.com/app/apikey"
	case "Mistral":
		return "https://console.mistral.ai/api-keys"
	default:
		return ""
	}
}

// openBrowser attempts to open url in the system browser.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}

// enterCredentialStep prepares the credential input for the current auth method.
func (p ProviderMenu) enterCredentialStep() (ProviderMenu, tea.Cmd) {
	p.focus = pmFocusCredential
	// Pre-fill from existing config if available.
	if p.selectedProv >= 0 {
		provLabel := p.providers[p.selectedProv].label
		if existing, ok := p.configured[provLabel]; ok && existing.Credential != "" {
			p.credInput.SetValue(existing.Credential)
		} else {
			p.credInput.SetValue("")
		}
	} else {
		p.credInput.SetValue("")
	}

	authMethod := ""
	if p.selectedProv >= 0 && p.selectedAuth >= 0 {
		authMethod = p.providers[p.selectedProv].authMethods[p.selectedAuth]
	}
	if authMethod == "API Key" {
		p.credInput.Placeholder = "sk-…"
		p.credInput.EchoMode = textinput.EchoPassword
		p.credInput.EchoCharacter = '•'
	} else {
		p.credInput.Placeholder = "paste token here"
		p.credInput.EchoMode = textinput.EchoNormal
	}
	return p, p.credInput.Focus()
}

// Update handles keyboard input and returns a new ProviderMenu and optional command.
func (p ProviderMenu) Update(msg tea.Msg) (ProviderMenu, tea.Cmd) {
	// Delegate to textinput when credential focus is active.
	if p.focus == pmFocusCredential {
		key, ok := msg.(tea.KeyMsg)
		if ok {
			switch key.String() {
			case "enter":
				p = p.confirmCurrentSelection()
				p.focus = pmFocusProviders
				p.credInput.Blur()
				return p, nil
			case "esc":
				p.focus = pmFocusAuth
				p.credInput.Blur()
				return p, nil
			case "ctrl+o":
				if p.selectedProv >= 0 {
					if u := oauthURL(p.providers[p.selectedProv].label); u != "" {
						openBrowser(u)
					}
				}
				return p, nil
			}
		}
		var cmd tea.Cmd
		p.credInput, cmd = p.credInput.Update(msg)
		return p, cmd
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	switch key.String() {

	// ── Vertical navigation ───────────────────────────────────────────────────
	case "j", "down":
		switch {
		case p.focus == pmFocusModels && p.dropdownOpen:
			vers := p.providers[p.cursor].models[p.modelCursor].versions
			if p.versionCursor < len(vers)-1 {
				p.versionCursor++
			}
		case p.focus == pmFocusProviders:
			if p.cursor < len(p.providers)-1 {
				p.cursor++
			}
			p.modelCursor, p.authCursor = 0, 0
			p.dropdownOpen = false
		case p.focus == pmFocusModels:
			models := p.providers[p.cursor].models
			if p.modelCursor < len(models)-1 {
				p.modelCursor++
			}
		case p.focus == pmFocusAuth:
			auths := p.providers[p.cursor].authMethods
			if p.authCursor < len(auths)-1 {
				p.authCursor++
			}
		}

	case "k", "up":
		switch {
		case p.focus == pmFocusModels && p.dropdownOpen:
			if p.versionCursor > 0 {
				p.versionCursor--
			}
		case p.focus == pmFocusProviders:
			if p.cursor > 0 {
				p.cursor--
			}
			p.modelCursor, p.authCursor = 0, 0
			p.dropdownOpen = false
		case p.focus == pmFocusModels:
			if p.modelCursor > 0 {
				p.modelCursor--
			}
		case p.focus == pmFocusAuth:
			if p.authCursor > 0 {
				p.authCursor--
			}
		}

	// ── Horizontal focus movement (blocked while dropdown open) ───────────────
	case "l", "tab":
		if !p.dropdownOpen {
			switch p.focus {
			case pmFocusProviders:
				p.focus = pmFocusModels
			case pmFocusModels:
				p.focus = pmFocusAuth
			}
		}

	case "h", "shift+tab":
		if !p.dropdownOpen {
			switch p.focus {
			case pmFocusModels:
				p.focus = pmFocusProviders
			case pmFocusAuth:
				p.focus = pmFocusModels
			}
		}

	// ── Clear current provider's configuration ────────────────────────────────
	case "x":
		if p.focus == pmFocusProviders {
			p = p.clearCurrentProvider()
		}

	// ── Confirm / open dropdown ───────────────────────────────────────────────
	case "enter":
		switch p.focus {
		case pmFocusProviders:
			// Start configuring the hovered provider; load existing config.
			p.selectedProv = p.cursor
			p.selectedModel = -1
			p.selectedVersion = -1
			p.selectedAuth = -1
			p.modelCursor = 0
			p.authCursor = 0
			p = p.loadStateForProvider(p.providers[p.cursor].label)
			p.focus = pmFocusModels

		case pmFocusModels:
			if p.dropdownOpen {
				p.selectedModel = p.modelCursor
				p.selectedVersion = p.versionCursor
				p.selectedAuth = -1
				p.dropdownOpen = false
				p.focus = pmFocusAuth
				p.authCursor = 0
			} else {
				p.dropdownOpen = true
				p.versionCursor = 0
				if p.selectedProv == p.cursor && p.selectedModel == p.modelCursor && p.selectedVersion >= 0 {
					p.versionCursor = p.selectedVersion
				}
			}

		case pmFocusAuth:
			p.selectedAuth = p.authCursor
			return p.enterCredentialStep()
		}

	// ── Cancel dropdown / step back ───────────────────────────────────────────
	case "esc":
		if p.dropdownOpen {
			p.dropdownOpen = false
			p.versionCursor = 0
		} else if p.focus != pmFocusProviders {
			switch p.focus {
			case pmFocusAuth:
				p.focus = pmFocusModels
			case pmFocusModels:
				p.focus = pmFocusProviders
				p.selectedProv = -1
			}
		}
	}

	return p, nil
}

// ── Layout constants ──────────────────────────────────────────────────────────

const (
	pmCol1W = 12 // provider column visible width
	pmCol2W = 16 // model column visible width
	pmCol3W = 12 // auth column visible width (last, not padded by pmRow)
	// pmBoxW is the Width() argument for StyleModalBorder.
	// StyleModalBorder has Padding(0,1) + RoundedBorder, so actual rendered
	// width = pmBoxW + 2 (padding) + 2 (border) = pmBoxW + 4.
	pmBoxW = pmCol1W + pmCol2W + pmCol3W // 40 → total box ≈ 44 chars
)

// ── View ──────────────────────────────────────────────────────────────────────

// View renders the provider menu as a self-contained bordered box string.
func (p ProviderMenu) View() string {
	var rows []string

	// ── Cyberpunk title bar ───────────────────────────────────────────────────
	deco := StyleHeaderDeco.Render(headerDecoFrames[AnimFrame])
	titleText := StyleNeonMagenta.Render("◈ AI PROVIDERS ◈")
	titleLine := lipgloss.NewStyle().
		Background(lipgloss.Color(clrBg2)).
		Width(pmBoxW).
		Align(lipgloss.Center).
		Render(deco + " " + titleText + " " + deco)
	rows = append(rows, titleLine)
	rows = append(rows, "") // padding below title
	rows = append(rows, p.renderHeaders())
	rows = append(rows, p.renderDividers())

	col1 := p.buildProviderCol()
	col2 := p.buildModelCol()
	col3 := p.buildAuthCol()

	h := max(max(len(col1), len(col2)), len(col3))
	for len(col1) < h {
		col1 = append(col1, "")
	}
	for len(col2) < h {
		col2 = append(col2, "")
	}
	for len(col3) < h {
		col3 = append(col3, "")
	}

	for i := 0; i < h; i++ {
		rows = append(rows, pmRow(col1[i], col2[i], col3[i]))
	}

	rows = append(rows, "")

	// Credential input step.
	if p.focus == pmFocusCredential {
		rows = append(rows, p.renderCredentialPanel())
		rows = append(rows, "")
	}

	// Show configured providers summary.
	if summary := p.renderConfiguredSummary(); summary != "" {
		rows = append(rows, summary)
		rows = append(rows, "")
	}

	// Context-sensitive hint bar.
	var hints string
	switch {
	case p.focus == pmFocusCredential:
		authMethod := ""
		if p.selectedProv >= 0 && p.selectedAuth >= 0 {
			authMethod = p.providers[p.selectedProv].authMethods[p.selectedAuth]
		}
		if authMethod == "OAuth" {
			hints = hintBar("Enter", "confirm", "Ctrl+O", "open browser", "Esc", "back")
		} else {
			hints = hintBar("Enter", "confirm", "Esc", "back")
		}
	case p.dropdownOpen:
		hints = hintBar("j/k", "version", "Enter", "confirm", "Esc", "cancel")
	case p.focus == pmFocusProviders:
		hints = hintBar("j/k", "navigate", "Enter", "configure", "x", "clear", "M", "close")
	default:
		hints = hintBar("j/k", "nav", "h/l", "col", "Enter", "pick", "Esc", "back")
	}
	rows = append(rows, hints)

	return StyleModalBorder.Width(pmBoxW).Render(strings.Join(rows, "\n"))
}

// renderConfiguredSummary shows all currently configured providers.
func (p ProviderMenu) renderConfiguredSummary() string {
	if len(p.configured) == 0 {
		return ""
	}
	var lines []string
	// Iterate in provider order for deterministic output.
	for _, prov := range p.providers {
		sel, ok := p.configured[prov.label]
		if !ok || !sel.IsSet() {
			continue
		}
		lines = append(lines, StyleNeonCyan.Render("  ◈ ")+StyleNeonGreen.Render(sel.Short()))
	}
	return strings.Join(lines, "\n")
}

// renderCredentialPanel renders the inline API key / OAuth token input.
func (p ProviderMenu) renderCredentialPanel() string {
	authMethod := ""
	provLabel := ""
	if p.selectedProv >= 0 {
		provLabel = p.providers[p.selectedProv].label
		if p.selectedAuth >= 0 {
			authMethod = p.providers[p.selectedProv].authMethods[p.selectedAuth]
		}
	}

	var label string
	var subText string
	if authMethod == "OAuth" {
		label = StyleNeonViolet.Render("◈ OAuth Token")
		if u := oauthURL(provLabel); u != "" {
			subText = StyleFgDimStyle.Render(fmt.Sprintf("  Get token: %s", u))
		} else {
			subText = StyleFgDimStyle.Render("  Paste your OAuth access token below")
		}
	} else {
		label = StyleNeonCyan.Render("◈ API Key")
		subText = StyleFgDimStyle.Render(fmt.Sprintf("  Enter %s API key", provLabel))
	}

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(clrMagenta)).
		Padding(0, 1).
		Width(pmBoxW - 4).
		Render(label + "  " + p.credInput.View())

	var lines []string
	lines = append(lines, subText)
	lines = append(lines, inputBox)
	return strings.Join(lines, "\n")
}

// renderHeaders returns the column header row.
func (p ProviderMenu) renderHeaders() string {
	bg := lipgloss.Color(clrBg2)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Background(bg)
	active := lipgloss.NewStyle().Foreground(lipgloss.Color(clrCyan)).Bold(true).Underline(true).Background(bg)
	dropdown := lipgloss.NewStyle().Foreground(lipgloss.Color(clrOrange)).Bold(true).Background(bg)

	h1, h2, h3 := dim, dim, dim
	switch p.focus {
	case pmFocusProviders:
		h1 = active
	case pmFocusModels:
		if p.dropdownOpen {
			h2 = dropdown
		} else {
			h2 = active
		}
	case pmFocusAuth:
		h3 = active
	}
	return pmRow(h1.Render("PROVIDER"), h2.Render("MODEL"), h3.Render("AUTH"))
}

// renderDividers returns the ─── separator row under the headers.
func (p ProviderMenu) renderDividers() string {
	s := lipgloss.NewStyle().Foreground(lipgloss.Color(clrViolet)).Background(lipgloss.Color(clrBg2))
	return pmRow(
		s.Render(strings.Repeat("─", 8)),
		s.Render(strings.Repeat("─", 9)),
		s.Render(strings.Repeat("─", 8)),
	)
}

// buildProviderCol returns one string per row for the provider column.
// ✓ is shown for any provider that is already configured.
func (p ProviderMenu) buildProviderCol() []string {
	lines := make([]string, 0, len(p.providers))
	for i, prov := range p.providers {
		isCur := i == p.cursor
		isConfigured := p.configured[prov.label].IsSet()
		isEditing := i == p.selectedProv
		isHL := isCur && p.focus == pmFocusProviders

		rowBg := lipgloss.Color(clrBg2)
		if isHL {
			rowBg = lipgloss.Color(clrBgHL)
		}

		arrow := lipgloss.NewStyle().Background(rowBg).Render("  ")
		if isCur {
			arrow = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Background(rowBg).Render("▶ ")
		}

		var label string
		switch {
		case isConfigured && !isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Bold(true).Background(lipgloss.Color(clrBg2)).Render("◈ " + prov.label)
		case isConfigured && isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Bold(true).Background(rowBg).Render("◈ " + prov.label)
		case isEditing && isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrCyan)).Bold(true).Background(rowBg).Render(prov.label)
		case isCur && p.focus == pmFocusProviders:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrCyan)).Bold(true).Background(rowBg).Render(prov.label)
		case isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg)).Background(lipgloss.Color(clrBg2)).Render(prov.label)
		default:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Background(lipgloss.Color(clrBg2)).Render(prov.label)
		}

		cell := arrow + label
		if isHL {
			cell = pmHighlight(cell, pmCol1W)
		}
		lines = append(lines, cell)
	}
	return lines
}

// buildModelCol returns one string per row for the model column.
// When the dropdown is open, version rows are injected after the active tier.
func (p ProviderMenu) buildModelCol() []string {
	models := p.providers[p.cursor].models
	var lines []string

	for i, tier := range models {
		isCur := i == p.modelCursor
		isSel := p.selectedProv == p.cursor && p.selectedModel == i
		isHL := isCur && p.focus == pmFocusModels && !p.dropdownOpen

		rowBg := lipgloss.Color(clrBg2)
		if isHL {
			rowBg = lipgloss.Color(clrBgHL)
		}

		displayName := tier.name
		if isSel && p.selectedVersion >= 0 && p.selectedVersion < len(tier.versions) {
			displayName = tier.name + " " + tier.versions[p.selectedVersion]
		}

		var indicator string
		if isCur && p.focus == pmFocusModels {
			if p.dropdownOpen {
				indicator = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Background(lipgloss.Color(clrBg2)).Render(" ▴")
			} else {
				indicator = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Background(rowBg).Render(" ▾")
			}
		}

		arrow := lipgloss.NewStyle().Background(rowBg).Render("  ")
		if isCur {
			arrow = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Background(rowBg).Render("▶ ")
		}

		var label string
		switch {
		case isSel && !isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Bold(true).Background(lipgloss.Color(clrBg2)).Render("◈ " + displayName)
		case isSel && isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Bold(true).Background(rowBg).Render("◈ " + displayName)
		case isCur && p.focus == pmFocusModels:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrCyan)).Bold(true).Background(rowBg).Render(displayName)
		case isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg)).Background(lipgloss.Color(clrBg2)).Render(displayName)
		default:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Background(lipgloss.Color(clrBg2)).Render(displayName)
		}

		cell := arrow + label + indicator
		if isHL {
			cell = pmHighlight(cell, pmCol2W)
		}
		lines = append(lines, cell)

		// ── Inject version dropdown rows ──────────────────────────────────────
		if isCur && p.dropdownOpen {
			for j, v := range tier.versions {
				isVCur := j == p.versionCursor
				vBg := lipgloss.Color(clrBg2)
				if isVCur {
					vBg = lipgloss.Color(clrBgHL)
				}

				var vArrow, vLabel string
				if isVCur {
					vArrow = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Background(vBg).Render("  ▸ ")
					vLabel = lipgloss.NewStyle().Foreground(lipgloss.Color(clrBlue)).Bold(true).Background(vBg).Render(v)
				} else {
					vArrow = lipgloss.NewStyle().Background(lipgloss.Color(clrBg2)).Render("    ")
					vLabel = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Background(lipgloss.Color(clrBg2)).Render(v)
				}

				vCell := vArrow + vLabel
				if isVCur {
					vCell = pmHighlight(vCell, pmCol2W)
				}
				lines = append(lines, vCell)
			}
		}
	}
	return lines
}

// buildAuthCol returns one string per row for the auth method column.
func (p ProviderMenu) buildAuthCol() []string {
	auths := p.providers[p.cursor].authMethods
	lines := make([]string, 0, len(auths))
	for i, a := range auths {
		isCur := i == p.authCursor
		isSel := p.selectedProv == p.cursor &&
			p.selectedModel >= 0 &&
			p.selectedVersion >= 0 &&
			i == p.selectedAuth
		isHL := isCur && p.focus == pmFocusAuth

		rowBg := lipgloss.Color(clrBg2)
		if isHL {
			rowBg = lipgloss.Color(clrBgHL)
		}

		arrow := lipgloss.NewStyle().Background(rowBg).Render("  ")
		if isCur && p.focus == pmFocusAuth {
			arrow = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Background(rowBg).Render("▶ ")
		}

		var label string
		switch {
		case isSel:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Bold(true).Background(lipgloss.Color(clrBg2)).Render("◈ " + a)
		case isCur && p.focus == pmFocusAuth:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrCyan)).Bold(true).Background(rowBg).Render(a)
		case isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg)).Background(lipgloss.Color(clrBg2)).Render(a)
		default:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Background(lipgloss.Color(clrBg2)).Render(a)
		}

		cell := arrow + label
		if isHL {
			cell = pmHighlight(cell, pmCol3W)
		}
		lines = append(lines, cell)
	}
	return lines
}

// ── Layout helpers ────────────────────────────────────────────────────────────

// pmRow assembles three column cells into one display line.
func pmRow(col1, col2, col3 string) string {
	return pmPad(col1, pmCol1W) + pmPad(col2, pmCol2W) + col3
}

// pmPad pads s with background-colored spaces until its visible width equals toW.
func pmPad(s string, toW int) string {
	if pad := toW - lipgloss.Width(s); pad > 0 {
		return s + lipgloss.NewStyle().Background(lipgloss.Color(clrBg2)).Render(strings.Repeat(" ", pad))
	}
	return s
}

// pmHighlight pads s to colW with highlight-colored spaces and applies the
// animated cursor-line background (breathing/pulse effect via AnimFrame).
func pmHighlight(s string, colW int) string {
	curStyle := activeCurLineStyle()
	bg := lipgloss.Color(clrBgHL)
	if AnimFrame == 1 {
		bg = lipgloss.Color(clrBgHL2)
	}
	if pad := colW - lipgloss.Width(s); pad > 0 {
		s = s + lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", pad))
	}
	return curStyle.Render(s)
}
