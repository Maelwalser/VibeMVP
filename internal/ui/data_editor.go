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

type deMode int

const (
	deNormal deMode = iota
	deInsert // typing in a column form text field
	deNaming // typing a new entity or column name
)

type deView int

const (
	deViewEntities       deView = iota
	deViewEntitySettings        // entity-level: database assignment, caching
	deViewColumns
	deViewColForm
)

// ── DataEditor ────────────────────────────────────────────────────────────────

// DataEditor is a self-contained entity/column schema editor embedded in the
// DATA section of the manifest TUI.
type DataEditor struct {
	Entities []manifest.EntityDef

	// availableDbs is synced from DBEditor so the entity settings form can
	// present live database/cache selects.
	availableDbs []manifest.DBSourceDef

	view       deView
	entityIdx  int
	columnIdx  int
	colFormIdx int

	// entForm holds mutable field state for the entity-level settings form.
	entForm    []Field
	entFormIdx int

	internalMode deMode

	// nameInput is used for typing new entity / column names.
	nameInput  textinput.Model
	nameTarget string // "entity" or "column"

	// formInput is the shared text input for KindText fields in any active form.
	formInput textinput.Model

	// colForm holds the mutable field state for the column currently being edited.
	colForm []Field

	width int
}

// newDataEditor returns an initialised, empty DataEditor.
func newDataEditor() DataEditor {
	ni := textinput.New()
	ni.Prompt = ""
	ni.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))

	fi := textinput.New()
	fi.Prompt = ""
	fi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg))
	fi.CursorStyle = StyleCursor
	fi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))

	return DataEditor{
		Entities:  []manifest.EntityDef{},
		nameInput: ni,
		formInput: fi,
	}
}

// Mode returns the equivalent app-level Mode for the parent status bar.
func (de DataEditor) Mode() Mode {
	if de.internalMode == deInsert || de.internalMode == deNaming {
		return ModeInsert
	}
	return ModeNormal
}

