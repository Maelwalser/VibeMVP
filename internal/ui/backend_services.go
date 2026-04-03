package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

func (be BackendEditor) updateServiceList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.serviceEditor
	n := len(ed.items)
	switch key.String() {
	case "j", "down":
		if n > 0 && ed.itemIdx < n-1 {
			ed.itemIdx++
		}
	case "k", "up":
		if ed.itemIdx > 0 {
			ed.itemIdx--
		}
	case "a":
		svc := manifest.ServiceDef{}
		be.Services = append(be.Services, svc)
		newFields := defaultServiceFields()
		be.applyServiceDiscoveryOpts(newFields)
		ed.items = append(ed.items, newFields)
		ed.itemIdx = len(ed.items) - 1
		ed.form = copyFields(ed.items[ed.itemIdx])
		existing := make([]string, 0, len(be.Services)-1)
		for i, s := range be.Services {
			if i != ed.itemIdx {
				existing = append(existing, s.Name)
			}
		}
		ed.form = setFieldValue(ed.form, "name", uniqueName("service", existing))
		ed.formIdx = 0
		ed.itemView = beListViewForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.Services = append(be.Services[:ed.itemIdx], be.Services[ed.itemIdx+1:]...)
			ed.items = append(ed.items[:ed.itemIdx], ed.items[ed.itemIdx+1:]...)
			if ed.itemIdx > 0 && ed.itemIdx >= len(ed.items) {
				ed.itemIdx = len(ed.items) - 1
			}
		}
	case "enter":
		if n > 0 {
			ed.form = copyFields(ed.items[ed.itemIdx])
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.activeField = 0
		}
	case "l", "right":
		tabs := be.activeTabs()
		if be.activeTabIdx < len(tabs)-1 {
			be.activeTabIdx++
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
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

// isServiceFieldHidden returns true when a service form field should be hidden for the current arch.
func (be BackendEditor) isServiceFieldHidden(key string) bool {
	arch := be.currentArch()
	if arch == "monolith" && (key == "language" || key == "framework" || key == "service_discovery") {
		return true
	}
	if arch != "hybrid" && key == "pattern_tag" {
		return true
	}
	return false
}

// nextServiceFormIdx advances formIdx skipping hidden fields.
func (be BackendEditor) nextServiceFormIdx(ed *beListEditor, delta int) int {
	n := len(ed.form)
	if n == 0 {
		return 0
	}
	idx := ed.formIdx
	for i := 0; i < n; i++ {
		idx = (idx + delta + n) % n
		if !be.isServiceFieldHidden(ed.form[idx].Key) {
			return idx
		}
	}
	return ed.formIdx
}

func (be BackendEditor) updateServiceForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.serviceEditor
	switch key.String() {
	case "j", "down":
		ed.formIdx = be.nextServiceFormIdx(ed, 1)
	case "k", "up":
		ed.formIdx = be.nextServiceFormIdx(ed, -1)
	case "enter", " ":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			be.dd.Open = true
			if f.Kind == KindSelect {
				be.dd.OptIdx = f.SelIdx
			} else {
				be.dd.OptIdx = f.DDCursor
			}
		} else {
			return be.enterServiceFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
			if f.Key == "language" {
				be.updateServiceFrameworkOptions(ed)
			}
		}
	case "i", "a":
		if ed.form[ed.formIdx].CanEditAsText() {
			return be.enterServiceFormInsert()
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
		be.saveServiceForm()
		ed.itemView = beListViewList
	}
	return be, nil
}

func (be *BackendEditor) updateServiceFrameworkOptions(ed *beListEditor) {
	lang := fieldGet(ed.form, "language")
	opts, ok := backendFrameworksByLang[lang]
	if !ok {
		opts = []string{"Other"}
	}
	for i := range ed.form {
		if ed.form[i].Key == "framework" {
			ed.form[i].Options = opts
			ed.form[i].SelIdx = 0
			ed.form[i].Value = opts[0]
			break
		}
	}
}

