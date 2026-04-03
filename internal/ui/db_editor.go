package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
)

// ── Modes & views ─────────────────────────────────────────────────────────────

type dbeView int

const (
	dbeViewList dbeView = iota
	dbeViewForm
)

// ── DBEditor ─────────────────────────────────────────────────────────────────

// DBEditor manages a list of named database/cache sources in the DATABASES section.
type DBEditor struct {
	Sources []manifest.DBSourceDef

	view   dbeView
	srcIdx int

	internalMode Mode

	dbForm    []Field
	formIdx   int
	formInput textinput.Model

	dd DropdownState

	width        int
	envNames     []string // injected from InfraPillar.Environments
}

// SetEnvironmentNames injects environment names for the environment selector field
// in the database form. A no-op when unchanged.
func (db *DBEditor) SetEnvironmentNames(names []string) {
	db.envNames = names
	applyEnvNamesToDBForm(db.dbForm, names)
}

func newDBEditor() DBEditor {
	fi := textinput.New()
	fi.Prompt = ""
	fi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg))
	fi.CursorStyle = StyleCursor
	fi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))

	return DBEditor{
		Sources:   []manifest.DBSourceDef{},
		formInput: fi,
	}
}

// Mode returns the equivalent app-level Mode for the parent status bar.
func (db DBEditor) Mode() Mode {
	if db.internalMode == ModeInsert {
		return ModeInsert
	}
	return ModeNormal
}

// HintLine returns context-sensitive key hints.
func (db DBEditor) HintLine() string {
	if db.internalMode == ModeInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch db.view {
	case dbeViewList:
		hints := []string{
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
			StyleHelpKey.Render("a") + StyleHelpDesc.Render(" add database"),
			StyleHelpKey.Render("d") + StyleHelpDesc.Render(" delete"),
			StyleHelpKey.Render("Enter") + StyleHelpDesc.Render(" edit"),
			StyleHelpKey.Render(":w") + StyleHelpDesc.Render(" save"),
		}
		return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
	case dbeViewForm:
		if db.dd.Open {
			hints := []string{
				StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
				StyleHelpKey.Render("Enter/Space") + StyleHelpDesc.Render(" select"),
				StyleHelpKey.Render("Esc") + StyleHelpDesc.Render(" cancel"),
			}
			return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
		}
		hints := []string{
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
			StyleHelpKey.Render("i") + StyleHelpDesc.Render(" edit text"),
			StyleHelpKey.Render("Enter/Space") + StyleHelpDesc.Render(" dropdown"),
			StyleHelpKey.Render("b/Esc") + StyleHelpDesc.Render(" save & back"),
		}
		return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
	}
	return ""
}

// ── Update ────────────────────────────────────────────────────────────────────

func (db DBEditor) Update(msg tea.Msg) (DBEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		db.width = wsz.Width
		db.formInput.Width = wsz.Width - 22
		return db, nil
	}
	switch db.internalMode {
	case ModeInsert:
		return db.updateInsert(msg)
	default:
		return db.updateNormal(msg)
	}
}

func (db DBEditor) updateInsert(msg tea.Msg) (DBEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			db.dbForm[db.formIdx].SaveTextInput(db.formInput.Value())
			db.internalMode = ModeNormal
			db.formInput.Blur()
			return db, nil

		case "tab":
			db.dbForm[db.formIdx].SaveTextInput(db.formInput.Value())
			if next := nextEditableIdx(db.dbForm, db.formIdx); next >= 0 {
				db.formIdx = next
			} else {
				db.formIdx = nextDBFormIdx(db.dbForm, db.formIdx)
			}
			f := db.dbForm[db.formIdx]
			if !f.CanEditAsText() {
				db.formInput.Blur()
				return db, nil
			}
			db.formInput.SetValue(f.TextInputValue())
			db.formInput.CursorEnd()
			return db, db.formInput.Focus()

		case "shift+tab":
			db.dbForm[db.formIdx].SaveTextInput(db.formInput.Value())
			if prev := prevEditableIdx(db.dbForm, db.formIdx); prev >= 0 {
				db.formIdx = prev
			} else {
				db.formIdx = prevDBFormIdx(db.dbForm, db.formIdx)
			}
			f := db.dbForm[db.formIdx]
			if !f.CanEditAsText() {
				db.formInput.Blur()
				return db, nil
			}
			db.formInput.SetValue(f.TextInputValue())
			db.formInput.CursorEnd()
			return db, db.formInput.Focus()
		}
	}
	var cmd tea.Cmd
	db.formInput, cmd = db.formInput.Update(msg)
	return db, cmd
}

