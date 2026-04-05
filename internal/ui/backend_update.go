package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (be BackendEditor) updateDropdown(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	opts := be.dropdownOptions()
	isMulti := be.isMultiSelectDropdown()
	be.dd.OptIdx = NavigateDropdown(key.String(), be.dd.OptIdx, len(opts))
	ddJustClosed := false
	switch key.String() {
	case " ":
		if isMulti {
			// Space toggles the highlighted option in a multi-select.
			be.toggleMultiSelectOption()
		} else {
			custom := be.applyDropdown()
			be.dd.Open = false
			ddJustClosed = true
			if custom {
				return be.tryEnterInsert()
			}
		}
	case "enter":
		if isMulti {
			be.toggleMultiSelectOption()
		} else {
			custom := be.applyDropdown()
			be.dd.Open = false
			ddJustClosed = true
			if custom {
				return be.tryEnterInsert()
			}
		}
	case "esc", "ctrl+c":
		if isMulti {
			be.saveMultiSelectCursor()
		}
		be.dd.Open = false
		ddJustClosed = true
	}
	// Auto-save the active form so changes persist without requiring b/esc.
	if be.repoSubView == beRepoSubViewForm && be.repoEditor.itemView == beListViewForm {
		be.saveRepoForm()
	} else if be.repoSubView == beRepoSubViewOpForm && be.opEditor.itemView == beListViewForm {
		be.saveOpForm()
	} else if be.serviceEditor.itemView == beListViewForm {
		be.saveServiceForm()
	} else if be.commEditor.itemView == beListViewForm {
		be.saveCommForm()
	} else if be.eventEditor.itemView == beListViewForm {
		be.saveEventForm()
	} else if be.jobsSubView == beViewForm {
		be.saveJobsForm()
	} else if be.authSubView == beAuthViewRoleForm {
		be.saveAuthRoleForm()
	} else if be.authSubView == beAuthViewPermForm {
		be.saveAuthPermForm()
	}
	// Only advance past hidden fields when the dropdown closes, not during j/k navigation.
	// Running these checks on every key (including j/k) causes activeField to jump to a
	// different field while the dropdown is still open, making it appear to switch menus.
	if ddJustClosed {
		// If strategy changed and the cursor now sits on a hidden auth field, advance it.
		if be.authSubView == beAuthViewConfig && be.activeField < len(be.AuthFields) &&
			be.isAuthFieldHidden(be.AuthFields[be.activeField].Key) {
			be.activeField = be.nextAuthFieldIdx(+1)
		}
		// If strategy changed and the cursor now sits on a hidden security field, advance it.
		if be.secEnabled && be.activeTab() == beTabSecurity && be.activeField < len(be.securityFields) &&
			be.isSecurityFieldHidden(be.securityFields[be.activeField].Key) {
			be.activeField = be.nextSecurityFieldIdx(+1)
		}
	}
	return be, nil
}

// isMultiSelectDropdown returns true when the active dropdown field is KindMultiSelect.
func (be BackendEditor) isMultiSelectDropdown() bool {
	if be.repoSubView == beRepoSubViewForm && be.repoEditor.itemView == beListViewForm {
		ed := &be.repoEditor
		if ed.formIdx < len(ed.form) {
			return ed.form[ed.formIdx].Kind == KindMultiSelect
		}
	}
	if be.repoSubView == beRepoSubViewOpForm && be.opEditor.itemView == beListViewForm {
		ed := &be.opEditor
		if ed.formIdx < len(ed.form) {
			return ed.form[ed.formIdx].Kind == KindMultiSelect
		}
	}
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) {
			return ed.form[ed.formIdx].Kind == KindMultiSelect
		}
	}
	if be.commEditor.itemView == beListViewForm {
		ed := &be.commEditor
		if ed.formIdx < len(ed.form) {
			return ed.form[ed.formIdx].Kind == KindMultiSelect
		}
	}
	if be.authSubView == beAuthViewRoleForm {
		if be.authRoleFormIdx < len(be.authRoleForm) {
			return be.authRoleForm[be.authRoleFormIdx].Kind == KindMultiSelect
		}
	}
	if f := be.mutableFieldPtr(); f != nil {
		return f.Kind == KindMultiSelect
	}
	return false
}

