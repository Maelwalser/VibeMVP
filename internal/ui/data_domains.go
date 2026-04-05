package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── Domain update ─────────────────────────────────────────────────────────────

func (dt DataTabEditor) updateDomains(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch dt.domainSubView {
	case domainViewList:
		return dt.updateDomainList(key)
	case domainViewForm:
		return dt.updateDomainForm(key)
	case domainViewAttrs:
		return dt.updateAttrList(key)
	case domainViewAttrForm:
		return dt.updateAttrForm(key)
	case domainViewRels:
		return dt.updateRelList(key)
	case domainViewRelForm:
		return dt.updateRelForm(key)
	}
	return dt, nil
}

func (dt DataTabEditor) updateDomainList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.domains)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.domainIdx < n-1 {
			dt.domainIdx++
		}
	case "k", "up":
		if dt.domainIdx > 0 {
			dt.domainIdx--
		}
	case "u":
		if snap, ok := dt.domainsUndo.Pop(); ok {
			dt.domains = snap
			if dt.domainIdx >= len(dt.domains) && dt.domainIdx > 0 {
				dt.domainIdx = len(dt.domains) - 1
			}
		}
	case "a":
		dt.domainsUndo.Push(copySlice(dt.domains))
		dt.domains = append(dt.domains, manifest.DomainDef{})
		dt.domainIdx = len(dt.domains) - 1
		dt.domainForm = defaultDomainFormFields(dt.dbNames())
		existing := make([]string, 0, len(dt.domains)-1)
		for i, d := range dt.domains {
			if i != dt.domainIdx {
				existing = append(existing, d.Name)
			}
		}
		dt.domainForm = setFieldValue(dt.domainForm, "name", uniqueName("domain", existing))
		dt.domainFormIdx = 0
		dt.attrItems = nil
		dt.relItems = nil
		dt.domainSubView = domainViewForm
		return dt.tryEnterInsert()
	case "d":
		if n > 0 {
			dt.domainsUndo.Push(copySlice(dt.domains))
			dt.domains = append(dt.domains[:dt.domainIdx], dt.domains[dt.domainIdx+1:]...)
			if dt.domainIdx > 0 && dt.domainIdx >= len(dt.domains) {
				dt.domainIdx = len(dt.domains) - 1
			}
		}
	case "enter":
		if n > 0 {
			d := dt.domains[dt.domainIdx]
			dbOpts := dt.dbNames()
			dt.domainForm = defaultDomainFormFields(dbOpts)
			dt.domainForm = setFieldValue(dt.domainForm, "name", d.Name)
			dt.domainForm = setFieldValue(dt.domainForm, "description", d.Description)
			// Restore multiselect for databases
			if d.Databases != "" {
				for i := range dt.domainForm {
					if dt.domainForm[i].Key == "databases" {
						for _, sel := range strings.Split(d.Databases, ", ") {
							for j, opt := range dt.domainForm[i].Options {
								if opt == sel {
									dt.domainForm[i].SelectedIdxs = append(dt.domainForm[i].SelectedIdxs, j)
								}
							}
						}
						break
					}
				}
			}
			dt.domainFormIdx = 0
			// Rebuild attr items
			attrTypes, _ := attrTypesForSources(d.Databases, dt.dbEditor.Sources)
			dt.attrItems = make([][]Field, len(d.Attributes))
			for i, attr := range d.Attributes {
				f := defaultAttrFields(attrTypes)
				f = setFieldValue(f, "name", attr.Name)
				f = setFieldValue(f, "type", attr.Type)
				f = restoreMultiSelectValue(f, "constraints", attr.Constraints)
				f = setFieldValue(f, "default", attr.Default)
				if attr.Sensitive {
					f = setFieldValue(f, "sensitive", "true")
				}
				f = restoreMultiSelectValue(f, "validation", attr.Validation)
				dt.attrItems[i] = f
			}
			// Rebuild rel items
			domOpts := dt.domainNames()
			dt.relItems = make([][]Field, len(d.Relationships))
			for i, rel := range d.Relationships {
				f := defaultRelFields(domOpts)
				if rel.RelatedDomain != "" {
					f = setFieldValue(f, "related_domain", rel.RelatedDomain)
				}
				if rel.RelType != "" {
					f = setFieldValue(f, "rel_type", rel.RelType)
				}
				if rel.Cascade != "" {
					f = setFieldValue(f, "cascade", rel.Cascade)
				}
				dt.relItems[i] = f
			}
			dt.domainSubView = domainViewForm
		}
	}
	return dt, nil
}