// HintLine returns context-sensitive key hints for the bottom help bar.
func (de DataEditor) HintLine() string {
	switch de.internalMode {
	case deNaming:
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Enter: confirm  Esc: cancel")
	case deInsert:
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch de.view {
	case deViewEntities:
		hints := []string{
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
			StyleHelpKey.Render("a") + StyleHelpDesc.Render(" add entity"),
			StyleHelpKey.Render("d") + StyleHelpDesc.Render(" delete"),
			StyleHelpKey.Render("Enter") + StyleHelpDesc.Render(" settings & columns"),
			StyleHelpKey.Render(":w") + StyleHelpDesc.Render(" save"),
		}
		return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
	case deViewEntitySettings:
		hints := []string{
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
			StyleHelpKey.Render("i") + StyleHelpDesc.Render(" edit"),
			StyleHelpKey.Render("Space") + StyleHelpDesc.Render(" cycle"),
			StyleHelpKey.Render("c") + StyleHelpDesc.Render(" columns"),
			StyleHelpKey.Render("b") + StyleHelpDesc.Render(" back"),
		}
		return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
	case deViewColumns:
		hints := []string{
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
			StyleHelpKey.Render("a") + StyleHelpDesc.Render(" add column"),
			StyleHelpKey.Render("d") + StyleHelpDesc.Render(" delete"),
			StyleHelpKey.Render("Enter") + StyleHelpDesc.Render(" edit"),
			StyleHelpKey.Render("b") + StyleHelpDesc.Render(" back"),
		}
		return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
	case deViewColForm:
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

// ── Column form helpers ───────────────────────────────────────────────────────

// defaultColForm returns a fresh, zeroed column form with all 16 fields.
func defaultColForm() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "type", Label: "type          ", Kind: KindSelect,
			Options: []string{
				"text", "varchar", "char", "int", "bigint", "smallint",
				"serial", "bigserial", "boolean", "float", "double", "decimal",
				"json", "jsonb", "uuid", "timestamp", "timestamptz",
				"date", "time", "bytea", "enum", "array", "other",
			},
			Value: "text",
		},
		{Key: "length", Label: "length        ", Kind: KindText},
		{Key: "nullable", Label: "nullable      ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{Key: "primary_key", Label: "primary_key   ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{Key: "unique", Label: "unique        ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "false",
		},
		{Key: "default", Label: "default       ", Kind: KindText},
		{Key: "check", Label: "check         ", Kind: KindText},
		{Key: "foreign_key", Label: "foreign_key   ", Kind: KindSelect,
			Options: []string{"no", "yes"}, Value: "no",
		},
		{Key: "fk_entity", Label: "  fk_entity   ", Kind: KindText},
		{Key: "fk_column", Label: "  fk_column   ", Kind: KindText},
		{Key: "fk_on_delete", Label: "  fk_on_delete", Kind: KindSelect,
			Options: []string{"NO ACTION", "RESTRICT", "CASCADE", "SET NULL", "SET DEFAULT"},
			Value:   "NO ACTION",
		},
		{Key: "fk_on_update", Label: "  fk_on_update", Kind: KindSelect,
			Options: []string{"NO ACTION", "RESTRICT", "CASCADE", "SET NULL", "SET DEFAULT"},
			Value:   "NO ACTION",
		},
		{Key: "indexed", Label: "indexed       ", Kind: KindSelect,
			Options: []string{"no", "yes"}, Value: "no",
		},
		{Key: "index_type", Label: "  index_type  ", Kind: KindSelect,
			Options: []string{"btree", "hash", "gin", "gist", "brin"},
			Value:   "btree",
		},
		{Key: "notes", Label: "notes         ", Kind: KindText},
	}
}

// colFormFromColumnDef populates a form from an existing ColumnDef.
func colFormFromColumnDef(col manifest.ColumnDef) []Field {
	f := defaultColForm()

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

	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}

	setVal("name", col.Name)
	if col.Type != "" {
		setVal("type", string(col.Type))
	}
	setVal("length", col.Length)
	setVal("nullable", boolStr(col.Nullable))
	setVal("primary_key", boolStr(col.PrimaryKey))
	setVal("unique", boolStr(col.Unique))
	setVal("default", col.Default)
	setVal("check", col.Check)
	if col.ForeignKey != nil {
		setVal("foreign_key", "yes")
		setVal("fk_entity", col.ForeignKey.RefEntity)
		setVal("fk_column", col.ForeignKey.RefColumn)
		setVal("fk_on_delete", string(col.ForeignKey.OnDelete))
		setVal("fk_on_update", string(col.ForeignKey.OnUpdate))
	}
	if col.Index {
		setVal("indexed", "yes")
		if col.IndexType != "" {
			setVal("index_type", string(col.IndexType))
		}
	}
	setVal("notes", col.Notes)
	return f
}

// colFormToColumnDef converts the current form state back to a ColumnDef.
func colFormToColumnDef(form []Field) manifest.ColumnDef {
	get := func(key string) string {
		for _, f := range form {
			if f.Key == key {
				return f.DisplayValue()
			}
		}
		return ""
	}

	col := manifest.ColumnDef{
		Name:       get("name"),
		Type:       manifest.ColumnType(get("type")),
		Length:     get("length"),
		Nullable:   get("nullable") == "true",
		PrimaryKey: get("primary_key") == "true",
		Unique:     get("unique") == "true",
		Default:    get("default"),
		Check:      get("check"),
		Index:      get("indexed") == "yes",
		Notes:      get("notes"),
	}
	if col.Index {
		col.IndexType = manifest.IndexType(get("index_type"))
	}
	if get("foreign_key") == "yes" {
		col.ForeignKey = &manifest.ForeignKey{
			RefEntity: get("fk_entity"),
			RefColumn: get("fk_column"),
			OnDelete:  manifest.CascadeAction(get("fk_on_delete")),
			OnUpdate:  manifest.CascadeAction(get("fk_on_update")),
		}
	}
	return col
}

