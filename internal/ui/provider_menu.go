package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// pmFocus tracks which column owns keyboard input.
type pmFocus int

const (
	pmFocusSections pmFocus = iota
	pmFocusProviders
	pmFocusModels
	pmFocusAuth
)

// sectionEntry is a tab section that can be assigned a provider.
type sectionEntry struct {
	id    string
	label string
	badge string // 1-char abbreviation shown in tab bar
}

// ProviderSelection holds a confirmed provider/model/auth triple.
type ProviderSelection struct {
	Provider string
	Model    string
	Version  string
	Auth     string
}

// IsSet reports whether this selection is fully configured.
func (p ProviderSelection) IsSet() bool {
	return p.Provider != "" && p.Model != "" && p.Auth != ""
}

// Short returns a compact display string like "Claude Sonnet 4.5 / API Key".
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

// ProviderMenu is the centered modal for picking section → provider → model version → auth.
// It carries no integration logic — UI only.
type ProviderMenu struct {
	// Section column
	sectionList   []sectionEntry
	sectionCursor int
	assignments   map[int]ProviderSelection // sectionList index → confirmed selection

	// Provider/model/auth columns
	providers       []providerEntry
	cursor          int     // hovered row in provider list
	modelCursor     int     // hovered row in model list
	authCursor      int     // hovered row in auth list
	focus           pmFocus // column that owns input
	dropdownOpen    bool    // version dropdown visible
	versionCursor   int     // hovered row inside the dropdown
	selectedProv    int     // -1 = none confirmed (for current section)
	selectedModel   int     // -1 = none confirmed (index in models slice)
	selectedVersion int     // -1 = none confirmed (index in tier.versions)
	selectedAuth    int     // -1 = none confirmed
}

