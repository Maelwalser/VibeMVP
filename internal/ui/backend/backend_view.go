package backend

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/ui/core"
)

func (be BackendEditor) HintLine() string {
	if !be.ArchConfirmed {
		if be.dropdownOpen {
			return core.HintBar("j/k", "navigate", "Enter/Space", "confirm", "Esc", "close")
		}
		return core.HintBar("Enter/Space", "open arch selector")
	}
	if be.dd.Open {
		return core.HintBar("j/k", "navigate", "Enter/Space", "select", "Esc", "cancel")
	}
	if be.internalMode == core.ModeInsert {
		return core.StyleInsertMode.Render(" -- INSERT -- ") +
			core.StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}

	tab := be.activeTab()
	switch tab {
	case beTabServices:
		switch be.repoSubView {
		case beRepoSubViewList:
			return core.HintBar("j/k", "navigate", "a", "add repo", "d", "delete", "Enter", "edit", "b/Esc", "back to service")
		case beRepoSubViewForm:
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "O", "operations", "b/Esc", "back to list")
		case beRepoSubViewOpList:
			return core.HintBar("j/k", "navigate", "a", "add op", "d", "delete", "Enter", "edit", "b/Esc", "back to repo")
		case beRepoSubViewOpForm:
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
		ed := be.serviceEditor
		if ed.itemView == beListViewList {
			return core.HintBar("j/k", "nav", "a", "add", "d", "del", "Enter", "edit", "h/l", "tabs")
		}
		return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	case beTabComm:
		ed := be.commEditor
		if ed.itemView == beListViewList {
			return core.HintBar("j/k", "navigate", "a", "add link", "d", "delete", "u", "undo", "Enter", "edit", "h/l", "sub-tab")
		}
		return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	case beTabMessaging:
		if be.eventEditor.itemView == beListViewForm {
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
		return core.HintBar("j/k", "navigate", "Space", "cycle", "a", "add event", "d", "del event", "u", "undo", "h/l", "sub-tab")
	case beTabJobs:
		if be.jobsSubView == beViewForm {
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
		return core.HintBar("j/k", "navigate", "a", "add job queue", "d", "delete", "u", "undo", "Enter", "edit", "h/l", "sub-tab")
	case beTabAuth:
		if !be.authEnabled {
			return core.HintBar("a", "configure", "h/l", "sub-tab", "b", "change arch")
		}
		switch be.authSubView {
		case beAuthViewConfig:
			return core.HintBar("j/k", "navigate", "a/i/Enter", "edit", "r", "roles", "p", "permissions", "D", "reset", "h/l", "sub-tab")
		case beAuthViewPermList:
			return core.HintBar("j/k", "navigate", "a", "add perm", "d", "delete", "u", "undo", "Enter", "edit", "b", "back to config", "h/l", "sub-tab")
		case beAuthViewPermForm:
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "b/Esc", "back to list")
		case beAuthViewRoleList:
			return core.HintBar("j/k", "navigate", "a", "add role", "d", "delete", "u", "undo", "Enter", "edit", "b", "back to config", "h/l", "sub-tab")
		case beAuthViewRoleForm:
			return core.HintBar("j/k", "navigate", "i/Enter", "edit text", "Space/Enter", "toggle perm", "b/Esc", "back to list")
		}
		return ""
	case beTabSecurity:
		if !be.secEnabled {
			return core.HintBar("a", "configure", "h/l", "sub-tab", "b", "change arch")
		}
		return core.HintBar("j/k", "navigate", "gg/G", "top/bottom", "a/Space/Enter", "cycle", "H", "cycle back", "D", "delete config", "h/l", "sub-tab")
	case beTabEnv:
		if be.currentArch() != "monolith" {
			ed := be.stackConfigEditor
			if ed.itemView == beListViewList {
				return core.HintBar("j/k", "navigate", "a", "add config", "d", "delete", "u", "undo", "Enter", "edit", "h/l", "sub-tab", "b", "change arch")
			}
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "H", "cycle back", "b/Esc", "back", "Tab", "next field")
		}
		configEnabled := be.envEnabled
		if !configEnabled {
			return core.HintBar("a", "configure", "h/l", "sub-tab", "b", "change arch")
		}
		return core.HintBar("j/k", "navigate", "gg/G", "top/bottom", "[n]j/k", "jump", "a/i/Enter", "edit", "Space", "cycle", "H", "cycle back", "D", "delete config", "h/l", "sub-tab", "b", "change arch")
	default:
		t := be.activeTab()
		configEnabled := true
		switch t {
		case beTabAPIGW:
			configEnabled = be.apiGWEnabled
		}
		if !configEnabled {
			return core.HintBar("a", "configure", "h/l", "sub-tab", "b", "change arch")
		}
		return core.HintBar("j/k", "navigate", "gg/G", "top/bottom", "[n]j/k", "jump", "a/i/Enter", "edit", "Space", "cycle", "H", "cycle back", "D", "delete config", "h/l", "sub-tab", "b", "change arch")
	}
}

