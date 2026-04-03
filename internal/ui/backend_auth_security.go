package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── Auth updates ──────────────────────────────────────────────────────────────

func (be BackendEditor) updateAuth(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	k := key.String()
	if !be.authEnabled {
		switch k {
		case "a":
			be.authEnabled = true
			be.activeField = 0
		case "h", "left":
			if be.activeTabIdx > 0 {
				be.activeTabIdx--
			}
		case "l", "right":
			if be.activeTabIdx < len(be.activeTabs())-1 {
				be.activeTabIdx++
			}
		case "b":
			be.ArchConfirmed = false
			be.dropdownOpen = false
			be.dropdownIdx = be.ArchIdx
			be.activeTabIdx = 0
			be.activeField = 0
		}
		return be, nil
	}
	switch be.authSubView {
	case beAuthViewConfig:
		return be.updateAuthConfig(key)
	case beAuthViewRoleList:
		return be.updateAuthRoleList(key)
	case beAuthViewRoleForm:
		return be.updateAuthRoleForm(key)
	case beAuthViewPermList:
		return be.updateAuthPermList(key)
	case beAuthViewPermForm:
		return be.updateAuthPermForm(key)
	}
	return be, nil
}

func (be BackendEditor) updateAuthConfig(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	k := key.String()
	n := len(be.AuthFields)

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

	switch k {
	case "j", "down":
		count := parseVimCount(be.countBuf)
		be.countBuf = ""
		be.gBuf = false
		for i := 0; i < count; i++ {
			if be.activeField < n-1 {
				be.activeField++
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
			be.activeField = 0
			be.gBuf = false
		} else {
			be.gBuf = true
		}
		be.countBuf = ""
	case "G":
		be.countBuf = ""
		be.gBuf = false
		if n > 0 {
			be.activeField = n - 1
		}
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
	case "b":
		be.countBuf = ""
		be.gBuf = false
		be.ArchConfirmed = false
		be.dropdownOpen = false
		be.dropdownIdx = be.ArchIdx
		be.activeTabIdx = 0
		be.activeField = 0
	case "r":
		be.countBuf = ""
		be.gBuf = false
		be.authSubView = beAuthViewRoleList
		be.activeField = 0
	case "p":
		be.countBuf = ""
		be.gBuf = false
		be.authSubView = beAuthViewPermList
		be.activeField = 0
	case "D":
		be.countBuf = ""
		be.gBuf = false
		be.authEnabled = false
		be.AuthFields = defaultAuthFields()
		be.authPerms = nil
		be.authPermsIdx = 0
		be.authRoles = nil
		be.authRolesIdx = 0
		be.authSubView = beAuthViewConfig
		be.activeField = 0
	case "enter", " ":
		be.countBuf = ""
		be.gBuf = false
		if be.activeField < n {
			f := &be.AuthFields[be.activeField]
			if f.Kind == KindSelect || f.Kind == KindMultiSelect {
				if f.Key == "service_unit" {
					be.refreshAuthServiceUnitOptions(f)
				}
				be.dd.Open = true
				if f.Kind == KindSelect {
					be.dd.OptIdx = f.SelIdx
				} else {
					be.dd.OptIdx = f.DDCursor
				}
			} else {
				return be.tryEnterInsert()
			}
		}
	case "H", "shift+left":
		be.countBuf = ""
		be.gBuf = false
		if be.activeField < n {
			f := &be.AuthFields[be.activeField]
			if f.Kind == KindSelect {
				if f.Key == "service_unit" {
					be.refreshAuthServiceUnitOptions(f)
				}
				f.CyclePrev()
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

func (be BackendEditor) updateAuthRoleList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.authRoles)
	switch key.String() {
	case "j", "down":
		if n > 0 && be.authRolesIdx < n-1 {
			be.authRolesIdx++
		}
	case "k", "up":
		if be.authRolesIdx > 0 {
			be.authRolesIdx--
		}
	case "a":
		be.authRoles = append(be.authRoles, manifest.RoleDef{})
		be.authRolesIdx = len(be.authRoles) - 1
		be.authRoleForm = defaultRoleFormFields(be.permissionNames(), be.roleNamesExcept(be.authRolesIdx))
		existing := make([]string, 0, len(be.authRoles)-1)
		for i, r := range be.authRoles {
			if i != be.authRolesIdx {
				existing = append(existing, r.Name)
			}
		}
		be.authRoleForm = setFieldValue(be.authRoleForm, "name", uniqueName("role", existing))
		be.authRoleFormIdx = 0
		be.authSubView = beAuthViewRoleForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.authRoles = append(be.authRoles[:be.authRolesIdx], be.authRoles[be.authRolesIdx+1:]...)
			if be.authRolesIdx > 0 && be.authRolesIdx >= len(be.authRoles) {
				be.authRolesIdx = len(be.authRoles) - 1
			}
		}
	case "enter", "i":
		if n > 0 {
			r := be.authRoles[be.authRolesIdx]
			be.authRoleForm = defaultRoleFormFields(be.permissionNames(), be.roleNamesExcept(be.authRolesIdx))
			be.authRoleForm = setFieldValue(be.authRoleForm, "name", r.Name)
			be.authRoleForm = setFieldValue(be.authRoleForm, "description", r.Description)
			be.authRoleForm = restoreMultiSelectValue(be.authRoleForm, "permissions", strings.Join(r.Permissions, ", "))
			be.authRoleForm = restoreMultiSelectValue(be.authRoleForm, "inherits", strings.Join(r.Inherits, ", "))
			be.authRoleFormIdx = 0
			be.authSubView = beAuthViewRoleForm
			be.activeField = 0
		}
	case "b", "esc":
		be.authSubView = beAuthViewConfig
		be.activeField = 0
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		if be.activeTabIdx < len(be.activeTabs())-1 {
			be.activeTabIdx++
		}
	}
	return be, nil
}

func (be BackendEditor) updateAuthRoleForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.authRoleForm)
	switch key.String() {
	case "j", "down":
		if be.authRoleFormIdx < n-1 {
			be.authRoleFormIdx++
		}
		be.activeField = be.authRoleFormIdx
	case "k", "up":
		if be.authRoleFormIdx > 0 {
			be.authRoleFormIdx--
		}
		be.activeField = be.authRoleFormIdx
	case "enter", " ":
		if be.authRoleFormIdx < n {
			f := &be.authRoleForm[be.authRoleFormIdx]
			if f.Kind == KindMultiSelect {
				be.dd.Open = true
				be.dd.OptIdx = f.DDCursor
			} else if f.Kind == KindText {
				return be.enterAuthRoleFormInsert()
			}
		}
	case "i", "a":
		if be.authRoleFormIdx < n && be.authRoleForm[be.authRoleFormIdx].Kind == KindText {
			return be.enterAuthRoleFormInsert()
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		tabs := be.activeTabs()
		if be.activeTabIdx < len(tabs)-1 {
			be.activeTabIdx++
		}
	case "b", "esc":
		be.saveAuthRoleForm()
		be.authSubView = beAuthViewRoleList
	}
	return be, nil
}

func (be BackendEditor) enterAuthRoleFormInsert() (BackendEditor, tea.Cmd) {
	n := len(be.authRoleForm)
	for i := 0; i < n; i++ {
		f := be.authRoleForm[be.authRoleFormIdx]
		if f.Kind == KindText {
			be.internalMode = ModeInsert
			be.formInput.SetValue(f.Value)
			be.formInput.Width = be.width - 22
			be.formInput.CursorEnd()
			return be, be.formInput.Focus()
		}
		be.authRoleFormIdx = (be.authRoleFormIdx + 1) % n
		be.activeField = be.authRoleFormIdx
	}
	return be, nil
}

func (be *BackendEditor) saveAuthRoleForm() {
	if be.authRolesIdx >= len(be.authRoles) {
		return
	}
	r := &be.authRoles[be.authRolesIdx]
	r.Name = fieldGet(be.authRoleForm, "name")
	r.Description = fieldGet(be.authRoleForm, "description")
	r.Permissions = splitCSV(fieldGetMulti(be.authRoleForm, "permissions"))
	r.Inherits = splitCSV(fieldGetMulti(be.authRoleForm, "inherits"))
}

func (be BackendEditor) updateAuthPermList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.authPerms)
	switch key.String() {
	case "j", "down":
		if n > 0 && be.authPermsIdx < n-1 {
			be.authPermsIdx++
		}
	case "k", "up":
		if be.authPermsIdx > 0 {
			be.authPermsIdx--
		}
	case "a":
		be.authPerms = append(be.authPerms, manifest.PermissionDef{})
		be.authPermsIdx = len(be.authPerms) - 1
		be.authPermForm = defaultPermFormFields()
		existing := make([]string, 0, len(be.authPerms)-1)
		for i, p := range be.authPerms {
			if i != be.authPermsIdx {
				existing = append(existing, p.Name)
			}
		}
		be.authPermForm = setFieldValue(be.authPermForm, "name", uniqueName("permission", existing))
		be.authPermFormIdx = 0
		be.authSubView = beAuthViewPermForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.authPerms = append(be.authPerms[:be.authPermsIdx], be.authPerms[be.authPermsIdx+1:]...)
			if be.authPermsIdx > 0 && be.authPermsIdx >= len(be.authPerms) {
				be.authPermsIdx = len(be.authPerms) - 1
			}
		}
	case "enter", "i":
		if n > 0 {
			p := be.authPerms[be.authPermsIdx]
			be.authPermForm = defaultPermFormFields()
			be.authPermForm = setFieldValue(be.authPermForm, "name", p.Name)
			be.authPermForm = setFieldValue(be.authPermForm, "description", p.Description)
			be.authPermFormIdx = 0
			be.authSubView = beAuthViewPermForm
			be.activeField = 0
		}
	case "b", "esc":
		be.authSubView = beAuthViewConfig
		be.activeField = 0
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		if be.activeTabIdx < len(be.activeTabs())-1 {
			be.activeTabIdx++
		}
	}
	return be, nil
}

