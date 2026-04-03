package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── Caching update ────────────────────────────────────────────────────────────

func (dt DataTabEditor) updateCaching(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch dt.cachingSubView {
	case cachingViewList:
		return dt.updateCachingList(key)
	case cachingViewForm:
		return dt.updateCachingForm(key)
	}
	return dt, nil
}

func (dt DataTabEditor) updateCachingList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.cachings)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.cachingIdx < n-1 {
			dt.cachingIdx++
		}
	case "k", "up":
		if dt.cachingIdx > 0 {
			dt.cachingIdx--
		}
	case "a":
		dt.cachings = append(dt.cachings, manifest.CachingConfig{})
		dt.cachingIdx = len(dt.cachings) - 1
		dt.cachingForm = defaultCachingFields()
		existing := make([]string, 0, len(dt.cachings)-1)
		for i, c := range dt.cachings {
			if i != dt.cachingIdx {
				existing = append(existing, c.Name)
			}
		}
		dt.cachingForm = setFieldValue(dt.cachingForm, "name", uniqueName("caching", existing))
		dt.cachingFormIdx = 0
		dt.cachingSubView = cachingViewForm
	case "d":
		if n > 0 {
			dt.cachings = append(dt.cachings[:dt.cachingIdx], dt.cachings[dt.cachingIdx+1:]...)
			if dt.cachingIdx > 0 && dt.cachingIdx >= len(dt.cachings) {
				dt.cachingIdx = len(dt.cachings) - 1
			}
		}
	case "enter":
		if n > 0 {
			dt.cachingForm = cachingFormFromDef(dt.cachings[dt.cachingIdx])
			dt.cachingFormIdx = 0
			dt.cachingSubView = cachingViewForm
		}
	}
	return dt, nil
}