// serviceDiscoveryByOrchestrator maps orchestrator → valid service discovery options.
var serviceDiscoveryByOrchestrator = map[string][]string{
	"K3s":            {"Kubernetes DNS", "Consul", "Static config"},
	"K8s (managed)":  {"Kubernetes DNS", "Consul", "Static config"},
	"Docker Compose": {"DNS-based", "Static config"},
	"ECS":            {"DNS-based (Cloud Map)", "Consul"},
	"Nomad":          {"Consul", "DNS-based"},
	"Cloud Run":      {"DNS-based"},
	"None":           {"Static config", "None"},
}

// applyServiceDiscoveryOpts sets the service_discovery options in a field slice
// to match the currently selected orchestrator, preserving the current value if valid.
func (be *BackendEditor) applyServiceDiscoveryOpts(fields []Field) {
	orch := fieldGet(be.EnvFields, "orchestrator")
	opts, ok := serviceDiscoveryByOrchestrator[orch]
	if !ok {
		opts = []string{"Static config", "None"}
	}
	for i := range fields {
		if fields[i].Key == "service_discovery" {
			current := fields[i].Value
			fields[i].Options = opts
			found := false
			for j, o := range opts {
				if o == current {
					fields[i].SelIdx = j
					found = true
					break
				}
			}
			if !found {
				fields[i].SelIdx = 0
				fields[i].Value = opts[0]
			}
			break
		}
	}
}

// updateServiceDiscoveryOptions refreshes the service_discovery dropdown
// in the service form and all existing service items to show only the options
// valid for the currently selected orchestrator.
func (be *BackendEditor) updateServiceDiscoveryOptions() {
	be.applyServiceDiscoveryOpts(be.serviceEditor.form)
	for _, item := range be.serviceEditor.items {
		be.applyServiceDiscoveryOpts(item)
	}
}

// orchestratorByComputeEnv maps compute_env values to the valid orchestrator options.
var orchestratorByComputeEnv = map[string][]string{
	"Bare Metal":          {"Docker Compose", "K3s", "Nomad", "None"},
	"VM":                  {"Docker Compose", "K3s", "Nomad", "None"},
	"Containers (Docker)": {"Docker Compose", "K3s", "K8s (managed)", "Nomad", "ECS", "None"},
	"Kubernetes":          {"K3s", "K8s (managed)"},
	"Serverless (FaaS)":   {"Cloud Run", "None"},
	// PaaS: orchestrator field is hidden entirely via visibleEnvFields.
}

// updateEnvOrchestratorOptions narrows the orchestrator dropdown to the options
// that are valid for the currently selected compute_env.
func (be *BackendEditor) updateEnvOrchestratorOptions() {
	computeEnv := fieldGet(be.EnvFields, "compute_env")
	opts, ok := orchestratorByComputeEnv[computeEnv]
	if !ok {
		// Unknown compute env: show all options.
		opts = []string{"Docker Compose", "K3s", "K8s (managed)", "Nomad", "ECS", "Cloud Run", "None"}
	}
	for i := range be.EnvFields {
		if be.EnvFields[i].Key != "orchestrator" {
			continue
		}
		be.EnvFields[i].Options = opts
		// Keep value if still valid; otherwise reset to first option.
		found := false
		for j, o := range opts {
			if o == be.EnvFields[i].Value {
				be.EnvFields[i].SelIdx = j
				found = true
				break
			}
		}
		if !found && len(opts) > 0 {
			be.EnvFields[i].Value = opts[0]
			be.EnvFields[i].SelIdx = 0
		}
		break
	}
}

// updateEnvMonolithOptions refreshes the monolith_fw and be_linter dropdowns
// to match the currently selected monolith_lang.
func (be *BackendEditor) updateEnvMonolithOptions() {
	lang := fieldGet(be.EnvFields, "monolith_lang")
	fwOpts, ok := backendFrameworksByLang[lang]
	if !ok {
		fwOpts = []string{"Other"}
	}
	lintOpts, ok := backendLintersByLang[lang]
	if !ok {
		lintOpts = []string{"Custom", "None"}
	}
	for i := range be.EnvFields {
		switch be.EnvFields[i].Key {
		case "monolith_fw":
			be.EnvFields[i].Options = fwOpts
			be.EnvFields[i].SelIdx = 0
			be.EnvFields[i].Value = fwOpts[0]
		case "be_linter":
			be.EnvFields[i].Options = lintOpts
			be.EnvFields[i].SelIdx = len(lintOpts) - 1
			be.EnvFields[i].Value = lintOpts[len(lintOpts)-1]
		}
	}
}

