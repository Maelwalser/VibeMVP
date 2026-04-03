package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

func (fe FrontendEditor) updateTech(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if !fe.techEnabled {
		if key.String() == "a" {
			fe.techEnabled = true
			fe.techFormIdx = 0
		}
		return fe, nil
	}
	if fe.dd.Open {
		return fe.updateTechDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.techFormIdx < len(fe.techFields)-1 {
			fe.techFormIdx++
		}
	case "k", "up":
		if fe.techFormIdx > 0 {
			fe.techFormIdx--
		}
	case "enter", " ":
		f := &fe.techFields[fe.techFormIdx]
		if f.Kind == KindSelect {
			fe.dd.Open = true
			fe.dd.OptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.techFields[fe.techFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
			if f.Key == "language" || f.Key == "platform" || f.Key == "framework" || f.Key == "language_version" {
				fe.updateFEDependentOptions()
			}
		}
	case "D":
		fe.techEnabled = false
		fe.techFields = defaultFETechFields()
		fe.techFormIdx = 0
	case "i", "a":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateTechDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.techFormIdx >= len(fe.techFields) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.techFields[fe.techFormIdx]
	fe.dd.OptIdx = NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = fe.dd.OptIdx
		if fe.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[fe.dd.OptIdx]
		}
		fe.dd.Open = false
		if f.Key == "language" || f.Key == "platform" || f.Key == "framework" || f.Key == "language_version" {
			fe.updateFEDependentOptions()
		}
		if f.PrepareCustomEntry() {
			return fe.tryEnterInsert()
		}
	case "esc", "b":
		fe.dd.Open = false
	}
	return fe, nil
}

func (fe FrontendEditor) updateTheme(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if !fe.themeEnabled {
		if key.String() == "a" {
			fe.themeEnabled = true
			fe.themeFormIdx = 0
		}
		return fe, nil
	}
	if fe.dd.Open {
		return fe.updateThemeDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.themeFormIdx < len(fe.themeFields)-1 {
			fe.themeFormIdx++
		}
	case "k", "up":
		if fe.themeFormIdx > 0 {
			fe.themeFormIdx--
		}
	case "enter", " ":
		f := &fe.themeFields[fe.themeFormIdx]
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
		f := &fe.themeFields[fe.themeFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "D":
		fe.themeEnabled = false
		fe.themeFields = defaultFEThemeFields()
		fe.themeFormIdx = 0
	case "i", "a":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateThemeDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.themeFormIdx >= len(fe.themeFields) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.themeFields[fe.themeFormIdx]
	fe.dd.OptIdx = NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == KindMultiSelect {
			f.ToggleMultiSelect(fe.dd.OptIdx)
			f.DDCursor = fe.dd.OptIdx
		} else {
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
		} else {
			f.SelIdx = fe.dd.OptIdx
			if fe.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[fe.dd.OptIdx]
			}
			if f.PrepareCustomEntry() {
				return fe.tryEnterInsert()
			}
		}
		fe.dd.Open = false
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		}
		fe.dd.Open = false
	}
	return fe, nil
}

func (fe FrontendEditor) updatePages(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.pageSubView == ceViewList {
		return fe.updatePageList(key)
	}
	if fe.inPageComp {
		if fe.compSubView == ceViewList {
			return fe.updateCompList(key)
		}
		return fe.updateCompForm(key)
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
		fe.pageForm = defaultPageFormFields(fe.availableAuthRoles, fe.pageRoutes(), fe.assetNames())
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
			fe.pageForm = defaultPageFormFields(fe.availableAuthRoles, otherRoutes, fe.assetNames())
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
			// Restore multiselect for auth_roles
			if p.AuthRoles != "" {
				for i := range fe.pageForm {
					if fe.pageForm[i].Key == "auth_roles" {
						for _, sel := range strings.Split(p.AuthRoles, ", ") {
							for j, opt := range fe.pageForm[i].Options {
								if opt == strings.TrimSpace(sel) {
									fe.pageForm[i].SelectedIdxs = append(fe.pageForm[i].SelectedIdxs, j)
								}
							}
						}
						break
					}
				}
			}
			// Restore multiselect for linked_pages
			if p.LinkedPages != "" {
				for i := range fe.pageForm {
					if fe.pageForm[i].Key == "linked_pages" {
						for _, sel := range strings.Split(p.LinkedPages, ", ") {
							for j, opt := range fe.pageForm[i].Options {
								if opt == strings.TrimSpace(sel) {
									fe.pageForm[i].SelectedIdxs = append(fe.pageForm[i].SelectedIdxs, j)
								}
							}
						}
						break
					}
				}
			}
			// Restore multiselect for assets
			if p.Assets != "" {
				for i := range fe.pageForm {
					if fe.pageForm[i].Key == "assets" {
						for _, sel := range strings.Split(p.Assets, ", ") {
							for j, opt := range fe.pageForm[i].Options {
								if opt == strings.TrimSpace(sel) {
									fe.pageForm[i].SelectedIdxs = append(fe.pageForm[i].SelectedIdxs, j)
								}
							}
						}
						break
					}
				}
			}
			fe.pageFormIdx = 0
			fe.pageSubView = ceViewForm
		}
	}
	return fe, nil
}

