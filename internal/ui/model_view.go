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
	if m.arch.show {
		return m.renderArchFullScreen()
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

func (m Model) renderArchFullScreen() string {
	w, h := m.width, m.height
	contentH := h - 1
	if contentH < 1 {
		contentH = 1
	}
	content := m.arch.screen.View(w, contentH)
	hint := m.arch.screen.HintLine()
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

// renderContent renders the main editor area with a description side panel.
//
// Panel sizing strategy:
//   - Always show a 44-column panel when the terminal is ≥ 120 columns wide.
//   - When a KindSelect field is highlighted and a description exists, show
//     the field/option description panel.
//   - Otherwise show a section overview panel.
//   - Below 120 columns the panel is hidden entirely.
func (m Model) renderContent(w int) string {
	ch := m.contentHeight()

	const minFormW = 72   // minimum columns to keep the form usable
	const descPanelW = 44 // fixed width for the right panel
	const minTermW = 120  // minimum terminal width to attempt the panel

	if w >= minTermW && (w-minFormW-1) >= descPanelW {
		contentW := w - descPanelW - 1
		label, value, desc := m.getActiveFieldDescription()
		var panelLines []string
		if desc != "" {
			panelLines = FormatDescriptionPanel(label, value, desc, descPanelW, ch)
		} else {
			panelLines = FormatSectionPanel(m.activeSection, descPanelW, ch)
		}
		return withDescriptionPanel(m.renderLeft(contentW, ch), panelLines, contentW, descPanelW, ch)
	}

	// Narrow terminal — render the editor at full width.
	return m.renderLeft(w, ch)
}

// renderLeft renders the left-side content area (editor or field list).
func (m Model) renderLeft(w, h int) string {
	if e := m.activeEditor(); e != nil {
		return e.View(w, h)
	}
	sec := m.sections[m.activeSection]
	return m.renderFieldList(w, h, sec)
}

// getActiveFieldDescription returns display info for the currently highlighted
// KindSelect field. Returns ("", "", "") when no description is registered.
func (m Model) getActiveFieldDescription() (label, value, desc string) {
	var f *Field
	switch m.activeSection {
	case 1:
		f = m.backendEditor.CurrentField()
	case 2:
		f = m.dataTabEditor.CurrentField()
	case 3:
		f = m.contractsEditor.CurrentField()
	case 4:
		f = m.frontendEditor.CurrentField()
	case 5:
		f = m.infraEditor.CurrentField()
	case 6:
		f = m.crossCutEditor.CurrentField()
	case 7:
		f = m.realizeEditor.CurrentField()
	}
	if f == nil || (f.Kind != KindSelect && f.Kind != KindMultiSelect) {
		return "", "", ""
	}
	d := GetOptionDescription(f.Key, f.Value)
	if d == "" {
		return "", "", ""
	}
	return strings.TrimSpace(f.Label), f.Value, d
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
	rightSeg := StyleStatusRight.Render("  "+sectionName+"  ") +
		StyleStatusSegmentPos.Render(pos+" ")

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
		line = lipgloss.NewStyle().MaxWidth(w).Render(line)
	}
	return line
}
