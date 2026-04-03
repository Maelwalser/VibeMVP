package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── DTO updates ───────────────────────────────────────────────────────────────

func (ce ContractsEditor) updateDTOs(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	switch ce.dtoSubView {
	case ceViewList:
		return ce.updateDTOList(key)
	case ceViewForm:
		return ce.updateDTOForm(key)
	case ceViewSubList:
		return ce.updateDTOFieldList(key)
	case ceViewSubForm:
		return ce.updateDTOFieldForm(key)
	}
	return ce, nil
}

func (ce ContractsEditor) updateDTOList(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	n := len(ce.dtos)
	switch key.String() {
	case "j", "down":
		if n > 0 && ce.dtoIdx < n-1 {
			ce.dtoIdx++
		}
	case "k", "up":
		if ce.dtoIdx > 0 {
			ce.dtoIdx--
		}
	case "a":
		ce.dtos = append(ce.dtos, manifest.DTODef{})
		ce.dtoIdx = len(ce.dtos) - 1
		ce.dtoForm = defaultDTOFormFields(ce.availableDomains)
		existing := make([]string, 0, len(ce.dtos)-1)
		for i, d := range ce.dtos {
			if i != ce.dtoIdx {
				existing = append(existing, d.Name)
			}
		}
		ce.dtoForm = setFieldValue(ce.dtoForm, "name", uniqueName("dto", existing))
		ce.dtoFormIdx = 0
		ce.dtoFieldItems = nil
		ce.dtoSubView = ceViewForm
		return ce.tryEnterInsert()
	case "d":
		if n > 0 {
			ce.dtos = append(ce.dtos[:ce.dtoIdx], ce.dtos[ce.dtoIdx+1:]...)
			if ce.dtoIdx > 0 && ce.dtoIdx >= len(ce.dtos) {
				ce.dtoIdx = len(ce.dtos) - 1
			}
		}
	case "enter":
		if n > 0 {
			d := ce.dtos[ce.dtoIdx]
			ce.dtoForm = defaultDTOFormFields(ce.availableDomains)
			ce.dtoForm = setFieldValue(ce.dtoForm, "name", d.Name)
			ce.dtoForm = setFieldValue(ce.dtoForm, "category", d.Category)
			// Restore multiselect for source_domains
			if d.SourceDomains != "" {
				for i := range ce.dtoForm {
					if ce.dtoForm[i].Key == "source_domains" {
						for _, sel := range splitCSV(d.SourceDomains) {
							for j, opt := range ce.dtoForm[i].Options {
								if opt == sel {
									ce.dtoForm[i].SelectedIdxs = append(ce.dtoForm[i].SelectedIdxs, j)
								}
							}
						}
						break
					}
				}
			}
			ce.dtoForm = setFieldValue(ce.dtoForm, "description", d.Description)
			if d.Protocol != "" {
				ce.dtoForm = setFieldValue(ce.dtoForm, "protocol", d.Protocol)
			}
			ce.dtoForm = setFieldValue(ce.dtoForm, "proto_package", d.ProtoPackage)
			ce.dtoForm = setFieldValue(ce.dtoForm, "proto_syntax", d.ProtoSyntax)
			ce.dtoForm = setFieldValue(ce.dtoForm, "proto_options", d.ProtoOptions)
			ce.dtoForm = setFieldValue(ce.dtoForm, "avro_namespace", d.AvroNamespace)
			ce.dtoForm = setFieldValue(ce.dtoForm, "schema_registry", d.SchemaRegistry)
			ce.dtoForm = setFieldValue(ce.dtoForm, "thrift_namespace", d.ThriftNamespace)
			if d.ThriftLanguage != "" {
				ce.dtoForm = setFieldValue(ce.dtoForm, "thrift_language", d.ThriftLanguage)
			}
			ce.dtoForm = setFieldValue(ce.dtoForm, "namespace", d.Namespace)
			ce.dtoFormIdx = 0
			// Rebuild field items
			proto := d.Protocol
			if proto == "" {
				proto = "REST/JSON"
			}
			ce.dtoFieldItems = make([][]Field, len(d.Fields))
			for i, df := range d.Fields {
				f := defaultDTOFieldForm(proto)
				f = setFieldValue(f, "name", df.Name)
				f = setFieldValue(f, "type", df.Type)
				if df.Required {
					f = setFieldValue(f, "required", "true")
				}
				if df.Nullable {
					f = setFieldValue(f, "nullable", "true")
				}
				f = restoreMultiSelectValue(f, "validation", df.Validation)
				f = setFieldValue(f, "default", df.Default)
				f = setFieldValue(f, "field_number", df.FieldNumber)
				if df.ProtoModifier != "" {
					f = setFieldValue(f, "proto_modifier", df.ProtoModifier)
				}
				f = setFieldValue(f, "json_name", df.JsonName)
				f = setFieldValue(f, "field_id", df.FieldID)
				if df.ThriftModifier != "" {
					f = setFieldValue(f, "thrift_mod", df.ThriftModifier)
				}
				if df.Deprecated {
					f = setFieldValue(f, "deprecated", "true")
				}
				f = setFieldValue(f, "notes", df.Notes)
				ce.dtoFieldItems[i] = f
			}
			ce.dtoSubView = ceViewForm
		}
	}
	return ce, nil
}


