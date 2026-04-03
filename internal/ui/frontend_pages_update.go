package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

func (fe FrontendEditor) updatePages(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.pageSubView == ceViewList {
		return fe.updatePageList(key)
	}
	return fe.updatePageForm(key)
}

func (fe FrontendEditor) updatePageList(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	n := len(fe.pages)
	switch key.String() {
	case "j", "down":
		if n > 0 && fe.pageIdx < n-1 {
			fe.pageIdx++
		}
	case "k", "up":
		if fe.pageIdx > 0 {
			fe.pageIdx--
		}
	case "a":
		fe.pages = append(fe.pages, manifest.PageDef{})
		fe.pageIdx = len(fe.pages) - 1
		fe.pageForm = defaultPageFormFields(fe.availableAuthRoles, fe.pageRoutes(), fe.assetNames(), fe.componentNames())
		existing := make([]string, 0, len(fe.pages)-1)
		for i, p := range fe.pages {
			if i != fe.pageIdx {
				existing = append(existing, p.Name)
			}
		}
		name := uniqueName("page", existing)
		fe.pageForm = setFieldValue(fe.pageForm, "name", name)
		fe.pageForm = setFieldValue(fe.pageForm, "route", "/"+name)
		fe.pageFormIdx = 0
		fe.pageSubView = ceViewForm
		return fe.tryEnterInsert()
	case "d":
		if n > 0 {
			fe.pages = append(fe.pages[:fe.pageIdx], fe.pages[fe.pageIdx+1:]...)
			if fe.pageIdx > 0 && fe.pageIdx >= len(fe.pages) {
				fe.pageIdx = len(fe.pages) - 1
			}
		}
	case "enter":
		if n > 0 {
			p := fe.pages[fe.pageIdx]
			// Exclude current page's route from linked_pages options
			otherRoutes := make([]string, 0, len(fe.pages))
			for i, pg := range fe.pages {
				if i != fe.pageIdx && pg.Route != "" {
					otherRoutes = append(otherRoutes, pg.Route)
				}
			}
			fe.pageForm = defaultPageFormFields(fe.availableAuthRoles, otherRoutes, fe.assetNames(), fe.componentNames())
			fe.pageForm = setFieldValue(fe.pageForm, "name", p.Name)
			fe.pageForm = setFieldValue(fe.pageForm, "route", p.Route)
			if p.Purpose != "" {
				fe.pageForm = setFieldValue(fe.pageForm, "purpose", p.Purpose)
			}
			fe.pageForm = setFieldValue(fe.pageForm, "auth_required", p.AuthRequired)
			if p.Layout != "" {
				fe.pageForm = setFieldValue(fe.pageForm, "layout", p.Layout)
			}
			fe.pageForm = setFieldValue(fe.pageForm, "description", p.Description)
			fe.pageForm = setFieldValue(fe.pageForm, "core_actions", p.CoreActions)
			if p.Loading != "" {
				fe.pageForm = setFieldValue(fe.pageForm, "loading", p.Loading)
			}
			if p.ErrorHandling != "" {
				fe.pageForm = setFieldValue(fe.pageForm, "error_handling", p.ErrorHandling)
			}
			restoreMultiField(fe.pageForm, "auth_roles", p.AuthRoles)
			restoreMultiField(fe.pageForm, "linked_pages", p.LinkedPages)
			restoreMultiField(fe.pageForm, "assets", p.Assets)
			restoreMultiField(fe.pageForm, "component_refs", p.ComponentRefs)
			fe.pageFormIdx = 0
			fe.pageSubView = ceViewForm
		}
	}
	return fe, nil
}

// restoreMultiField restores a comma-separated saved value into a KindMultiSelect field.
func restoreMultiField(fields []Field, key, saved string) {
	if saved == "" {
		return
	}
	for i := range fields {
		if fields[i].Key == key {
			for _, sel := range strings.Split(saved, ", ") {
				for j, opt := range fields[i].Options {
					if opt == strings.TrimSpace(sel) {
						fields[i].SelectedIdxs = append(fields[i].SelectedIdxs, j)
					}
				}
			}
			return
		}
	}
}

func (fe FrontendEditor) updatePageForm(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.dd.Open {
		return fe.updatePageFormDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.pageFormIdx < len(fe.pageForm)-1 {
			fe.pageFormIdx++
		}
	case "k", "up":
		if fe.pageFormIdx > 0 {
			fe.pageFormIdx--
		}
	case "enter", " ":
		f := &fe.pageForm[fe.pageFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			fe.dd.Open = true
			if f.Kind == KindSelect {
				fe.dd.OptIdx = f.SelIdx
			} else {
				fe.dd.OptIdx = f.DDCursor
			}
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.pageForm[fe.pageFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if fe.pageForm[fe.pageFormIdx].CanEditAsText() {
			return fe.tryEnterInsert()
		}
	case "b", "esc":
		fe.savePageForm()
		fe.pageSubView = ceViewList
	}
	fe.savePageForm()
	return fe, nil
}

func (fe FrontendEditor) updatePageFormDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.pageFormIdx >= len(fe.pageForm) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.pageForm[fe.pageFormIdx]
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
			f.ToggleMultiSelect(fe.dd.OptIdx)
			f.DDCursor = fe.dd.OptIdx
		} else if f.Kind == KindSelect {
			f.SelIdx = fe.dd.OptIdx
			if fe.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[fe.dd.OptIdx]
			}
			fe.dd.Open = false
			if f.PrepareCustomEntry() {
				fe.savePageForm()
				return fe.tryEnterInsert()
			}
		}
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		}
		fe.dd.Open = false
	}
	fe.savePageForm()
	return fe, nil
}

func (fe *FrontendEditor) savePageForm() {
	if fe.pageIdx >= len(fe.pages) {
		return
	}
	p := &fe.pages[fe.pageIdx]
	p.Name = fieldGet(fe.pageForm, "name")
	p.Route = fieldGet(fe.pageForm, "route")
	p.Purpose = fieldGet(fe.pageForm, "purpose")
	p.AuthRequired = fieldGet(fe.pageForm, "auth_required")
	p.Layout = fieldGet(fe.pageForm, "layout")
	p.Description = fieldGet(fe.pageForm, "description")
	p.CoreActions = fieldGet(fe.pageForm, "core_actions")
	p.Loading = fieldGet(fe.pageForm, "loading")
	p.ErrorHandling = fieldGet(fe.pageForm, "error_handling")
	p.AuthRoles = fieldGetMulti(fe.pageForm, "auth_roles")
	p.LinkedPages = fieldGetMulti(fe.pageForm, "linked_pages")
	p.Assets = fieldGetMulti(fe.pageForm, "assets")
	p.ComponentRefs = fieldGetMulti(fe.pageForm, "component_refs")
}