// isColFormFieldDisabled returns true when a field is gated behind a parent toggle.
func isColFormFieldDisabled(form []Field, idx int) bool {
	key := form[idx].Key
	switch key {
	case "fk_entity", "fk_column", "fk_on_delete", "fk_on_update":
		for _, f := range form {
			if f.Key == "foreign_key" {
				return f.DisplayValue() != "yes"
			}
		}
	case "index_type":
		for _, f := range form {
			if f.Key == "indexed" {
				return f.DisplayValue() != "yes"
			}
		}
	}
	return false
}

func nextColFormIdx(form []Field, cur int) int {
	n := len(form)
	next := (cur + 1) % n
	for next != cur && isColFormFieldDisabled(form, next) {
		next = (next + 1) % n
	}
	return next
}

func prevColFormIdx(form []Field, cur int) int {
	n := len(form)
	prev := (cur - 1 + n) % n
	for prev != cur && isColFormFieldDisabled(form, prev) {
		prev = (prev - 1 + n) % n
	}
	return prev
}

// ── Entity settings form helpers ──────────────────────────────────────────────

// buildEntitySettingsForm constructs the entity-level settings form, dynamically
// populating database and cache_store selects from the available DBSources.
func buildEntitySettingsForm(ent manifest.EntityDef, dbs []manifest.DBSourceDef) []Field {
	// Build db options list
	dbOptions := []string{"(none)"}
	for _, db := range dbs {
		dbOptions = append(dbOptions, db.Alias)
	}

	// Build cache options list (only sources marked as cache)
	cacheOptions := []string{"(none)"}
	for _, db := range dbs {
		if db.IsCache {
			cacheOptions = append(cacheOptions, db.Alias)
		}
	}

	findIdx := func(opts []string, val string) int {
		for i, o := range opts {
			if o == val {
				return i
			}
		}
		return 0
	}

	cachedVal := "no"
	cachedIdx := 0
	if ent.Cached {
		cachedVal = "yes"
		cachedIdx = 1
	}

	dbVal := ent.Database
	if dbVal == "" {
		dbVal = "(none)"
	}
	cacheVal := ent.CacheStore
	if cacheVal == "" {
		cacheVal = "(none)"
	}

	return []Field{
		{Key: "database", Label: "database      ", Kind: KindSelect,
			Options: dbOptions,
			SelIdx:  findIdx(dbOptions, dbVal),
			Value:   dbVal,
		},
		{Key: "description", Label: "description   ", Kind: KindText, Value: ent.Description},
		{Key: "cached", Label: "cached        ", Kind: KindSelect,
			Options: []string{"no", "yes"},
			SelIdx:  cachedIdx,
			Value:   cachedVal,
		},
		{Key: "cache_store", Label: "  cache_store ", Kind: KindSelect,
			Options: cacheOptions,
			SelIdx:  findIdx(cacheOptions, cacheVal),
			Value:   cacheVal,
		},
		{Key: "cache_ttl", Label: "  cache_ttl   ", Kind: KindText, Value: ent.CacheTTL},
		{Key: "notes", Label: "notes         ", Kind: KindText, Value: ent.Notes},
	}
}

func entitySettingsToEntityDef(form []Field, ent manifest.EntityDef) manifest.EntityDef {
	get := func(key string) string {
		for _, f := range form {
			if f.Key == key {
				return f.DisplayValue()
			}
		}
		return ""
	}
	db := get("database")
	if db == "(none)" {
		db = ""
	}
	ent.Database = db
	ent.Description = get("description")
	ent.Cached = get("cached") == "yes"
	cs := get("cache_store")
	if cs == "(none)" {
		cs = ""
	}
	ent.CacheStore = cs
	ent.CacheTTL = get("cache_ttl")
	ent.Notes = get("notes")
	return ent
}