func (dt DataTabEditor) updateCachingForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	// Refresh dynamic options
	dt = dt.withRefreshedCachingEntities()
	dt = dt.withRefreshedCachingDBs()
	switch key.String() {
	case "j", "down":
		dt.cachingFormIdx = nextCachingFormIdx(dt.cachingForm, dt.cachingFormIdx)
	case "k", "up":
		dt.cachingFormIdx = prevCachingFormIdx(dt.cachingForm, dt.cachingFormIdx)
	case "enter", " ":
		f := &dt.cachingForm[dt.cachingFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			dt.dd.Open = true
			if f.Kind == KindSelect {
				dt.dd.OptIdx = f.SelIdx
			} else {
				dt.dd.OptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.cachingForm[dt.cachingFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.cachingForm[dt.cachingFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.cachingIdx < len(dt.cachings) {
			dt.cachings[dt.cachingIdx] = cachingDefFromForm(dt.cachingForm)
		}
		dt.cachingSubView = cachingViewList
	}
	if dt.cachingIdx < len(dt.cachings) {
		dt.cachings[dt.cachingIdx] = cachingDefFromForm(dt.cachingForm)
	}
	return dt, nil
}

// ── Governance update ─────────────────────────────────────────────────────────

func (dt DataTabEditor) updateGovernance(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	if !dt.govEnabled {
		if key.String() == "a" {
			dt.govEnabled = true
			dt.govFormIdx = 0
		}
		return dt, nil
	}
	switch key.String() {
	case "j", "down":
		if dt.govFormIdx < len(dt.governanceFields)-1 {
			dt.govFormIdx++
		}
	case "k", "up":
		if dt.govFormIdx > 0 {
			dt.govFormIdx--
		}
	case "enter", " ":
		f := &dt.governanceFields[dt.govFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			dt.dd.Open = true
			if f.Kind == KindSelect {
				dt.dd.OptIdx = f.SelIdx
			} else {
				dt.dd.OptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.governanceFields[dt.govFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "D":
		dt.govEnabled = false
		dt.governanceFields = defaultGovernanceFields()
		dt.govFormIdx = 0
	case "i", "a":
		if dt.governanceFields[dt.govFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	}
	return dt, nil
}

// ── File storage update ───────────────────────────────────────────────────────

func (dt DataTabEditor) updateFileStorage(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch dt.fsSubView {
	case fsViewList:
		return dt.updateFSList(key)
	case fsViewForm:
		return dt.updateFSForm(key)
	}
	return dt, nil
}

func (dt DataTabEditor) updateFSList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.fileStorages)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.fsIdx < n-1 {
			dt.fsIdx++
		}
	case "k", "up":
		if dt.fsIdx > 0 {
			dt.fsIdx--
		}
	case "a":
		dt.fileStorages = append(dt.fileStorages, manifest.FileStorageDef{})
		dt.fsIdx = len(dt.fileStorages) - 1
		dt.fsForm = defaultFSFormFields(dt.serviceNames, dt.domainNames())
		existing := make([]string, 0, len(dt.fileStorages)-1)
		for i, fs := range dt.fileStorages {
			if i != dt.fsIdx {
				existing = append(existing, fs.Purpose)
			}
		}
		dt.fsForm = setFieldValue(dt.fsForm, "purpose", uniqueName("storage", existing))
		dt.fsFormIdx = 0
		dt.fsSubView = fsViewForm
	case "d":
		if n > 0 {
			dt.fileStorages = append(dt.fileStorages[:dt.fsIdx], dt.fileStorages[dt.fsIdx+1:]...)
			if dt.fsIdx > 0 && dt.fsIdx >= len(dt.fileStorages) {
				dt.fsIdx = len(dt.fileStorages) - 1
			}
		}
	case "enter":
		if n > 0 {
			dt.fsForm = fsFormFromDef(dt.fileStorages[dt.fsIdx], dt.serviceNames, dt.domainNames())
			dt.fsFormIdx = 0
			dt.fsSubView = fsViewForm
		}
	}
	return dt, nil
}

func (dt DataTabEditor) updateFSForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.fsFormIdx < len(dt.fsForm)-1 {
			dt.fsFormIdx++
		}
	case "k", "up":
		if dt.fsFormIdx > 0 {
			dt.fsFormIdx--
		}
	case "enter", " ":
		f := &dt.fsForm[dt.fsFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			dt.dd.Open = true
			if f.Kind == KindSelect {
				dt.dd.OptIdx = f.SelIdx
			} else {
				dt.dd.OptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.fsForm[dt.fsFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.fsForm[dt.fsFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.fsIdx < len(dt.fileStorages) {
			dt.fileStorages[dt.fsIdx] = fsDefFromForm(dt.fsForm)
		}
		dt.fsSubView = fsViewList
	}
	if dt.fsIdx < len(dt.fileStorages) {
		dt.fileStorages[dt.fsIdx] = fsDefFromForm(dt.fsForm)
	}
	return dt, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (dt DataTabEditor) viewCaching(w int) []string {
	var lines []string
	switch dt.cachingSubView {
	case cachingViewList:
		lines = append(lines, StyleSectionDesc.Render("  # Caching Strategies — a: add  d: delete  Enter: edit"), "")
		if len(dt.cachings) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no caching strategies yet — press 'a' to add)"))
		} else {
			for i, c := range dt.cachings {
				name := c.Name
				if name == "" {
					name = fmt.Sprintf("(strategy #%d)", i+1)
				}
				detail := c.Layer
				if c.Strategy != "" {
					detail += " / " + c.Strategy
				}
				lines = append(lines, renderListItem(w, i == dt.cachingIdx, "  ▶ ", name, detail))
			}
		}
	case cachingViewForm:
		dt = dt.withRefreshedCachingEntities()
		dt = dt.withRefreshedCachingDBs()
		name := fieldGet(dt.cachingForm, "name")
		if name == "" {
			name = "(new strategy)"
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		visible := cachingVisibleFields(dt.cachingForm)
		visIdx := cachingVisibleIdx(dt.cachingForm, dt.cachingFormIdx)
		lines = append(lines, renderFormFields(w, visible, visIdx, dt.internalMode == ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
	}
	return lines
}

func (dt DataTabEditor) viewFileStorage(w int) []string {
	switch dt.fsSubView {
	case fsViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # File / Object Storage — a: add  d: delete  Enter: edit"), "")
		if len(dt.fileStorages) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no storage buckets yet — press 'a' to add)"))
		} else {
			for i, fs := range dt.fileStorages {
				tech := fs.Technology
				if tech == "" {
					tech = "?"
				}
				name := fs.Purpose
				if name == "" {
					name = fmt.Sprintf("(storage #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == dt.fsIdx, "  ▶ ", name, tech+" / "+fs.Access))
			}
		}
		return lines

	case fsViewForm:
		var lines []string
		tech := fieldGet(dt.fsForm, "technology")
		if tech == "" {
			tech = "(new storage)"
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(tech), "")
		lines = append(lines, renderFormFields(w, dt.fsForm, dt.fsFormIdx, dt.internalMode == ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
		return lines
	}
	return nil
}

func (dt DataTabEditor) viewGovernance(w int) []string {
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  # Data Governance & Privacy"), "")
	if !dt.govEnabled {
		lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		return lines
	}
	lines = append(lines, renderFormFields(w, dt.governanceFields, dt.govFormIdx, dt.internalMode == ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
	return lines
}

// Expose db sources for syncing into the DataEditor.
func (dt DataTabEditor) DBSources() []manifest.DBSourceDef {
	return dt.dbEditor.Sources
}