func (ce ContractsEditor) updateDTOForm(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	visible := ce.visibleDTOFields()
	switch key.String() {
	case "j", "down":
		if ce.dtoFormIdx < len(visible)-1 {
			ce.dtoFormIdx++
		}
	case "k", "up":
		if ce.dtoFormIdx > 0 {
			ce.dtoFormIdx--
		}
	case "enter", " ":
		f := ce.activeCEFieldPtr()
		if f == nil {
			break
		}
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			ce.dd.Open = true
			if f.Kind == KindSelect {
				ce.dd.OptIdx = f.SelIdx
			} else {
				ce.dd.OptIdx = f.DDCursor
			}
		} else {
			return ce.tryEnterInsert()
		}
	case "H", "shift+left":
		f := ce.activeCEFieldPtr()
		if f != nil && f.Kind == KindSelect {
			f.CyclePrev()
			ce.updateDTODependentFields()
		}
	case "i", "a":
		f := ce.activeCEFieldPtr()
		if f != nil && f.CanEditAsText() {
			return ce.tryEnterInsert()
		}
	case "F":
		ce.saveDTOForm()
		ce.populateDTOFieldsFromDomains()
		ce.dtoFieldIdx = 0
		ce.dtoSubView = ceViewSubList
	case "b", "esc":
		ce.saveDTOForm()
		ce.dtoSubView = ceViewList
	}
	ce.saveDTOForm()
	return ce, nil
}

// populateDTOFieldsFromDomains auto-populates DTO fields from selected source domains
// when navigating to the fields sub-list. Only runs when the field list is empty.
func (ce *ContractsEditor) populateDTOFieldsFromDomains() {
	if len(ce.dtoFieldItems) > 0 {
		return
	}
	sourceDomains := fieldGetMulti(ce.dtoForm, "source_domains")
	if sourceDomains == "" {
		return
	}
	for _, domainName := range strings.Split(sourceDomains, ", ") {
		domainName = strings.TrimSpace(domainName)
		if domainName == "" {
			continue
		}
		for _, domainDef := range ce.availableDomainDefs {
			if domainDef.Name != domainName {
				continue
			}
			for _, attr := range domainDef.Attributes {
				f := defaultDTOFieldForm(ce.currentDTOProtocol())
				f = setFieldValue(f, "name", attr.Name)
				f = setFieldValue(f, "type", domainTypeToDTOType(attr.Type))
				if attr.Sensitive {
					f = setFieldValue(f, "nullable", "true")
				}
				if attr.Validation != "" {
					f = setFieldValue(f, "validation", attr.Validation)
				}
				ce.dtoFieldItems = append(ce.dtoFieldItems, f)
			}
			break
		}
	}
}

func domainTypeToDTOType(t string) string {
	switch t {
	case "String":
		return "string"
	case "Int":
		return "int"
	case "Float":
		return "float"
	case "Boolean":
		return "boolean"
	case "DateTime":
		return "datetime"
	case "UUID":
		return "uuid"
	case "Enum(values)":
		return "enum(values)"
	case "JSON/Map":
		return "map(key,value)"
	case "Array(type)":
		return "array(type)"
	case "Ref(Domain)":
		return "nested(DTO)"
	default:
		return "string"
	}
}

