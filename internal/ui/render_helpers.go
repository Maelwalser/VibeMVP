package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// withoutField returns a copy of fields with the entry matching key removed.
func withoutField(fields []Field, key string) []Field {
	out := make([]Field, 0, len(fields))
	for _, f := range fields {
		if f.Key != key {
			out = append(out, f)
		}
	}
	return out
}

// copyFields makes a deep copy of a field slice, duplicating the Options slice
// so mutations to one copy do not affect others.
func copyFields(src []Field) []Field {
	dst := make([]Field, len(src))
	for i, f := range src {
		dst[i] = f
		if f.Options != nil {
			dst[i].Options = make([]string, len(f.Options))
			copy(dst[i].Options, f.Options)
		}
	}
	return dst
}

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

		// Capture the cursor-row background once per row so every individually-
		// styled component (line number, label, separator, value) carries it.
		// Without this the ANSI reset at the end of each styled segment clears the
		// background set by the outer row wrapper, leaving only bare spaces lit up.
		var curBG lipgloss.TerminalColor
		if isCur {
			curBG = activeCurLineStyle().GetBackground()
		}

		var lineNo string
		if isCur {
			lineNo = StyleCurLineNum.Background(curBG).Render(fmt.Sprintf("%3d ", i+1))
		} else {
			lineNo = StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		}

		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Background(curBG).Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}

		var eq string
		if isCur {
			eq = StyleEquals.Background(curBG).Render(" = ")
		} else {
			eq = StyleEquals.Render(" = ")
		}

		var valStr string
		switch {
		case insertMode && isCur && f.CanEditAsText():
			valStr = input.View()
		case f.Kind == KindSelect:
			val := f.DisplayValue()
			// Reserve 2 chars for the " ▾/▴" arrow to prevent line wrapping.
			if valW > 3 && len(val) > valW-2 {
				val = val[:valW-3] + "…"
			}
			if isCur {
				val = StyleFieldValActive.Background(curBG).Render(val)
				arrow := StyleSelectArrow.Background(curBG)
				if ddOpen {
					valStr = val + arrow.Render(" ▴")
				} else {
					valStr = val + arrow.Render(" ▾")
				}
			} else {
				val = StyleFieldVal.Render(val)
				if ddOpen {
					valStr = val + StyleSelectArrow.Render(" ▴")
				} else {
					valStr = val + StyleSelectArrow.Render(" ▾")
				}
			}
		case f.Kind == KindMultiSelect && f.ColorSwatch:
			if isCur {
				arrow := StyleSelectArrow.Background(curBG)
				arrowStr := arrow.Render(" ▾")
				if ddOpen {
					arrowStr = arrow.Render(" ▴")
				}
				if len(f.SelectedIdxs) == 0 {
					valStr = StyleFieldValActive.Background(curBG).Render("(none)") + arrowStr
				} else {
					var pieces []string
					for _, idx := range f.SelectedIdxs {
						if idx >= 0 && idx < len(f.Options) {
							hex := f.Options[idx]
							if isCustomOption(hex) {
								hex = f.CustomText
							}
							if strings.HasPrefix(hex, "#") {
								swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Background(curBG).Bold(true).Render("■")
								pieces = append(pieces, swatch+StyleFieldVal.Background(curBG).Render(" "+hex))
							} else {
								pieces = append(pieces, StyleSectionDesc.Background(curBG).Render("■")+StyleFieldVal.Background(curBG).Render(" custom"))
							}
						}
					}
					val := strings.Join(pieces, StyleSectionDesc.Background(curBG).Render(" · "))
					if lipgloss.Width(val) > valW-2 {
						val = StyleFieldVal.Background(curBG).Render(fmt.Sprintf("%d colors", len(f.SelectedIdxs)))
					}
					valStr = val + arrowStr
				}
			} else {
				arrow := StyleSelectArrow.Render(" ▾")
				if ddOpen {
					arrow = StyleSelectArrow.Render(" ▴")
				}
				if len(f.SelectedIdxs) == 0 {
					valStr = StyleFieldVal.Render("(none)") + arrow
				} else {
					var pieces []string
					for _, idx := range f.SelectedIdxs {
						if idx >= 0 && idx < len(f.Options) {
							hex := f.Options[idx]
							if isCustomOption(hex) {
								hex = f.CustomText
							}
							if strings.HasPrefix(hex, "#") {
								swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Bold(true).Render("■")
								pieces = append(pieces, swatch+StyleFieldVal.Render(" "+hex))
							} else {
								pieces = append(pieces, StyleSectionDesc.Render("■")+StyleFieldVal.Render(" custom"))
							}
						}
					}
					val := strings.Join(pieces, StyleSectionDesc.Render(" · "))
					if lipgloss.Width(val) > valW-2 {
						val = StyleFieldVal.Render(fmt.Sprintf("%d colors", len(f.SelectedIdxs)))
					}
					valStr = val + arrow
				}
			}
		case f.Kind == KindMultiSelect:
			val := f.DisplayValue()
			if val == "" {
				val = "(none)"
			}
			// Reserve 2 chars for the " ▾/▴" arrow to prevent line wrapping.
			if valW > 3 && len(val) > valW-2 {
				val = val[:valW-3] + "…"
			}
			if isCur {
				val = StyleFieldValActive.Background(curBG).Render(val)
				arrow := StyleSelectArrow.Background(curBG)
				if ddOpen {
					valStr = val + arrow.Render(" ▴")
				} else {
					valStr = val + arrow.Render(" ▾")
				}
			} else {
				val = StyleFieldVal.Render(val)
				if ddOpen {
					valStr = val + StyleSelectArrow.Render(" ▴")
				} else {
					valStr = val + StyleSelectArrow.Render(" ▾")
				}
			}
		default:
			dv := f.DisplayValue()
			if len(dv) > valW {
				dv = dv[:valW-1] + "…"
			}
			if dv == "" {
				if isCur {
					valStr = StyleSectionDesc.Background(curBG).Render("_")
				} else {
					valStr = StyleSectionDesc.Render("_")
				}
			} else if isCur {
				valStr = StyleFieldValActive.Background(curBG).Render(dv)
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

		// When typing a custom hex for a ColorSwatch field, show the currently
		// selected colors as swatches on a hint line below the text input.
		if insertMode && isCur && f.ColorSwatch && len(f.SelectedIdxs) > 0 {
			indent := strings.Repeat(" ", ddIndent)
			var pieces []string
			for _, idx := range f.SelectedIdxs {
				if idx < 0 || idx >= len(f.Options) {
					continue
				}
				hex := f.Options[idx]
				if isCustomOption(hex) {
					hex = f.CustomText
				}
				if strings.HasPrefix(hex, "#") {
					swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Bold(true).Render("■")
					pieces = append(pieces, swatch+" "+StyleSectionDesc.Render(hex))
				}
			}
			if len(pieces) > 0 {
				lines = append(lines, indent+StyleSectionDesc.Render("selected: ")+strings.Join(pieces, StyleSectionDesc.Render(" · ")))
			}
		}

		// Inject scrollable dropdown options below the active select/multiselect field.
		// Skip dropdown when in insert mode for a custom-option field (text input is active).
		if isCur && ddOpen && (f.Kind == KindSelect || f.Kind == KindMultiSelect) && !(insertMode && f.CanEditAsText()) {
			const ddMaxVisible = 8
			indent := strings.Repeat(" ", ddIndent)
			total := len(f.Options)

			// Compute scroll window centered around the highlighted option.
			start := ddOptIdx - ddMaxVisible/2
			if start+ddMaxVisible > total {
				start = total - ddMaxVisible
			}
			if start < 0 {
				start = 0
			}
			end := start + ddMaxVisible
			if end > total {
				end = total
			}

			if start > 0 {
				lines = append(lines, StyleSectionDesc.Render(fmt.Sprintf("%s↑ %d more", indent, start)))
			}

			for j := start; j < end; j++ {
				opt := f.Options[j]
				isHL := j == ddOptIdx
				var optRow string
				if f.ColorSwatch && isCustomOption(opt) {
					// Custom hex entry row: show a live preview swatch of the typed hex.
					previewHex := f.CustomText
					var swatch string
					if strings.HasPrefix(previewHex, "#") {
						swatch = lipgloss.NewStyle().Foreground(lipgloss.Color(previewHex)).Bold(true).Render("■")
					} else {
						swatch = StyleSectionDesc.Render("■")
					}
					check := "  "
					if f.IsMultiSelected(j) {
						check = StyleNeonGreen.Render("✓") + " "
					}
					label := opt
					if previewHex != "" {
						label = opt + " " + previewHex
					} else {
						label = opt + " (enter hex)"
					}
					if isHL {
						optRow = indent + StyleFieldValActive.Render("► ") + check + swatch + " " + StyleFieldValActive.Render(label)
						rw := lipgloss.Width(optRow)
						if rw < w {
							optRow += strings.Repeat(" ", w-rw)
						}
						optRow = activeCurLineStyle().Render(optRow)
					} else {
						optRow = indent + "   " + check + swatch + " " + StyleFieldVal.Render(label)
					}
				} else if f.ColorSwatch && strings.HasPrefix(opt, "#") {
					// Render a colored swatch block (foreground-only, survives cursor highlight).
					swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(opt)).Bold(true).Render("■")
					check := "  "
					if f.IsMultiSelected(j) {
						check = StyleNeonGreen.Render("✓") + " "
					}
					if isHL {
						optRow = indent + StyleFieldValActive.Render("► ") + check + swatch + " " + StyleFieldValActive.Render(opt)
						rw := lipgloss.Width(optRow)
						if rw < w {
							optRow += strings.Repeat(" ", w-rw)
						}
						optRow = activeCurLineStyle().Render(optRow)
					} else {
						optRow = indent + "   " + check + swatch + " " + StyleFieldVal.Render(opt)
					}
				} else if f.Kind == KindMultiSelect {
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

			if end < total {
				lines = append(lines, StyleSectionDesc.Render(fmt.Sprintf("%s↓ %d more", indent, total-end)))
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

// hintBarBg is like hintBar but applies bg to all text segments — for use
// inside modals where the background must be explicit on every styled span.
func hintBarBg(bg lipgloss.Color, pairs ...string) string {
	if len(pairs)%2 != 0 {
		return ""
	}
	keyStyle := StyleHelpKey.Background(bg)
	descStyle := StyleHelpDesc.Background(bg)
	var hints []string
	for i := 0; i < len(pairs); i += 2 {
		hints = append(hints, keyStyle.Render(pairs[i])+descStyle.Render(" "+pairs[i+1]))
	}
	sep := descStyle.Render("  │  ")
	prefix := lipgloss.NewStyle().Background(bg).Render("  ")
	return prefix + strings.Join(hints, sep)
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

// fieldGetSelectedSlice returns the selected option names for a KindMultiSelect
// field as a []string, or nil if the field is not found or nothing is selected.
func fieldGetSelectedSlice(fields []Field, key string) []string {
	for _, f := range fields {
		if f.Key != key {
			continue
		}
		var out []string
		for _, idx := range f.SelectedIdxs {
			if idx < len(f.Options) {
				out = append(out, f.Options[idx])
			}
		}
		return out
	}
	return nil
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
// For KindSelect fields where val is not found in Options: if a "Custom" or "Other"
// option exists it is selected and val is stored in CustomText (round-trip support).
func setFieldValue(fields []Field, key, val string) []Field {
	for i := range fields {
		if fields[i].Key != key {
			continue
		}
		fields[i].Value = val
		if fields[i].Kind == KindSelect {
			matched := false
			for j, opt := range fields[i].Options {
				if opt == val {
					fields[i].SelIdx = j
					matched = true
					break
				}
			}
			// Value not in options — try to use a Custom/Other sentinel option.
			if !matched && val != "" {
				for j, opt := range fields[i].Options {
					if isCustomOption(opt) {
						fields[i].SelIdx = j
						fields[i].CustomText = val
						break
					}
				}
			}
		}
		return fields
	}
	return fields
}

// restoreMultiSelectValue restores SelectedIdxs for a KindMultiSelect field from
// a comma-separated value string (as produced by DisplayValue). Skips unknown tokens.
func restoreMultiSelectValue(fields []Field, key, val string) []Field {
	if val == "" {
		return fields
	}
	for i := range fields {
		if fields[i].Key != key || fields[i].Kind != KindMultiSelect {
			continue
		}
		fields[i].SelectedIdxs = nil
		for _, part := range strings.Split(val, ", ") {
			part = strings.TrimSpace(part)
			found := false
			for j, opt := range fields[i].Options {
				if opt == part {
					fields[i].SelectedIdxs = append(fields[i].SelectedIdxs, j)
					found = true
					break
				}
			}
			// ColorSwatch fields: custom hexes are not in the static palette,
			// so inject them dynamically (preserving round-trip through manifest).
			if !found && fields[i].ColorSwatch && strings.HasPrefix(part, "#") {
				fields[i].AddCustomHexColor(part)
			}
		}
		fields[i].Value = val
		return fields
	}
	return fields
}

// stringSlicesEqual returns true when a and b have the same length and elements.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// nextFormIdx returns the next non-disabled field index after cur, wrapping around.
// disabled is a predicate called with the full field slice and a candidate index.
func nextFormIdx(form []Field, cur int, disabled func([]Field, int) bool) int {
	n := len(form)
	if n == 0 {
		return cur
	}
	next := (cur + 1) % n
	for next != cur && disabled(form, next) {
		next = (next + 1) % n
	}
	return next
}

// prevFormIdx returns the previous non-disabled field index before cur, wrapping around.
func prevFormIdx(form []Field, cur int, disabled func([]Field, int) bool) int {
	n := len(form)
	if n == 0 {
		return cur
	}
	prev := (cur - 1 + n) % n
	for prev != cur && disabled(form, prev) {
		prev = (prev - 1 + n) % n
	}
	return prev
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
// uniqueName returns base if not present in existing, otherwise appends an
// incrementing suffix (base1, base2, …) until a free name is found.
func uniqueName(base string, existing []string) string {
	taken := make(map[string]bool, len(existing))
	for _, n := range existing {
		taken[n] = true
	}
	if !taken[base] {
		return base
	}
	for i := 1; ; i++ {
		if candidate := fmt.Sprintf("%s%d", base, i); !taken[candidate] {
			return candidate
		}
	}
}

func newFormInput() textinput.Model {
	fi := textinput.New()
	fi.Prompt = ""
	fi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg))
	fi.CursorStyle = StyleCursor
	fi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))
	return fi
}

// newFormTextArea creates a styled multi-line textarea for use in form editors.
func newFormTextArea() textarea.Model {
	ta := textarea.New()
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.SetHeight(8)
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg)).
		Background(lipgloss.Color(clrBg)).
		BorderStyle(sharpBorder).
		BorderForeground(lipgloss.Color(clrYellow))
	ta.BlurredStyle.Base = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg)).
		Background(lipgloss.Color(clrBg)).
		BorderStyle(sharpBorder).
		BorderForeground(lipgloss.Color(clrComment))
	ta.FocusedStyle.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg)).
		Background(lipgloss.Color(clrBg))
	ta.BlurredStyle.Text = ta.FocusedStyle.Text
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color(clrBgHL))
	ta.CharLimit = 0
	return ta
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