func newProviderMenu() ProviderMenu {
	pm := ProviderMenu{
		sectionList: []sectionEntry{
			{id: "backend", label: "Backend", badge: "B"},
			{id: "data", label: "Data", badge: "D"},
			{id: "contracts", label: "Contracts", badge: "C"},
			{id: "frontend", label: "Frontend", badge: "F"},
			{id: "infrastructure", label: "Infra", badge: "I"},
			{id: "crosscut", label: "CrossCut", badge: "X"},
			{id: "realize", label: "Realize", badge: "R"},
		},
		assignments: make(map[int]ProviderSelection),
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

// SectionAssignment returns the confirmed ProviderSelection for the given section ID,
// and whether one exists.
func (p ProviderMenu) SectionAssignment(sectionID string) (ProviderSelection, bool) {
	for i, s := range p.sectionList {
		if s.id == sectionID {
			sel, ok := p.assignments[i]
			return sel, ok && sel.IsSet()
		}
	}
	return ProviderSelection{}, false
}

// loadSectionState restores the provider/model/auth cursor positions from the
// assignment saved for the currently selected section.
func (p ProviderMenu) loadSectionState() ProviderMenu {
	sel, ok := p.assignments[p.sectionCursor]
	if !ok || !sel.IsSet() {
		p.selectedProv = -1
		p.selectedModel = -1
		p.selectedVersion = -1
		p.selectedAuth = -1
		p.cursor = 0
		p.modelCursor = 0
		p.authCursor = 0
		return p
	}

	// Restore provider cursor.
	for i, prov := range p.providers {
		if prov.label == sel.Provider {
			p.cursor = i
			p.selectedProv = i
			break
		}
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

	return p
}

// confirmCurrentSelection saves the current provider/model/auth choices as the
// assignment for the active section, if all three are confirmed.
func (p ProviderMenu) confirmCurrentSelection() ProviderMenu {
	if p.selectedProv < 0 || p.selectedModel < 0 || p.selectedVersion < 0 || p.selectedAuth < 0 {
		return p
	}

	prov := p.providers[p.selectedProv]
	tier := prov.models[p.selectedModel]

	sel := ProviderSelection{
		Provider: prov.label,
		Model:    tier.name,
		Version:  tier.versions[p.selectedVersion],
		Auth:     prov.authMethods[p.selectedAuth],
	}

	// Copy map (immutable pattern).
	newAssignments := make(map[int]ProviderSelection, len(p.assignments)+1)
	for k, v := range p.assignments {
		newAssignments[k] = v
	}
	newAssignments[p.sectionCursor] = sel
	p.assignments = newAssignments
	return p
}

// clearCurrentSection removes the assignment for the active section.
func (p ProviderMenu) clearCurrentSection() ProviderMenu {
	newAssignments := make(map[int]ProviderSelection, len(p.assignments))
	for k, v := range p.assignments {
		if k != p.sectionCursor {
			newAssignments[k] = v
		}
	}
	p.assignments = newAssignments
	p.selectedProv = -1
	p.selectedModel = -1
	p.selectedVersion = -1
	p.selectedAuth = -1
	p.cursor = 0
	p.modelCursor = 0
	p.authCursor = 0
	return p
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
		case p.focus == pmFocusSections:
			if p.sectionCursor < len(p.sectionList)-1 {
				p.sectionCursor++
				p = p.loadSectionState()
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
		case p.focus == pmFocusSections:
			if p.sectionCursor > 0 {
				p.sectionCursor--
				p = p.loadSectionState()
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
			case pmFocusSections:
				p.focus = pmFocusProviders
			case pmFocusProviders:
				p.focus = pmFocusModels
			case pmFocusModels:
				p.focus = pmFocusAuth
			}
		}

	case "h", "shift+tab":
		if !p.dropdownOpen {
			switch p.focus {
			case pmFocusProviders:
				p.focus = pmFocusSections
			case pmFocusModels:
				p.focus = pmFocusProviders
			case pmFocusAuth:
				p.focus = pmFocusModels
			}
		}

	// ── Clear current section assignment ─────────────────────────────────────
	case "x":
		if p.focus == pmFocusSections {
			p = p.clearCurrentSection()
		}

	// ── Confirm / open dropdown ───────────────────────────────────────────────
	case "enter":
		switch p.focus {
		case pmFocusSections:
			// Move focus to providers, loading existing state.
			p = p.loadSectionState()
			p.focus = pmFocusProviders

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
			// Save assignment for the current section.
			p = p.confirmCurrentSelection()
			// Return focus to sections so the user can configure the next one.
			p.focus = pmFocusSections
		}

	// ── Cancel dropdown (handled here; modal-level Esc is in model.go) ────────
	case "esc":
		if p.dropdownOpen {
			p.dropdownOpen = false
			p.versionCursor = 0
		} else if p.focus != pmFocusSections {
			// Step back one column.
			switch p.focus {
			case pmFocusAuth:
				p.focus = pmFocusModels
			case pmFocusModels:
				p.focus = pmFocusProviders
			case pmFocusProviders:
				p.focus = pmFocusSections
			}
		}
	}

	return p
}

// ── Layout constants ──────────────────────────────────────────────────────────

const (
	pmCol0W = 12 // section column visible width
	pmCol1W = 12 // provider column visible width
	pmCol2W = 16 // model column visible width
	pmCol3W = 12 // auth column visible width (last, not padded by pmRow)
	// pmBoxW is the Width() argument for StyleModalBorder.
	// StyleModalBorder has Padding(0,1) + RoundedBorder, so actual rendered
	// width = pmBoxW + 2 (padding) + 2 (border) = pmBoxW + 4.
	pmBoxW = pmCol0W + pmCol1W + pmCol2W + pmCol3W // 52 → total box ≈ 56 chars
)

// ── View ──────────────────────────────────────────────────────────────────────

// View renders the provider menu as a self-contained bordered box string.
func (p ProviderMenu) View() string {
	var rows []string

	rows = append(rows, "") // top padding
	rows = append(rows, p.renderHeaders())
	rows = append(rows, p.renderDividers())

	// Build each column independently so the model dropdown can expand freely.
	col0 := p.buildSectionCol()
	col1 := p.buildProviderCol()
	col2 := p.buildModelCol()
	col3 := p.buildAuthCol()

	// Pad shorter columns to the same height.
	h := max(max(len(col0), len(col1)), max(len(col2), len(col3)))
	for len(col0) < h {
		col0 = append(col0, "")
	}
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
		rows = append(rows, pmRow(col0[i], col1[i], col2[i], col3[i]))
	}

	rows = append(rows, "") // spacer before hints

	// Context-sensitive hint bar.
	var hints string
	switch {
	case p.dropdownOpen:
		hints = hintBar("j/k", "version", "Enter", "confirm", "Esc", "cancel")
	case p.focus == pmFocusSections:
		hints = hintBar("j/k", "section", "Enter/l", "configure", "x", "clear", "M", "close")
	default:
		hints = hintBar("j/k", "nav", "h/l", "col", "Enter", "pick", "Esc", "back")
	}
	rows = append(rows, hints)

	return StyleModalBorder.Width(pmBoxW).Render(strings.Join(rows, "\n"))
}

// renderHeaders returns the column header row.
func (p ProviderMenu) renderHeaders() string {
	bg := lipgloss.Color(clrBg2)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Background(bg)
	active := lipgloss.NewStyle().Foreground(lipgloss.Color(clrCyan)).Bold(true).Underline(true).Background(bg)
	dropdown := lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Bold(true).Background(bg)

	h0, h1, h2, h3 := dim, dim, dim, dim
	switch p.focus {
	case pmFocusSections:
		h0 = active
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
	return pmRow(h0.Render("SECTION"), h1.Render("PROVIDER"), h2.Render("MODEL"), h3.Render("AUTH"))
}

// renderDividers returns the ─── separator row under the headers.
func (p ProviderMenu) renderDividers() string {
	s := lipgloss.NewStyle().Foreground(lipgloss.Color(clrComment)).Background(lipgloss.Color(clrBg2))
	return pmRow(
		s.Render(strings.Repeat("─", 8)),
		s.Render(strings.Repeat("─", 8)),
		s.Render(strings.Repeat("─", 9)),
		s.Render(strings.Repeat("─", 8)),
	)
}

// buildSectionCol returns one string per row for the section column.
func (p ProviderMenu) buildSectionCol() []string {
	lines := make([]string, 0, len(p.sectionList))
	for i, sec := range p.sectionList {
		isCur := i == p.sectionCursor
		_, hasAssignment := p.assignments[i]
		isAssigned := hasAssignment && p.assignments[i].IsSet()

		arrow := "  "
		if isCur {
			arrow = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Render("▶ ")
		}

		var label string
		switch {
		case isAssigned && !isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Render("✓ " + sec.label)
		case isAssigned && isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Render(sec.label)
		case isCur && p.focus == pmFocusSections:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrBlue)).Bold(true).Render(sec.label)
		case isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg)).Render(sec.label)
		default:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Render(sec.label)
		}

		cell := arrow + label
		if isCur && p.focus == pmFocusSections {
			cell = pmHighlight(cell, pmCol0W)
		}
		lines = append(lines, cell)
	}
	return lines
}