func (be BackendEditor) updateAuthPermForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.authPermForm)
	switch key.String() {
	case "j", "down":
		if be.authPermFormIdx < n-1 {
			be.authPermFormIdx++
		}
		be.activeField = be.authPermFormIdx
	case "k", "up":
		if be.authPermFormIdx > 0 {
			be.authPermFormIdx--
		}
		be.activeField = be.authPermFormIdx
	case "enter", "i", "a":
		if be.authPermFormIdx < n && be.authPermForm[be.authPermFormIdx].Kind == KindText {
			return be.enterAuthPermFormInsert()
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		tabs := be.activeTabs()
		if be.activeTabIdx < len(tabs)-1 {
			be.activeTabIdx++
		}
	case "b", "esc":
		be.saveAuthPermForm()
		be.authSubView = beAuthViewPermList
	}
	return be, nil
}

func (be BackendEditor) enterAuthPermFormInsert() (BackendEditor, tea.Cmd) {
	if be.authPermFormIdx < len(be.authPermForm) {
		f := be.authPermForm[be.authPermFormIdx]
		if f.Kind == KindText {
			be.internalMode = ModeInsert
			be.formInput.SetValue(f.Value)
			be.formInput.Width = be.width - 22
			be.formInput.CursorEnd()
			return be, be.formInput.Focus()
		}
	}
	return be, nil
}