// toggleMultiSelectOption toggles ddOptIdx in the active KindMultiSelect field.
func (be *BackendEditor) toggleMultiSelectOption() {
	if be.repoSubView == beRepoSubViewForm && be.repoEditor.itemView == beListViewForm {
		ed := &be.repoEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindMultiSelect {
			ed.form[ed.formIdx].ToggleMultiSelect(be.dd.OptIdx)
			ed.form[ed.formIdx].DDCursor = be.dd.OptIdx
		}
		return
	}
	if be.repoSubView == beRepoSubViewOpForm && be.opEditor.itemView == beListViewForm {
		ed := &be.opEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindMultiSelect {
			ed.form[ed.formIdx].ToggleMultiSelect(be.dd.OptIdx)
			ed.form[ed.formIdx].DDCursor = be.dd.OptIdx
		}
		return
	}
	if be.authSubView == beAuthViewRoleForm {
		if be.authRoleFormIdx < len(be.authRoleForm) && be.authRoleForm[be.authRoleFormIdx].Kind == KindMultiSelect {
			be.authRoleForm[be.authRoleFormIdx].ToggleMultiSelect(be.dd.OptIdx)
			be.authRoleForm[be.authRoleFormIdx].DDCursor = be.dd.OptIdx
		}
		return
	}
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindMultiSelect {
			ed.form[ed.formIdx].ToggleMultiSelect(be.dd.OptIdx)
			ed.form[ed.formIdx].DDCursor = be.dd.OptIdx
			if ed.form[ed.formIdx].Key == "technologies" {
				be.updateServiceErrorFormatOptions(ed)
			}
		}
		return
	}
	if be.commEditor.itemView == beListViewForm {
		ed := &be.commEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindMultiSelect {
			ed.form[ed.formIdx].ToggleMultiSelect(be.dd.OptIdx)
			ed.form[ed.formIdx].DDCursor = be.dd.OptIdx
		}
		return
	}
	if f := be.mutableFieldPtr(); f != nil && f.Kind == KindMultiSelect {
		f.ToggleMultiSelect(be.dd.OptIdx)
		f.DDCursor = be.dd.OptIdx
		if be.activeTab() == beTabAuth && be.authSubView == beAuthViewConfig && f.Key == "strategy" {
			be.updateAuthTokenStorageOptions()
		}
	}
}

// saveMultiSelectCursor saves the current dropdown cursor back to the field.
func (be *BackendEditor) saveMultiSelectCursor() {
	if be.repoSubView == beRepoSubViewForm && be.repoEditor.itemView == beListViewForm {
		ed := &be.repoEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindMultiSelect {
			ed.form[ed.formIdx].DDCursor = be.dd.OptIdx
		}
		return
	}
	if be.repoSubView == beRepoSubViewOpForm && be.opEditor.itemView == beListViewForm {
		ed := &be.opEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindMultiSelect {
			ed.form[ed.formIdx].DDCursor = be.dd.OptIdx
		}
		return
	}
	if be.authSubView == beAuthViewRoleForm {
		if be.authRoleFormIdx < len(be.authRoleForm) && be.authRoleForm[be.authRoleFormIdx].Kind == KindMultiSelect {
			be.authRoleForm[be.authRoleFormIdx].DDCursor = be.dd.OptIdx
		}
		return
	}
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindMultiSelect {
			ed.form[ed.formIdx].DDCursor = be.dd.OptIdx
		}
		return
	}
	if be.commEditor.itemView == beListViewForm {
		ed := &be.commEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindMultiSelect {
			ed.form[ed.formIdx].DDCursor = be.dd.OptIdx
		}
		return
	}
	if f := be.mutableFieldPtr(); f != nil && f.Kind == KindMultiSelect {
		f.DDCursor = be.dd.OptIdx
	}
}

