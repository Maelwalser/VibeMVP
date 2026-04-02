package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// renderFormFields renders a list of Fields into display lines using the
// shared vim-style form layout. It is the canonical rendering helper for all
// editors and can be called from any sub-editor's view methods.
//
// When ddOpen is true, an inline scrollable dropdown is injected below the
// active KindSelect/KindMultiSelect field; ddOptIdx is the highlighted index.
// Pass ddOpen=false and ddOptIdx=0 for plain form rendering without dropdowns.
func renderFormFields(w int, fields []Field, activeIdx int, insertMode bool, input textinput.Model, ddOpen bool, ddOptIdx int) []string {
	if len(fields) == 0 {
		return nil
	}
	const labelW = 14
	const eqW = 3
	const lineNumW = 4
	const ddIndent = lineNumW + labelW + eqW // 21 spaces to align with value column
	valW := w - lineNumW - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	lines := make([]string, 0, len(fields))
	for i, f := range fields {
		isCur := i == activeIdx

		var lineNo string
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
		} else {
			lineNo = StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		}

		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}

		eq := StyleEquals.Render(" = ")

		var valStr string
		switch {
		case insertMode && isCur && f.Kind == KindText:
			valStr = input.View()
		case f.Kind == KindSelect:
			val := f.DisplayValue()
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			if isCur && ddOpen {
				valStr = val + StyleSelectArrow.Render(" ▴")
			} else {
				valStr = val + StyleSelectArrow.Render(" ▾")
			}
		case f.Kind == KindMultiSelect:
			val := f.DisplayValue()
			if val == "" {
				val = "(none)"
			}
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			if isCur && ddOpen {
				valStr = val + StyleSelectArrow.Render(" ▴")
			} else {
				valStr = val + StyleSelectArrow.Render(" ▾")
			}
		default:
			dv := f.DisplayValue()
			if len(dv) > valW {
				dv = dv[:valW-1] + "…"
			}
			if dv == "" {
				valStr = StyleSectionDesc.Render("_")
			} else if isCur {
				valStr = StyleFieldValActive.Render(dv)
			} else {
				valStr = StyleFieldVal.Render(dv)
			}
		}

		row := lineNo + keyStr + eq + valStr
		if isCur {
			raw := lipgloss.Width(row)
			if raw < w {
				row += strings.Repeat(" ", w-raw)
			}
			row = activeCurLineStyle().Render(row)
		}
		lines = append(lines, row)

		// Inject scrollable dropdown options below the active select/multiselect field
		if isCur && ddOpen && (f.Kind == KindSelect || f.Kind == KindMultiSelect) {
			indent := strings.Repeat(" ", ddIndent)
			for j, opt := range f.Options {
				isHL := j == ddOptIdx
				var optRow string
				if f.Kind == KindMultiSelect {
					check := "  "
					if f.IsMultiSelected(j) {
						check = StyleNeonGreen.Render("✓") + " "
					}
					if isHL {
						optRow = indent + StyleFieldValActive.Render("► "+check+opt)
						rw := lipgloss.Width(optRow)
						if rw < w {
							optRow += strings.Repeat(" ", w-rw)
						}
						optRow = activeCurLineStyle().Render(optRow)
					} else {
						optRow = indent + StyleFieldVal.Render("  "+check+opt)
					}
				} else {
					if isHL {
						optRow = indent + StyleFieldValActive.Render("► "+opt)
						rw := lipgloss.Width(optRow)
						if rw < w {
							optRow += strings.Repeat(" ", w-rw)
						}
						optRow = activeCurLineStyle().Render(optRow)
					} else {
						optRow = indent + StyleFieldVal.Render("  "+opt)
					}
				}
				lines = append(lines, optRow)
			}
		}
	}
	return lines
}