func (be *BackendEditor) saveAuthPermForm() {
	if be.authPermsIdx >= len(be.authPerms) {
		return
	}
	p := &be.authPerms[be.authPermsIdx]
	p.Name = fieldGet(be.authPermForm, "name")
	p.Description = fieldGet(be.authPermForm, "description")
}

// permissionNames returns names of all defined permissions.
func (be BackendEditor) permissionNames() []string {
	names := make([]string, 0, len(be.authPerms))
	for _, p := range be.authPerms {
		if p.Name != "" {
			names = append(names, p.Name)
		}
	}
	return names
}

// authProviderNeedsService returns true for providers that are self-hosted and
// require a backend service unit to handle authentication.
func authProviderNeedsService(provider string) bool {
	switch provider {
	case "Self-managed", "Keycloak":
		return true
	}
	return false
}

// refreshAuthServiceUnitOptions updates the service_unit field options based on
// the currently selected provider. External providers get a single "None (external)"
// option; self-managed providers get the list of configured service names.
func (be *BackendEditor) refreshAuthServiceUnitOptions(f *Field) {
	provider := fieldGet(be.AuthFields, "provider")
	var opts []string
	if authProviderNeedsService(provider) {
		svcNames := be.ServiceNames()
		if len(svcNames) == 0 {
			opts = []string{"(no services configured)"}
		} else {
			opts = append([]string{"None"}, svcNames...)
		}
	} else {
		opts = []string{"None (external)"}
	}
	f.Options = opts
	// Keep current value if still present; otherwise reset to first option.
	found := false
	for j, o := range opts {
		if o == f.Value {
			f.SelIdx = j
			found = true
			break
		}
	}
	if !found {
		f.SelIdx = 0
		f.Value = opts[0]
	}
}

// roleNamesExcept returns names of all roles except the one at excludeIdx.
func (be BackendEditor) roleNamesExcept(excludeIdx int) []string {
	names := make([]string, 0, len(be.authRoles))
	for i, r := range be.authRoles {
		if i != excludeIdx && r.Name != "" {
			names = append(names, r.Name)
		}
	}
	return names
}


