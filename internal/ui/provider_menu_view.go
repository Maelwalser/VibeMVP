package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Layout constants ──────────────────────────────────────────────────────────

const (
	pmCol1W = 12 // provider column visible width
	pmCol2W = 16 // model column visible width
	pmCol3W = 12 // auth column visible width (last, not padded by pmRow)
	// pmBoxW is the Width() argument for StyleModalBorder.
	// StyleModalBorder has Padding(0,1) + RoundedBorder, so actual rendered
	// width = pmBoxW + 2 (padding) + 2 (border) = pmBoxW + 4.
	// +2 for the two │ column separators inserted by pmRow.
	pmBoxW = pmCol1W + 1 + pmCol2W + 1 + pmCol3W // 42 → total box ≈ 46 chars
)

// ── View ──────────────────────────────────────────────────────────────────────

// View renders the provider menu as a self-contained bordered box string.
func (p ProviderMenu) View() string {
	var rows []string

	// ── Cyberpunk title bar ───────────────────────────────────────────────────
	// Opposing frames on left/right produce the same scanning animation as the
	// main header bar: light appears to sweep across the title.
	decoL := StyleHeaderDeco.Render(headerDecoFrames[AnimFrame])
	decoR := StyleHeaderDeco.Render(headerDecoFrames[1-AnimFrame])
	titleText := StyleNeonMagenta.Background(lipgloss.Color(clrBg2)).Render("◈ AI PROVIDERS ◈")
	spaceBg := lipgloss.NewStyle().Background(lipgloss.Color(clrBg2)).Render(" ")
	titleLine := lipgloss.NewStyle().
		Background(lipgloss.Color(clrBg2)).
		Width(pmBoxW).
		Align(lipgloss.Center).
		Render(decoL + spaceBg + titleText + spaceBg + decoR)
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

	// Context-sensitive hint bar — use hintBarBg so all spans carry clrBg2.
	modalBg := lipgloss.Color(clrBg2)
	var hints string
	switch {
	case p.focus == pmFocusCredential:
		authMethod := ""
		if p.selectedProv >= 0 && p.selectedAuth >= 0 {
			authMethod = p.providers[p.selectedProv].authMethods[p.selectedAuth]
		}
		switch {
		case authMethod == "OAuth" && p.oauthAwaitingClientID:
			hints = hintBarBg(modalBg, "Enter", "open browser →", "Esc", "back")
		case authMethod == "OAuth":
			hints = hintBarBg(modalBg, "Enter", "confirm token", "Ctrl+O", "re-authorize", "Esc", "back")
		default:
			hints = hintBarBg(modalBg, "Enter", "confirm", "Esc", "back")
		}
	case p.dropdownOpen:
		hints = hintBarBg(modalBg, "j/k", "version", "Enter", "confirm", "Esc", "cancel")
	case p.focus == pmFocusProviders:
		hints = hintBarBg(modalBg, "j/k", "navigate", "Enter", "configure", "x", "clear", "M", "close")
	default:
		hints = hintBarBg(modalBg, "j/k", "nav", "h/l", "col", "Enter", "pick", "Esc", "back")
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

	var lines []string

	if authMethod == "OAuth" {
		if p.oauthAwaitingClientID {
			// Step 1: collect the OAuth Client ID before we can open the browser.
			lines = append(lines, StyleNeonViolet.Render("  ◈ Browser Sign-In  ·  Step 1 of 2"))
			lines = append(lines, StyleFgDimStyle.Render("  Enter your OAuth Client ID below, then press Enter to open the browser."))
			switch provLabel {
			case "Gemini":
				lines = append(lines, StyleFgDimStyle.Render("  Get one at: console.cloud.google.com → APIs & Services → Credentials → OAuth 2.0 Client ID (Desktop app)"))
			default:
				lines = append(lines, StyleFgDimStyle.Render("  Create an OAuth 2.0 Client ID (Desktop app) for your registered application."))
			}
			lines = append(lines, "")

			inputBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(clrMagenta)).
				Padding(0, 1).
				Width(pmBoxW - 4).
				Render(StyleNeonCyan.Render("Client ID") + "  " + p.credInput.View())
			lines = append(lines, inputBox)
		} else {
			// Step 2: browser has been / is being opened.
			lines = append(lines, StyleNeonGreen.Render("  ◈ Browser Sign-In  ·  Step 2 of 2"))
			if p.oauthStatus != "" {
				lines = append(lines, StyleFgDimStyle.Render("  Approve access in the browser, then return here."))
			} else {
				lines = append(lines, StyleFgDimStyle.Render("  Ctrl+O to re-authorize  ·  Enter to confirm  ·  Esc to go back"))
			}
			lines = append(lines, "")

			inputBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(clrMagenta)).
				Padding(0, 1).
				Width(pmBoxW - 4).
				Render(StyleNeonViolet.Render("◈ OAuth Token") + "  " + p.credInput.View())
			lines = append(lines, inputBox)
			if p.oauthStatus != "" {
				lines = append(lines, StyleNeonOrange.Render("  "+p.oauthStatus))
			}
		}
	} else {
		lines = append(lines, StyleFgDimStyle.Render(fmt.Sprintf("  Enter %s API key", provLabel)))
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(clrMagenta)).
			Padding(0, 1).
			Width(pmBoxW - 4).
			Render(StyleNeonCyan.Render("◈ API Key") + "  " + p.credInput.View())
		lines = append(lines, inputBox)
	}
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
// Each segment fills its full column width so the grid lines are flush.
func (p ProviderMenu) renderDividers() string {
	s := lipgloss.NewStyle().Foreground(lipgloss.Color(clrViolet)).Background(lipgloss.Color(clrBg2))
	return pmRow(
		s.Render(strings.Repeat("─", pmCol1W)),
		s.Render(strings.Repeat("─", pmCol2W)),
		s.Render(strings.Repeat("─", pmCol3W)),
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

// pmRow assembles three column cells into one display line, with │ separators
// between each column for a clean grid layout.
func pmRow(col1, col2, col3 string) string {
	sep := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrViolet)).
		Background(lipgloss.Color(clrBg2)).
		Render("│")
	return pmPad(col1, pmCol1W) + sep + pmPad(col2, pmCol2W) + sep + pmPad(col3, pmCol3W)
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