// renderSubTabBar renders a horizontal sub-tab bar scaled to fit within w columns.
// Three levels of fidelity:
//  1. Full labels — all tabs with their text labels, expanded to fill w.
//  2. Compact — active tab keeps its label; others shrink to their 1-based index, expanded to fill w.
//  3. Minimal — "[N/total] LABEL" single-item indicator.
func renderSubTabBar(labels []string, active, w int) string {
	sep := StyleTabSep.Render("│")
	n := len(labels)
	const margin = 2 // leading "  " prefix

	// build renders lbls as a tab bar. If the natural width fits within w,
	// extra space is distributed evenly among tabs so the bar fills the full width.
	// Returns (rendered, fits).
	build := func(lbls []string) (string, bool) {
		var parts []string
		for i, lbl := range lbls {
			if i == active {
				parts = append(parts, StyleTabActive.Render(" "+lbl+" "))
			} else {
				parts = append(parts, StyleTabInactive.Render(" "+lbl+" "))
			}
		}
		naturalW := margin + lipgloss.Width(strings.Join(parts, sep))
		if w > 0 && naturalW > w {
			return "", false
		}
		if w <= 0 || naturalW >= w {
			return strings.Repeat(" ", margin) + strings.Join(parts, sep), true
		}
		// Distribute extra space inside each tab's right padding.
		extra := w - naturalW
		perTab := extra / n
		rem := extra % n
		var expanded []string
		for i, lbl := range lbls {
			pad := perTab
			if i < rem {
				pad++
			}
			padded := " " + lbl + strings.Repeat(" ", 1+pad)
			if i == active {
				expanded = append(expanded, StyleTabActive.Render(padded))
			} else {
				expanded = append(expanded, StyleTabInactive.Render(padded))
			}
		}
		return strings.Repeat(" ", margin) + strings.Join(expanded, sep), true
	}

	// Level 1: full labels.
	if result, ok := build(labels); ok {
		return result
	}

	// Level 2: active tab shows label, others collapse to index number.
	compact := make([]string, n)
	for i := range labels {
		if i == active {
			compact[i] = labels[i]
		} else {
			compact[i] = fmt.Sprintf("%d", i+1)
		}
	}
	if result, ok := build(compact); ok {
		return result
	}

	// Level 3: single "[N/total] LABEL" indicator — always fits.
	pos := fmt.Sprintf("[%d/%d]", active+1, n)
	lbl := labels[active]
	maxLblW := w - lipgloss.Width("  "+pos+" ") - 4
	if maxLblW > 0 && len([]rune(lbl)) > maxLblW {
		lbl = string([]rune(lbl)[:maxLblW-1]) + "…"
	}
	return "  " + StyleTabInactive.Render(" "+pos+" ") + StyleTabActive.Render(" "+lbl+" ")
}

// renderListItem renders one row in a list view.
func renderListItem(w int, isCur bool, arrow, name, extra string) string {
	var arrowStr, nameStr string
	if isCur {
		arrowStr = StyleCurLineNum.Render(arrow)
		nameStr = StyleFieldKeyActive.Render(name)
	} else {
		arrowStr = StyleFgDimStyle.Render("  ► ")
		nameStr = StyleFieldKey.Render(name)
	}
	row := arrowStr + nameStr
	if extra != "" {
		row += StyleSectionDesc.Render("  " + extra)
	}
	// Truncate row to terminal width to prevent horizontal overflow.
	if w > 0 {
		if rw := lipgloss.Width(row); rw > w {
			row = lipgloss.NewStyle().MaxWidth(w).Render(row)
		}
	}
	if isCur {
		raw := lipgloss.Width(row)
		if raw < w {
			row += strings.Repeat(" ", w-raw)
		}
		row = activeCurLineStyle().Render(row)
	}
	return row
}

// StyleFgDimStyle is an inline style for dim inactive list arrows.
var StyleFgDimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))

// hintBar builds a hint line from key-description pairs.
func hintBar(pairs ...string) string {
	if len(pairs)%2 != 0 {
		return ""
	}
	var hints []string
	for i := 0; i < len(pairs); i += 2 {
		hints = append(hints, StyleHelpKey.Render(pairs[i])+StyleHelpDesc.Render(" "+pairs[i+1]))
	}
	sep := StyleHelpDesc.Render("  │  ")
	return "  " + strings.Join(hints, sep)
}

// fieldGet returns the DisplayValue for the field with the given key in a slice.
func fieldGet(fields []Field, key string) string {
	for _, f := range fields {
		if f.Key == key {
			return f.DisplayValue()
		}
	}
	return ""
}

