package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (be BackendEditor) HintLine() string {
	if !be.ArchConfirmed {
		if be.dropdownOpen {
			return hintBar("j/k", "navigate", "Enter/Space", "confirm", "Esc", "close")
		}
		return hintBar("Enter/Space", "open arch selector")
	}
	if be.dd.Open {
		return hintBar("j/k", "navigate", "Enter/Space", "select", "Esc", "cancel")
	}
	if be.internalMode == ModeInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}

	tab := be.activeTab()
	switch tab {
	case beTabServices:
		ed := be.serviceEditor
		if ed.itemView == beListViewList {
			return hintBar("j/k", "navigate", "a", "add service", "d", "delete", "Enter", "edit", "h/l", "sub-tab", "b", "change arch")
		}
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back", "Tab", "next field")
	case beTabComm:
		ed := be.commEditor
		if ed.itemView == beListViewList {
			return hintBar("j/k", "navigate", "a", "add link", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	case beTabMessaging:
		if be.eventEditor.itemView == beListViewForm {
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
		return hintBar("j/k", "navigate", "Space", "cycle", "a", "add event", "d", "del event", "h/l", "sub-tab")
	case beTabJobs:
		if be.jobsSubView == beViewForm {
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
		return hintBar("j/k", "navigate", "a", "add job queue", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
	case beTabAuth:
		if !be.authEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab", "b", "change arch")
		}
		switch be.authSubView {
		case beAuthViewConfig:
			return hintBar("j/k", "navigate", "a/i/Enter", "edit", "r", "roles", "p", "permissions", "D", "reset", "h/l", "sub-tab")
		case beAuthViewPermList:
			return hintBar("j/k", "navigate", "a", "add perm", "d", "delete", "Enter", "edit", "b", "back to config", "h/l", "sub-tab")
		case beAuthViewPermForm:
			return hintBar("j/k", "navigate", "i/Enter", "edit", "b/Esc", "back to list")
		case beAuthViewRoleList:
			return hintBar("j/k", "navigate", "a", "add role", "d", "delete", "Enter", "edit", "b", "back to config", "h/l", "sub-tab")
		case beAuthViewRoleForm:
			return hintBar("j/k", "navigate", "i/Enter", "edit text", "Space/Enter", "toggle perm", "b/Esc", "back to list")
		}
		return ""
	case beTabSecurity:
		if !be.secEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab", "b", "change arch")
		}
		return hintBar("j/k", "navigate", "gg/G", "top/bottom", "a/Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "h/l", "sub-tab")
	default:
		t := be.activeTab()
		configEnabled := true
		switch t {
		case beTabEnv:
			configEnabled = be.envEnabled
		case beTabAPIGW:
			configEnabled = be.apiGWEnabled
		}
		if !configEnabled {
			return hintBar("a", "configure", "h/l", "sub-tab", "b", "change arch")
		}
		return hintBar("j/k", "navigate", "gg/G", "top/bottom", "[n]j/k", "jump", "a/i/Enter", "edit", "Space", "cycle", "H", "cycle back", "D", "delete config", "h/l", "sub-tab", "b", "change arch")
	}
}

func (be BackendEditor) visibleEnvFields() []Field {
	arch := be.currentArch()
	corsStrategy := fieldGet(be.EnvFields, "cors_strategy")
	var out []Field
	for _, f := range be.EnvFields {
		if (f.Key == "monolith_lang" || f.Key == "monolith_lang_ver" || f.Key == "monolith_fw" || f.Key == "monolith_fw_ver") && arch != "monolith" {
			continue
		}
		if f.Key == "cors_origins" && corsStrategy != "Strict allowlist" {
			continue
		}
		out = append(out, f)
	}
	return out
}

// currentEditableFields returns a pointer to the current tab's field slice.
// For ENV, we return nil (use visibleEnvFields instead) but we keep it for
// generic field navigation — actual navigation uses visibleEnvFieldIdx.
func (be *BackendEditor) currentEditableFields() *[]Field {
	switch be.activeTab() {
	case beTabEnv:
		return &be.EnvFields
	case beTabMessaging:
		return &be.MessagingFields
	case beTabAPIGW:
		return &be.APIGWFields
	case beTabAuth:
		if be.authSubView == beAuthViewConfig {
			return &be.AuthFields
		}
		return nil
	case beTabSecurity:
		return &be.securityFields
	}
	return nil
}

// mutableFieldPtr returns a pointer to the active field for the current tab.
// For the ENV tab, it resolves through the visible fields to find the correct
// pointer in the underlying EnvFields slice.
func (be *BackendEditor) mutableFieldPtr() *Field {
	if be.activeTab() == beTabEnv {
		visible := be.visibleEnvFields()
		if be.activeField < 0 || be.activeField >= len(visible) {
			return nil
		}
		key := visible[be.activeField].Key
		for i := range be.EnvFields {
			if be.EnvFields[i].Key == key {
				return &be.EnvFields[i]
			}
		}
		return nil
	}
	fields := be.currentEditableFields()
	if fields == nil {
		return nil
	}
	if be.activeField >= 0 && be.activeField < len(*fields) {
		return &(*fields)[be.activeField]
	}
	return nil
}

func (be BackendEditor) tryEnterInsert() (BackendEditor, tea.Cmd) {
	fields := be.currentEditableFields()
	if fields == nil {
		return be, nil
	}
	n := len(*fields)
	for range n {
		if be.activeField >= n {
			break
		}
		f := (*fields)[be.activeField]
		if f.CanEditAsText() {
			be.internalMode = ModeInsert
			be.formInput.SetValue(f.TextInputValue())
			be.formInput.Width = be.width - 22
			be.formInput.CursorEnd()
			return be, be.formInput.Focus()
		}
		be.activeField = (be.activeField + 1) % n
	}
	return be, nil
}

func (be *BackendEditor) saveInput() {
	val := be.formInput.Value()

	// Check if we're in a service form
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].CanEditAsText() {
			ed.form[ed.formIdx].SaveTextInput(val)
		}
		return
	}
	// Check if we're in a comm form
	if be.commEditor.itemView == beListViewForm {
		ed := &be.commEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].CanEditAsText() {
			ed.form[ed.formIdx].SaveTextInput(val)
		}
		return
	}
	// Check if we're in an event form
	if be.eventEditor.itemView == beListViewForm {
		ed := &be.eventEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].CanEditAsText() {
			ed.form[ed.formIdx].SaveTextInput(val)
		}
		return
	}
	// Check if we're in an auth permission form
	if be.authSubView == beAuthViewPermForm {
		if be.authPermFormIdx < len(be.authPermForm) && be.authPermForm[be.authPermFormIdx].Kind == KindText {
			be.authPermForm[be.authPermFormIdx].Value = val
		}
		return
	}
	// Check if we're in an auth role form
	if be.authSubView == beAuthViewRoleForm {
		if be.authRoleFormIdx < len(be.authRoleForm) && be.authRoleForm[be.authRoleFormIdx].Kind == KindText {
			be.authRoleForm[be.authRoleFormIdx].Value = val
		}
		return
	}
	// Check if we're in a jobs form
	if be.jobsSubView == beViewForm {
		if be.jobsFormIdx < len(be.jobsForm) && be.jobsForm[be.jobsFormIdx].CanEditAsText() {
			be.jobsForm[be.jobsFormIdx].SaveTextInput(val)
		}
		return
	}
	// Generic field stores
	fields := be.currentEditableFields()
	if fields == nil {
		return
	}
	if be.activeField < len(*fields) && (*fields)[be.activeField].CanEditAsText() {
		(*fields)[be.activeField].SaveTextInput(val)
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (be BackendEditor) View(w, h int) string {
	be.width = w
	be.formInput.Width = w - 22
	if !be.ArchConfirmed {
		return be.viewArchSelect(w, h)
	}
	return be.viewSubTabs(w, h)
}

func (be BackendEditor) viewArchSelect(w, h int) string {
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Backend — Choose an architecture pattern"),
		"",
	)

	current := beArchOptions[be.ArchIdx]
	label := StyleFieldKey.Render("arch_pattern  ")
	val := StyleFieldValActive.Render(current.label) + StyleSelectArrow.Render(" ▾")
	row := "     " + label + StyleEquals.Render(" = ") + val
	raw := lipgloss.Width(row)
	if raw < w {
		row += strings.Repeat(" ", w-raw)
	}
	lines = append(lines, activeCurLineStyle().Render(row))

	if be.dropdownOpen {
		lines = append(lines, "")
		for i, opt := range beArchOptions {
			isCur := i == be.dropdownIdx
			var cursor string
			if isCur {
				cursor = StyleCurLineNum.Render("  ▶ ")
			} else {
				cursor = "      "
			}
			labelPart := fmt.Sprintf("%-20s", opt.label)
			var optRow string
			if isCur {
				optRow = cursor +
					StyleFieldValActive.Render(labelPart) +
					StyleSectionDesc.Render(opt.desc)
				rw := lipgloss.Width(optRow)
				if rw < w {
					optRow += strings.Repeat(" ", w-rw)
				}
				optRow = activeCurLineStyle().Render(optRow)
			} else {
				optRow = cursor +
					StyleFieldKey.Render(labelPart) +
					StyleSectionDesc.Render(opt.desc)
			}
			lines = append(lines, optRow)
		}
	}

	return fillTildes(lines, h)
}