// dropdownOptions returns the options of the currently active KindSelect or KindMultiSelect field.
func (be BackendEditor) dropdownOptions() []string {
	if be.repoSubView == beRepoSubViewForm && be.repoEditor.itemView == beListViewForm {
		ed := &be.repoEditor
		if ed.formIdx < len(ed.form) && (ed.form[ed.formIdx].Kind == KindSelect || ed.form[ed.formIdx].Kind == KindMultiSelect) {
			return ed.form[ed.formIdx].Options
		}
	}
	if be.repoSubView == beRepoSubViewOpForm && be.opEditor.itemView == beListViewForm {
		ed := &be.opEditor
		if ed.formIdx < len(ed.form) && (ed.form[ed.formIdx].Kind == KindSelect || ed.form[ed.formIdx].Kind == KindMultiSelect) {
			return ed.form[ed.formIdx].Options
		}
	}
	if be.stackConfigEditor.itemView == beListViewForm {
		ed := &be.stackConfigEditor
		if ed.formIdx < len(ed.form) && (ed.form[ed.formIdx].Kind == KindSelect || ed.form[ed.formIdx].Kind == KindMultiSelect) {
			return ed.form[ed.formIdx].Options
		}
	}
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) && (ed.form[ed.formIdx].Kind == KindSelect || ed.form[ed.formIdx].Kind == KindMultiSelect) {
			return ed.form[ed.formIdx].Options
		}
	}
	if be.commEditor.itemView == beListViewForm {
		ed := &be.commEditor
		if ed.formIdx < len(ed.form) && (ed.form[ed.formIdx].Kind == KindSelect || ed.form[ed.formIdx].Kind == KindMultiSelect) {
			return ed.form[ed.formIdx].Options
		}
	}
	if be.eventEditor.itemView == beListViewForm {
		ed := &be.eventEditor
		if ed.formIdx < len(ed.form) && (ed.form[ed.formIdx].Kind == KindSelect || ed.form[ed.formIdx].Kind == KindMultiSelect) {
			return ed.form[ed.formIdx].Options
		}
	}
	if be.jobsSubView == beViewForm {
		if be.jobsFormIdx < len(be.jobsForm) && (be.jobsForm[be.jobsFormIdx].Kind == KindSelect || be.jobsForm[be.jobsFormIdx].Kind == KindMultiSelect) {
			return be.jobsForm[be.jobsFormIdx].Options
		}
	}
	if be.authSubView == beAuthViewRoleForm {
		if be.authRoleFormIdx < len(be.authRoleForm) && (be.authRoleForm[be.authRoleFormIdx].Kind == KindSelect || be.authRoleForm[be.authRoleFormIdx].Kind == KindMultiSelect) {
			return be.authRoleForm[be.authRoleFormIdx].Options
		}
	}
	if f := be.mutableFieldPtr(); f != nil {
		return f.Options
	}
	return nil
}