func isEntFormFieldDisabled(form []Field, idx int) bool {
	key := form[idx].Key
	if key == "cache_store" || key == "cache_ttl" {
		for _, f := range form {
			if f.Key == "cached" {
				return f.DisplayValue() != "yes"
			}
		}
	}
	return false
}

func nextEntFormIdx(form []Field, cur int) int {
	n := len(form)
	next := (cur + 1) % n
	for next != cur && isEntFormFieldDisabled(form, next) {
		next = (next + 1) % n
	}
	return next
}

func prevEntFormIdx(form []Field, cur int) int {
	n := len(form)
	prev := (cur - 1 + n) % n
	for prev != cur && isEntFormFieldDisabled(form, prev) {
		prev = (prev - 1 + n) % n
	}
	return prev
}

// ── Update ────────────────────────────────────────────────────────────────────

// Update handles all keyboard input and returns the new state plus any tea.Cmd.
func (de DataEditor) Update(msg tea.Msg) (DataEditor, tea.Cmd) {
	switch de.internalMode {
	case deNaming:
		return de.updateNaming(msg)
	case deInsert:
		return de.updateInsert(msg)
	default:
		return de.updateNormal(msg)
	}
}

func (de DataEditor) updateNaming(msg tea.Msg) (DataEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			de.internalMode = deNormal
			de.nameInput.Blur()
			return de, nil

		case "enter":
			name := strings.TrimSpace(de.nameInput.Value())
			de.internalMode = deNormal
			de.nameInput.Blur()
			if name == "" {
				return de, nil
			}
			switch de.nameTarget {
			case "entity":
				de.Entities = append(de.Entities, manifest.EntityDef{
					Name:    name,
					Columns: []manifest.ColumnDef{},
				})
				de.entityIdx = len(de.Entities) - 1
				// Go to entity settings so the user assigns a database first
				de.entForm = buildEntitySettingsForm(de.Entities[de.entityIdx], de.availableDbs)
				de.entFormIdx = 0
				de.view = deViewEntitySettings
			case "column":
				col := manifest.ColumnDef{Name: name, Type: manifest.ColTypeText}
				ent := de.Entities[de.entityIdx]
				ent.Columns = append(ent.Columns, col)
				de.Entities[de.entityIdx] = ent
				de.columnIdx = len(ent.Columns) - 1
				de.colForm = colFormFromColumnDef(col)
				de.colFormIdx = 1 // start on type field (name is pre-filled)
				de.view = deViewColForm
			}
			return de, nil
		}
	}
	var cmd tea.Cmd
	de.nameInput, cmd = de.nameInput.Update(msg)
	return de, cmd
}

func (de DataEditor) updateInsert(msg tea.Msg) (DataEditor, tea.Cmd) {
	// Route to the correct active form based on current view.
	if de.view == deViewEntitySettings {
		return de.updateInsertEntForm(msg)
	}
	return de.updateInsertColForm(msg)
}

func (de DataEditor) updateInsertColForm(msg tea.Msg) (DataEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			de.colForm[de.colFormIdx].Value = de.formInput.Value()
			de.internalMode = deNormal
			de.formInput.Blur()
			return de, nil

		case "tab":
			de.colForm[de.colFormIdx].Value = de.formInput.Value()
			de.colFormIdx = nextColFormIdx(de.colForm, de.colFormIdx)
			f := de.colForm[de.colFormIdx]
			if f.Kind == KindSelect {
				de.internalMode = deNormal
				de.formInput.Blur()
				return de, nil
			}
			de.formInput.SetValue(f.Value)
			de.formInput.CursorEnd()
			return de, de.formInput.Focus()

		case "shift+tab":
			de.colForm[de.colFormIdx].Value = de.formInput.Value()
			de.colFormIdx = prevColFormIdx(de.colForm, de.colFormIdx)
			f := de.colForm[de.colFormIdx]
			if f.Kind == KindSelect {
				de.internalMode = deNormal
				de.formInput.Blur()
				return de, nil
			}
			de.formInput.SetValue(f.Value)
			de.formInput.CursorEnd()
			return de, de.formInput.Focus()
		}
	}
	var cmd tea.Cmd
	de.formInput, cmd = de.formInput.Update(msg)
	return de, cmd
}