func (ce *ContractsEditor) saveDTOForm() {
	if ce.dtoIdx >= len(ce.dtos) {
		return
	}
	d := &ce.dtos[ce.dtoIdx]
	d.Name = fieldGet(ce.dtoForm, "name")
	d.Category = fieldGet(ce.dtoForm, "category")
	d.SourceDomains = fieldGetMulti(ce.dtoForm, "source_domains")
	d.Description = fieldGet(ce.dtoForm, "description")
	d.Protocol = fieldGet(ce.dtoForm, "protocol")
	d.ProtoPackage = fieldGet(ce.dtoForm, "proto_package")
	d.ProtoSyntax = fieldGet(ce.dtoForm, "proto_syntax")
	d.ProtoOptions = fieldGet(ce.dtoForm, "proto_options")
	d.AvroNamespace = fieldGet(ce.dtoForm, "avro_namespace")
	d.SchemaRegistry = fieldGet(ce.dtoForm, "schema_registry")
	d.ThriftNamespace = fieldGet(ce.dtoForm, "thrift_namespace")
	d.ThriftLanguage = fieldGet(ce.dtoForm, "thrift_language")
	d.Namespace = fieldGet(ce.dtoForm, "namespace")

	d.Fields = make([]manifest.DTOField, len(ce.dtoFieldItems))
	for i, item := range ce.dtoFieldItems {
		d.Fields[i] = manifest.DTOField{
			Name:           fieldGet(item, "name"),
			Type:           fieldGet(item, "type"),
			Required:       fieldGet(item, "required") == "true",
			Nullable:       fieldGet(item, "nullable") == "true",
			Validation:     fieldGetMulti(item, "validation"),
			Default:        fieldGet(item, "default"),
			FieldNumber:    fieldGet(item, "field_number"),
			ProtoModifier:  fieldGet(item, "proto_modifier"),
			JsonName:       fieldGet(item, "json_name"),
			FieldID:        fieldGet(item, "field_id"),
			ThriftModifier: fieldGet(item, "thrift_mod"),
			Deprecated:     fieldGet(item, "deprecated") == "true",
			Notes:          fieldGet(item, "notes"),
		}
	}
}

func (ce ContractsEditor) updateDTOFieldList(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	n := len(ce.dtoFieldItems)
	switch key.String() {
	case "j", "down":
		if n > 0 && ce.dtoFieldIdx < n-1 {
			ce.dtoFieldIdx++
		}
	case "k", "up":
		if ce.dtoFieldIdx > 0 {
			ce.dtoFieldIdx--
		}
	case "a":
		ce.dtoFieldItems = append(ce.dtoFieldItems, defaultDTOFieldForm(ce.currentDTOProtocol()))
		ce.dtoFieldIdx = len(ce.dtoFieldItems) - 1
		ce.dtoFieldForm = copyFields(ce.dtoFieldItems[ce.dtoFieldIdx])
		existing := make([]string, 0, len(ce.dtoFieldItems)-1)
		for i, f := range ce.dtoFieldItems {
			if i != ce.dtoFieldIdx {
				existing = append(existing, fieldGet(f, "name"))
			}
		}
		ce.dtoFieldForm = setFieldValue(ce.dtoFieldForm, "name", uniqueName("field", existing))
		ce.dtoFieldFormIdx = 0
		ce.dtoSubView = ceViewSubForm
		return ce.tryEnterInsert()
	case "d":
		if n > 0 {
			ce.dtoFieldItems = append(ce.dtoFieldItems[:ce.dtoFieldIdx], ce.dtoFieldItems[ce.dtoFieldIdx+1:]...)
			if ce.dtoFieldIdx > 0 && ce.dtoFieldIdx >= len(ce.dtoFieldItems) {
				ce.dtoFieldIdx = len(ce.dtoFieldItems) - 1
			}
		}
	case "enter":
		if n > 0 {
			ce.dtoFieldForm = copyFields(ce.dtoFieldItems[ce.dtoFieldIdx])
			ce.dtoFieldForm = refreshDTOFieldTypeOptions(ce.dtoFieldForm, ce.currentDTOProtocol())
			ce.dtoFieldFormIdx = 0
			ce.dtoSubView = ceViewSubForm
		}
	case "b", "esc":
		ce.dtoSubView = ceViewForm
	}
	return ce, nil
}

func (ce ContractsEditor) updateDTOFieldForm(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	visible := ce.visibleDTOFieldFormFields()
	switch key.String() {
	case "j", "down":
		if ce.dtoFieldFormIdx < len(visible)-1 {
			ce.dtoFieldFormIdx++
		}
	case "k", "up":
		if ce.dtoFieldFormIdx > 0 {
			ce.dtoFieldFormIdx--
		}
	case "enter", " ":
		f := ce.activeCEFieldPtr()
		if f == nil {
			break
		}
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			ce.dd.Open = true
			if f.Kind == KindSelect {
				ce.dd.OptIdx = f.SelIdx
			} else {
				ce.dd.OptIdx = f.DDCursor
			}
		} else {
			return ce.tryEnterInsert()
		}
	case "H", "shift+left":
		f := ce.activeCEFieldPtr()
		if f != nil && f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		f := ce.activeCEFieldPtr()
		if f != nil && f.CanEditAsText() {
			return ce.tryEnterInsert()
		}
	case "b", "esc":
		if ce.dtoFieldIdx < len(ce.dtoFieldItems) {
			ce.dtoFieldItems[ce.dtoFieldIdx] = copyFields(ce.dtoFieldForm)
		}
		ce.dtoSubView = ceViewSubList
	}
	if ce.dtoFieldIdx < len(ce.dtoFieldItems) {
		ce.dtoFieldItems[ce.dtoFieldIdx] = copyFields(ce.dtoFieldForm)
	}
	return ce, nil
}

