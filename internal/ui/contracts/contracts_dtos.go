package contracts

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── DTO updates ───────────────────────────────────────────────────────────────

func (ce ContractsEditor) updateDTOs(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	switch ce.dtoSubView {
	case core.ViewList:
		return ce.updateDTOList(key)
	case core.ViewForm:
		return ce.updateDTOForm(key)
	case core.ViewSubList:
		return ce.updateDTOFieldList(key)
	case core.ViewSubForm:
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
	case "u":
		if snap, ok := ce.dtosUndo.Pop(); ok {
			ce.dtos = snap
			if ce.dtoIdx >= len(ce.dtos) && ce.dtoIdx > 0 {
				ce.dtoIdx = len(ce.dtos) - 1
			}
		}
	case "a":
		ce.dtosUndo.Push(core.CopySlice(ce.dtos))
		ce.dtos = append(ce.dtos, manifest.DTODef{})
		ce.dtoIdx = len(ce.dtos) - 1
		ce.dtoForm = defaultDTOFormFields(ce.availableDomains)
		existing := make([]string, 0, len(ce.dtos)-1)
		for i, d := range ce.dtos {
			if i != ce.dtoIdx {
				existing = append(existing, d.Name)
			}
		}
		ce.dtoForm = core.SetFieldValue(ce.dtoForm, "name", core.UniqueName("dto", existing))
		ce.dtoFormIdx = 0
		ce.dtoFieldItems = nil
		ce.dtoSubView = core.ViewForm
		return ce.tryEnterInsert()
	case "d":
		if n > 0 {
			ce.dtosUndo.Push(core.CopySlice(ce.dtos))
			ce.dtos = append(ce.dtos[:ce.dtoIdx], ce.dtos[ce.dtoIdx+1:]...)
			if ce.dtoIdx > 0 && ce.dtoIdx >= len(ce.dtos) {
				ce.dtoIdx = len(ce.dtos) - 1
			}
			ce.ClearStaleDTORefs()
		}
	case "enter":
		if n > 0 {
			d := ce.dtos[ce.dtoIdx]
			ce.dtoForm = defaultDTOFormFields(ce.availableDomains)
			ce.dtoForm = core.SetFieldValue(ce.dtoForm, "name", d.Name)
			ce.dtoForm = core.SetFieldValue(ce.dtoForm, "category", d.Category)
			// Restore multiselect for source_domains
			if d.SourceDomains != "" {
				for i := range ce.dtoForm {
					if ce.dtoForm[i].Key == "source_domains" {
						for _, sel := range core.SplitCSV(d.SourceDomains) {
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
			ce.dtoForm = core.SetFieldValue(ce.dtoForm, "description", d.Description)
			if d.Protocol != "" {
				ce.dtoForm = core.SetFieldValue(ce.dtoForm, "protocol", d.Protocol)
			}
			ce.dtoForm = core.SetFieldValue(ce.dtoForm, "proto_package", d.ProtoPackage)
			ce.dtoForm = core.SetFieldValue(ce.dtoForm, "proto_syntax", d.ProtoSyntax)
			ce.dtoForm = core.SetFieldValue(ce.dtoForm, "proto_options", d.ProtoOptions)
			ce.dtoForm = core.SetFieldValue(ce.dtoForm, "avro_namespace", d.AvroNamespace)
			ce.dtoForm = core.SetFieldValue(ce.dtoForm, "schema_registry", d.SchemaRegistry)
			ce.dtoForm = core.SetFieldValue(ce.dtoForm, "thrift_namespace", d.ThriftNamespace)
			if d.ThriftLanguage != "" {
				ce.dtoForm = core.SetFieldValue(ce.dtoForm, "thrift_language", d.ThriftLanguage)
			}
			ce.dtoForm = core.SetFieldValue(ce.dtoForm, "namespace", d.Namespace)
			ce.dtoFormIdx = 0
			// Rebuild field items
			proto := d.Protocol
			if proto == "" {
				proto = "REST/JSON"
			}
			ce.dtoFieldItems = make([][]core.Field, len(d.Fields))
			for i, df := range d.Fields {
				f := defaultDTOFieldForm(proto)
				f = core.SetFieldValue(f, "name", df.Name)
				f = core.SetFieldValue(f, "type", df.Type)
				if df.Required {
					f = core.SetFieldValue(f, "required", "true")
				}
				if df.Nullable {
					f = core.SetFieldValue(f, "nullable", "true")
				}
				f = core.RestoreMultiSelectValue(f, "validation", df.Validation)
				f = core.SetFieldValue(f, "default", df.Default)
				f = core.SetFieldValue(f, "field_number", df.FieldNumber)
				if df.ProtoModifier != "" {
					f = core.SetFieldValue(f, "proto_modifier", df.ProtoModifier)
				}
				f = core.SetFieldValue(f, "json_name", df.JsonName)
				f = core.SetFieldValue(f, "field_id", df.FieldID)
				if df.ThriftModifier != "" {
					f = core.SetFieldValue(f, "thrift_mod", df.ThriftModifier)
				}
				if df.Deprecated {
					f = core.SetFieldValue(f, "deprecated", "true")
				}
				f = core.SetFieldValue(f, "notes", df.Notes)
				ce.dtoFieldItems[i] = f
			}
			ce.dtoSubView = core.ViewForm
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
		if (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
			ce.dd.Open = true
			if f.Kind == core.KindSelect {
				ce.dd.OptIdx = f.SelIdx
			} else {
				ce.dd.OptIdx = f.DDCursor
			}
		} else {
			return ce.tryEnterInsert()
		}
	case "H", "shift+left":
		f := ce.activeCEFieldPtr()
		if f != nil && f.Kind == core.KindSelect {
			f.CyclePrev()
			ce.updateDTODependentFields()
		}
	case "i", "a":
		f := ce.activeCEFieldPtr()
		if f != nil && f.CanEditAsText() {
			return ce.tryEnterInsert()
		}
	case "A":
		ce.saveDTOForm()
		ce.populateDTOFieldsFromDomains()
		ce.dtoFieldIdx = 0
		ce.dtoSubView = core.ViewSubList
	case "b", "esc":
		ce.saveDTOForm()
		ce.dtoSubView = core.ViewList
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
	sourceDomains := core.FieldGetMulti(ce.dtoForm, "source_domains")
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
				f = core.SetFieldValue(f, "name", attr.Name)
				f = core.SetFieldValue(f, "type", domainTypeToDTOType(attr.Type))
				if attr.Sensitive {
					f = core.SetFieldValue(f, "nullable", "true")
				}
				if attr.Validation != "" {
					f = core.SetFieldValue(f, "validation", attr.Validation)
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
	d.Name = core.FieldGet(ce.dtoForm, "name")
	d.Category = core.FieldGet(ce.dtoForm, "category")
	d.SourceDomains = core.FieldGetMulti(ce.dtoForm, "source_domains")
	d.Description = core.FieldGet(ce.dtoForm, "description")
	d.Protocol = core.FieldGet(ce.dtoForm, "protocol")
	d.ProtoPackage = core.FieldGet(ce.dtoForm, "proto_package")
	d.ProtoSyntax = core.FieldGet(ce.dtoForm, "proto_syntax")
	d.ProtoOptions = core.FieldGet(ce.dtoForm, "proto_options")
	d.AvroNamespace = core.FieldGet(ce.dtoForm, "avro_namespace")
	d.SchemaRegistry = core.FieldGet(ce.dtoForm, "schema_registry")
	d.ThriftNamespace = core.FieldGet(ce.dtoForm, "thrift_namespace")
	d.ThriftLanguage = core.FieldGet(ce.dtoForm, "thrift_language")
	d.Namespace = core.FieldGet(ce.dtoForm, "namespace")

	d.Fields = make([]manifest.DTOField, len(ce.dtoFieldItems))
	for i, item := range ce.dtoFieldItems {
		d.Fields[i] = manifest.DTOField{
			Name:           core.FieldGet(item, "name"),
			Type:           core.FieldGet(item, "type"),
			Required:       core.FieldGet(item, "required") == "true",
			Nullable:       core.FieldGet(item, "nullable") == "true",
			Validation:     core.FieldGetMulti(item, "validation"),
			Default:        core.FieldGet(item, "default"),
			FieldNumber:    core.FieldGet(item, "field_number"),
			ProtoModifier:  core.FieldGet(item, "proto_modifier"),
			JsonName:       core.FieldGet(item, "json_name"),
			FieldID:        core.FieldGet(item, "field_id"),
			ThriftModifier: core.FieldGet(item, "thrift_mod"),
			Deprecated:     core.FieldGet(item, "deprecated") == "true",
			Notes:          core.FieldGet(item, "notes"),
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
		ce.dtoFieldForm = core.CopyFields(ce.dtoFieldItems[ce.dtoFieldIdx])
		existing := make([]string, 0, len(ce.dtoFieldItems)-1)
		for i, f := range ce.dtoFieldItems {
			if i != ce.dtoFieldIdx {
				existing = append(existing, core.FieldGet(f, "name"))
			}
		}
		ce.dtoFieldForm = core.SetFieldValue(ce.dtoFieldForm, "name", core.UniqueName("field", existing))
		ce.dtoFieldFormIdx = 0
		ce.dtoSubView = core.ViewSubForm
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
			ce.dtoFieldForm = core.CopyFields(ce.dtoFieldItems[ce.dtoFieldIdx])
			ce.dtoFieldForm = refreshDTOFieldTypeOptions(ce.dtoFieldForm, ce.currentDTOProtocol())
			ce.dtoFieldFormIdx = 0
			ce.dtoSubView = core.ViewSubForm
		}
	case "b", "esc":
		ce.dtoSubView = core.ViewForm
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
		if (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
			ce.dd.Open = true
			if f.Kind == core.KindSelect {
				ce.dd.OptIdx = f.SelIdx
			} else {
				ce.dd.OptIdx = f.DDCursor
			}
		} else {
			return ce.tryEnterInsert()
		}
	case "H", "shift+left":
		f := ce.activeCEFieldPtr()
		if f != nil && f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		f := ce.activeCEFieldPtr()
		if f != nil && f.CanEditAsText() {
			return ce.tryEnterInsert()
		}
	case "b", "esc":
		if ce.dtoFieldIdx < len(ce.dtoFieldItems) {
			ce.dtoFieldItems[ce.dtoFieldIdx] = core.CopyFields(ce.dtoFieldForm)
		}
		ce.dtoSubView = core.ViewSubList
	}
	if ce.dtoFieldIdx < len(ce.dtoFieldItems) {
		ce.dtoFieldItems[ce.dtoFieldIdx] = core.CopyFields(ce.dtoFieldForm)
	}
	return ce, nil
}

func (ce ContractsEditor) viewDTOs(w int) []string {
	switch ce.dtoSubView {
	case core.ViewList:
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  # DTOs — a: add  d: delete  Enter: edit"), "")
		if len(ce.dtos) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no DTOs yet — press 'a' to add one)"))
		} else {
			for i, dto := range ce.dtos {
				cat := dto.Category
				lines = append(lines, core.RenderListItem(w, i == ce.dtoIdx, "  ▶ ", dto.Name, cat))
			}
		}
		return lines

	case core.ViewForm:
		name := core.FieldGet(ce.dtoForm, "name")
		if name == "" {
			name = "(new DTO)"
		}
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(name), "")
		lines = append(lines, core.RenderFormFields(w, ce.visibleDTOFields(), ce.dtoFormIdx, ce.internalMode == core.ModeInsert, ce.formInput, ce.dd.Open, ce.dd.OptIdx)...)
		lines = append(lines, "", core.StyleSectionDesc.Render(fmt.Sprintf("  A: edit fields  (%d field(s))", len(ce.dtoFieldItems))))
		return lines

	case core.ViewSubList:
		var lines []string
		dtoName := ""
		if ce.dtoIdx < len(ce.dtos) {
			dtoName = ce.dtos[ce.dtoIdx].Name
		}
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(dtoName)+core.StyleSectionDesc.Render(" › Fields"), "")
		if len(ce.dtoFieldItems) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no fields — press 'a' to add)"))
		} else {
			for i, item := range ce.dtoFieldItems {
				fname := core.FieldGet(item, "name")
				ftype := core.FieldGet(item, "type")
				req := core.FieldGet(item, "required")
				extra := ftype
				if req == "true" {
					extra += " *required"
				}
				lines = append(lines, core.RenderListItem(w, i == ce.dtoFieldIdx, "  ▶ ", fname, extra))
			}
		}
		return lines

	case core.ViewSubForm:
		fname := core.FieldGet(ce.dtoFieldForm, "name")
		if fname == "" {
			fname = "(new field)"
		}
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(fname), "")
		lines = append(lines, core.RenderFormFields(w, ce.visibleDTOFieldFormFields(), ce.dtoFieldFormIdx, ce.internalMode == core.ModeInsert, ce.formInput, ce.dd.Open, ce.dd.OptIdx)...)
		return lines
	}
	return nil
}