func (dt DataTabEditor) updateDomainForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.domainFormIdx < len(dt.domainForm)-1 {
			dt.domainFormIdx++
		}
	case "k", "up":
		if dt.domainFormIdx > 0 {
			dt.domainFormIdx--
		}
	case "enter", " ":
		f := &dt.domainForm[dt.domainFormIdx]
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
		f := &dt.domainForm[dt.domainFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.domainForm[dt.domainFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "A":
		dt.saveDomainForm()
		dt.attrIdx = 0
		dt.domainSubView = domainViewAttrs
	case "R":
		dt.saveDomainForm()
		dt.relIdx = 0
		dt.domainSubView = domainViewRels
	case "b", "esc":
		dt.saveDomainForm()
		dt.domainSubView = domainViewList
	}
	dt.saveDomainForm()
	return dt, nil
}

func (dt *DataTabEditor) saveDomainForm() {
	if dt.domainIdx >= len(dt.domains) {
		return
	}
	d := &dt.domains[dt.domainIdx]
	d.Name = fieldGet(dt.domainForm, "name")
	d.Description = fieldGet(dt.domainForm, "description")
	d.Databases = fieldGetMulti(dt.domainForm, "databases")

	// Save attrs
	d.Attributes = make([]manifest.DomainAttribute, len(dt.attrItems))
	for i, item := range dt.attrItems {
		d.Attributes[i] = manifest.DomainAttribute{
			Name:        fieldGet(item, "name"),
			Type:        fieldGet(item, "type"),
			Constraints: fieldGet(item, "constraints"),
			Default:     fieldGet(item, "default"),
			Sensitive:   fieldGet(item, "sensitive") == "true",
			Validation:  fieldGet(item, "validation"),
			Indexed:     fieldGet(item, "indexed") == "true",
			Unique:      fieldGet(item, "unique") == "true",
		}
	}

	// Save rels (no FK field — auto-inferred from rel_type)
	d.Relationships = make([]manifest.DomainRelationship, len(dt.relItems))
	for i, item := range dt.relItems {
		relType := fieldGet(item, "rel_type")
		relDomain := fieldGet(item, "related_domain")
		// Auto-generate FK name
		fk := ""
		if relDomain != "" {
			switch relType {
			case "One-to-Many":
				fk = strings.ToLower(relDomain) + "_id"
			case "One-to-One":
				fk = strings.ToLower(relDomain) + "_id"
			case "Many-to-Many":
				fk = "" // junction table; no single FK
			}
		}
		d.Relationships[i] = manifest.DomainRelationship{
			RelatedDomain: relDomain,
			RelType:       relType,
			ForeignKey:    fk,
			Cascade:       fieldGet(item, "cascade"),
		}
	}
}

// saveDomainAttrItemsOnly saves attrItems back to the current domain's Attributes
// without touching name/description/databases/attr_names fields.
func (dt *DataTabEditor) saveDomainAttrItemsOnly() {
	if dt.domainIdx >= len(dt.domains) {
		return
	}
	d := &dt.domains[dt.domainIdx]
	d.Attributes = make([]manifest.DomainAttribute, len(dt.attrItems))
	for i, item := range dt.attrItems {
		d.Attributes[i] = manifest.DomainAttribute{
			Name:        fieldGet(item, "name"),
			Type:        fieldGet(item, "type"),
			Constraints: fieldGet(item, "constraints"),
			Default:     fieldGet(item, "default"),
			Sensitive:   fieldGet(item, "sensitive") == "true",
			Validation:  fieldGet(item, "validation"),
			Indexed:     fieldGet(item, "indexed") == "true",
			Unique:      fieldGet(item, "unique") == "true",
		}
	}
}