// fieldGetMulti returns the comma-separated DisplayValue for a KindMultiSelect field,
// or the plain DisplayValue for any other kind.
func fieldGetMulti(fields []Field, key string) string {
	for _, f := range fields {
		if f.Key == key {
			return f.DisplayValue()
		}
	}
	return ""
}

// setFieldValue sets the value (and SelIdx for select fields) for the field
// with the given key in a slice, returning the modified slice.
func setFieldValue(fields []Field, key, val string) []Field {
	for i := range fields {
		if fields[i].Key != key {
			continue
		}
		fields[i].Value = val
		if fields[i].Kind == KindSelect {
			for j, opt := range fields[i].Options {
				if opt == val {
					fields[i].SelIdx = j
					break
				}
			}
		}
		return fields
	}
	return fields
}

// parseVimCount converts a digit buffer (e.g. "3", "12") to an integer count.
// Returns 1 when the buffer is empty. Caps at 999 for sanity.
func parseVimCount(buf string) int {
	if buf == "" {
		return 1
	}
	n := 0
	for _, c := range buf {
		if c < '0' || c > '9' {
			return 1
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return 1
	}
	if n > 999 {
		return 999
	}
	return n
}

// newFormInput creates a standard textinput for use in form editors.
func newFormInput() textinput.Model {
	fi := textinput.New()
	fi.Prompt = ""
	fi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg))
	fi.CursorStyle = StyleCursor
	fi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))
	return fi
}

// placeOverlay paints fg on top of bg at position (x, y), where x and y are
// zero-based visible-column and line indices. Lines outside bg bounds are
// skipped. The portion of each bg line to the right of the overlay is
// preserved as plain (un-styled) text so the overlay always looks clean.
func placeOverlay(bg, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	for i, fgLine := range fgLines {
		idx := y + i
		if idx < 0 || idx >= len(bgLines) {
			continue
		}
		fgW := lipgloss.Width(fgLine)
		bgW := lipgloss.Width(bgLines[idx])

		// Left part: bg up to column x (ANSI-aware truncation).
		left := lipgloss.NewStyle().MaxWidth(x).Render(bgLines[idx])
		leftW := lipgloss.Width(left)
		if leftW < x {
			left += strings.Repeat(" ", x-leftW)
		}

		// Right part: plain text after the overlay's right edge.
		right := ""
		rightStart := x + fgW
		if rightStart < bgW {
			plain := stripANSI(bgLines[idx])
			runes := []rune(plain)
			if rightStart < len(runes) {
				right = string(runes[rightStart:])
			}
		}

		bgLines[idx] = left + fgLine + right
	}

	return strings.Join(bgLines, "\n")
}

// appendViewport applies a scrolling viewport to a list-with-header layout.
// It preserves the first headerH lines unchanged and applies viewportSlice to
// the remaining item lines, keeping the item at itemIdx visible within the
// available height. Returns the combined fixed+scrolled slice.
func appendViewport(lines []string, headerH, itemIdx, available int) []string {
	if available <= 0 || len(lines) <= available {
		return lines
	}
	if headerH > len(lines) {
		headerH = len(lines)
	}
	header := lines[:headerH]
	items := viewportSlice(lines[headerH:], itemIdx, available-headerH)
	result := make([]string, 0, len(header)+len(items))
	result = append(result, header...)
	result = append(result, items...)
	return result
}

// viewportSlice returns a height-bounded window of lines keeping activeLine visible.
// The active line is kept roughly centered. Returns lines unchanged if height <= 0
// or len(lines) <= height.
func viewportSlice(lines []string, activeLine, height int) []string {
	if height <= 0 || len(lines) <= height {
		return lines
	}
	if activeLine < 0 {
		activeLine = 0
	}
	if activeLine >= len(lines) {
		activeLine = len(lines) - 1
	}
	half := height / 2
	start := activeLine - half
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
		start = end - height
		if start < 0 {
			start = 0
		}
	}
	return lines[start:end]
}

// stripANSI removes ANSI CSI escape sequences from s, returning plain text.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip CSI sequence: ESC [ ... <terminator>
			i += 2
			for i < len(s) && (s[i] < 0x40 || s[i] > 0x7e) {
				i++
			}
			if i < len(s) {
				i++ // consume terminator
			}
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}