func (fe FrontendEditor) updatePageForm(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	// Handle dropdown if open
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
	case "c", "C":
		fe.savePageForm()
		if fe.pageIdx < len(fe.pages) {
			fe.pageComps = make([]manifest.PageComponentDef, len(fe.pages[fe.pageIdx].Components))
			copy(fe.pageComps, fe.pages[fe.pageIdx].Components)
		} else {
			fe.pageComps = nil
		}
		fe.inPageComp = true
		fe.compSubView = ceViewList
		fe.compIdx = 0
	case "b", "esc":
		fe.savePageForm()
		fe.inPageComp = false
		fe.pageSubView = ceViewList
	}
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
}

func (fe *FrontendEditor) saveCompsToPage() {
	if fe.pageIdx >= len(fe.pages) {
		return
	}
	comps := make([]manifest.PageComponentDef, len(fe.pageComps))
	copy(comps, fe.pageComps)
	fe.pages[fe.pageIdx].Components = comps
}

func (fe *FrontendEditor) saveCompForm() {
	if fe.compIdx >= len(fe.pageComps) {
		return
	}
	c := &fe.pageComps[fe.compIdx]
	c.Name = fieldGet(fe.compForm, "name")
	c.ComponentType = fieldGet(fe.compForm, "comp_type")
	c.ConnectedEndpoints = fieldGetMulti(fe.compForm, "endpoints")
	reqDTO := fieldGet(fe.compForm, "request_dto")
	if reqDTO == "None" {
		reqDTO = ""
	}
	c.RequestDTO = reqDTO
	respDTO := fieldGet(fe.compForm, "response_dto")
	if respDTO == "None" {
		respDTO = ""
	}
	c.ResponseDTO = respDTO
	c.Description = fieldGet(fe.compForm, "description")
}

func (fe FrontendEditor) updateCompList(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	n := len(fe.pageComps)
	switch key.String() {
	case "j", "down":
		if n > 0 && fe.compIdx < n-1 {
			fe.compIdx++
		}
	case "k", "up":
		if fe.compIdx > 0 {
			fe.compIdx--
		}
	case "a":
		fe.pageComps = append(fe.pageComps, manifest.PageComponentDef{})
		fe.compIdx = len(fe.pageComps) - 1
		existing := make([]string, 0, len(fe.pageComps)-1)
		for i, c := range fe.pageComps {
			if i != fe.compIdx {
				existing = append(existing, c.Name)
			}
		}
		fe.compForm = defaultComponentFormFields(fe.availableEndpoints, fe.availableDTOs)
		fe.compForm = setFieldValue(fe.compForm, "name", uniqueName("component", existing))
		fe.compFormIdx = 0
		fe.compSubView = ceViewForm
		return fe.tryEnterInsert()
	case "d":
		if n > 0 {
			fe.pageComps = append(fe.pageComps[:fe.compIdx], fe.pageComps[fe.compIdx+1:]...)
			if fe.compIdx > 0 && fe.compIdx >= len(fe.pageComps) {
				fe.compIdx = len(fe.pageComps) - 1
			}
			fe.saveCompsToPage()
		}
	case "enter":
		if n > 0 {
			c := fe.pageComps[fe.compIdx]
			fe.compForm = defaultComponentFormFields(fe.availableEndpoints, fe.availableDTOs)
			fe.compForm = setFieldValue(fe.compForm, "name", c.Name)
			fe.compForm = setFieldValue(fe.compForm, "comp_type", c.ComponentType)
			reqDTO := c.RequestDTO
			if reqDTO == "" {
				reqDTO = "None"
			}
			fe.compForm = setFieldValue(fe.compForm, "request_dto", reqDTO)
			respDTO := c.ResponseDTO
			if respDTO == "" {
				respDTO = "None"
			}
			fe.compForm = setFieldValue(fe.compForm, "response_dto", respDTO)
			fe.compForm = setFieldValue(fe.compForm, "description", c.Description)
			// Restore multiselect for endpoints
			if c.ConnectedEndpoints != "" {
				for i := range fe.compForm {
					if fe.compForm[i].Key == "endpoints" {
						for _, sel := range strings.Split(c.ConnectedEndpoints, ", ") {
							for j, opt := range fe.compForm[i].Options {
								if opt == strings.TrimSpace(sel) {
									fe.compForm[i].SelectedIdxs = append(fe.compForm[i].SelectedIdxs, j)
								}
							}
						}
						break
					}
				}
			}
			fe.compFormIdx = 0
			fe.compSubView = ceViewForm
		}
	case "b", "esc":
		fe.saveCompsToPage()
		fe.inPageComp = false
	}
	return fe, nil
}