func (db DBEditor) updateNormal(msg tea.Msg) (DBEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return db, nil
	}
	if db.dd.Open && db.view == dbeViewForm {
		return db.updateDBDropdown(key)
	}
	switch db.view {
	case dbeViewList:
		return db.updateNormalList(key)
	case dbeViewForm:
		return db.updateNormalForm(key)
	}
	return db, nil
}

func (db DBEditor) updateNormalList(key tea.KeyMsg) (DBEditor, tea.Cmd) {
	n := len(db.Sources)
	switch key.String() {
	case "j", "down":
		if n > 0 && db.srcIdx < n-1 {
			db.srcIdx++
		}
	case "k", "up":
		if db.srcIdx > 0 {
			db.srcIdx--
		}
	case "g":
		db.srcIdx = 0
	case "G":
		if n > 0 {
			db.srcIdx = n - 1
		}
	case "d":
		if n > 0 {
			db.Sources = append(db.Sources[:db.srcIdx], db.Sources[db.srcIdx+1:]...)
			if db.srcIdx >= len(db.Sources) && db.srcIdx > 0 {
				db.srcIdx--
			}
		}
	case "a":
		db.Sources = append(db.Sources, manifest.DBSourceDef{
			Type: manifest.DBPostgres,
		})
		db.srcIdx = len(db.Sources) - 1
		db.dbForm = dbFormFromSourceWithEnvs(db.Sources[db.srcIdx], db.envNames)
		existing := make([]string, 0, len(db.Sources)-1)
		for i, s := range db.Sources {
			if i != db.srcIdx {
				existing = append(existing, s.Alias)
			}
		}
		db.dbForm = setFieldValue(db.dbForm, "alias", uniqueName("database", existing))
		db.formIdx = 0
		db.view = dbeViewForm
		// Auto-focus alias field
		return db.enterDBFormInsert()

	case "enter", "l", "right":
		if n > 0 {
			db.dbForm = dbFormFromSourceWithEnvs(db.Sources[db.srcIdx], db.envNames)
			db.formIdx = 0
			db.view = dbeViewForm
		}
	}
	return db, nil
}

func (db DBEditor) updateNormalForm(key tea.KeyMsg) (DBEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		db.formIdx = nextDBFormIdx(db.dbForm, db.formIdx)
	case "k", "up":
		db.formIdx = prevDBFormIdx(db.dbForm, db.formIdx)
	case "g":
		db.formIdx = 0
	case "G":
		db.formIdx = len(db.dbForm) - 1
	case "enter", " ":
		f := &db.dbForm[db.formIdx]
		if f.Kind == KindSelect {
			db.dd.Open = true
			db.dd.OptIdx = f.SelIdx
		} else {
			return db.enterDBFormInsert()
		}
	case "H", "shift+left":
		f := &db.dbForm[db.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if db.dbForm[db.formIdx].CanEditAsText() {
			return db.enterDBFormInsert()
		}
	case "b", "esc":
		db.saveFormBack()
		db.view = dbeViewList
	}
	return db, nil
}

func (db DBEditor) updateDBDropdown(key tea.KeyMsg) (DBEditor, tea.Cmd) {
	if db.formIdx >= len(db.dbForm) {
		db.dd.Open = false
		return db, nil
	}
	f := &db.dbForm[db.formIdx]
	db.dd.OptIdx = NavigateDropdown(key.String(), db.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = db.dd.OptIdx
		if db.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[db.dd.OptIdx]
		}
		db.dd.Open = false
		if f.PrepareCustomEntry() {
			return db.enterDBFormInsert()
		}
	case "esc", "b":
		db.dd.Open = false
	}
	return db, nil
}

func (db DBEditor) enterDBFormInsert() (DBEditor, tea.Cmd) {
	f := db.dbForm[db.formIdx]
	if !f.CanEditAsText() {
		return db, nil
	}
	db.internalMode = ModeInsert
	db.formInput.SetValue(f.TextInputValue())
	db.formInput.Width = db.width - 22
	db.formInput.CursorEnd()
	return db, db.formInput.Focus()
}