// buildProviderCol returns one string per row for the provider column.
func (p ProviderMenu) buildProviderCol() []string {
	lines := make([]string, 0, len(p.providers))
	for i, prov := range p.providers {
		isCur := i == p.cursor
		isSel := i == p.selectedProv
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
		case isSel && !isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Background(lipgloss.Color(clrBg2)).Render("✓ " + prov.label)
		case isSel && isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Background(rowBg).Render(prov.label)
		case isCur && p.focus == pmFocusProviders:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrBlue)).Bold(true).Background(rowBg).Render(prov.label)
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

		// Choose display name: show "Tier Ver" when a version is confirmed.
		displayName := tier.name
		if isSel && p.selectedVersion >= 0 && p.selectedVersion < len(tier.versions) {
			displayName = tier.name + " " + tier.versions[p.selectedVersion]
		}

		// Dropdown open/close indicator (only on the focused tier).
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
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Background(lipgloss.Color(clrBg2)).Render("✓ " + displayName)
		case isSel && isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Background(rowBg).Render(displayName)
		case isCur && p.focus == pmFocusModels:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrBlue)).Bold(true).Background(rowBg).Render(displayName)
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
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Background(lipgloss.Color(clrBg2)).Render("✓ " + a)
		case isCur && p.focus == pmFocusAuth:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrBlue)).Bold(true).Background(rowBg).Render(a)
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

// pmRow assembles four column cells into one display line.
func pmRow(col0, col1, col2, col3 string) string {
	return pmPad(col0, pmCol0W) + pmPad(col1, pmCol1W) + pmPad(col2, pmCol2W) + col3
}

// pmPad pads s with background-colored spaces until its visible width equals toW.
func pmPad(s string, toW int) string {
	if pad := toW - lipgloss.Width(s); pad > 0 {
		return s + lipgloss.NewStyle().Background(lipgloss.Color(clrBg2)).Render(strings.Repeat(" ", pad))
	}
	return s
}

// pmHighlight pads s to colW with highlight-colored spaces and applies the cursor-line background.
func pmHighlight(s string, colW int) string {
	if pad := colW - lipgloss.Width(s); pad > 0 {
		s = s + lipgloss.NewStyle().Background(lipgloss.Color(clrBgHL)).Render(strings.Repeat(" ", pad))
	}
	return StyleCurLine.Render(s)
}
