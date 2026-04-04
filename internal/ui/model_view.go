package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)


func (m Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}
	if m.width < minTermWidth || m.height < minTermHeight {
		msg := fmt.Sprintf(" Terminal too small (%d×%d). Resize to at least %d×%d. ",
			m.width, m.height, minTermWidth, minTermHeight)
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			StyleMsgErr.Render(msg))
	}

	base := m.renderBaseView()

	if m.modal.open {
		modal := m.modal.menu.View()
		modalLines := strings.Split(modal, "\n")
		modalH := len(modalLines)
		modalW := 0
		for _, l := range modalLines {
			if w := lipgloss.Width(l); w > modalW {
				modalW = w
			}
		}
		x := (m.width - modalW) / 2
		y := (m.height - modalH) / 2
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		return placeOverlay(base, modal, x, y)
	}

	return base
}

func (m Model) renderBaseView() string {
	if m.realize.show {
		return m.renderRealizeFullScreen()
	}
	var b strings.Builder
	w := m.width
	b.WriteString(m.renderHeader(w))
	b.WriteString("\n")
	b.WriteString(m.renderContent(w))
	b.WriteString(m.renderTabBar(w))
	b.WriteString("\n")
	b.WriteString(m.renderStatusLine(w))
	b.WriteString("\n")
	b.WriteString(m.renderCmdLine(w))
	return b.String()
}

func (m Model) renderRealizeFullScreen() string {
	w, h := m.width, m.height
	contentH := h - 1
	if contentH < 1 {
		contentH = 1
	}
	content := m.realize.screen.View(w, contentH)
	hint := m.realize.screen.HintLine()
	return content + "\n" + hint
}

// renderHeader renders the top application bar with editorial typography.
//
//	VIBEMENU  ── 01  SECTION.MANIFEST  ──────────────────  02/08
func (m Model) renderHeader(w int) string {
	sec := m.sections[m.activeSection]

	// Left: VIBEMENU wordmark + section label.
	wordmark := StyleHeaderTitle.Render("VIBEMENU")
	divL := StyleHeaderDeco.Render("  ──  ")
	idx := StyleHeaderDeco.Render(fmt.Sprintf("%02d", m.activeSection+1))
	sectionLabel := StyleSectionTitle.Render("  " + strings.ToUpper(sec.ID) + ".MANIFEST")
	modMark := ""
	if m.modified {
		modMark = StyleHeaderMod.Render(" [+]")
	}
	leftSeg := " " + wordmark + divL + idx + sectionLabel + modMark

	// Right: position counter.
	pos := StyleHeaderTitle.Render(fmt.Sprintf("%02d/%02d", m.activeSection+1, len(m.sections)))
	rightSeg := pos + " "

	leftW := lipgloss.Width(leftSeg)
	rightW := lipgloss.Width(rightSeg)
	gap := w - leftW - rightW - 2
	if gap < 2 {
		gap = 2
	}
	mid := StyleHeaderDeco.Render("  " + strings.Repeat("─", gap))

	line := leftSeg + mid + rightSeg
	return StyleHeaderBar.Width(w).Render(line)
}

// renderContent renders the main editor area with an ASCII art side panel.
//
// Panel sizing strategy:
//   - Always show the panel when the terminal is ≥ 120 columns wide.
//   - Give the art as much horizontal space as possible up to its natural width.
//   - If the available space is narrower than the art, center-crop horizontally
//     so the most interesting content (the center of symmetric art) stays visible.
//   - Never crop vertically — art is shown top-to-bottom and leaves the lower
//     portion of the panel empty if shorter than the content area.
//   - Below 120 columns the panel is hidden entirely.
func (m Model) renderContent(w int) string {
	ch := m.contentHeight()

	const minFormW = 72  // minimum columns to keep the form usable
	const minArtW = 36   // minimum art panel width worth showing
	const minTermW = 120 // minimum terminal width to attempt the panel

	artW := 0
	art := SectionAsciiArt(m.activeSection)
	if natW := ArtNaturalWidth(art); natW > 0 && w >= minTermW {
		available := w - minFormW - 1 // 1 for │
		artW = natW
		if artW > available {
			artW = available // center-crop will handle the rest
		}
		if artW < minArtW {
			artW = 0 // not enough room to be useful
		}
	}

	if artW > 0 {
		contentW := w - artW - 1
		var leftContent string
		if e := m.activeEditor(); e != nil {
			leftContent = e.View(contentW, ch)
		} else {
			sec := m.sections[m.activeSection]
			leftContent = m.renderFieldList(contentW, ch, sec)
		}
		return withSidePanel(leftContent, art, contentW, artW, ch)
	}

	// No panel — render the editor at full width.
	if e := m.activeEditor(); e != nil {
		return e.View(w, ch)
	}
	sec := m.sections[m.activeSection]
	return m.renderFieldList(w, ch, sec)
}