func (de DataEditor) updateInsertEntForm(msg tea.Msg) (DataEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			de.entForm[de.entFormIdx].Value = de.formInput.Value()
			de.internalMode = deNormal
			de.formInput.Blur()
			return de, nil

		case "tab":
			de.entForm[de.entFormIdx].Value = de.formInput.Value()
			de.entFormIdx = nextEntFormIdx(de.entForm, de.entFormIdx)
			f := de.entForm[de.entFormIdx]
			if f.Kind == KindSelect {
				de.internalMode = deNormal
				de.formInput.Blur()
				return de, nil
			}
			de.formInput.SetValue(f.Value)
			de.formInput.CursorEnd()
			return de, de.formInput.Focus()

		case "shift+tab":
			de.entForm[de.entFormIdx].Value = de.formInput.Value()
			de.entFormIdx = prevEntFormIdx(de.entForm, de.entFormIdx)
			f := de.entForm[de.entFormIdx]
			if f.Kind == KindSelect {
				de.internalMode = deNormal
				de.formInput.Blur()
				return de, nil
			}
			de.formInput.SetValue(f.Value)
			de.formInput.CursorEnd()
			return de, de.formInput.Focus()
		}
	}
	var cmd tea.Cmd
	de.formInput, cmd = de.formInput.Update(msg)
	return de, cmd
}

func (de DataEditor) updateNormal(msg tea.Msg) (DataEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return de, nil
	}
	switch de.view {
	case deViewEntities:
		return de.updateNormalEntities(key)
	case deViewEntitySettings:
		return de.updateNormalEntitySettings(key)
	case deViewColumns:
		return de.updateNormalColumns(key)
	case deViewColForm:
		return de.updateNormalColForm(key)
	}
	return de, nil
}

func (de DataEditor) updateNormalEntities(key tea.KeyMsg) (DataEditor, tea.Cmd) {
	n := len(de.Entities)
	switch key.String() {
	case "j", "down":
		if n > 0 && de.entityIdx < n-1 {
			de.entityIdx++
		}
	case "k", "up":
		if de.entityIdx > 0 {
			de.entityIdx--
		}
	case "g":
		de.entityIdx = 0
	case "G":
		if n > 0 {
			de.entityIdx = n - 1
		}
	case "d":
		if n > 0 {
			de.Entities = append(de.Entities[:de.entityIdx], de.Entities[de.entityIdx+1:]...)
			if de.entityIdx >= len(de.Entities) && de.entityIdx > 0 {
				de.entityIdx--
			}
		}
	case "a":
		de.nameTarget = "entity"
		de.nameInput.Placeholder = "entity name…"
		de.nameInput.SetValue("")
		de.internalMode = deNaming
		return de, de.nameInput.Focus()
	case "enter", "l", "right":
		if n > 0 {
			de.entForm = buildEntitySettingsForm(de.Entities[de.entityIdx], de.availableDbs)
			de.entFormIdx = 0
			de.view = deViewEntitySettings
		}
	}
	return de, nil
}