// visibleEnvFields returns the env fields appropriate for the current arch.
// The env tab only appears for monolith, so all fields here are always shown.
func (be BackendEditor) visibleEnvFields() []core.Field {
	return be.EnvFields
}

// currentEditableFields returns a pointer to the current tab's field slice.
// For ENV, we return nil (use visibleEnvFields instead) but we keep it for
// generic field navigation — actual navigation uses visibleEnvFieldIdx.
func (be *BackendEditor) currentEditableFields() *[]core.Field {
	switch be.activeTab() {
	case beTabEnv:
		if be.currentArch() != "monolith" && be.stackConfigEditor.itemView == beListViewForm {
			return &be.stackConfigEditor.form
		}
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

// CurrentField returns the currently highlighted form field for the description panel.
// Returns nil when in list/non-form views (services list, comm list, etc.) or when
// the active tab has not been configured yet.
func (be *BackendEditor) CurrentField() *core.Field {
	switch be.activeTab() {
	case beTabEnv:
		if be.currentArch() == "monolith" && !be.envEnabled {
			return nil
		}
	case beTabAPIGW:
		if !be.apiGWEnabled {
			return nil
		}
	case beTabSecurity:
		if !be.secEnabled {
			return nil
		}
	case beTabAuth:
		if !be.authEnabled {
			return nil
		}
	}
	return be.mutableFieldPtr()
}

// mutableFieldPtr returns a pointer to the active field for the current tab.
// For the ENV tab, it resolves through the visible fields to find the correct
// pointer in the underlying EnvFields slice.
func (be *BackendEditor) mutableFieldPtr() *core.Field {
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
			be.internalMode = core.ModeInsert
			be.formInput.SetValue(f.TextInputValue())
			be.formInput.Width = be.width - 22
			be.formInput.CursorEnd()
			return be, be.formInput.Focus()
		}
		be.activeField = (be.activeField + 1) % n
	}
	return be, nil
}

func (be BackendEditor) clearAndEnterInsert() (BackendEditor, tea.Cmd) {
	be, cmd := be.tryEnterInsert()
	if be.internalMode == core.ModeInsert {
		be.formInput.SetValue("")
	}
	return be, cmd
}