func (be BackendEditor) enterServiceFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.serviceEditor
	f := ed.form[ed.formIdx]
	if !f.CanEditAsText() {
		return be, nil
	}
	be.internalMode = ModeInsert
	be.formInput.SetValue(f.TextInputValue())
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveServiceForm() {
	ed := &be.serviceEditor
	if ed.itemIdx >= len(ed.items) {
		return
	}
	ed.items[ed.itemIdx] = copyFields(ed.form)
	svc := serviceDefFromFields(ed.form)
	if ed.itemIdx < len(be.Services) {
		be.Services[ed.itemIdx] = svc
	}
}

func (be BackendEditor) updateCommList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.commEditor
	n := len(ed.items)
	switch key.String() {
	case "j", "down":
		if n > 0 && ed.itemIdx < n-1 {
			ed.itemIdx++
		}
	case "k", "up":
		if ed.itemIdx > 0 {
			ed.itemIdx--
		}
	case "a":
		be.CommLinks = append(be.CommLinks, manifest.CommLink{})
		ed.items = append(ed.items, be.withDTONames(be.withServiceNames(defaultCommFields())))
		ed.itemIdx = len(ed.items) - 1
		ed.form = copyFields(ed.items[ed.itemIdx])
		ed.formIdx = 0
		ed.itemView = beListViewForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.CommLinks = append(be.CommLinks[:ed.itemIdx], be.CommLinks[ed.itemIdx+1:]...)
			ed.items = append(ed.items[:ed.itemIdx], ed.items[ed.itemIdx+1:]...)
			if ed.itemIdx > 0 && ed.itemIdx >= len(ed.items) {
				ed.itemIdx = len(ed.items) - 1
			}
		}
	case "enter":
		if n > 0 {
			ed.form = be.withDTONames(be.withServiceNames(copyFields(ed.items[ed.itemIdx])))
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.activeField = 0
		}
	case "l", "right":
		tabs := be.activeTabs()
		if be.activeTabIdx < len(tabs)-1 {
			be.activeTabIdx++
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "b":
		be.ArchConfirmed = false
	}
	return be, nil
}

func (be BackendEditor) updateCommForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.commEditor
	switch key.String() {
	case "j", "down":
		ed.formIdx = (ed.formIdx + 1) % len(ed.form)
	case "k", "up":
		n := len(ed.form)
		ed.formIdx = (ed.formIdx - 1 + n) % n
	case "enter", " ":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			be.dd.Open = true
			if f.Kind == KindSelect {
				be.dd.OptIdx = f.SelIdx
			} else {
				be.dd.OptIdx = f.DDCursor
			}
		} else {
			return be.enterCommFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if ed.form[ed.formIdx].CanEditAsText() {
			return be.enterCommFormInsert()
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
		be.saveCommForm()
		ed.itemView = beListViewList
	}
	return be, nil
}

func (be BackendEditor) enterCommFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.commEditor
	f := ed.form[ed.formIdx]
	if !f.CanEditAsText() {
		return be, nil
	}
	be.internalMode = ModeInsert
	be.formInput.SetValue(f.TextInputValue())
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveCommForm() {
	ed := &be.commEditor
	if ed.itemIdx >= len(ed.items) {
		return
	}
	ed.items[ed.itemIdx] = copyFields(ed.form)
	link := commLinkFromFields(ed.form)
	if ed.itemIdx < len(be.CommLinks) {
		be.CommLinks[ed.itemIdx] = link
	}
}