func (m Model) renderFieldList(w, h int, sec Section) string {
	const lineNumW = 4
	const labelW = 14
	const eqW = 3
	valW := w - lineNumW - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	var lines []string
	descLine := StyleSectionDesc.Render(fmt.Sprintf("  # %s", sec.Desc))
	lines = append(lines, descLine, "")

	for i, f := range sec.Fields {
		lineNo := i + 1
		isCur := i == m.activeField

		var numStr string
		if isCur {
			numStr = StyleCurLineNum.Render(fmt.Sprintf("%3d ", lineNo))
		} else {
			numStr = StyleLineNum.Render(fmt.Sprintf("%3d ", lineNo))
		}

		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}

		eq := StyleEquals.Render(" = ")

		var valStr string
		if m.mode == ModeInsert && isCur && f.Kind == KindText {
			valStr = m.textInput.View()
		} else if f.Kind == KindSelect {
			arrow := StyleSelectArrow.Render(" ▾")
			val := f.DisplayValue()
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			valStr = val + arrow
		} else {
			dv := f.DisplayValue()
			if len(dv) > valW {
				dv = dv[:valW-1] + "…"
			}
			if dv == "" && !isCur {
				dv = StyleFieldVal.Foreground(lipgloss.Color(clrFgDim)).Render("_")
			} else if isCur {
				valStr = StyleFieldValActive.Render(dv)
			} else {
				valStr = StyleFieldVal.Render(dv)
			}
			if valStr == "" {
				valStr = StyleFieldVal.Render(dv)
			}
		}

		row := numStr + keyStr + eq + valStr
		if isCur {
			rawW := lipgloss.Width(row)
			if rawW < w {
				row += strings.Repeat(" ", w-rawW)
			}
			row = activeCurLineStyle().Render(row)
		}
		lines = append(lines, row)
	}

	return fillTildes(lines, h)
}

// renderTabBar renders the main section tab bar at the bottom of the content area.
func (m Model) renderTabBar(w int) string {
	sep := StyleTabSep.Render("│")
	sepW := lipgloss.Width(sep)
	n := len(m.sections)

	buildTabs := func(labels []string) (string, bool) {
		var parts []string
		for i, lbl := range labels {
			if i == m.activeSection {
				parts = append(parts, StyleTabActive.Render(" "+lbl+" "))
			} else {
				parts = append(parts, StyleTabInactive.Render(" "+lbl+" "))
			}
		}
		naturalW := lipgloss.Width(strings.Join(parts, sep))
		if naturalW > w {
			return "", false
		}
		extra := w - naturalW
		if extra == 0 {
			return strings.Join(parts, sep), true
		}
		perTab := extra / n
		rem := extra % n
		_ = sepW
		var expanded []string
		for i, lbl := range labels {
			pad := perTab
			if i < rem {
				pad++
			}
			padded := " " + lbl + strings.Repeat(" ", 1+pad)
			if i == m.activeSection {
				expanded = append(expanded, StyleTabActive.Render(padded))
			} else {
				expanded = append(expanded, StyleTabInactive.Render(padded))
			}
		}
		return strings.Join(expanded, sep), true
	}

	// Level 1: full Abbr labels.
	fullLabels := make([]string, n)
	for i, s := range m.sections {
		fullLabels[i] = s.Abbr
	}
	if tabs, ok := buildTabs(fullLabels); ok {
		return tabs
	}

	// Level 2: icon only (first word of Abbr).
	iconLabels := make([]string, n)
	for i, s := range m.sections {
		parts := strings.Fields(s.Abbr)
		if len(parts) > 0 {
			iconLabels[i] = parts[0]
		} else {
			iconLabels[i] = fmt.Sprintf("%d", i+1)
		}
	}
	if tabs, ok := buildTabs(iconLabels); ok {
		return tabs
	}

	// Level 3: bare index numbers.
	numLabels := make([]string, n)
	for i := range m.sections {
		numLabels[i] = fmt.Sprintf("%d", i+1)
	}
	tabs, _ := buildTabs(numLabels)
	return tabs
}