func (be *BackendEditor) saveInput() {
	val := be.formInput.Value()

	// Check if we're in a stack config form
	if be.stackConfigEditor.itemView == beListViewForm {
		ed := &be.stackConfigEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].CanEditAsText() {
			ed.form[ed.formIdx].SaveTextInput(val)
			// Persist to items and propagate name changes to dependent forms immediately.
			be.saveStackConfigForm()
		}
		return
	}
	// Check if we're in a repo form (within service)
	if be.repoSubView == beRepoSubViewForm && be.repoEditor.itemView == beListViewForm {
		ed := &be.repoEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].CanEditAsText() {
			ed.form[ed.formIdx].SaveTextInput(val)
			be.saveRepoForm()
		}
		return
	}
	// Check if we're in an op form (within repo)
	if be.repoSubView == beRepoSubViewOpForm && be.opEditor.itemView == beListViewForm {
		ed := &be.opEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].CanEditAsText() {
			ed.form[ed.formIdx].SaveTextInput(val)
			be.saveOpForm()
		}
		return
	}
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
		if be.authPermFormIdx < len(be.authPermForm) && be.authPermForm[be.authPermFormIdx].Kind == core.KindText {
			be.authPermForm[be.authPermFormIdx].Value = val
		}
		return
	}
	// Check if we're in an auth role form
	if be.authSubView == beAuthViewRoleForm {
		if be.authRoleFormIdx < len(be.authRoleForm) && be.authRoleForm[be.authRoleFormIdx].Kind == core.KindText {
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
		core.StyleSectionDesc.Render("  # Backend — Choose an architecture pattern"),
		"",
	)

	current := beArchOptions[be.ArchIdx]
	label := core.StyleFieldKey.Render("arch_pattern  ")
	val := core.StyleFieldValActive.Render(current.label) + core.StyleSelectArrow.Render(" ▾")
	row := "     " + label + core.StyleEquals.Render(" = ") + val
	raw := lipgloss.Width(row)
	if raw < w {
		row += strings.Repeat(" ", w-raw)
	}
	lines = append(lines, core.ActiveCurLineStyle().Render(row))

	if be.dropdownOpen {
		lines = append(lines, "")
		for i, opt := range beArchOptions {
			isCur := i == be.dropdownIdx
			var cursor string
			if isCur {
				cursor = core.StyleCurLineNum.Render("  ▶ ")
			} else {
				cursor = "      "
			}
			labelPart := fmt.Sprintf("%-20s", opt.label)
			var optRow string
			if isCur {
				optRow = cursor +
					core.StyleFieldValActive.Render(labelPart) +
					core.StyleSectionDesc.Render(opt.desc)
				rw := lipgloss.Width(optRow)
				if rw < w {
					optRow += strings.Repeat(" ", w-rw)
				}
				optRow = core.ActiveCurLineStyle().Render(optRow)
			} else {
				optRow = cursor +
					core.StyleFieldKey.Render(labelPart) +
					core.StyleSectionDesc.Render(opt.desc)
			}
			lines = append(lines, optRow)
		}
	}

	return core.FillTildes(lines, h)
}

