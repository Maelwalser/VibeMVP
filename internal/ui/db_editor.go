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
		// Security / network integrity (conditionally shown by type)
		{Key: "ssl_mode", Label: "  ssl_mode    ", Kind: KindSelect,
			Options: []string{"require", "disable", "verify-ca", "verify-full"},
			Value:   "require",
		},
		{Key: "consistency", Label: "  consistency ", Kind: KindSelect,
			Options: []string{"strong", "eventual", "LOCAL_QUORUM", "ONE", "QUORUM", "ALL", "LOCAL_ONE"},
			Value:   "strong",
		},
		// Availability topology (conditionally shown by type)
		{Key: "replication", Label: "  replication ", Kind: KindSelect,
			Options: []string{"single-node", "primary-replica", "multi-region"},
			Value:   "single-node",
		},
		// Connection pooling
		{Key: "pool_min", Label: "  pool_min    ", Kind: KindText},
		{Key: "pool_max", Label: "  pool_max    ", Kind: KindText},
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
		// Only relational databases support explicit SSL mode configuration
		return dbType != "PostgreSQL" && dbType != "MySQL"
	case "consistency":
		// Distributed DBs with tunable consistency
		return dbType != "Cassandra" && dbType != "MongoDB" && dbType != "DynamoDB"
	case "replication":
		// Cache stores and SQLite don't have meaningful replication topology options
		return dbType == "Redis" || dbType == "Memcached" || dbType == "SQLite"
	case "pool_min", "pool_max":
		// Connection pooling doesn't apply to cache stores
		return dbType == "Redis" || dbType == "Memcached"
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
	setVal("ssl_mode", src.SSLMode)
	setVal("consistency", src.Consistency)
	setVal("replication", src.Replication)
	if src.PoolMinSize > 0 {
		setVal("pool_min", fmt.Sprintf("%d", src.PoolMinSize))
	}
	if src.PoolMaxSize > 0 {
		setVal("pool_max", fmt.Sprintf("%d", src.PoolMaxSize))
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
	getInt := func(key string) int {
		v := get(key)
		if v == "" {
			return 0
		}
		n := 0
		fmt.Sscanf(v, "%d", &n)
		return n
	}
	src := manifest.DBSourceDef{
		Alias:       get("alias"),
		Type:        manifest.DatabaseType(get("type")),
		Version:     get("version"),
		Namespace:   get("namespace"),
		IsCache:     get("is_cache") == "yes",
		SSLMode:     get("ssl_mode"),
		Consistency: get("consistency"),
		Replication: get("replication"),
		PoolMinSize: getInt("pool_min"),
		PoolMaxSize: getInt("pool_max"),
		Notes:       get("notes"),
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

	ddOpen   bool
	ddOptIdx int

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
		if db.ddOpen {
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
			db.dbForm[db.formIdx].SaveTextInput(db.formInput.Value())
			db.internalMode = dbeNormal
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
	if db.ddOpen && db.view == dbeViewForm {
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
		db.dbForm = dbFormFromSource(db.Sources[db.srcIdx])
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
			db.ddOpen = true
			db.ddOptIdx = f.SelIdx
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
		db.ddOpen = false
		return db, nil
	}
	f := &db.dbForm[db.formIdx]
	switch key.String() {
	case "j", "down":
		if db.ddOptIdx < len(f.Options)-1 {
			db.ddOptIdx++
		}
	case "k", "up":
		if db.ddOptIdx > 0 {
			db.ddOptIdx--
		}
	case " ", "enter":
		f.SelIdx = db.ddOptIdx
		if db.ddOptIdx < len(f.Options) {
			f.Value = f.Options[db.ddOptIdx]
		}
		db.ddOpen = false
		if f.PrepareCustomEntry() {
			return db.enterDBFormInsert()
		}
	case "esc", "b":
		db.ddOpen = false
	}
	return db, nil
}

func (db DBEditor) enterDBFormInsert() (DBEditor, tea.Cmd) {
	f := db.dbForm[db.formIdx]
	if !f.CanEditAsText() {
		return db, nil
	}
	db.internalMode = dbeInsert
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
		case db.internalMode == dbeInsert && isCur && f.Kind == KindText:
			valStr = db.formInput.View()
		case f.Kind == KindSelect:
			val := f.DisplayValue()
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			if isCur && db.ddOpen {
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
		if isCur && db.ddOpen && f.Kind == KindSelect {
			const ddIndent = 4 + 14 + 3 // lineNumW + labelW + eqW
			indent := strings.Repeat(" ", ddIndent)
			for j, opt := range f.Options {
				isHL := j == db.ddOptIdx
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