func (de DataEditor) updateNormalEntitySettings(key tea.KeyMsg) (DataEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		de.entFormIdx = nextEntFormIdx(de.entForm, de.entFormIdx)
	case "k", "up":
		de.entFormIdx = prevEntFormIdx(de.entForm, de.entFormIdx)
	case "g":
		de.entFormIdx = 0
	case "G":
		de.entFormIdx = len(de.entForm) - 1
	case "enter", " ":
		f := &de.entForm[de.entFormIdx]
		if f.Kind == KindSelect {
			f.CycleNext()
		} else {
			return de.enterEntFormInsert()
		}
	case "H", "shift+left":
		f := &de.entForm[de.entFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if de.entForm[de.entFormIdx].Kind == KindText {
			return de.enterEntFormInsert()
		}
	case "c": // go to columns
		de.saveEntFormBack()
		de.columnIdx = 0
		de.view = deViewColumns
	case "b", "esc":
		de.saveEntFormBack()
		de.view = deViewEntities
	}
	return de, nil
}

func (de DataEditor) enterEntFormInsert() (DataEditor, tea.Cmd) {
	f := de.entForm[de.entFormIdx]
	if f.Kind != KindText {
		return de, nil
	}
	de.internalMode = deInsert
	de.formInput.SetValue(f.Value)
	de.formInput.Width = de.width - 22
	de.formInput.CursorEnd()
	return de, de.formInput.Focus()
}

func (de *DataEditor) saveEntFormBack() {
	if de.entityIdx >= len(de.Entities) {
		return
	}
	de.Entities[de.entityIdx] = entitySettingsToEntityDef(de.entForm, de.Entities[de.entityIdx])
}

func (de DataEditor) updateNormalColumns(key tea.KeyMsg) (DataEditor, tea.Cmd) {
	if de.entityIdx >= len(de.Entities) {
		return de, nil
	}
	ent := de.Entities[de.entityIdx]
	nc := len(ent.Columns)

	switch key.String() {
	case "j", "down":
		if nc > 0 && de.columnIdx < nc-1 {
			de.columnIdx++
		}
	case "k", "up":
		if de.columnIdx > 0 {
			de.columnIdx--
		}
	case "g":
		de.columnIdx = 0
	case "G":
		if nc > 0 {
			de.columnIdx = nc - 1
		}
	case "d":
		if nc > 0 {
			ent.Columns = append(ent.Columns[:de.columnIdx], ent.Columns[de.columnIdx+1:]...)
			de.Entities[de.entityIdx] = ent
			if de.columnIdx >= len(ent.Columns) && de.columnIdx > 0 {
				de.columnIdx--
			}
		}
	case "a":
		de.nameTarget = "column"
		de.nameInput.Placeholder = "column name…"
		de.nameInput.SetValue("")
		de.internalMode = deNaming
		return de, de.nameInput.Focus()
	case "enter", "e":
		if nc > 0 {
			de.colForm = colFormFromColumnDef(ent.Columns[de.columnIdx])
			de.colFormIdx = 0
			de.view = deViewColForm
		}
	case "b", "esc", "h", "left":
		// Back to entity settings instead of entity list
		de.entForm = buildEntitySettingsForm(de.Entities[de.entityIdx], de.availableDbs)
		de.entFormIdx = 0
		de.view = deViewEntitySettings
	}
	return de, nil
}

func (de DataEditor) updateNormalColForm(key tea.KeyMsg) (DataEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		de.colFormIdx = nextColFormIdx(de.colForm, de.colFormIdx)
	case "k", "up":
		de.colFormIdx = prevColFormIdx(de.colForm, de.colFormIdx)
	case "g":
		de.colFormIdx = 0
	case "G":
		de.colFormIdx = len(de.colForm) - 1
	case "enter", " ":
		f := &de.colForm[de.colFormIdx]
		if f.Kind == KindSelect {
			f.CycleNext()
		} else {
			return de.enterFormInsert()
		}
	case "H", "shift+left":
		f := &de.colForm[de.colFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if de.colForm[de.colFormIdx].Kind == KindText {
			return de.enterFormInsert()
		}
	case "b", "esc":
		de.saveColFormBack()
		de.view = deViewColumns
	}
	return de, nil
}

