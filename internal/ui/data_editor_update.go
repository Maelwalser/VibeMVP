package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── Update ────────────────────────────────────────────────────────────────────

// Update handles all keyboard input and returns the new state plus any tea.Cmd.
func (de DataEditor) Update(msg tea.Msg) (DataEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		de.width = wsz.Width
		de.formInput.Width = wsz.Width - 22
		return de, nil
	}
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
			de.colForm[de.colFormIdx].SaveTextInput(de.formInput.Value())
			de.internalMode = deNormal
			de.formInput.Blur()
			return de, nil

		case "tab":
			de.colForm[de.colFormIdx].SaveTextInput(de.formInput.Value())
			de.colFormIdx = nextColFormIdx(de.colForm, de.colFormIdx)
			f := de.colForm[de.colFormIdx]
			if !f.CanEditAsText() {
				de.internalMode = deNormal
				de.formInput.Blur()
				return de, nil
			}
			de.formInput.SetValue(f.TextInputValue())
			de.formInput.CursorEnd()
			return de, de.formInput.Focus()

		case "shift+tab":
			de.colForm[de.colFormIdx].SaveTextInput(de.formInput.Value())
			de.colFormIdx = prevColFormIdx(de.colForm, de.colFormIdx)
			f := de.colForm[de.colFormIdx]
			if !f.CanEditAsText() {
				de.internalMode = deNormal
				de.formInput.Blur()
				return de, nil
			}
			de.formInput.SetValue(f.TextInputValue())
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
			de.entForm[de.entFormIdx].SaveTextInput(de.formInput.Value())
			de.internalMode = deNormal
			de.formInput.Blur()
			return de, nil

		case "tab":
			de.entForm[de.entFormIdx].SaveTextInput(de.formInput.Value())
			de.entFormIdx = nextEntFormIdx(de.entForm, de.entFormIdx)
			f := de.entForm[de.entFormIdx]
			if !f.CanEditAsText() {
				de.internalMode = deNormal
				de.formInput.Blur()
				return de, nil
			}
			de.formInput.SetValue(f.TextInputValue())
			de.formInput.CursorEnd()
			return de, de.formInput.Focus()

		case "shift+tab":
			de.entForm[de.entFormIdx].SaveTextInput(de.formInput.Value())
			de.entFormIdx = prevEntFormIdx(de.entForm, de.entFormIdx)
			f := de.entForm[de.entFormIdx]
			if !f.CanEditAsText() {
				de.internalMode = deNormal
				de.formInput.Blur()
				return de, nil
			}
			de.formInput.SetValue(f.TextInputValue())
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
	if de.dd.Open {
		switch de.view {
		case deViewEntitySettings:
			return de.updateEntFormDropdown(key)
		case deViewColForm:
			return de.updateColFormDropdown(key)
		default:
			de.dd.Open = false
		}
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
			de.dd.Open = true
			de.dd.OptIdx = f.SelIdx
		} else {
			return de.enterEntFormInsert()
		}
	case "H", "shift+left":
		f := &de.entForm[de.entFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if de.entForm[de.entFormIdx].CanEditAsText() {
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
	de.saveEntFormBack()
	return de, nil
}

func (de DataEditor) enterEntFormInsert() (DataEditor, tea.Cmd) {
	f := de.entForm[de.entFormIdx]
	if !f.CanEditAsText() {
		return de, nil
	}
	de.internalMode = deInsert
	de.formInput.SetValue(f.TextInputValue())
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
			de.dd.Open = true
			de.dd.OptIdx = f.SelIdx
		} else {
			return de.enterFormInsert()
		}
	case "H", "shift+left":
		f := &de.colForm[de.colFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if de.colForm[de.colFormIdx].CanEditAsText() {
			return de.enterFormInsert()
		}
	case "b", "esc":
		de.saveColFormBack()
		de.view = deViewColumns
	}
	de.saveColFormBack()
	return de, nil
}

func (de DataEditor) updateEntFormDropdown(key tea.KeyMsg) (DataEditor, tea.Cmd) {
	if de.entFormIdx >= len(de.entForm) {
		de.dd.Open = false
		return de, nil
	}
	f := &de.entForm[de.entFormIdx]
	de.dd.OptIdx = NavigateDropdown(key.String(), de.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = de.dd.OptIdx
		if de.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[de.dd.OptIdx]
		}
		de.dd.Open = false
		if f.PrepareCustomEntry() {
			return de.enterEntFormInsert()
		}
	case "esc", "b":
		de.dd.Open = false
	}
	de.saveEntFormBack()
	return de, nil
}

func (de DataEditor) updateColFormDropdown(key tea.KeyMsg) (DataEditor, tea.Cmd) {
	if de.colFormIdx >= len(de.colForm) {
		de.dd.Open = false
		return de, nil
	}
	f := &de.colForm[de.colFormIdx]
	de.dd.OptIdx = NavigateDropdown(key.String(), de.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = de.dd.OptIdx
		if de.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[de.dd.OptIdx]
		}
		de.dd.Open = false
		if f.PrepareCustomEntry() {
			return de.enterFormInsert()
		}
	case "esc", "b":
		de.dd.Open = false
	}
	de.saveColFormBack()
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
	if !f.CanEditAsText() {
		return de, nil
	}
	de.internalMode = deInsert
	de.formInput.SetValue(f.TextInputValue())
	de.formInput.Width = de.width - 22
	de.formInput.CursorEnd()
	return de, de.formInput.Focus()
}