// applyDropdown writes ddOptIdx back to the active KindSelect field.
// applyDropdown applies the highlighted dropdown option to the active field.
// Returns true if the selected option is "Custom"/"Other" (caller should enter insert mode).
func (be *BackendEditor) applyDropdown() bool {
	applyTo := func(f *Field) bool {
		if f == nil || f.Kind != KindSelect || be.dd.OptIdx >= len(f.Options) {
			return false
		}
		f.SelIdx = be.dd.OptIdx
		f.Value = f.Options[be.dd.OptIdx]
		return f.PrepareCustomEntry()
	}
	if be.repoSubView == beRepoSubViewForm && be.repoEditor.itemView == beListViewForm {
		ed := &be.repoEditor
		if ed.formIdx < len(ed.form) {
			f := &ed.form[ed.formIdx]
			custom := applyTo(f)
			if f.Key == "entity_ref" {
				be.refreshRepoFieldOptions()
			} else if f.Key == "target_db" {
				be.refreshRepoEntityRefOptions()
			}
			return custom
		}
		return false
	}
	if be.repoSubView == beRepoSubViewOpForm && be.opEditor.itemView == beListViewForm {
		ed := &be.opEditor
		if ed.formIdx < len(ed.form) {
			return applyTo(&ed.form[ed.formIdx])
		}
		return false
	}
	if be.stackConfigEditor.itemView == beListViewForm {
		ed := &be.stackConfigEditor
		if ed.formIdx < len(ed.form) {
			f := &ed.form[ed.formIdx]
			custom := applyTo(f)
			if f.Key == "language" {
				be.updateStackConfigFrameworkOptions(ed)
			} else if f.Key == "language_version" || f.Key == "framework" {
				be.updateStackConfigVersionOptions(ed)
			}
			return custom
		}
		return false
	}
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) {
			f := &ed.form[ed.formIdx]
			custom := applyTo(f)
			if f.Key == "language" {
				be.updateServiceFrameworkOptions(ed)
			} else if f.Key == "language_version" || f.Key == "framework" {
				be.updateServiceVersionOptions(ed)
			}
			return custom
		}
		return false
	}
	if be.commEditor.itemView == beListViewForm {
		ed := &be.commEditor
		if ed.formIdx < len(ed.form) {
			return applyTo(&ed.form[ed.formIdx])
		}
		return false
	}
	if be.eventEditor.itemView == beListViewForm {
		ed := &be.eventEditor
		if ed.formIdx < len(ed.form) {
			return applyTo(&ed.form[ed.formIdx])
		}
		return false
	}
	if be.jobsSubView == beViewForm {
		if be.jobsFormIdx < len(be.jobsForm) {
			f := &be.jobsForm[be.jobsFormIdx]
			custom := applyTo(f)
			if f.Key == "config_ref" {
				be.updateJobQueueTechOptions()
			}
			return custom
		}
		return false
	}
	if be.authSubView == beAuthViewRoleForm {
		if be.authRoleFormIdx < len(be.authRoleForm) {
			f := &be.authRoleForm[be.authRoleFormIdx]
			if f.Kind == KindSelect && be.dd.OptIdx < len(f.Options) {
				f.SelIdx = be.dd.OptIdx
				f.Value = f.Options[be.dd.OptIdx]
			}
			// KindMultiSelect handled via toggleMultiSelectOption
		}
		return false
	}
	if f := be.mutableFieldPtr(); f != nil && f.Kind == KindSelect && be.dd.OptIdx < len(f.Options) {
		f.SelIdx = be.dd.OptIdx
		f.Value = f.Options[be.dd.OptIdx]
		switch be.activeTab() {
		case beTabEnv:
			switch f.Key {
			case "monolith_lang":
				be.updateEnvMonolithOptions()
			case "monolith_lang_ver", "monolith_fw":
				be.updateEnvMonolithVersionOptions()
			case "compute_env":
				be.updateEnvOrchestratorOptions()
			case "orchestrator":
				be.updateServiceDiscoveryOptions()
			}
		case beTabAPIGW:
			if f.Key == "environment" {
				be.updateAPIGWTechOptions()
			}
		case beTabAuth:
			if be.authSubView == beAuthViewConfig && f.Key == "provider" {
				be.updateAuthMFAOptions()
			}
		case beTabMessaging:
			if f.Key == "broker_tech" {
				be.refreshMessagingDeploymentOptions()
			}
		}
	}
	return applyTo(be.mutableFieldPtr())
}

