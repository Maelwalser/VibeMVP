package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// pmFocus tracks which column owns keyboard input.
type pmFocus int

const (
	pmFocusProviders pmFocus = iota
	pmFocusModels
	pmFocusAuth
)

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

// ProviderMenu is the centered modal for picking provider → model version → auth.
// It carries no integration logic — UI only.
type ProviderMenu struct {
	providers       []providerEntry
	cursor          int     // hovered row in provider list
	modelCursor     int     // hovered row in model list
	authCursor      int     // hovered row in auth list
	focus           pmFocus // column that owns input
	dropdownOpen    bool    // version dropdown visible
	versionCursor   int     // hovered row inside the dropdown
	selectedProv    int     // -1 = none confirmed
	selectedModel   int     // -1 = none confirmed (index in models slice)
	selectedVersion int     // -1 = none confirmed (index in tier.versions)
	selectedAuth    int     // -1 = none confirmed
}

func newProviderMenu() ProviderMenu {
	return ProviderMenu{
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
}

// Update handles keyboard input and returns a new ProviderMenu (immutable).
func (p ProviderMenu) Update(msg tea.Msg) ProviderMenu {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return p
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

	// ── Confirm / open dropdown ───────────────────────────────────────────────
	case "enter":
		switch p.focus {
		case pmFocusProviders:
			p.selectedProv = p.cursor
			p.selectedModel, p.selectedVersion, p.selectedAuth = -1, -1, -1
			p.focus = pmFocusModels
			p.modelCursor = 0

		case pmFocusModels:
			if p.dropdownOpen {
				// Confirm version selection, advance to auth.
				p.selectedModel = p.modelCursor
				p.selectedVersion = p.versionCursor
				p.selectedAuth = -1
				p.dropdownOpen = false
				p.focus = pmFocusAuth
				p.authCursor = 0
			} else {
				// Open the version dropdown for the current tier.
				p.dropdownOpen = true
				p.versionCursor = 0
				// Pre-select the previously confirmed version if on same tier.
				if p.selectedProv == p.cursor && p.selectedModel == p.modelCursor && p.selectedVersion >= 0 {
					p.versionCursor = p.selectedVersion
				}
			}

		case pmFocusAuth:
			p.selectedAuth = p.authCursor
		}

	// ── Cancel dropdown (handled here; modal-level Esc is in model.go) ────────
	case "esc":
		if p.dropdownOpen {
			p.dropdownOpen = false
			p.versionCursor = 0
		}
	}

	return p
}

// ── Layout constants ──────────────────────────────────────────────────────────

const (
	pmCol1W = 16 // provider column visible width
	pmCol2W = 16 // model column visible width
	pmCol3W = 14 // auth column visible width (last, not padded by pmRow)
	// pmBoxW is the Width() argument for StyleModalBorder.
	// StyleModalBorder has Padding(0,1) + RoundedBorder, so actual rendered
	// width = pmBoxW + 2 (padding) + 2 (border) = pmBoxW + 4.
	pmBoxW = pmCol1W + pmCol2W + pmCol3W // 46 → total box ≈ 50 chars
)

// ── View ──────────────────────────────────────────────────────────────────────

// View renders the provider menu as a self-contained bordered box string.
func (p ProviderMenu) View() string {
	var rows []string

	rows = append(rows, "") // top padding
	rows = append(rows, p.renderHeaders())
	rows = append(rows, p.renderDividers())

	// Build each column independently so the model dropdown can expand freely.
	col1 := p.buildProviderCol()
	col2 := p.buildModelCol()
	col3 := p.buildAuthCol()

	// Pad shorter columns to the same height.
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

	rows = append(rows, "") // spacer before hints

	// Context-sensitive hint bar.
	var hints string
	if p.dropdownOpen {
		hints = hintBar("j/k", "version", "Enter", "confirm", "Esc", "cancel")
	} else {
		hints = hintBar("j/k", "nav", "h/l", "col", "Enter", "pick", "M", "close")
	}
	rows = append(rows, hints)

	return StyleModalBorder.Width(pmBoxW).Render(strings.Join(rows, "\n"))
}

