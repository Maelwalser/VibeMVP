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
// Parameters:
//   - w         : total available width
//   - fields    : the slice of Field values to render
//   - activeIdx : which field index is currently focused
//   - insertMode: true when the active field is being edited
//   - input     : the live textinput widget (used only when insertMode && KindText)
func renderFormFields(w int, fields []Field, activeIdx int, insertMode bool, input textinput.Model) []string {
	if len(fields) == 0 {
		return nil
	}
	const labelW = 14
	const eqW = 3
	const lineNumW = 4
	valW := w - lineNumW - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	lines := make([]string, 0, len(fields))
	for i, f := range fields {
		isCur := i == activeIdx

		// Line number
		var lineNo string
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
		} else {
			lineNo = StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		}

		// Key label
		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}

		// Equals
		eq := StyleEquals.Render(" = ")

		// Value
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
			valStr = val + StyleSelectArrow.Render(" ▾")
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
			valStr = val + StyleSelectArrow.Render(" ▾")
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
			row = StyleCurLine.Render(row)
		}
		lines = append(lines, row)
	}
	return lines
}

// renderFormFieldsWithDropdown is like renderFormFields but renders an inline
// dropdown list below the active KindSelect field when ddOpen is true.
// ddOptIdx is the currently highlighted option index in the dropdown.
func renderFormFieldsWithDropdown(w int, fields []Field, activeIdx int, insertMode bool, input textinput.Model, ddOpen bool, ddOptIdx int) []string {
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
			row = StyleCurLine.Render(row)
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
						check = "✓ "
					}
					if isHL {
						optRow = indent + StyleFieldValActive.Render("▶ "+check+opt)
						rw := lipgloss.Width(optRow)
						if rw < w {
							optRow += strings.Repeat(" ", w-rw)
						}
						optRow = StyleCurLine.Render(optRow)
					} else {
						optRow = indent + StyleFieldVal.Render("  "+check+opt)
					}
				} else {
					if isHL {
						optRow = indent + StyleFieldValActive.Render("▶ "+opt)
						rw := lipgloss.Width(optRow)
						if rw < w {
							optRow += strings.Repeat(" ", w-rw)
						}
						optRow = StyleCurLine.Render(optRow)
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

// renderFormFieldsWithDisabled is like renderFormFields but also renders
// disabled (grayed-out) fields with a dash value and no highlighting.
func renderFormFieldsWithDisabled(w int, fields []Field, activeIdx int, insertMode bool, input textinput.Model, isDisabled func([]Field, int) bool) []string {
	if len(fields) == 0 {
		return nil
	}
	const labelW = 14
	const eqW = 3
	const lineNumW = 4
	valW := w - lineNumW - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	lines := make([]string, 0, len(fields))
	for i, f := range fields {
		isCur := i == activeIdx
		disabled := isDisabled(fields, i)

		var lineNo string
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
		} else {
			lineNo = StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		}

		var keyStr string
		switch {
		case disabled:
			keyStr = StyleSectionDesc.Render(f.Label)
		case isCur:
			keyStr = StyleFieldKeyActive.Render(f.Label)
		default:
			keyStr = StyleFieldKey.Render(f.Label)
		}

		eq := StyleEquals.Render(" = ")

		var valStr string
		switch {
		case disabled:
			valStr = StyleSectionDesc.Render("—")
		case insertMode && isCur && f.Kind == KindText:
			valStr = input.View()
		case f.Kind == KindSelect:
			val := f.DisplayValue()
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			valStr = val + StyleSelectArrow.Render(" ▾")
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
			valStr = val + StyleSelectArrow.Render(" ▾")
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
		if isCur && !disabled {
			raw := lipgloss.Width(row)
			if raw < w {
				row += strings.Repeat(" ", w-raw)
			}
			row = StyleCurLine.Render(row)
		}
		lines = append(lines, row)
	}
	return lines
}

// renderSubTabBar renders a horizontal sub-tab bar and returns the string.
func renderSubTabBar(labels []string, active int) string {
	var parts []string
	for i, lbl := range labels {
		if i == active {
			parts = append(parts, StyleTabActive.Render(" "+lbl+" "))
		} else {
			parts = append(parts, StyleTabInactive.Render(" "+lbl+" "))
		}
	}
	return "  " + strings.Join(parts, "")
}

// renderListItem renders one row in a list view.
func renderListItem(w int, isCur bool, arrow, name, extra string) string {
	var arrowStr, nameStr string
	if isCur {
		arrowStr = StyleCurLineNum.Render(arrow)
		nameStr = StyleFieldKeyActive.Render(name)
	} else {
		arrowStr = "  ▸ "
		nameStr = StyleFieldKey.Render(name)
	}
	row := arrowStr + nameStr
	if extra != "" {
		row += StyleSectionDesc.Render("  "+extra)
	}
	if isCur {
		raw := lipgloss.Width(row)
		if raw < w {
			row += strings.Repeat(" ", w-raw)
		}
		row = StyleCurLine.Render(row)
	}
	return row
}

// hintBar builds a hint line from key-description pairs.
func hintBar(pairs ...string) string {
	if len(pairs)%2 != 0 {
		return ""
	}
	var hints []string
	for i := 0; i < len(pairs); i += 2 {
		hints = append(hints, StyleHelpKey.Render(pairs[i])+StyleHelpDesc.Render(" "+pairs[i+1]))
	}
	return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
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