func (be BackendEditor) updateInsert(msg tea.Msg) (BackendEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			be.saveInput()
			be.internalMode = ModeNormal
			be.formInput.Blur()
			return be, nil
		case "tab":
			be.saveInput()
			if be.stackConfigEditor.itemView == beListViewForm {
				n := len(be.stackConfigEditor.form)
				if n > 0 {
					be.stackConfigEditor.formIdx = (be.stackConfigEditor.formIdx + 1) % n
				}
				return be.enterStackConfigFormInsert()
			}
			if be.authSubView == beAuthViewPermForm {
				n := len(be.authPermForm)
				if n > 0 {
					be.authPermFormIdx = (be.authPermFormIdx + 1) % n
					be.activeField = be.authPermFormIdx
				}
				return be.enterAuthPermFormInsert()
			}
			if be.authSubView == beAuthViewRoleForm {
				n := len(be.authRoleForm)
				if n > 0 {
					be.authRoleFormIdx = (be.authRoleFormIdx + 1) % n
					be.activeField = be.authRoleFormIdx
				}
				return be.enterAuthRoleFormInsert()
			}
			if be.jobsSubView == beViewForm {
				n := len(be.jobsForm)
				if n > 0 {
					be.jobsFormIdx = (be.jobsFormIdx + 1) % n
					be.activeField = be.jobsFormIdx
				}
				return be.enterJobsFormInsert()
			}
			fields := be.currentEditableFields()
			if fields != nil {
				be.activeField = (be.activeField + 1) % len(*fields)
				return be.tryEnterInsert()
			}
		case "shift+tab":
			be.saveInput()
			if be.stackConfigEditor.itemView == beListViewForm {
				n := len(be.stackConfigEditor.form)
				if n > 0 {
					be.stackConfigEditor.formIdx = (be.stackConfigEditor.formIdx - 1 + n) % n
				}
				return be.enterStackConfigFormInsert()
			}
			if be.authSubView == beAuthViewPermForm {
				n := len(be.authPermForm)
				if n > 0 {
					be.authPermFormIdx = (be.authPermFormIdx - 1 + n) % n
					be.activeField = be.authPermFormIdx
				}
				return be.enterAuthPermFormInsert()
			}
			if be.authSubView == beAuthViewRoleForm {
				n := len(be.authRoleForm)
				if n > 0 {
					be.authRoleFormIdx = (be.authRoleFormIdx - 1 + n) % n
					be.activeField = be.authRoleFormIdx
				}
				return be.enterAuthRoleFormInsert()
			}
			if be.jobsSubView == beViewForm {
				n := len(be.jobsForm)
				if n > 0 {
					be.jobsFormIdx = (be.jobsFormIdx - 1 + n) % n
					be.activeField = be.jobsFormIdx
				}
				return be.enterJobsFormInsert()
			}
			fields := be.currentEditableFields()
			if fields != nil {
				n := len(*fields)
				be.activeField = (be.activeField - 1 + n) % n
				return be.tryEnterInsert()
			}
		}
	}
	var cmd tea.Cmd
	be.formInput, cmd = be.formInput.Update(msg)
	return be, cmd
}

