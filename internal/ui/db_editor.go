package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-mvp/internal/manifest"
)

// ── Modes & views ─────────────────────────────────────────────────────────────

type dbeMode int

const (
	dbeNormal dbeMode = iota
	dbeInsert
)

type dbeView int

const (
	dbeViewList dbeView = iota
	dbeViewForm
)

// ── DB form fields ────────────────────────────────────────────────────────────

// defaultDBForm returns a blank database source form.
func defaultDBForm() []Field {
	return []Field{
		{Key: "alias", Label: "alias         ", Kind: KindText},
		{Key: "type", Label: "type          ", Kind: KindSelect,
			Options: []string{
				"PostgreSQL", "MySQL", "SQLite",
				"MongoDB", "DynamoDB",
				"Cassandra",
				"Redis", "Memcached",
				"ClickHouse", "Elasticsearch", "other",
			},
			Value: "PostgreSQL",
		},
		{Key: "version", Label: "version       ", Kind: KindText},
		{Key: "namespace", Label: "namespace     ", Kind: KindText},
		{Key: "is_cache", Label: "is_cache      ", Kind: KindSelect,
			Options: []string{"no", "yes"}, Value: "no",
		},
		// Type-specific fields (conditionally shown)
		{Key: "ssl_mode", Label: "  ssl_mode    ", Kind: KindSelect,
			Options: []string{"require", "disable", "verify-ca", "verify-full"},
			Value:   "require",
		},
		{Key: "consistency", Label: "  consistency ", Kind: KindSelect,
			Options: []string{"LOCAL_QUORUM", "ONE", "QUORUM", "ALL", "LOCAL_ONE"},
			Value:   "LOCAL_QUORUM",
		},
		{Key: "notes", Label: "notes         ", Kind: KindText},
	}
}

// isDBFormFieldDisabled returns true when a field is gated by the current db type.
func isDBFormFieldDisabled(form []Field, idx int) bool {
	key := form[idx].Key
	var dbType string
	for _, f := range form {
		if f.Key == "type" {
			dbType = f.DisplayValue()
			break
		}
	}
	switch key {
	case "ssl_mode":
		return dbType != "PostgreSQL" && dbType != "MySQL"
	case "consistency":
		return dbType != "Cassandra"
	}
	return false
}

func nextDBFormIdx(form []Field, cur int) int {
	n := len(form)
	next := (cur + 1) % n
	for next != cur && isDBFormFieldDisabled(form, next) {
		next = (next + 1) % n
	}
	return next
}

func prevDBFormIdx(form []Field, cur int) int {
	n := len(form)
	prev := (cur - 1 + n) % n
	for prev != cur && isDBFormFieldDisabled(form, prev) {
		prev = (prev - 1 + n) % n
	}
	return prev
}

func dbFormFromSource(src manifest.DBSourceDef) []Field {
	f := defaultDBForm()
	setVal := func(key, val string) {
		for i := range f {
			if f[i].Key != key {
				continue
			}
			f[i].Value = val
			if f[i].Kind == KindSelect {
				for j, opt := range f[i].Options {
					if opt == val {
						f[i].SelIdx = j
						break
					}
				}
			}
			return
		}
	}
	setVal("alias", src.Alias)
	if src.Type != "" {
		setVal("type", string(src.Type))
	}
	setVal("version", src.Version)
	setVal("namespace", src.Namespace)
	if src.IsCache {
		setVal("is_cache", "yes")
	}
	setVal("notes", src.Notes)
	return f
}

func dbFormToSource(form []Field) manifest.DBSourceDef {
	get := func(key string) string {
		for _, f := range form {
			if f.Key == key {
				return f.DisplayValue()
			}
		}
		return ""
	}
	src := manifest.DBSourceDef{
		Alias:     get("alias"),
		Type:      manifest.DatabaseType(get("type")),
		Version:   get("version"),
		Namespace: get("namespace"),
		IsCache:   get("is_cache") == "yes",
		Notes:     get("notes"),
	}
	return src
}