// renderHeaders returns the column header row, with the active column
// highlighted in cyan; turns yellow while a dropdown is open.
func (p ProviderMenu) renderHeaders() string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))
	active := lipgloss.NewStyle().Foreground(lipgloss.Color(clrCyan)).Bold(true).Underline(true)
	dropdown := lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Bold(true)

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
	s := lipgloss.NewStyle().Foreground(lipgloss.Color(clrComment))
	return pmRow(
		s.Render(strings.Repeat("─", 10)),
		s.Render(strings.Repeat("─", 9)),
		s.Render(strings.Repeat("─", 8)),
	)
}

// buildProviderCol returns one string per row for the provider column.
func (p ProviderMenu) buildProviderCol() []string {
	lines := make([]string, 0, len(p.providers))
	for i, prov := range p.providers {
		isCur := i == p.cursor
		isSel := i == p.selectedProv

		arrow := "  "
		if isCur {
			arrow = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Render("▶ ")
		}

		var label string
		switch {
		case isSel && !isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Render("✓ " + prov.label)
		case isSel && isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Render(prov.label)
		case isCur && p.focus == pmFocusProviders:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrBlue)).Bold(true).Render(prov.label)
		case isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg)).Render(prov.label)
		default:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Render(prov.label)
		}

		cell := arrow + label
		if isCur && p.focus == pmFocusProviders {
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

		// Choose display name: show "Tier Ver" when a version is confirmed.
		displayName := tier.name
		if isSel && p.selectedVersion >= 0 && p.selectedVersion < len(tier.versions) {
			displayName = tier.name + " " + tier.versions[p.selectedVersion]
		}

		// Dropdown open/close indicator (only on the focused tier).
		var indicator string
		if isCur && p.focus == pmFocusModels {
			if p.dropdownOpen {
				indicator = " " + StyleSelectArrow.Render("▴")
			} else {
				indicator = " " + StyleSelectArrow.Render("▾")
			}
		}

		arrow := "  "
		if isCur {
			arrow = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Render("▶ ")
		}

		var label string
		switch {
		case isSel && !isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Render("✓ " + displayName)
		case isSel && isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Render(displayName)
		case isCur && p.focus == pmFocusModels:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrBlue)).Bold(true).Render(displayName)
		case isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg)).Render(displayName)
		default:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Render(displayName)
		}

		cell := arrow + label + indicator
		if isCur && p.focus == pmFocusModels && !p.dropdownOpen {
			cell = pmHighlight(cell, pmCol2W)
		}
		lines = append(lines, cell)

		// ── Inject version dropdown rows ──────────────────────────────────────
		if isCur && p.dropdownOpen {
			for j, v := range tier.versions {
				isVCur := j == p.versionCursor

				var vArrow, vLabel string
				if isVCur {
					vArrow = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Render("  ▸ ")
					vLabel = lipgloss.NewStyle().Foreground(lipgloss.Color(clrBlue)).Bold(true).Render(v)
				} else {
					vArrow = "    "
					vLabel = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Render(v)
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

		arrow := "  "
		if isCur && p.focus == pmFocusAuth {
			arrow = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Render("▶ ")
		}

		var label string
		switch {
		case isSel:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Render("✓ " + a)
		case isCur && p.focus == pmFocusAuth:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrBlue)).Bold(true).Render(a)
		case isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg)).Render(a)
		default:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Render(a)
		}

		cell := arrow + label
		if isCur && p.focus == pmFocusAuth {
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

// pmPad pads s with spaces until its visible width equals toW.
func pmPad(s string, toW int) string {
	if pad := toW - lipgloss.Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// pmHighlight pads s to colW and applies the cursor-line background.
func pmHighlight(s string, colW int) string {
	padded := pmPad(s, colW)
	return StyleCurLine.Render(padded)
}