func (dt DataTabEditor) updateAttrList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.attrItems)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.attrIdx < n-1 {
			dt.attrIdx++
		}
	case "k", "up":
		if dt.attrIdx > 0 {
			dt.attrIdx--
		}
	case "a":
		dt.attrItems = append(dt.attrItems, defaultAttrFields(dt.currentDomainAttrTypes()))
		dt.saveDomainAttrItemsOnly()
		dt.attrIdx = len(dt.attrItems) - 1
		dt.attrForm = copyFields(dt.attrItems[dt.attrIdx])
		existing := make([]string, 0, len(dt.attrItems)-1)
		for i, item := range dt.attrItems {
			if i != dt.attrIdx {
				existing = append(existing, fieldGet(item, "name"))
			}
		}
		dt.attrForm = setFieldValue(dt.attrForm, "name", uniqueName("attribute", existing))
		dt.attrFormIdx = 0
		dt.domainSubView = domainViewAttrForm
		return dt.tryEnterInsert()
	case "d":
		if n > 0 {
			dt.attrItems = append(dt.attrItems[:dt.attrIdx], dt.attrItems[dt.attrIdx+1:]...)
			if dt.attrIdx > 0 && dt.attrIdx >= len(dt.attrItems) {
				dt.attrIdx = len(dt.attrItems) - 1
			}
			dt.saveDomainAttrItemsOnly()
		}
	case "enter":
		if n > 0 {
			dt.attrForm = copyFields(dt.attrItems[dt.attrIdx])
			dt.attrForm = refreshAttrTypeOptions(dt.attrForm, dt.currentDomainAttrTypes())
			dt.attrFormIdx = 0
			dt.domainSubView = domainViewAttrForm
		}
	case "b", "esc":
		dt.saveDomainAttrItemsOnly()
		dt.domainSubView = domainViewForm
	}
	return dt, nil
}

func (dt DataTabEditor) updateAttrForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.attrFormIdx < len(dt.attrForm)-1 {
			dt.attrFormIdx++
		}
	case "k", "up":
		if dt.attrFormIdx > 0 {
			dt.attrFormIdx--
		}
	case "enter", " ":
		f := &dt.attrForm[dt.attrFormIdx]
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
		f := &dt.attrForm[dt.attrFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.attrForm[dt.attrFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.attrIdx < len(dt.attrItems) {
			dt.attrItems[dt.attrIdx] = copyFields(dt.attrForm)
		}
		dt.saveDomainAttrItemsOnly()
		dt.domainSubView = domainViewAttrs
	}
	if dt.attrIdx < len(dt.attrItems) {
		dt.attrItems[dt.attrIdx] = copyFields(dt.attrForm)
	}
	dt.saveDomainAttrItemsOnly()
	return dt, nil
}

func (dt DataTabEditor) updateRelList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.relItems)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.relIdx < n-1 {
			dt.relIdx++
		}
	case "k", "up":
		if dt.relIdx > 0 {
			dt.relIdx--
		}
	case "a":
		dt.relItems = append(dt.relItems, defaultRelFields(dt.domainNames()))
		dt.relIdx = len(dt.relItems) - 1
		dt.relForm = copyFields(dt.relItems[dt.relIdx])
		dt.relFormIdx = 0
		dt.domainSubView = domainViewRelForm
	case "d":
		if n > 0 {
			dt.relItems = append(dt.relItems[:dt.relIdx], dt.relItems[dt.relIdx+1:]...)
			if dt.relIdx > 0 && dt.relIdx >= len(dt.relItems) {
				dt.relIdx = len(dt.relItems) - 1
			}
		}
	case "enter":
		if n > 0 {
			dt.relForm = copyFields(dt.relItems[dt.relIdx])
			dt.relFormIdx = 0
			dt.domainSubView = domainViewRelForm
		}
	case "b", "esc":
		dt.domainSubView = domainViewForm
	}
	return dt, nil
}