// renderStatusLine renders the vim-style status bar.
//
//	[NRM]  SECTION  ──────────────────────────────────────  02/08
func (m Model) renderStatusLine(w int) string {
	glyphs := modeSpinFrames[0]

	var modeStyle lipgloss.Style
	var modeText string
	switch m.activeMode() {
	case ModeNormal:
		modeStyle = StyleNormalMode
		modeText = "NRM"
	case ModeInsert:
		modeStyle = StyleInsertMode
		modeText = "INS"
	case ModeCommand:
		modeStyle = StyleCommandMode
		modeText = "CMD"
	}
	modeLabel := modeStyle.Render(glyphs[0] + " " + modeText + " " + glyphs[1])

	// Right segment: section + position counter.
	sec := m.sections[m.activeSection]
	sectionName := strings.ToUpper(sec.ID)
	pos := fmt.Sprintf("%02d/%02d", m.activeSection+1, len(m.sections))
	rightSeg := StyleStatusRight.Render("  " + sectionName + "  ") +
		StyleStatusSegmentPos.Render(pos + " ")

	// Centre: status message.
	msg := ""
	if m.cmd.status != "" {
		if m.cmd.isErr {
			msg = StyleMsgErr.Render(" ✗ " + m.cmd.status + " ")
		} else {
			msg = StyleMsgOK.Render(" ✓ " + m.cmd.status + " ")
		}
	}

	leftW := lipgloss.Width(modeLabel)
	rightW := lipgloss.Width(rightSeg)
	msgW := lipgloss.Width(msg)
	gapW := w - leftW - rightW - msgW
	if gapW < 1 {
		gapW = 1
	}

	line := modeLabel + StyleStatusLine.Render(strings.Repeat(" ", gapW/2)) + msg +
		StyleStatusLine.Render(strings.Repeat(" ", gapW-gapW/2)) + rightSeg
	return line
}

func (m Model) renderCmdLine(w int) string {
	if m.mode == ModeCommand {
		cursor := StyleCursor.Render(" ")
		return StyleCmdLine.Render(":"+m.cmd.buffer) + cursor
	}

	var line string
	if e := m.activeEditor(); e != nil {
		line = e.HintLine()
	} else {
		switch m.mode {
		case ModeNormal:
			hints := []string{
				StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
				StyleHelpKey.Render("i") + StyleHelpDesc.Render(" insert"),
				StyleHelpKey.Render("Tab") + StyleHelpDesc.Render(" section"),
				StyleHelpKey.Render("Enter") + StyleHelpDesc.Render(" cycle"),
				StyleHelpKey.Render(":w") + StyleHelpDesc.Render(" save"),
				StyleHelpKey.Render(":q") + StyleHelpDesc.Render(" quit"),
			}
			sep := StyleHelpDesc.Render("  │  ")
			line = "  " + strings.Join(hints, sep)
		case ModeInsert:
			line = StyleInsertMode.Render(" ── INSERT ── ") + StyleHelpDesc.Render("  Esc: normal  │  Tab: next field")
		}
	}

	if lipgloss.Width(line) > w {
		line = line[:w-1]
	}
	return line
}