func (fe FrontendEditor) updateCompForm(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.dd.Open {
		return fe.updateCompFormDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.compFormIdx < len(fe.compForm)-1 {
			fe.compFormIdx++
		}
	case "k", "up":
		if fe.compFormIdx > 0 {
			fe.compFormIdx--
		}
	case "enter", " ":
		f := &fe.compForm[fe.compFormIdx]
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
		f := &fe.compForm[fe.compFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if fe.compForm[fe.compFormIdx].CanEditAsText() {
			return fe.tryEnterInsert()
		}
	case "b", "esc":
		fe.saveCompForm()
		fe.saveCompsToPage()
		fe.compSubView = ceViewList
	}
	return fe, nil
}

func (fe FrontendEditor) updateCompFormDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.compFormIdx >= len(fe.compForm) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.compForm[fe.compFormIdx]
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
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		}
		fe.dd.Open = false
	}
	return fe, nil
}

func (fe FrontendEditor) updateNav(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if !fe.navEnabled {
		if key.String() == "a" {
			fe.navEnabled = true
			fe.navFormIdx = 0
		}
		return fe, nil
	}
	if fe.dd.Open {
		return fe.updateNavDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.navFormIdx < len(fe.navFields)-1 {
			fe.navFormIdx++
		}
	case "k", "up":
		if fe.navFormIdx > 0 {
			fe.navFormIdx--
		}
	case "enter", " ":
		f := &fe.navFields[fe.navFormIdx]
		if f.Kind == KindSelect {
			fe.dd.Open = true
			fe.dd.OptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.navFields[fe.navFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "D":
		fe.navEnabled = false
		fe.navFields = defaultNavFields()
		fe.navFormIdx = 0
	case "i", "a":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateNavDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.navFormIdx >= len(fe.navFields) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.navFields[fe.navFormIdx]
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

// ── View ──────────────────────────────────────────────────────────────────────

func (fe FrontendEditor) View(w, h int) string {
	fe.width = w
	fe.formInput.Width = w - 22
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Frontend — technologies, theming, pages, and navigation"),
		"",
		renderSubTabBar(feTabLabels, int(fe.activeTab), w),
		"",
	)
	const feHeaderH = 4

	switch fe.activeTab {
	case feTabTech:
		if fe.techEnabled {
			fl := renderFormFields(w, fe.techFields, fe.techFormIdx, fe.internalMode == ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)
			lines = append(lines, appendViewport(fl, 0, fe.techFormIdx, h-feHeaderH)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case feTabTheme:
		if fe.themeEnabled {
			if fe.inTextArea && fe.internalMode == ModeInsert {
				// Render all fields in normal mode, then show the textarea expanded below.
				lines = append(lines, renderFormFields(w, fe.themeFields, fe.themeFormIdx, false, fe.formInput, false, 0)...)
				lines = append(lines, "")
				lines = append(lines, StyleFieldKeyActive.Render("  ── description ─────────────────────────────────────"))
				taHeight := h - feHeaderH - len(fe.themeFields) - 3
				if taHeight < 4 {
					taHeight = 4
				}
				fe.formTextArea.SetHeight(taHeight)
				fe.formTextArea.SetWidth(w - 4)
				lines = append(lines, fe.formTextArea.View())
			} else {
				fl := renderFormFields(w, fe.themeFields, fe.themeFormIdx, fe.internalMode == ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)
				lines = append(lines, appendViewport(fl, 0, fe.themeFormIdx, h-feHeaderH)...)
			}
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case feTabPages:
		pageLines := fe.viewPages(w)
		if fe.pageSubView == ceViewList {
			pageLines = appendViewport(pageLines, 2, fe.pageIdx, h-feHeaderH)
		} else if fe.inPageComp && fe.compSubView == ceViewList {
			pageLines = appendViewport(pageLines, 2, fe.compIdx, h-feHeaderH)
		}
		lines = append(lines, pageLines...)
	case feTabNav:
		if fe.navEnabled {
			fl := renderFormFields(w, fe.navFields, fe.navFormIdx, fe.internalMode == ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)
			lines = append(lines, appendViewport(fl, 0, fe.navFormIdx, h-feHeaderH)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case feTabI18n:
		if fe.i18nEnabled {
			fl := renderFormFields(w, fe.i18nFields, fe.i18nFormIdx, fe.internalMode == ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)
			lines = append(lines, appendViewport(fl, 0, fe.i18nFormIdx, h-feHeaderH)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case feTabA11ySEO:
		if fe.a11yEnabled {
			fl := renderFormFields(w, fe.a11yFields, fe.a11yFormIdx, fe.internalMode == ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)
			lines = append(lines, appendViewport(fl, 0, fe.a11yFormIdx, h-feHeaderH)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case feTabAssets:
		assetLines := fe.viewAssets(w)
		if fe.assetSubView == ceViewList {
			assetLines = appendViewport(assetLines, 2, fe.assetIdx, h-feHeaderH)
		}
		lines = append(lines, assetLines...)
	}

	return fillTildes(lines, h)
}