// splitCSV splits a comma-separated string into trimmed, non-empty parts.
// Both "a, b" and "a,b" are handled identically.
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// fillTildes pads lines with vim-style tilde lines to height h.
func fillTildes(lines []string, h int) string {
	for len(lines) < h {
		lines = append(lines, StyleTilde.Render("·"))
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n") + "\n"
}

// withSidePanel places left content and a right ASCII art panel side by side,
// separated by a thin vertical rule.
//
// Cropping behaviour:
//   - Horizontal: if artW < natural art width, a center-crop window is used so
//     the middle section of symmetric art stays visible. When artW >= natural
//     width no cropping occurs and trailing spaces are stripped.
//   - Vertical: no cropping — art renders top-to-bottom; rows beyond the art's
//     line count leave the right panel empty (art need not fill the full height).
func withSidePanel(left string, artLines []string, leftW, artW, h int) string {
	leftLines := strings.Split(strings.TrimRight(left, "\n"), "\n")
	sep := StyleArtSep.Render("│")

	// Compute the natural (trimmed) width of the widest art line.
	natW := 0
	for _, l := range artLines {
		if rw := len([]rune(strings.TrimRight(l, " "))); rw > natW {
			natW = rw
		}
	}

	// Horizontal center-crop offset: positive only when panel is narrower than art.
	hOffset := 0
	if natW > artW {
		hOffset = (natW - artW) / 2
	}

	var sb strings.Builder
	for i := 0; i < h; i++ {
		// ── Left content line — padded to leftW so the separator aligns. ──────
		var leftLine string
		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		lw := lipgloss.Width(leftLine)
		if lw < leftW {
			leftLine += strings.Repeat(" ", leftW-lw)
		}

		// ── Right art line — center-crop then strip trailing spaces. ──────────
		var rightLine string
		if i < len(artLines) {
			runes := []rune(artLines[i])
			start := hOffset
			if start > len(runes) {
				start = len(runes)
			}
			end := start + artW
			if end > len(runes) {
				end = len(runes)
			}
			if start < end {
				trimmed := strings.TrimRight(string(runes[start:end]), " ")
				if trimmed != "" {
					rightLine = StyleArtPanel.Render(trimmed)
				}
			}
		}

		sb.WriteString(leftLine)
		sb.WriteString(sep)
		sb.WriteString(rightLine)
		if i < h-1 {
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