func (be BackendEditor) updateNormal(msg tea.Msg) (BackendEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return be, nil
	}

	tab := be.activeTab()

	// Delegate list editors
	switch tab {
	case beTabEnv:
		if be.currentArch() != "monolith" {
			if be.stackConfigEditor.itemView == beListViewList {
				return be.updateStackConfigList(key)
			}
			return be.updateStackConfigForm(key)
		}
	case beTabServices:
		if be.serviceEditor.itemView == beListViewList {
			return be.updateServiceList(key)
		}
		switch be.repoSubView {
		case beRepoSubViewList:
			return be.updateRepoList(key)
		case beRepoSubViewForm:
			return be.updateRepoForm(key)
		case beRepoSubViewOpList:
			return be.updateOpList(key)
		case beRepoSubViewOpForm:
			return be.updateOpForm(key)
		}
		return be.updateServiceForm(key)
	case beTabComm:
		if be.commEditor.itemView == beListViewList {
			return be.updateCommList(key)
		}
		return be.updateCommForm(key)
	case beTabMessaging:
		if be.eventEditor.itemView == beListViewForm {
			return be.updateEventForm(key)
		}
		return be.updateMessaging(key)
	case beTabJobs:
		if be.jobsSubView == beViewList {
			return be.updateJobsList(key)
		}
		return be.updateJobsForm(key)
	case beTabAuth:
		return be.updateAuth(key)
	case beTabSecurity:
		return be.updateSecurity(key)
	}

	// Enabled guard for config-only tabs (ENV, API GW)
	k := key.String()
	activeConfigEnabled := true
	switch tab {
	case beTabEnv:
		activeConfigEnabled = be.envEnabled
	case beTabAPIGW:
		activeConfigEnabled = be.apiGWEnabled
	}
	if !activeConfigEnabled {
		if k == "a" {
			switch tab {
			case beTabEnv:
				be.envEnabled = true
			case beTabAPIGW:
				be.apiGWEnabled = true
			case beTabAuth:
				be.authEnabled = true
			}
			be.activeField = 0
		} else if k == "h" || k == "left" {
			be.countBuf = ""
			be.gBuf = false
			if be.activeTabIdx > 0 {
				be.activeTabIdx--
			}
		} else if k == "l" || k == "right" {
			be.countBuf = ""
			be.gBuf = false
			if be.activeTabIdx < len(be.activeTabs())-1 {
				be.activeTabIdx++
			}
		} else if k == "b" {
			be.countBuf = ""
			be.gBuf = false
			be.ArchConfirmed = false
			be.dropdownOpen = false
			be.dropdownIdx = be.ArchIdx
			be.activeTabIdx = 0
			be.activeField = 0
		}
		return be, nil
	}

	// Vim count prefix (digits 1-9, or 0 when count already started)
	if len(k) == 1 && k[0] >= '1' && k[0] <= '9' {
		be.countBuf += k
		be.gBuf = false
		return be, nil
	}
	if k == "0" && be.countBuf != "" {
		be.countBuf += "0"
		be.gBuf = false
		return be, nil
	}

	// Shift+D resets the active single-config tab (ENV, API GW, AUTH).
	if k == "D" {
		switch tab {
		case beTabEnv:
			be.envEnabled = false
			be.EnvFields = defaultEnvFields()
			be.activeField = 0
		case beTabAPIGW:
			be.apiGWEnabled = false
			be.APIGWFields = defaultAPIGWFields()
			be.activeField = 0
		}
		return be, nil
	}

	// Generic field navigation for ENV, API GW, AUTH
	switch k {
	case "b":
		be.countBuf = ""
		be.gBuf = false
		be.ArchConfirmed = false
		be.dropdownOpen = false
		be.dropdownIdx = be.ArchIdx
		be.activeTabIdx = 0
		be.activeField = 0
	case "h", "left":
		be.countBuf = ""
		be.gBuf = false
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		be.countBuf = ""
		be.gBuf = false
		if be.activeTabIdx < len(be.activeTabs())-1 {
			be.activeTabIdx++
		}
	case "j", "down":
		count := parseVimCount(be.countBuf)
		be.countBuf = ""
		be.gBuf = false
		if be.activeTab() == beTabEnv {
			visible := be.visibleEnvFields()
			for i := 0; i < count; i++ {
				if be.activeField < len(visible)-1 {
					be.activeField++
				}
			}
		} else if fields := be.currentEditableFields(); fields != nil {
			for i := 0; i < count; i++ {
				if be.activeField < len(*fields)-1 {
					be.activeField++
				}
			}
		}
	case "k", "up":
		count := parseVimCount(be.countBuf)
		be.countBuf = ""
		be.gBuf = false
		for i := 0; i < count; i++ {
			if be.activeField > 0 {
				be.activeField--
			}
		}
	case "g":
		if be.gBuf {
			// gg — go to top
			be.activeField = 0
			be.gBuf = false
		} else {
			be.gBuf = true
		}
		be.countBuf = ""
	case "G":
		be.countBuf = ""
		be.gBuf = false
		if be.activeTab() == beTabEnv {
			visible := be.visibleEnvFields()
			if len(visible) > 0 {
				be.activeField = len(visible) - 1
			}
		} else if fields := be.currentEditableFields(); fields != nil {
			be.activeField = len(*fields) - 1
		}
	case "enter", " ":
		be.countBuf = ""
		be.gBuf = false
		if f := be.mutableFieldPtr(); f != nil && (f.Kind == KindSelect || f.Kind == KindMultiSelect) {
			be.dd.Open = true
			if f.Kind == KindSelect {
				be.dd.OptIdx = f.SelIdx
			} else {
				be.dd.OptIdx = f.DDCursor
			}
		} else {
			return be.tryEnterInsert()
		}
	case "H", "shift+left":
		be.countBuf = ""
		be.gBuf = false
		if f := be.mutableFieldPtr(); f != nil && f.Kind == KindSelect {
			f.CyclePrev()
			if be.activeTab() == beTabEnv {
				switch f.Key {
				case "monolith_lang":
					be.updateEnvMonolithOptions()
				case "monolith_lang_ver", "monolith_fw":
					be.updateEnvMonolithVersionOptions()
				case "compute_env":
					be.updateEnvOrchestratorOptions()
				case "orchestrator":
					be.updateServiceDiscoveryOptions()
				}
			} else if be.activeTab() == beTabAPIGW && f.Key == "environment" {
				be.updateAPIGWTechOptions()
			} else if be.activeTab() == beTabAuth && be.authSubView == beAuthViewConfig && f.Key == "provider" {
				be.updateAuthMFAOptions()
			}
		}
	case "i", "a":
		be.countBuf = ""
		be.gBuf = false
		return be.tryEnterInsert()
	default:
		be.countBuf = ""
		be.gBuf = false
	}
	return be, nil
}