func (ce ContractsEditor) viewDTOs(w int) []string {
	switch ce.dtoSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # DTOs — a: add  d: delete  Enter: edit"), "")
		if len(ce.dtos) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no DTOs yet — press 'a' to add one)"))
		} else {
			for i, dto := range ce.dtos {
				cat := dto.Category
				lines = append(lines, renderListItem(w, i == ce.dtoIdx, "  ▶ ", dto.Name, cat))
			}
		}
		return lines

	case ceViewForm:
		name := fieldGet(ce.dtoForm, "name")
		if name == "" {
			name = "(new DTO)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFields(w, ce.visibleDTOFields(), ce.dtoFormIdx, ce.internalMode == ModeInsert, ce.formInput, ce.dd.Open, ce.dd.OptIdx)...)
		lines = append(lines, "", StyleSectionDesc.Render(fmt.Sprintf("  F: edit fields  (%d field(s))", len(ce.dtoFieldItems))))
		return lines

	case ceViewSubList:
		var lines []string
		dtoName := ""
		if ce.dtoIdx < len(ce.dtos) {
			dtoName = ce.dtos[ce.dtoIdx].Name
		}
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(dtoName)+StyleSectionDesc.Render(" › Fields"), "")
		if len(ce.dtoFieldItems) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no fields — press 'a' to add)"))
		} else {
			for i, item := range ce.dtoFieldItems {
				fname := fieldGet(item, "name")
				ftype := fieldGet(item, "type")
				req := fieldGet(item, "required")
				extra := ftype
				if req == "true" {
					extra += " *required"
				}
				lines = append(lines, renderListItem(w, i == ce.dtoFieldIdx, "  ▶ ", fname, extra))
			}
		}
		return lines

	case ceViewSubForm:
		fname := fieldGet(ce.dtoFieldForm, "name")
		if fname == "" {
			fname = "(new field)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(fname), "")
		lines = append(lines, renderFormFields(w, ce.visibleDTOFieldFormFields(), ce.dtoFieldFormIdx, ce.internalMode == ModeInsert, ce.formInput, ce.dd.Open, ce.dd.OptIdx)...)
		return lines
	}
	return nil
}

func (ce ContractsEditor) viewEndpoints(w int) []string {
	switch ce.epSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Endpoints — a: add  d: delete  Enter: edit"), "")
		if len(ce.endpoints) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no endpoints yet — press 'a' to add one)"))
		} else {
			for i, ep := range ce.endpoints {
				proto := ep.Protocol
				if proto == "" {
					proto = "?"
				}
				name := ep.NamePath
				if name == "" {
					name = fmt.Sprintf("(endpoint #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == ce.epIdx, "  ▶ ", name, ep.ServiceUnit+" / "+proto))
			}
		}
		return lines

	case ceViewForm:
		visible := ce.visibleEPFields()
		title := fieldGet(ce.epForm, "name_path")
		if title == "" {
			title = "(new endpoint)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(title), "")
		lines = append(lines, renderFormFields(w, visible, ce.epFormIdx, ce.internalMode == ModeInsert, ce.formInput, ce.dd.Open, ce.dd.OptIdx)...)
		return lines
	}
	return nil
}

func (ce ContractsEditor) viewVersioning(w int) []string {
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  # API Versioning"), "")
	if !ce.versioningEnabled {
		lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		return lines
	}
	lines = append(lines, renderFormFields(w, ce.versioningFields, ce.verFormIdx, ce.internalMode == ModeInsert, ce.formInput, ce.dd.Open, ce.dd.OptIdx)...)
	return lines
}

// Expose endpoint names for cross-reference in other editors.
func (ce ContractsEditor) EndpointNames() []string {
	names := make([]string, len(ce.endpoints))
	for i, ep := range ce.endpoints {
		names[i] = ep.NamePath
	}
	return names
}

// DTONames returns the names of all DTOs for cross-reference.
func (ce ContractsEditor) DTONames() []string {
	names := make([]string, len(ce.dtos))
	for i, dto := range ce.dtos {
		names[i] = dto.Name
	}
	return names
}