func (be BackendEditor) updateMessaging(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.eventEditor
	// Upper section: messaging broker config fields
	// Lower section: event catalog list
	// We use a split: first len(MessagingFields) positions are broker fields,
	// then event list items below.
	brokerCount := len(be.MessagingFields)
	eventCount := len(ed.items)
	total := brokerCount + eventCount

	switch key.String() {
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
	case "j", "down":
		if be.activeField < total-1 {
			be.activeField++
		}
	case "k", "up":
		if be.activeField > 0 {
			be.activeField--
		}
	case "enter", " ":
		if be.activeField < brokerCount {
			f := &be.MessagingFields[be.activeField]
			if f.Kind == KindSelect {
				be.dd.Open = true
				be.dd.OptIdx = f.SelIdx
			}
		} else {
			eventIdx := be.activeField - brokerCount
			if eventIdx < eventCount {
				ed.form = be.withDomainNames(copyFields(ed.items[eventIdx]))
				ed.formIdx = 0
				ed.itemIdx = eventIdx
				ed.itemView = beListViewForm
			}
		}
	case "H", "shift+left":
		if be.activeField < brokerCount {
			f := &be.MessagingFields[be.activeField]
			if f.Kind == KindSelect {
				f.CyclePrev()
			}
		}
	case "a":
		be.Events = append(be.Events, manifest.EventDef{})
		ed.items = append(ed.items, be.withDomainNames(defaultEventFields()))
		ed.itemIdx = len(ed.items) - 1
		ed.form = copyFields(ed.items[ed.itemIdx])
		existing := make([]string, 0, len(be.Events)-1)
		for i, ev := range be.Events {
			if i != ed.itemIdx {
				existing = append(existing, ev.Name)
			}
		}
		ed.form = setFieldValue(ed.form, "name", uniqueName("event", existing))
		ed.formIdx = 0
		ed.itemView = beListViewForm
		be.activeField = brokerCount + ed.itemIdx
	case "d":
		eventIdx := be.activeField - brokerCount
		if eventIdx >= 0 && eventIdx < eventCount {
			be.Events = append(be.Events[:eventIdx], be.Events[eventIdx+1:]...)
			ed.items = append(ed.items[:eventIdx], ed.items[eventIdx+1:]...)
			if be.activeField > brokerCount && be.activeField >= brokerCount+len(ed.items) {
				be.activeField = brokerCount + len(ed.items) - 1
			}
		}
	case "i":
		if be.activeField < brokerCount {
			return be.tryEnterInsert()
		}
	}
	return be, nil
}

func (be BackendEditor) updateEventForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.eventEditor
	switch key.String() {
	case "j", "down":
		ed.formIdx = (ed.formIdx + 1) % len(ed.form)
	case "k", "up":
		n := len(ed.form)
		ed.formIdx = (ed.formIdx - 1 + n) % n
	case "enter", " ":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			be.dd.Open = true
			be.dd.OptIdx = f.SelIdx
		} else {
			return be.enterEventFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if ed.form[ed.formIdx].CanEditAsText() {
			return be.enterEventFormInsert()
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
		be.saveEventForm()
		ed.itemView = beListViewList
	}
	return be, nil
}

func (be BackendEditor) enterEventFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.eventEditor
	f := ed.form[ed.formIdx]
	if !f.CanEditAsText() {
		return be, nil
	}
	be.internalMode = ModeInsert
	be.formInput.SetValue(f.TextInputValue())
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveEventForm() {
	ed := &be.eventEditor
	if ed.itemIdx >= len(ed.items) {
		return
	}
	ed.items[ed.itemIdx] = copyFields(ed.form)
	evt := manifest.EventDef{
		Name:        fieldGet(ed.form, "name"),
		Domain:      fieldGet(ed.form, "domain"),
		Description: fieldGet(ed.form, "description"),
	}
	if ed.itemIdx < len(be.Events) {
		be.Events[ed.itemIdx] = evt
	}
}

// visibleEnvFields returns the ENV fields filtered by the current arch and
// compute environment.
// - monolith_lang / monolith_fw only shown for monolith arch.
// - cors_origins only shown when cors_strategy is "Strict allowlist".
// - orchestrator hidden when compute_env is "PaaS" (irrelevant — no orchestration layer).