func (be BackendEditor) viewSubTabs(w, h int) string {
	var lines []string

	opt := beArchOptions[be.ArchIdx]
	archStr := StyleFieldValActive.Render(opt.label)
	hint := StyleSectionDesc.Render("  (b: change arch)")
	lines = append(lines,
		StyleSectionDesc.Render("  # Backend · ")+archStr+hint,
		"",
		renderSubTabBar(be.tabLabels(), be.activeTabIdx, w),
		"",
	)

	tab := be.activeTab()
	switch tab {
	case beTabEnv:
		if be.envEnabled {
			envFields := be.visibleEnvFields()
			lines = append(lines, renderFormFields(w, envFields, be.activeField, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case beTabServices:
		const beListHeaderH = 2
		svcLines := be.viewServiceEditor(w)
		if be.serviceEditor.itemView == beListViewList {
			svcLines = appendViewport(svcLines, beListHeaderH, be.serviceEditor.itemIdx, h-4)
		}
		lines = append(lines, svcLines...)
	case beTabComm:
		commLines := be.viewCommEditor(w)
		if be.commEditor.itemView == beListViewList {
			commLines = appendViewport(commLines, 2, be.commEditor.itemIdx, h-4)
		}
		lines = append(lines, commLines...)
	case beTabMessaging:
		msgLines := be.viewMessaging(w)
		if be.eventEditor.itemView == beListViewList {
			msgLines = appendViewport(msgLines, 2, be.eventEditor.itemIdx, h-4)
		}
		lines = append(lines, msgLines...)
	case beTabAPIGW:
		if be.apiGWEnabled {
			lines = append(lines, renderFormFields(w, be.APIGWFields, be.activeField, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case beTabJobs:
		jobLines := be.viewJobs(w)
		if be.jobsSubView == beViewList {
			jobLines = appendViewport(jobLines, 2, be.jobsIdx, h-4)
		}
		lines = append(lines, jobLines...)
	case beTabSecurity:
		if be.secEnabled {
			lines = append(lines, renderFormFields(w, be.securityFields, be.activeField, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case beTabAuth:
		authLines := be.viewAuth(w)
		switch be.authSubView {
		case beAuthViewPermList:
			authLines = appendViewport(authLines, 2, be.authPermsIdx, h-4)
		case beAuthViewRoleList:
			authLines = appendViewport(authLines, 2, be.authRolesIdx, h-4)
		}
		lines = append(lines, authLines...)
	}

	return fillTildes(lines, h)
}

func (be BackendEditor) viewServiceEditor(w int) []string {
	ed := be.serviceEditor
	if ed.itemView == beListViewList {
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Service Units — a: add  d: delete  Enter: edit"), "")
		if len(ed.items) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no services yet — press 'a' to add)"))
		} else {
			for i, item := range ed.items {
				name := fieldGet(item, "name")
				if name == "" {
					name = fmt.Sprintf("(service #%d)", i+1)
				}
				lang := fieldGet(item, "language")
				langVer := fieldGet(item, "language_version")
				fw := fieldGet(item, "framework")
				fwVer := fieldGet(item, "framework_version")
				extra := lang
				if langVer != "" {
					extra += " " + langVer
				}
				if fw != "" {
					extra += " / " + fw
					if fwVer != "" {
						extra += " " + fwVer
					}
				}
				lines = append(lines, renderListItem(w, i == ed.itemIdx, "  ▶ ", name, extra))
			}
		}
		return lines
	}
	// Form view
	idx := ed.itemIdx
	name := "(new service)"
	if idx < len(ed.items) {
		n := fieldGet(ed.form, "name")
		if n != "" {
			name = n
		}
	}

	arch := be.currentArch()
	var fields []Field
	filteredActiveIdx := ed.formIdx
	skippedBefore := 0
	for i, f := range ed.form {
		// For monolith: language, framework, and service_discovery are defined at top level (ENV tab)
		if arch == "monolith" && (f.Key == "language" || f.Key == "framework" || f.Key == "service_discovery") {
			if i < ed.formIdx {
				skippedBefore++
			}
			continue
		}
		// Hide pattern_tag for non-hybrid arches
		if arch != "hybrid" && f.Key == "pattern_tag" {
			if i < ed.formIdx {
				skippedBefore++
			}
			continue
		}
		fields = append(fields, f)
	}
	filteredActiveIdx = ed.formIdx - skippedBefore
	if filteredActiveIdx < 0 {
		filteredActiveIdx = 0
	}

	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
	lines = append(lines, renderFormFields(w, fields, filteredActiveIdx, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
	return lines
}

func (be BackendEditor) viewCommEditor(w int) []string {
	ed := be.commEditor
	if ed.itemView == beListViewList {
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Communication Links — a: add  d: delete  Enter: edit"), "")
		if len(ed.items) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no links yet — press 'a' to add)"))
		} else {
			for i, item := range ed.items {
				from := fieldGet(item, "from")
				to := fieldGet(item, "to")
				if from == "" {
					from = "?"
				}
				if to == "" {
					to = "?"
				}
				proto := fieldGet(item, "protocol")
				name := from + " → " + to
				lines = append(lines, renderListItem(w, i == ed.itemIdx, "  ▶ ", name, proto))
			}
		}
		return lines
	}
	from := fieldGet(be.commEditor.form, "from")
	to := fieldGet(be.commEditor.form, "to")
	title := from + " → " + to
	if from == "" && to == "" {
		title = "(new link)"
	}
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(title), "")
	lines = append(lines, renderFormFields(w, ed.form, ed.formIdx, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
	return lines
}

func (be BackendEditor) viewMessaging(w int) []string {
	ed := be.eventEditor
	if ed.itemView == beListViewForm {
		name := fieldGet(ed.form, "name")
		if name == "" {
			name = "(new event)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFields(w, ed.form, ed.formIdx, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
		return lines
	}

	// Combined view: broker config fields + event catalog list
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  # Messaging Broker + Event Catalog"), "")

	brokerCount := len(be.MessagingFields)
	// Render broker fields in upper section
	const msgDDIndent = 21 // lineNumW(4) + labelW(14) + eqW(3)
	for i, f := range be.MessagingFields {
		isCur := i == be.activeField
		lineNo := StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
		}
		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}
		eq := StyleEquals.Render(" = ")
		val := f.DisplayValue()
		var valStr string
		if isCur && be.dd.Open {
			valStr = StyleFieldValActive.Render(val) + StyleSelectArrow.Render(" ▴")
		} else if isCur {
			valStr = StyleFieldValActive.Render(val) + StyleSelectArrow.Render(" ▾")
		} else {
			valStr = StyleFieldVal.Render(val) + StyleSelectArrow.Render(" ▾")
		}
		row := lineNo + keyStr + eq + valStr
		if isCur {
			rw := lipgloss.Width(row)
			if rw < w {
				row += strings.Repeat(" ", w-rw)
			}
			row = activeCurLineStyle().Render(row)
		}
		lines = append(lines, row)
		// Inline dropdown for active broker field
		if isCur && be.dd.Open {
			indent := strings.Repeat(" ", msgDDIndent)
			for j, opt := range f.Options {
				isHL := j == be.dd.OptIdx
				var optRow string
				if isHL {
					optRow = indent + StyleFieldValActive.Render("▶ "+opt)
					rw := lipgloss.Width(optRow)
					if rw < w {
						optRow += strings.Repeat(" ", w-rw)
					}
					optRow = activeCurLineStyle().Render(optRow)
				} else {
					optRow = indent + StyleFieldVal.Render("  "+opt)
				}
				lines = append(lines, optRow)
			}
		}
	}

	// Divider + event catalog
	lines = append(lines, "", StyleSectionDesc.Render("  ── Event Catalog (a: add  d: delete  Enter: edit) ──"), "")

	if len(ed.items) == 0 {
		lines = append(lines, StyleSectionDesc.Render("  (no events yet — press 'a' to add)"))
	} else {
		for i, item := range ed.items {
			globalIdx := brokerCount + i
			isCur := globalIdx == be.activeField
			name := fieldGet(item, "name")
			if name == "" {
				name = fmt.Sprintf("(event #%d)", i+1)
			}
			domain := fieldGet(item, "domain")
			lines = append(lines, renderListItem(w, isCur, "  ▶ ", name, domain))
		}
	}
	return lines
}

// ── Jobs updates ──────────────────────────────────────────────────────────────