func (dt DataTabEditor) updateRelForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.relFormIdx < len(dt.relForm)-1 {
			dt.relFormIdx++
		}
	case "k", "up":
		if dt.relFormIdx > 0 {
			dt.relFormIdx--
		}
	case "enter", " ":
		f := &dt.relForm[dt.relFormIdx]
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
		f := &dt.relForm[dt.relFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.relForm[dt.relFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.relIdx < len(dt.relItems) {
			dt.relItems[dt.relIdx] = copyFields(dt.relForm)
		}
		dt.domainSubView = domainViewRels
	}
	if dt.relIdx < len(dt.relItems) {
		dt.relItems[dt.relIdx] = copyFields(dt.relForm)
	}
	return dt, nil
}

func (dt DataTabEditor) viewDomains(w int) []string {
	switch dt.domainSubView {
	case domainViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Domains — a: add  d: delete  Enter: edit"), "")
		if len(dt.domains) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no domains yet — press 'a' to add one)"))
		} else {
			for i, d := range dt.domains {
				desc := d.Description
				if len(desc) > 40 {
					desc = desc[:39] + "…"
				}
				lines = append(lines, renderListItem(w, i == dt.domainIdx, "  ▶ ", d.Name, desc))
			}
		}
		return lines

	case domainViewForm:
		var lines []string
		name := fieldGet(dt.domainForm, "name")
		if name == "" {
			name = "(new domain)"
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFields(w, dt.domainForm, dt.domainFormIdx, dt.internalMode == ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
		lines = append(lines, "", StyleSectionDesc.Render("  A: edit fields  R: edit relationships"))
		attrCount := len(dt.attrItems)
		relCount := len(dt.relItems)
		lines = append(lines, StyleSectionDesc.Render(fmt.Sprintf("  %d field(s)  %d relationship(s)", attrCount, relCount)))
		return lines

	case domainViewAttrs:
		var lines []string
		if dt.domainIdx < len(dt.domains) {
			lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(dt.domains[dt.domainIdx].Name)+StyleSectionDesc.Render(" › Fields"), "")
		}
		if len(dt.attrItems) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no fields — press 'a' to add)"))
		} else {
			for i, item := range dt.attrItems {
				attrName := fieldGet(item, "name")
				attrType := fieldGet(item, "type")
				if attrName == "" {
					attrName = fmt.Sprintf("(attr #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == dt.attrIdx, "  ▶ ", attrName, attrType))
			}
		}
		return lines

	case domainViewAttrForm:
		var lines []string
		attrName := fieldGet(dt.attrForm, "name")
		if attrName == "" {
			attrName = "(new attribute)"
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(attrName), "")
		lines = append(lines, renderFormFields(w, dt.attrForm, dt.attrFormIdx, dt.internalMode == ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
		return lines

	case domainViewRels:
		var lines []string
		if dt.domainIdx < len(dt.domains) {
			lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(dt.domains[dt.domainIdx].Name)+StyleSectionDesc.Render(" › Relationships"), "")
		}
		if len(dt.relItems) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no relationships — press 'a' to add)"))
		} else {
			for i, item := range dt.relItems {
				relName := fieldGet(item, "related_domain")
				relType := fieldGet(item, "rel_type")
				if relName == "" {
					relName = fmt.Sprintf("(rel #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == dt.relIdx, "  ▶ ", relName, relType))
			}
		}
		return lines

	case domainViewRelForm:
		var lines []string
		relName := fieldGet(dt.relForm, "related_domain")
		if relName == "" {
			relName = "(new relationship)"
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(relName), "")
		lines = append(lines, renderFormFields(w, dt.relForm, dt.relFormIdx, dt.internalMode == ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
		return lines
	}
	return nil
}