// viewAuth renders the AUTH tab content.
func (be BackendEditor) viewAuth(w int) []string {
	if !be.authEnabled {
		return []string{StyleSectionDesc.Render("  (not configured — press 'a' to configure)")}
	}
	switch be.authSubView {
	case beAuthViewConfig:
		lines := renderFormFields(w, be.AuthFields, be.activeField, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)
		permCount := fmt.Sprintf("%d", len(be.authPerms))
		roleCount := fmt.Sprintf("%d", len(be.authRoles))
		lines = append(lines,
			"",
			StyleSectionDesc.Render("  # Permissions ("+permCount+" defined) — press 'p' to manage"),
			StyleSectionDesc.Render("  # Roles ("+roleCount+" defined) — press 'r' to manage"),
		)
		return lines
	case beAuthViewPermList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Permissions — a: add  d: delete  Enter: edit  b: back"), "")
		if len(be.authPerms) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no permissions yet — press 'a' to add)"))
		} else {
			for i, p := range be.authPerms {
				name := p.Name
				if name == "" {
					name = fmt.Sprintf("(perm #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == be.authPermsIdx, "  ▶ ", name, p.Description))
			}
		}
		return lines
	case beAuthViewPermForm:
		name := fieldGet(be.authPermForm, "name")
		if name == "" {
			name = "(new permission)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFields(w, be.authPermForm, be.authPermFormIdx, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
		return lines
	case beAuthViewRoleList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Roles — a: add  d: delete  Enter: edit  b: back"), "")
		if len(be.authRoles) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no roles yet — press 'a' to add)"))
		} else {
			for i, r := range be.authRoles {
				name := r.Name
				if name == "" {
					name = fmt.Sprintf("(role #%d)", i+1)
				}
				detail := ""
				if len(r.Permissions) > 0 {
					detail = strings.Join(r.Permissions[:min(3, len(r.Permissions))], ", ")
					if len(r.Permissions) > 3 {
						detail += "…"
					}
				}
				lines = append(lines, renderListItem(w, i == be.authRolesIdx, "  ▶ ", name, detail))
			}
		}
		return lines
	case beAuthViewRoleForm:
		name := fieldGet(be.authRoleForm, "name")
		if name == "" {
			name = "(new role)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFields(w, be.authRoleForm, be.authRoleFormIdx, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
		return lines
	}
	return nil
}

// ── Security updates ──────────────────────────────────────────────────────────

func (be BackendEditor) updateSecurity(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	k := key.String()
	if !be.secEnabled {
		switch k {
		case "a":
			be.secEnabled = true
			be.activeField = 0
		case "h", "left":
			if be.activeTabIdx > 0 {
				be.activeTabIdx--
			}
		case "l", "right":
			if be.activeTabIdx < len(be.activeTabs())-1 {
				be.activeTabIdx++
			}
		case "b":
			be.ArchConfirmed = false
			be.dropdownOpen = false
			be.dropdownIdx = be.ArchIdx
			be.activeTabIdx = 0
			be.activeField = 0
		}
		return be, nil
	}
	// Security uses generic field navigation via currentEditableFields / mutableFieldPtr
	// which already handles beTabSecurity. Just fall through to normal key handling.
	n := len(be.securityFields)

	// Vim count prefix
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

	switch k {
	case "j", "down":
		count := parseVimCount(be.countBuf)
		be.countBuf = ""
		be.gBuf = false
		for i := 0; i < count; i++ {
			if be.activeField < n-1 {
				be.activeField++
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
			be.activeField = 0
			be.gBuf = false
		} else {
			be.gBuf = true
		}
		be.countBuf = ""
	case "G":
		be.countBuf = ""
		be.gBuf = false
		be.activeField = n - 1
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
	case "b":
		be.countBuf = ""
		be.gBuf = false
		be.ArchConfirmed = false
		be.dropdownOpen = false
		be.dropdownIdx = be.ArchIdx
		be.activeTabIdx = 0
		be.activeField = 0
	case "enter", " ":
		be.countBuf = ""
		be.gBuf = false
		if be.activeField < n {
			f := &be.securityFields[be.activeField]
			if f.Kind == KindSelect {
				// Refresh options for cache-dependent fields just before opening.
				if f.Key == "rate_limit_backend" {
					opts := be.rateBackendOptions()
					f.Options = opts
					// Reconcile SelIdx: keep if still valid, else reset to "None".
					found := false
					for j, o := range opts {
						if o == f.Value {
							f.SelIdx = j
							found = true
							break
						}
					}
					if !found {
						f.SelIdx = len(opts) - 1
						f.Value = opts[len(opts)-1]
					}
				}
				be.dd.Open = true
				be.dd.OptIdx = f.SelIdx
			}
		}
	case "H", "shift+left":
		be.countBuf = ""
		be.gBuf = false
		if be.activeField < n {
			f := &be.securityFields[be.activeField]
			if f.Kind == KindSelect {
				if f.Key == "rate_limit_backend" {
					opts := be.rateBackendOptions()
					if len(f.Options) != len(opts) {
						f.Options = opts
						f.SelIdx = len(opts) - 1
						f.Value = opts[f.SelIdx]
					}
				}
				f.CyclePrev()
			}
		}
	case "D":
		be.countBuf = ""
		be.gBuf = false
		be.secEnabled = false
		be.securityFields = defaultSecurityFields()
		be.activeField = 0
	default:
		be.countBuf = ""
		be.gBuf = false
	}
	return be, nil
}


// CloudProvider returns the selected cloud provider from the Env tab.
// Returns an empty string if the env section has not been configured.