// saveColFormBack writes the current form state into de.Entities.
func (de *DataEditor) saveColFormBack() {
	if de.entityIdx >= len(de.Entities) {
		return
	}
	ent := de.Entities[de.entityIdx]
	if de.columnIdx < len(ent.Columns) {
		ent.Columns[de.columnIdx] = colFormToColumnDef(de.colForm)
		de.Entities[de.entityIdx] = ent
	}
}

func (de DataEditor) enterFormInsert() (DataEditor, tea.Cmd) {
	f := de.colForm[de.colFormIdx]
	if f.Kind != KindText {
		return de, nil
	}
	de.internalMode = deInsert
	de.formInput.SetValue(f.Value)
	de.formInput.Width = de.width - 22
	de.formInput.CursorEnd()
	return de, de.formInput.Focus()
}

// ── View ──────────────────────────────────────────────────────────────────────

// View renders the editor into a w×h content block.
func (de DataEditor) View(w, h int) string {
	de.width = w
	switch de.view {
	case deViewEntities:
		return de.viewEntities(w, h)
	case deViewEntitySettings:
		return de.viewEntitySettings(w, h)
	case deViewColumns:
		return de.viewColumns(w, h)
	case deViewColForm:
		return de.viewColForm(w, h)
	}
	return ""
}

