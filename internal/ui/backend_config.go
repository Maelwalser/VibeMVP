package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// ── Stack Config list/form update handlers ────────────────────────────────────

func (be BackendEditor) updateStackConfigList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.stackConfigEditor
	n := len(ed.items)
	switch key.String() {
	case "j", "down":
		if n > 0 && ed.itemIdx < n-1 {
			ed.itemIdx++
		}
	case "k", "up":
		if ed.itemIdx > 0 {
			ed.itemIdx--
		}
	case "a":
		newFields := defaultStackConfigFields()
		existing := make([]string, 0, len(ed.items))
		for _, item := range ed.items {
			if name := fieldGet(item, "name"); name != "" {
				existing = append(existing, name)
			}
		}
		newFields = setFieldValue(newFields, "name", uniqueName("stack", existing))
		ed.items = append(ed.items, newFields)
		ed.itemIdx = len(ed.items) - 1
		ed.form = copyFields(ed.items[ed.itemIdx])
		ed.formIdx = 0
		ed.itemView = beListViewForm
		be.activeField = 0
	case "d":
		if n > 0 {
			ed.items = append(ed.items[:ed.itemIdx], ed.items[ed.itemIdx+1:]...)
			if ed.itemIdx > 0 && ed.itemIdx >= len(ed.items) {
				ed.itemIdx = len(ed.items) - 1
			}
			be.applyStackConfigNamesToServices()
		}
	case "enter":
		if n > 0 {
			ed.form = copyFields(ed.items[ed.itemIdx])
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.activeField = 0
		}
	case "l", "right":
		tabs := be.activeTabs()
		if be.activeTabIdx < len(tabs)-1 {
			be.activeTabIdx++
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "b":
		be.ArchConfirmed = false
		be.dropdownOpen = false
		be.dropdownIdx = be.ArchIdx
		be.activeTabIdx = 0
		be.activeField = 0
	}
	return be, nil
}

func (be BackendEditor) updateStackConfigForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.stackConfigEditor
	n := len(ed.form)
	switch key.String() {
	case "j", "down":
		if ed.formIdx < n-1 {
			ed.formIdx++
		}
	case "k", "up":
		if ed.formIdx > 0 {
			ed.formIdx--
		}
	case "enter", " ":
		if ed.formIdx < n {
			f := &ed.form[ed.formIdx]
			if f.Kind == KindSelect {
				be.dd.Open = true
				be.dd.OptIdx = f.SelIdx
			} else {
				return be.enterStackConfigFormInsert()
			}
		}
	case "H", "shift+left":
		if ed.formIdx < n {
			f := &ed.form[ed.formIdx]
			if f.Kind == KindSelect {
				f.CyclePrev()
				if f.Key == "language" {
					be.updateStackConfigFrameworkOptions(ed)
				} else if f.Key == "language_version" || f.Key == "framework" {
					be.updateStackConfigVersionOptions(ed)
				}
			}
		}
	case "i", "a":
		if ed.formIdx < n && ed.form[ed.formIdx].CanEditAsText() {
			return be.enterStackConfigFormInsert()
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		tabs := be.activeTabs()
		if be.activeTabIdx < len(tabs)-1 {
			be.activeTabIdx++
		}
	case "b", "esc":
		be.saveStackConfigForm()
		ed.itemView = beListViewList
	}
	return be, nil
}

func (be *BackendEditor) saveStackConfigForm() {
	ed := &be.stackConfigEditor
	if ed.itemIdx < len(ed.items) {
		ed.items[ed.itemIdx] = copyFields(ed.form)
	}
	be.applyStackConfigNamesToServices()
}

func (be BackendEditor) enterStackConfigFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.stackConfigEditor
	if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].CanEditAsText() {
		be.internalMode = ModeInsert
		be.formInput.SetValue(ed.form[ed.formIdx].TextInputValue())
		be.formInput.Width = be.width - 22
		be.formInput.CursorEnd()
		return be, be.formInput.Focus()
	}
	return be, nil
}

func (be *BackendEditor) updateStackConfigFrameworkOptions(ed *beListEditor) {
	lang := fieldGet(ed.form, "language")
	opts, ok := backendFrameworksByLang[lang]
	if !ok {
		opts = []string{"Other"}
	}
	langVers, hasVers := langVersions[lang]
	for i := range ed.form {
		switch ed.form[i].Key {
		case "framework":
			ed.form[i].Options = opts
			ed.form[i].SelIdx = 0
			ed.form[i].Value = opts[0]
		case "language_version":
			if hasVers {
				ed.form[i].Options = langVers
				ed.form[i].SelIdx = 0
				ed.form[i].Value = langVers[0]
			}
		}
	}
	be.updateStackConfigVersionOptions(ed)
}

func (be *BackendEditor) updateStackConfigVersionOptions(ed *beListEditor) {
	lang := fieldGet(ed.form, "language")
	langVer := fieldGet(ed.form, "language_version")
	fw := fieldGet(ed.form, "framework")
	fwVers := compatibleFrameworkVersions(lang, langVer, fw)
	for i := range ed.form {
		if ed.form[i].Key == "framework_version" {
			ed.form[i].Options = fwVers
			ed.form[i].SelIdx = 0
			ed.form[i].Value = fwVers[0]
			break
		}
	}
}

// ── Stack Config view ─────────────────────────────────────────────────────────

func (be BackendEditor) viewStackConfigEditor(w int) []string {
	ed := be.stackConfigEditor
	if ed.itemView == beListViewList {
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Stack Configs — a: add  d: delete  Enter: edit"), "")
		if len(ed.items) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no configs yet — press 'a' to add)"))
		} else {
			for i, item := range ed.items {
				name := fieldGet(item, "name")
				if name == "" {
					name = fmt.Sprintf("(config #%d)", i+1)
				}
				lang := fieldGet(item, "language")
				fw := fieldGet(item, "framework")
				extra := lang
				if fw != "" {
					extra += " / " + fw
				}
				lines = append(lines, renderListItem(w, i == ed.itemIdx, "  ▶ ", name, extra))
			}
		}
		return lines
	}

	// Form view
	name := "(new stack config)"
	if n := fieldGet(ed.form, "name"); n != "" {
		name = n
	}

	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
	lines = append(lines, renderFormFields(w, ed.form, ed.formIdx, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
	return lines
}
