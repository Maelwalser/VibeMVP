package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (fe FrontendEditor) updateI18n(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if !fe.i18nEnabled {
		if key.String() == "a" {
			fe.i18nEnabled = true
			fe.i18nFormIdx = 0
		}
		return fe, nil
	}
	if fe.dd.Open {
		return fe.updateI18nDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.i18nFormIdx < len(fe.i18nFields)-1 {
			fe.i18nFormIdx++
		}
	case "k", "up":
		if fe.i18nFormIdx > 0 {
			fe.i18nFormIdx--
		}
	case "enter", " ":
		f := &fe.i18nFields[fe.i18nFormIdx]
		switch f.Kind {
		case KindSelect:
			fe.dd.Open = true
			fe.dd.OptIdx = f.SelIdx
		case KindMultiSelect:
			fe.dd.Open = true
			fe.dd.OptIdx = f.DDCursor
		default:
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.i18nFields[fe.i18nFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "D":
		fe.i18nEnabled = false
		fe.i18nFields = defaultI18nFields()
		fe.i18nFormIdx = 0
	case "i", "a":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateI18nDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.i18nFormIdx >= len(fe.i18nFields) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.i18nFields[fe.i18nFormIdx]
	fe.dd.OptIdx = NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == KindMultiSelect {
			f.ToggleMultiSelect(fe.dd.OptIdx)
			f.DDCursor = fe.dd.OptIdx
		} else if f.Kind == KindSelect {
			f.SelIdx = fe.dd.OptIdx
			if fe.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[fe.dd.OptIdx]
			}
			fe.dd.Open = false
			if f.PrepareCustomEntry() {
				return fe.tryEnterInsert()
			}
		}
	case "enter":
		if f.Kind == KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		} else if f.Kind == KindSelect {
			f.SelIdx = fe.dd.OptIdx
			if fe.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[fe.dd.OptIdx]
			}
		}
		fe.dd.Open = false
		if f.Kind == KindSelect && f.PrepareCustomEntry() {
			return fe.tryEnterInsert()
		}
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		}
		fe.dd.Open = false
	}
	return fe, nil
}

func (fe FrontendEditor) updateA11ySEO(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if !fe.a11yEnabled {
		if key.String() == "a" {
			fe.a11yEnabled = true
			fe.a11yFormIdx = 0
		}
		return fe, nil
	}
	if fe.dd.Open {
		return fe.updateA11ySEODropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.a11yFormIdx < len(fe.a11yFields)-1 {
			fe.a11yFormIdx++
		}
	case "k", "up":
		if fe.a11yFormIdx > 0 {
			fe.a11yFormIdx--
		}
	case "enter", " ":
		f := &fe.a11yFields[fe.a11yFormIdx]
		if f.Kind == KindSelect {
			if f.Key == "seo_render_strategy" {
				fe.a11yFields = refreshSEORenderOptions(fe.a11yFields, fe.Platform(), fieldGet(fe.techFields, "meta_framework"))
				f = &fe.a11yFields[fe.a11yFormIdx]
			}
			if f.Key == "meta_tag_injection" {
				fe.a11yFields = refreshMetaTagOptions(fe.a11yFields, fieldGet(fe.techFields, "framework"))
				f = &fe.a11yFields[fe.a11yFormIdx]
			}
			fe.dd.Open = true
			fe.dd.OptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.a11yFields[fe.a11yFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "D":
		fe.a11yEnabled = false
		fe.a11yFields = defaultA11ySEOFields()
		fe.a11yFormIdx = 0
	case "i", "a":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateA11ySEODropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.a11yFormIdx >= len(fe.a11yFields) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.a11yFields[fe.a11yFormIdx]
	fe.dd.OptIdx = NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = fe.dd.OptIdx
		if fe.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[fe.dd.OptIdx]
		}
		fe.dd.Open = false
	case "esc", "b":
		fe.dd.Open = false
	}
	return fe, nil
}

func (fe FrontendEditor) viewComponents(w int) []string {
	switch fe.compSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Components — a: add  d: delete  Enter: edit"), "")
		if len(fe.components) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no components yet — press 'a' to add)"))
		} else {
			for i, c := range fe.components {
				name := c.Name
				if name == "" {
					name = fmt.Sprintf("(component #%d)", i+1)
				}
				detail := c.ComponentType
				nAct := len(c.Actions)
				if nAct == 1 {
					detail += "  [1 action]"
				} else if nAct > 1 {
					detail += fmt.Sprintf("  [%d actions]", nAct)
				}
				lines = append(lines, renderListItem(w, i == fe.compIdx, "  ▶ ", name, detail))
			}
		}
		return lines

	case ceViewForm:
		if fe.inCompAction {
			return fe.viewComponentActions(w)
		}
		compName := fieldGet(fe.compForm, "name")
		if compName == "" {
			compName = "(new component)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(compName)+StyleSectionDesc.Render("  [A: actions]"), "")
		lines = append(lines, renderFormFields(w, fe.compForm, fe.compFormIdx, fe.internalMode == ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)...)
		return lines
	}
	return nil
}

func (fe FrontendEditor) viewPages(w int) []string {
	switch fe.pageSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Pages — a: add  d: delete  Enter: edit"), "")
		if len(fe.pages) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no pages yet — press 'a' to add)"))
		} else {
			for i, p := range fe.pages {
				name := p.Name
				if name == "" {
					name = fmt.Sprintf("(page #%d)", i+1)
				}
				detail := p.Route
				if p.ComponentRefs != "" {
					detail += "  [components: " + p.ComponentRefs + "]"
				}
				lines = append(lines, renderListItem(w, i == fe.pageIdx, "  ▶ ", name, detail))
			}
		}
		return lines

	case ceViewForm:
		name := fieldGet(fe.pageForm, "name")
		if name == "" {
			name = "(new page)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFields(w, fe.pageForm, fe.pageFormIdx, fe.internalMode == ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)...)
		return lines
	}
	return nil
}

func (fe FrontendEditor) viewComponentActions(w int) []string {
	compName := ""
	if fe.compIdx < len(fe.components) {
		compName = fe.components[fe.compIdx].Name
	}
	if compName == "" {
		compName = "(component)"
	}

	switch fe.actionSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(compName)+StyleSectionDesc.Render(" › Actions — a: add  d: delete  Enter: edit"), "")
		if len(fe.compActions) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no actions yet — press 'a' to add)"))
		} else {
			for i, a := range fe.compActions {
				label := a.Trigger
				if label == "" {
					label = fmt.Sprintf("(action #%d)", i+1)
				}
				detail := a.ActionType
				switch {
				case a.Endpoint != "":
					detail += " → " + a.Endpoint
				case a.FormTarget != "":
					detail += " → " + a.FormTarget
				case a.ModalTarget != "":
					detail += " → " + a.ModalTarget
				case a.TargetPage != "":
					detail += " → " + a.TargetPage
				case a.ToastMessage != "":
					detail += ": " + a.ToastMessage
				case a.StateKey != "":
					detail += " → " + a.StateKey
				}
				lines = append(lines, renderListItem(w, i == fe.actionIdx, "  ▶ ", label, detail))
			}
		}
		return lines

	case ceViewForm:
		trigger := fieldGet(fe.actionForm, "trigger")
		if trigger == "" {
			trigger = "(new action)"
		}
		vis := actionVisibleFields(fe.actionForm)
		visIdx := actionVisibleIdx(fe.actionForm, fe.actionFormIdx)
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(compName+" › "+trigger), "")
		lines = append(lines, renderFormFields(w, vis, visIdx, fe.internalMode == ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)...)
		return lines
	}
	return nil
}