// ── DBEditor ─────────────────────────────────────────────────────────────────

// DBEditor manages a list of named database/cache sources in the DATABASES section.
type DBEditor struct {
	Sources []manifest.DBSourceDef

	view   dbeView
	srcIdx int

	internalMode dbeMode

	dbForm    []Field
	formIdx   int
	formInput textinput.Model

	width int
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
	if db.internalMode == dbeInsert {
		return ModeInsert
	}
	return ModeNormal
}

// HintLine returns context-sensitive key hints.
func (db DBEditor) HintLine() string {
	if db.internalMode == dbeInsert {
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
		hints := []string{
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
			StyleHelpKey.Render("i") + StyleHelpDesc.Render(" edit text"),
			StyleHelpKey.Render("Space") + StyleHelpDesc.Render(" cycle option"),
			StyleHelpKey.Render("b/Esc") + StyleHelpDesc.Render(" save & back"),
		}
		return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
	}
	return ""
}

// ── Update ────────────────────────────────────────────────────────────────────

func (db DBEditor) Update(msg tea.Msg) (DBEditor, tea.Cmd) {
	switch db.internalMode {
	case dbeInsert:
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
			db.dbForm[db.formIdx].Value = db.formInput.Value()
			db.internalMode = dbeNormal
			db.formInput.Blur()
			return db, nil

		case "tab":
			db.dbForm[db.formIdx].Value = db.formInput.Value()
			db.formIdx = nextDBFormIdx(db.dbForm, db.formIdx)
			f := db.dbForm[db.formIdx]
			if f.Kind == KindSelect {
				db.internalMode = dbeNormal
				db.formInput.Blur()
				return db, nil
			}
			db.formInput.SetValue(f.Value)
			db.formInput.CursorEnd()
			return db, db.formInput.Focus()

		case "shift+tab":
			db.dbForm[db.formIdx].Value = db.formInput.Value()
			db.formIdx = prevDBFormIdx(db.dbForm, db.formIdx)
			f := db.dbForm[db.formIdx]
			if f.Kind == KindSelect {
				db.internalMode = dbeNormal
				db.formInput.Blur()
				return db, nil
			}
			db.formInput.SetValue(f.Value)
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
		db.dbForm = dbFormFromSource(db.Sources[db.srcIdx])
		db.formIdx = 0
		db.view = dbeViewForm
		// Auto-focus alias field
		return db.enterDBFormInsert()

	case "enter", "l", "right":
		if n > 0 {
			db.dbForm = dbFormFromSource(db.Sources[db.srcIdx])
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
			f.CycleNext()
		} else {
			return db.enterDBFormInsert()
		}
	case "H", "shift+left":
		f := &db.dbForm[db.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if db.dbForm[db.formIdx].Kind == KindText {
			return db.enterDBFormInsert()
		}
	case "b", "esc":
		db.saveFormBack()
		db.view = dbeViewList
	}
	return db, nil
}

func (db DBEditor) enterDBFormInsert() (DBEditor, tea.Cmd) {
	f := db.dbForm[db.formIdx]
	if f.Kind != KindText {
		return db, nil
	}
	db.internalMode = dbeInsert
	db.formInput.SetValue(f.Value)
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

	if len(db.Sources) == 0 {
		lines = append(lines, StyleSectionDesc.Render("  (no databases yet — press 'a' to add one)"))
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
			lines = append(lines, row)
		}
	}

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

	for i, f := range db.dbForm {
		isCur := i == db.formIdx
		disabled := isDBFormFieldDisabled(db.dbForm, i)

		lineNo := StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
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
		case db.internalMode == dbeInsert && isCur && f.Kind == KindText:
			valStr = db.formInput.View()
		case f.Kind == KindSelect:
			val := f.DisplayValue()
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
			row = activeCurLineStyle().Render(row)
		}
		lines = append(lines, row)
	}

	return fillTildes(lines, h)
}