func (ce ContractsEditor) viewEndpoints(w int) []string {
	switch ce.epSubView {
	case core.ViewList:
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  # Endpoints — a: add  d: delete  Enter: edit"), "")
		if len(ce.endpoints) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no endpoints yet — press 'a' to add one)"))
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
				lines = append(lines, core.RenderListItem(w, i == ce.epIdx, "  ▶ ", name, ep.ServiceUnit+" / "+proto))
			}
		}
		return lines

	case core.ViewForm:
		visible := ce.visibleEPFields()
		title := core.FieldGet(ce.epForm, "name_path")
		if title == "" {
			title = "(new endpoint)"
		}
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(title), "")
		lines = append(lines, core.RenderFormFields(w, visible, ce.epFormIdx, ce.internalMode == core.ModeInsert, ce.formInput, ce.dd.Open, ce.dd.OptIdx)...)
		return lines
	}
	return nil
}

func (ce ContractsEditor) viewVersioning(w int) []string {
	var lines []string
	lines = append(lines, core.StyleSectionDesc.Render("  # API Versioning"), "")
	if !ce.versioningEnabled {
		lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		return lines
	}
	lines = append(lines, core.RenderFormFields(w, ce.versioningFields, ce.verFormIdx, ce.internalMode == core.ModeInsert, ce.formInput, ce.dd.Open, ce.dd.OptIdx)...)
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

// DTOProtocols returns the unique serialisation protocols used across all DTOs.
// Used by BackendEditor to suggest a matching messaging serialization default.
func (ce ContractsEditor) DTOProtocols() []string {
	seen := make(map[string]bool)
	var result []string
	for _, dto := range ce.dtos {
		if dto.Protocol != "" && !seen[dto.Protocol] {
			seen[dto.Protocol] = true
			result = append(result, dto.Protocol)
		}
	}
	return result
}