func (be BackendEditor) viewSubTabs(w, h int) string {
	var lines []string

	opt := beArchOptions[be.ArchIdx]
	archStr := core.StyleFieldValActive.Render(opt.label)
	hint := core.StyleSectionDesc.Render("  (b: change arch)")
	lines = append(lines,
		core.StyleSectionDesc.Render("  # Backend · ")+archStr+hint,
		"",
		core.RenderSubTabBar(be.tabLabels(), be.activeTabIdx, w),
		"",
	)

	const beOuterH = 4
	tab := be.activeTab()
	switch tab {
	case beTabEnv:
		if be.currentArch() != "monolith" {
			// Non-monolith: list+form stack config editor.
			cfgLines := be.viewStackConfigEditor(w)
			switch be.stackConfigEditor.itemView {
			case beListViewList:
				cfgLines = core.AppendViewport(cfgLines, 2, be.stackConfigEditor.itemIdx, h-beOuterH)
			case beListViewForm:
				cfgLines = core.AppendViewport(cfgLines, 2, be.stackConfigEditor.formIdx, h-beOuterH)
			}
			lines = append(lines, cfgLines...)
		} else if be.envEnabled {
			envFields := be.visibleEnvFields()
			fl := core.RenderFormFields(w, envFields, be.activeField, be.internalMode == core.ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)
			lines = append(lines, core.AppendViewport(fl, 0, be.activeField, h-beOuterH)...)
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case beTabServices:
		const beListHeaderH = 2
		var svcLines []string
		switch be.repoSubView {
		case beRepoSubViewList:
			svcLines = be.viewRepoEditor(w)
			svcLines = core.AppendViewport(svcLines, beListHeaderH, be.repoEditor.itemIdx, h-beOuterH)
		case beRepoSubViewForm:
			svcLines = be.viewRepoEditor(w)
			svcLines = core.AppendViewport(svcLines, 2, be.repoEditor.formIdx, h-beOuterH)
		case beRepoSubViewOpList:
			svcLines = be.viewOpEditor(w)
			svcLines = core.AppendViewport(svcLines, beListHeaderH, be.opEditor.itemIdx, h-beOuterH)
		case beRepoSubViewOpForm:
			svcLines = be.viewOpEditor(w)
			svcLines = core.AppendViewport(svcLines, 2, be.opEditor.formIdx, h-beOuterH)
		default:
			svcLines = be.viewServiceEditor(w)
			switch be.serviceEditor.itemView {
			case beListViewList:
				svcLines = core.AppendViewport(svcLines, beListHeaderH, be.serviceEditor.itemIdx, h-beOuterH)
			case beListViewForm:
				svcLines = core.AppendViewport(svcLines, 2, be.serviceEditor.formIdx, h-beOuterH)
			}
		}
		lines = append(lines, svcLines...)
	case beTabComm:
		commLines := be.viewCommEditor(w)
		switch be.commEditor.itemView {
		case beListViewList:
			commLines = core.AppendViewport(commLines, 2, be.commEditor.itemIdx, h-beOuterH)
		case beListViewForm:
			commLines = core.AppendViewport(commLines, 2, be.commEditor.formIdx, h-beOuterH)
		}
		lines = append(lines, commLines...)
	case beTabMessaging:
		msgLines := be.viewMessaging(w)
		switch be.eventEditor.itemView {
		case beListViewList:
			msgLines = core.AppendViewport(msgLines, 2, be.eventEditor.itemIdx, h-beOuterH)
		case beListViewForm:
			msgLines = core.AppendViewport(msgLines, 2, be.eventEditor.formIdx, h-beOuterH)
		}
		lines = append(lines, msgLines...)
	case beTabAPIGW:
		if be.apiGWEnabled {
			fl := core.RenderFormFields(w, be.APIGWFields, be.activeField, be.internalMode == core.ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)
			lines = append(lines, core.AppendViewport(fl, 0, be.activeField, h-beOuterH)...)
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case beTabJobs:
		jobLines := be.viewJobs(w)
		switch be.jobsSubView {
		case beViewList:
			jobLines = core.AppendViewport(jobLines, 2, be.jobsIdx, h-beOuterH)
		case beViewForm:
			jobLines = core.AppendViewport(jobLines, 2, be.jobsFormIdx, h-beOuterH)
		}
		lines = append(lines, jobLines...)
	case beTabSecurity:
		if be.secEnabled {
			var visibleSecFields []core.Field
			skippedBefore := 0
			for i, f := range be.securityFields {
				if be.isSecurityFieldHidden(f.Key) {
					if i < be.activeField {
						skippedBefore++
					}
					continue
				}
				visibleSecFields = append(visibleSecFields, f)
			}
			filteredSecIdx := be.activeField - skippedBefore
			if filteredSecIdx < 0 {
				filteredSecIdx = 0
			}
			fl := core.RenderFormFields(w, visibleSecFields, filteredSecIdx, be.internalMode == core.ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)
			lines = append(lines, core.AppendViewport(fl, 0, filteredSecIdx, h-beOuterH)...)
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case beTabAuth:
		authLines := be.viewAuth(w)
		switch be.authSubView {
		case beAuthViewConfig:
			authLines = core.AppendViewport(authLines, 0, be.activeField, h-beOuterH)
		case beAuthViewPermList:
			authLines = core.AppendViewport(authLines, 2, be.authPermsIdx, h-beOuterH)
		case beAuthViewPermForm:
			authLines = core.AppendViewport(authLines, 2, be.authPermFormIdx, h-beOuterH)
		case beAuthViewRoleList:
			authLines = core.AppendViewport(authLines, 2, be.authRolesIdx, h-beOuterH)
		case beAuthViewRoleForm:
			authLines = core.AppendViewport(authLines, 2, be.authRoleFormIdx, h-beOuterH)
		}
		lines = append(lines, authLines...)
	}

	return core.FillTildes(lines, h)
}

func (be BackendEditor) viewServiceEditor(w int) []string {
	ed := be.serviceEditor
	if ed.itemView == beListViewList {
		arch := be.currentArch()
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  # Service Units — a: add  d: delete  Enter: edit"), "")
		if len(ed.items) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no services yet — press 'a' to add)"))
		} else {
			for i, item := range ed.items {
				name := core.FieldGet(item, "name")
				if name == "" {
					name = fmt.Sprintf("(service #%d)", i+1)
				}
				var extra string
				if arch == "monolith" {
					// Stack defined globally in CONFIG tab — show nothing per-service.
				} else {
					cfg := core.FieldGet(item, "config_ref")
					if cfg != "" && cfg != "(no configs defined)" {
						extra = cfg
					}
				}
				// Show repo count if any repositories are defined.
				if i < len(be.Services) && len(be.Services[i].Repositories) > 0 {
					rc := len(be.Services[i].Repositories)
					repoStr := "1 repo"
					if rc > 1 {
						repoStr = fmt.Sprintf("%d repos", rc)
					}
					if extra != "" {
						extra = extra + "  " + repoStr
					} else {
						extra = repoStr
					}
				}
				lines = append(lines, core.RenderListItem(w, i == ed.itemIdx, "  ▶ ", name, extra))
			}
		}
		return lines
	}
	// Form view
	idx := ed.itemIdx
	name := "(new service)"
	if idx < len(ed.items) {
		n := core.FieldGet(ed.form, "name")
		if n != "" {
			name = n
		}
	}

	var fields []core.Field
	skippedBefore := 0
	for i, f := range ed.form {
		if be.isServiceFieldHidden(f.Key) {
			if i < ed.formIdx {
				skippedBefore++
			}
			continue
		}
		fields = append(fields, f)
	}
	filteredActiveIdx := ed.formIdx - skippedBefore
	if filteredActiveIdx < 0 {
		filteredActiveIdx = 0
	}

	// Count repos for this service.
	repoCountHint := ""
	if ed.itemIdx < len(be.Services) {
		nRepos := len(be.Services[ed.itemIdx].Repositories)
		if nRepos == 0 {
			repoCountHint = "  no repos — press R to add"
		} else if nRepos == 1 {
			repoCountHint = "  1 repo — press R to manage"
		} else {
			repoCountHint = fmt.Sprintf("  %d repos — press R to manage", nRepos)
		}
	}

	var lines []string
	lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(name), "")
	lines = append(lines, core.StyleSectionDesc.Render("  (R: data access"+repoCountHint+")"), "")
	// For monolith: show which global stack config is in use.
	if be.currentArch() == "monolith" {
		if be.envEnabled {
			lang := core.FieldGet(be.EnvFields, "monolith_lang")
			fw := core.FieldGet(be.EnvFields, "monolith_fw")
			info := "  stack: " + lang
			if fw != "" {
				info += " / " + fw
			}
			lines = append(lines, core.StyleSectionDesc.Render(info), "")
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  ⚠  Configure the global stack in the CONFIG tab"), "")
		}
	}
	lines = append(lines, core.RenderFormFields(w, fields, filteredActiveIdx, be.internalMode == core.ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
	return lines
}

func (be BackendEditor) viewCommEditor(w int) []string {
	ed := be.commEditor
	if ed.itemView == beListViewList {
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  # Communication Links — a: add  d: delete  Enter: edit"), "")
		if len(ed.items) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no links yet — press 'a' to add)"))
		} else {
			for i, item := range ed.items {
				from := core.FieldGet(item, "from")
				to := core.FieldGet(item, "to")
				if from == "" {
					from = "?"
				}
				if to == "" {
					to = "?"
				}
				proto := core.FieldGet(item, "protocol")
				name := from + " → " + to
				lines = append(lines, core.RenderListItem(w, i == ed.itemIdx, "  ▶ ", name, proto))
			}
		}
		return lines
	}
	from := core.FieldGet(be.commEditor.form, "from")
	to := core.FieldGet(be.commEditor.form, "to")
	title := from + " → " + to
	if from == "" && to == "" {
		title = "(new link)"
	}

	// Filter fields based on direction; response_dto only shown when bidirectional.
	var visibleFields []core.Field
	skippedBefore := 0
	for i, f := range ed.form {
		if be.isCommFieldHidden(f.Key) {
			if i < ed.formIdx {
				skippedBefore++
			}
			continue
		}
		visibleFields = append(visibleFields, f)
	}
	filteredIdx := ed.formIdx - skippedBefore
	if filteredIdx < 0 {
		filteredIdx = 0
	}

	var lines []string
	lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(title), "")
	lines = append(lines, core.RenderFormFields(w, visibleFields, filteredIdx, be.internalMode == core.ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
	return lines
}

func (be BackendEditor) viewMessaging(w int) []string {
	ed := be.eventEditor
	if ed.itemView == beListViewForm {
		name := core.FieldGet(ed.form, "name")
		if name == "" {
			name = "(new event)"
		}
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(name), "")
		lines = append(lines, core.RenderFormFields(w, ed.form, ed.formIdx, be.internalMode == core.ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
		return lines
	}

	// Combined view: broker config fields + event catalog list
	var lines []string
	lines = append(lines, core.StyleSectionDesc.Render("  # Messaging Broker + Event Catalog"), "")

	brokerCount := len(be.MessagingFields)
	// Render broker fields in upper section
	const msgDDIndent = 21 // lineNumW(4) + labelW(14) + eqW(3)
	for i, f := range be.MessagingFields {
		isCur := i == be.activeField
		lineNo := core.StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		if isCur {
			lineNo = core.StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
		}
		var keyStr string
		if isCur {
			keyStr = core.StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = core.StyleFieldKey.Render(f.Label)
		}
		eq := core.StyleEquals.Render(" = ")
		val := f.DisplayValue()
		var valStr string
		if isCur && be.dd.Open {
			valStr = core.StyleFieldValActive.Render(val) + core.StyleSelectArrow.Render(" ▴")
		} else if isCur {
			valStr = core.StyleFieldValActive.Render(val) + core.StyleSelectArrow.Render(" ▾")
		} else {
			valStr = core.StyleFieldVal.Render(val) + core.StyleSelectArrow.Render(" ▾")
		}
		row := lineNo + keyStr + eq + valStr
		if isCur {
			rw := lipgloss.Width(row)
			if rw < w {
				row += strings.Repeat(" ", w-rw)
			}
			row = core.ActiveCurLineStyle().Render(row)
		}
		lines = append(lines, row)
		// Inline dropdown for active broker field
		if isCur && be.dd.Open {
			indent := strings.Repeat(" ", msgDDIndent)
			for j, opt := range f.Options {
				isHL := j == be.dd.OptIdx
				var optRow string
				if isHL {
					optRow = indent + core.StyleFieldValActive.Render("▶ "+opt)
					rw := lipgloss.Width(optRow)
					if rw < w {
						optRow += strings.Repeat(" ", w-rw)
					}
					optRow = core.ActiveCurLineStyle().Render(optRow)
				} else {
					optRow = indent + core.StyleFieldVal.Render("  "+opt)
				}
				lines = append(lines, optRow)
			}
		}
	}

	// Divider + event catalog
	lines = append(lines, "", core.StyleSectionDesc.Render("  ── Event Catalog (a: add  d: delete  Enter: edit) ──"), "")

	if len(ed.items) == 0 {
		lines = append(lines, core.StyleSectionDesc.Render("  (no events yet — press 'a' to add)"))
	} else {
		for i, item := range ed.items {
			globalIdx := brokerCount + i
			isCur := globalIdx == be.activeField
			name := core.FieldGet(item, "name")
			if name == "" {
				name = fmt.Sprintf("(event #%d)", i+1)
			}
			domain := core.FieldGet(item, "domain")
			lines = append(lines, core.RenderListItem(w, isCur, "  ▶ ", name, domain))
		}
	}
	return lines
}

// ── Jobs updates ──────────────────────────────────────────────────────────────