func (db *DBEditor) saveFormBack() {
	if db.srcIdx >= len(db.Sources) {
		return
	}
	db.Sources[db.srcIdx] = dbFormToSource(db.dbForm)
}

// ── View ──────────────────────────────────────────────────────────────────────

func (db DBEditor) View(w, h int) string {
	db.width = w
	db.formInput.Width = w - 22
	switch db.view {
	case dbeViewList:
		return db.viewList(w, h)
	case dbeViewForm:
		return db.viewForm(w, h)
	}
	return ""
}

func (db DBEditor) viewList(w, h int) string {
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Database sources — a: add  d: delete  Enter: configure"),
		"",
	)

	const dbListHeaderH = 2
	var itemLines []string
	if len(db.Sources) == 0 {
		itemLines = append(itemLines, StyleSectionDesc.Render("  (no databases yet — press 'a' to add one)"))
	} else {
		for i, src := range db.Sources {
			isCur := i == db.srcIdx

			arrow := "  ▸ "
			alias := src.Alias
			if alias == "" {
				alias = "(unnamed)"
			}
			if isCur {
				arrow = StyleCurLineNum.Render("  ▶ ")
				alias = StyleFieldKeyActive.Render(alias)
			} else {
				alias = StyleFieldKey.Render(alias)
			}

			typeStr := string(src.Type)
			var tags []string
			if src.IsCache {
				tags = append(tags, StyleMsgOK.Render("CACHE"))
			}
			if src.Version != "" {
				tags = append(tags, StyleSectionDesc.Render("v"+src.Version))
			}
			if src.Namespace != "" {
				tags = append(tags, StyleSectionDesc.Render(src.Namespace))
			}
			tagStr := ""
			if len(tags) > 0 {
				tagStr = "  " + strings.Join(tags, " ")
			}

			pad := max(1, 20-len(src.Alias))
			typeRendered := StyleFieldVal.Render(fmt.Sprintf("%-14s", typeStr))
			row := arrow + alias + strings.Repeat(" ", pad) + typeRendered + tagStr

			if isCur {
				raw := lipgloss.Width(row)
				if raw < w {
					row += strings.Repeat(" ", w-raw)
				}
				row = activeCurLineStyle().Render(row)
			}
			itemLines = append(itemLines, row)
		}
	}

	itemLines = viewportSlice(itemLines, db.srcIdx, h-dbListHeaderH)
	lines = append(lines, itemLines...)
	return fillTildes(lines, h)
}

func (db DBEditor) viewForm(w, h int) string {
	srcAlias := "(new database)"
	if db.srcIdx < len(db.Sources) && db.Sources[db.srcIdx].Alias != "" {
		srcAlias = db.Sources[db.srcIdx].Alias
	}

	var lines []string
	breadcrumb := StyleSectionDesc.Render("  ← ") + StyleFieldKey.Render(srcAlias)
	lines = append(lines, breadcrumb, "")

	const labelW = 14
	const eqW = 3
	valW := w - 4 - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	visIdx := 0
	for i, f := range db.dbForm {
		if isDBFormFieldDisabled(db.dbForm, i) {
			continue
		}
		isCur := i == db.formIdx
		visIdx++

		lineNo := StyleLineNum.Render(fmt.Sprintf("%3d ", visIdx))
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", visIdx))
		}

		var keyStr string
		switch {
		case isCur:
			keyStr = StyleFieldKeyActive.Render(f.Label)
		default:
			keyStr = StyleFieldKey.Render(f.Label)
		}

		eq := StyleEquals.Render(" = ")

		var valStr string
		switch {
		case db.internalMode == ModeInsert && isCur && f.Kind == KindText:
			valStr = db.formInput.View()
		case f.Kind == KindSelect:
			val := f.DisplayValue()
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			if isCur && db.dd.Open {
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

		// Inject dropdown options below the active KindSelect field
		if isCur && db.dd.Open && f.Kind == KindSelect {
			const ddIndent = 4 + 14 + 3 // lineNumW + labelW + eqW
			indent := strings.Repeat(" ", ddIndent)
			for j, opt := range f.Options {
				isHL := j == db.dd.OptIdx
				var optRow string
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
				lines = append(lines, optRow)
			}
		}
	}

	return fillTildes(lines, h)
}