func (de DataEditor) viewEntities(w, h int) string {
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Entities — a: add  d: delete  Enter: settings & columns"),
		"",
	)

	if len(de.Entities) == 0 {
		lines = append(lines, StyleSectionDesc.Render("  (no entities yet — press 'a' to add one)"))
	} else {
		for i, ent := range de.Entities {
			isCur := i == de.entityIdx
			nCols := len(ent.Columns)
			colLabel := fmt.Sprintf("%d col", nCols)

			arrow := "  ▸ "
			nameStr := ent.Name
			if isCur {
				arrow = StyleCurLineNum.Render("  ▶ ")
				nameStr = StyleFieldKeyActive.Render(nameStr)
			} else {
				nameStr = StyleFieldKey.Render(nameStr)
			}

			// Database badge
			dbBadge := ""
			if ent.Database != "" {
				dbBadge = StyleSectionTitle.Render("[" + ent.Database + "]")
			} else {
				dbBadge = StyleSectionDesc.Render("[?]")
			}

			// Cache badge
			cacheBadge := ""
			if ent.Cached {
				cs := ent.CacheStore
				if cs == "" {
					cs = "cache"
				}
				ttl := ""
				if ent.CacheTTL != "" {
					ttl = " " + ent.CacheTTL
				}
				cacheBadge = " " + StyleMsgOK.Render("⚡"+cs+ttl)
			}

			pad := max(1, 22-len(ent.Name))
			cols := StyleSectionDesc.Render(colLabel)
			row := arrow + nameStr + strings.Repeat(" ", pad) + dbBadge + cacheBadge + "  " + cols

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

	if de.internalMode == deNaming && de.nameTarget == "entity" {
		lines = append(lines, "")
		lines = append(lines, StyleTextAreaLabel.Render("  New entity: ")+de.nameInput.View())
	}

	return fillTildes(lines, h)
}

func (de DataEditor) viewEntitySettings(w, h int) string {
	if de.entityIdx >= len(de.Entities) {
		return fillTildes(nil, h)
	}
	ent := de.Entities[de.entityIdx]
	var lines []string

	breadcrumb := StyleSectionDesc.Render("  ← ") + StyleSectionTitle.Render(ent.Name) +
		StyleSectionDesc.Render("  (c: columns  b: back)")
	lines = append(lines, breadcrumb, "")

	const labelW = 14
	const eqW = 3
	valW := w - 4 - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	for i, f := range de.entForm {
		isCur := i == de.entFormIdx
		disabled := isEntFormFieldDisabled(de.entForm, i)

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
		case de.internalMode == deInsert && isCur && f.Kind == KindText:
			valStr = de.formInput.View()
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

func (de DataEditor) viewColumns(w, h int) string {
	if de.entityIdx >= len(de.Entities) {
		return fillTildes(nil, h)
	}
	ent := de.Entities[de.entityIdx]
	var lines []string

	dbLabel := ""
	if ent.Database != "" {
		dbLabel = "  " + StyleSectionTitle.Render("["+ent.Database+"]")
	}
	breadcrumb := StyleSectionDesc.Render("  ← ") + StyleSectionTitle.Render(ent.Name) + dbLabel
	lines = append(lines, breadcrumb, "")

	if len(ent.Columns) == 0 {
		lines = append(lines, StyleSectionDesc.Render("  (no columns yet — press 'a' to add one)"))
	} else {
		for i, col := range ent.Columns {
			isCur := i == de.columnIdx

			numStr := fmt.Sprintf("%3d ", i+1)
			if isCur {
				numStr = StyleCurLineNum.Render(numStr)
			} else {
				numStr = StyleLineNum.Render(numStr)
			}

			typeStr := string(col.Type)
			if col.Length != "" {
				typeStr += "(" + col.Length + ")"
			}

			var badges []string
			if col.PrimaryKey {
				badges = append(badges, StyleSelectArrow.Render("PK"))
			}
			if !col.Nullable {
				badges = append(badges, StyleSectionDesc.Render("NOT NULL"))
			}
			if col.Unique {
				badges = append(badges, StyleMsgOK.Render("UNIQUE"))
			}
			if col.ForeignKey != nil {
				ref := fmt.Sprintf("FK→%s.%s", col.ForeignKey.RefEntity, col.ForeignKey.RefColumn)
				onDel := ""
				if col.ForeignKey.OnDelete != "" && col.ForeignKey.OnDelete != manifest.CascadeNoAction {
					onDel = " " + string(col.ForeignKey.OnDelete)
				}
				badges = append(badges, StyleSectionTitle.Render(ref+onDel))
			}
			if col.Index {
				idxType := string(col.IndexType)
				if idxType == "" {
					idxType = "idx"
				}
				badges = append(badges, StyleHelpKey.Render(idxType))
			}

			badgeStr := ""
			if len(badges) > 0 {
				badgeStr = "  " + strings.Join(badges, " ")
			}

			colName := col.Name
			if isCur {
				colName = StyleFieldKeyActive.Render(colName)
			} else {
				colName = StyleFieldKey.Render(colName)
			}

			pad := max(1, 20-len(col.Name))
			typeRendered := StyleFieldVal.Render(fmt.Sprintf("%-14s", typeStr))
			row := numStr + colName + strings.Repeat(" ", pad) + typeRendered + badgeStr

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

	if de.internalMode == deNaming && de.nameTarget == "column" {
		lines = append(lines, "")
		lines = append(lines, StyleTextAreaLabel.Render("  New column: ")+de.nameInput.View())
	}

	return fillTildes(lines, h)
}

func (de DataEditor) viewColForm(w, h int) string {
	if de.entityIdx >= len(de.Entities) {
		return fillTildes(nil, h)
	}
	ent := de.Entities[de.entityIdx]

	colLabel := "(new column)"
	if de.columnIdx < len(ent.Columns) {
		colLabel = ent.Columns[de.columnIdx].Name
	}

	var lines []string
	breadcrumb := StyleSectionDesc.Render("  ← ") +
		StyleSectionTitle.Render(ent.Name) +
		StyleSectionDesc.Render(" → ") +
		StyleFieldKey.Render(colLabel)
	lines = append(lines, breadcrumb, "")

	const labelW = 14
	const eqW = 3
	valW := w - 4 - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	for i, f := range de.colForm {
		isCur := i == de.colFormIdx
		disabled := isColFormFieldDisabled(de.colForm, i)

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
		case de.internalMode == deInsert && isCur && f.Kind == KindText:
			valStr = de.formInput.View()
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
