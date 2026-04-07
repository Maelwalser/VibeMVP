package backend

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── field constructors ────────────────────────────────────────────────────────

func defaultRepoFields() []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{
			Key:     "entity_ref",
			Label:   "entity_ref    ",
			Kind:    core.KindSelect,
			Options: []string{"(no domains configured)"},
			Value:   "(no domains configured)",
		},
		{
			Key:     "fields",
			Label:   "fields        ",
			Kind:    core.KindMultiSelect,
			Options: []string{},
		},
		{
			Key:     "target_db",
			Label:   "target_db     ",
			Kind:    core.KindSelect,
			Options: []string{"(no databases configured)"},
			Value:   "(no databases configured)",
		},
	}
}

// ── manifest converters ───────────────────────────────────────────────────────

func repoDefFromFields(fields []core.Field) manifest.RepositoryDef {
	entityRef := core.FieldGet(fields, "entity_ref")
	if entityRef == "(no domains configured)" || entityRef == "(none)" {
		entityRef = ""
	}
	targetDB := core.FieldGet(fields, "target_db")
	if targetDB == "(no databases configured)" || targetDB == "(none)" {
		targetDB = ""
	}
	var selectedFields []string
	for _, f := range fields {
		if f.Key == "fields" {
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					selectedFields = append(selectedFields, f.Options[idx])
				}
			}
			break
		}
	}
	return manifest.RepositoryDef{
		Name:           core.FieldGet(fields, "name"),
		EntityRef:      entityRef,
		TargetDB:       targetDB,
		SelectedFields: selectedFields,
	}
}

func repoFieldsFromDef(r manifest.RepositoryDef) []core.Field {
	f := defaultRepoFields()
	f = core.SetFieldValue(f, "name", r.Name)
	if r.EntityRef != "" {
		f = core.SetFieldValue(f, "entity_ref", r.EntityRef)
	}
	if r.TargetDB != "" {
		f = core.SetFieldValue(f, "target_db", r.TargetDB)
	}
	// Store selected field names in Value for lazy restoration via withRepoRefs.
	if len(r.SelectedFields) > 0 {
		for i := range f {
			if f[i].Key == "fields" {
				f[i].Value = strings.Join(r.SelectedFields, ", ")
				break
			}
		}
	}
	return f
}

// ── reference injection helpers ───────────────────────────────────────────────

// withRepoRefs injects target_db, entity_ref (filtered by target_db), and
// fields options based on the BackendEditor's current domain and DB source references.
func (be *BackendEditor) withRepoRefs(fields []core.Field) []core.Field {
	f := core.CopyFields(fields)

	// target_db: DB source aliases — resolve first so entity_ref can be filtered.
	dbOpts, _ := core.NoneOrPlaceholder(be.dbSourceAliases, "(no databases configured)")
	for i := range f {
		if f[i].Key != "target_db" {
			continue
		}
		cur := f[i].Value
		if len(dbOpts) == 0 {
			f[i].Options = []string{"(no databases configured)"}
			f[i].Value = "(no databases configured)"
			f[i].SelIdx = 0
			break
		}
		f[i].Kind = core.KindSelect
		f[i].Options = dbOpts
		found := false
		for j, o := range dbOpts {
			if o == cur {
				f[i].SelIdx = j
				f[i].Value = o
				found = true
				break
			}
		}
		if !found {
			f[i].SelIdx = 0
			f[i].Value = dbOpts[0]
		}
		break
	}

	// Determine the resolved target_db value to filter entity_ref options.
	targetDB := ""
	for _, fld := range f {
		if fld.Key == "target_db" {
			targetDB = fld.Value
			break
		}
	}
	if targetDB == "(no databases configured)" || targetDB == "(none)" {
		targetDB = ""
	}

	// entity_ref: filter to domains linked to the selected target_db, or all if
	// no target_db is selected yet.
	var domainCandidates []string
	if targetDB != "" {
		if linked, ok := be.domainsByDB[targetDB]; ok && len(linked) > 0 {
			domainCandidates = linked
		}
	}
	if len(domainCandidates) == 0 {
		domainCandidates = be.DomainNames
	}
	entityOpts, _ := core.NoneOrPlaceholder(domainCandidates, "(no domains configured)")
	for i := range f {
		if f[i].Key != "entity_ref" {
			continue
		}
		cur := f[i].Value
		if len(entityOpts) == 0 {
			f[i].Options = []string{"(no domains configured)"}
			f[i].Value = "(no domains configured)"
			f[i].SelIdx = 0
			break
		}
		f[i].Kind = core.KindSelect
		f[i].Options = entityOpts
		// Try to restore current selection.
		found := false
		for j, o := range entityOpts {
			if o == cur {
				f[i].SelIdx = j
				f[i].Value = o
				found = true
				break
			}
		}
		if !found {
			f[i].SelIdx = 0
			f[i].Value = entityOpts[0]
		}
		break
	}

	// Resolve which entity is now selected and populate the fields multiselect.
	entityRef := ""
	for _, fld := range f {
		if fld.Key == "entity_ref" {
			entityRef = fld.Value
			break
		}
	}
	if entityRef == "(none)" || entityRef == "(no domains configured)" {
		entityRef = ""
	}
	be.applyDomainFieldsToRepo(f, entityRef)

	return f
}

// applyDomainFieldsToRepo populates the "fields" multiselect in a repo field
// slice from the domain attributes of entityRef. When entityRef changes the
// selection is reset to all attributes selected by default. If the field
// already has a Value with comma-separated names (lazy restore), those are
// used instead.
func (be *BackendEditor) applyDomainFieldsToRepo(fields []core.Field, entityRef string) {
	attrs := be.domainAttributes[entityRef]
	for i := range fields {
		if fields[i].Key != "fields" {
			continue
		}
		if len(attrs) == 0 {
			fields[i].Options = []string{}
			fields[i].SelectedIdxs = nil
			break
		}
		fields[i].Options = attrs
		// Collect currently selected names for restoration.
		var selectedNames []string
		if len(fields[i].SelectedIdxs) > 0 && len(fields[i].Options) > 0 {
			// Already have selections in the current options — try to preserve.
			for _, idx := range fields[i].SelectedIdxs {
				if idx < len(fields[i].Options) {
					selectedNames = append(selectedNames, fields[i].Options[idx])
				}
			}
		} else if fields[i].Value != "" {
			// Lazy restore from manifest (comma-sep names stored in Value).
			for _, name := range strings.Split(fields[i].Value, ", ") {
				name = strings.TrimSpace(name)
				if name != "" {
					selectedNames = append(selectedNames, name)
				}
			}
		}
		fields[i].Value = ""
		fields[i].SelectedIdxs = nil
		if len(selectedNames) > 0 {
			// Restore the preserved selection.
			for _, name := range selectedNames {
				for j, opt := range attrs {
					if opt == name {
						fields[i].SelectedIdxs = append(fields[i].SelectedIdxs, j)
						break
					}
				}
			}
		} else {
			// Default: select all attributes.
			for j := range attrs {
				fields[i].SelectedIdxs = append(fields[i].SelectedIdxs, j)
			}
		}
		break
	}
}

// refreshRepoFieldOptions refreshes the "fields" multiselect after entity_ref
// changes, selecting all attributes by default (preserving any prior selection
// only if the entity_ref did not change).
func (be *BackendEditor) refreshRepoFieldOptions() {
	ed := &be.repoEditor
	entityRef := core.FieldGet(ed.form, "entity_ref")
	if entityRef == "(none)" || entityRef == "(no domains configured)" {
		entityRef = ""
	}
	attrs := be.domainAttributes[entityRef]
	for i := range ed.form {
		if ed.form[i].Key != "fields" {
			continue
		}
		ed.form[i].Options = attrs
		ed.form[i].Value = ""
		ed.form[i].SelectedIdxs = nil
		// Select all by default when entity changes.
		for j := range attrs {
			ed.form[i].SelectedIdxs = append(ed.form[i].SelectedIdxs, j)
		}
		break
	}
}

// refreshRepoEntityRefOptions refreshes the "entity_ref" options in the repo form
// after "target_db" changes, filtering to only domains linked to the new DB.
// If no DB is selected, falls back to all domain names.
func (be *BackendEditor) refreshRepoEntityRefOptions() {
	ed := &be.repoEditor
	targetDB := core.FieldGet(ed.form, "target_db")
	if targetDB == "(no databases configured)" || targetDB == "(none)" {
		targetDB = ""
	}

	var candidates []string
	if targetDB != "" {
		if linked, ok := be.domainsByDB[targetDB]; ok && len(linked) > 0 {
			candidates = linked
		}
	}
	if len(candidates) == 0 {
		candidates = be.DomainNames
	}

	opts, _ := core.NoneOrPlaceholder(candidates, "(no domains configured)")
	curEntity := core.FieldGet(ed.form, "entity_ref")

	for i := range ed.form {
		if ed.form[i].Key != "entity_ref" {
			continue
		}
		ed.form[i].Options = opts
		// Try to keep the current selection if still valid.
		found := false
		for j, o := range opts {
			if o == curEntity {
				ed.form[i].SelIdx = j
				ed.form[i].Value = o
				found = true
				break
			}
		}
		if !found {
			ed.form[i].SelIdx = 0
			if len(opts) > 0 {
				ed.form[i].Value = opts[0]
			}
			// Also reset the fields multiselect since the entity changed.
			be.refreshRepoFieldOptions()
		}
		break
	}
}

// ── data loading helpers ──────────────────────────────────────────────────────

// loadRepoEditorItems populates repoEditor.items from the current service's repositories.
func (be *BackendEditor) loadRepoEditorItems() {
	svcIdx := be.serviceEditor.itemIdx
	if svcIdx >= len(be.Services) {
		be.repoEditor.items = nil
		return
	}
	repos := be.Services[svcIdx].Repositories
	be.repoEditor.items = make([][]core.Field, len(repos))
	for i, r := range repos {
		be.repoEditor.items[i] = repoFieldsFromDef(r)
	}
}

// currentServiceRepos returns the repositories for the currently selected service.
func (be BackendEditor) currentServiceRepos() []manifest.RepositoryDef {
	svcIdx := be.serviceEditor.itemIdx
	if svcIdx >= len(be.Services) {
		return nil
	}
	return be.Services[svcIdx].Repositories
}

// ── save helpers ──────────────────────────────────────────────────────────────

func (be *BackendEditor) saveRepoForm() {
	ed := &be.repoEditor
	svcIdx := be.serviceEditor.itemIdx
	if ed.itemIdx >= len(ed.items) || svcIdx >= len(be.Services) {
		return
	}
	ed.items[ed.itemIdx] = core.CopyFields(ed.form)
	repo := repoDefFromFields(ed.form)
	repos := be.Services[svcIdx].Repositories
	if ed.itemIdx < len(repos) {
		repo.Operations = repos[ed.itemIdx].Operations // preserve ops
		be.Services[svcIdx].Repositories[ed.itemIdx] = repo
	}
}

// ── repo list update ──────────────────────────────────────────────────────────

func (be BackendEditor) updateRepoList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.repoEditor
	svcIdx := be.serviceEditor.itemIdx
	repos := be.currentServiceRepos()
	n := len(repos)
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
		if svcIdx < len(be.Services) {
			existing := make([]string, 0, len(be.Services[svcIdx].Repositories))
			for _, r := range be.Services[svcIdx].Repositories {
				existing = append(existing, r.Name)
			}
			newRepo := manifest.RepositoryDef{Name: core.UniqueName("repository", existing)}
			be.Services[svcIdx].Repositories = append(be.Services[svcIdx].Repositories, newRepo)
			newFields := be.withRepoRefs(defaultRepoFields())
			newFields = core.SetFieldValue(newFields, "name", newRepo.Name)
			ed.items = append(ed.items, newFields)
			ed.itemIdx = len(ed.items) - 1
			ed.form = core.CopyFields(ed.items[ed.itemIdx])
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.repoSubView = beRepoSubViewForm
		}
	case "d":
		if n > 0 && svcIdx < len(be.Services) {
			r := be.Services[svcIdx].Repositories
			be.Services[svcIdx].Repositories = append(r[:ed.itemIdx], r[ed.itemIdx+1:]...)
			ed.items = append(ed.items[:ed.itemIdx], ed.items[ed.itemIdx+1:]...)
			if ed.itemIdx > 0 && ed.itemIdx >= len(ed.items) {
				ed.itemIdx = len(ed.items) - 1
			}
		}
	case "enter":
		if n > 0 {
			ed.form = be.withRepoRefs(core.CopyFields(ed.items[ed.itemIdx]))
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.repoSubView = beRepoSubViewForm
		}
	case "b", "esc":
		be.repoSubView = beRepoSubViewNone
		ed.itemView = beListViewList
	}
	return be, nil
}

// ── repo form update ──────────────────────────────────────────────────────────

func (be BackendEditor) updateRepoForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.repoEditor
	switch key.String() {
	case "j", "down":
		if ed.formIdx < len(ed.form)-1 {
			ed.formIdx++
		}
	case "k", "up":
		if ed.formIdx > 0 {
			ed.formIdx--
		}
	case "enter", " ":
		f := &ed.form[ed.formIdx]
		if (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
			be.dd.Open = true
			if f.Kind == core.KindSelect {
				be.dd.OptIdx = f.SelIdx
			} else {
				be.dd.OptIdx = f.DDCursor
			}
		} else {
			return be.enterRepoFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
			if f.Key == "entity_ref" {
				be.refreshRepoFieldOptions()
			} else if f.Key == "target_db" {
				be.refreshRepoEntityRefOptions()
			}
		}
	case "i", "a":
		if ed.form[ed.formIdx].CanEditAsText() {
			return be.enterRepoFormInsert()
		}
	case "O":
		// Drill into operations for this repo.
		be.saveRepoForm()
		be.loadOpEditorItems()
		be.opEditor.itemIdx = 0
		be.opEditor.itemView = beListViewList
		be.repoSubView = beRepoSubViewOpList
	case "b", "esc":
		be.saveRepoForm()
		ed.itemView = beListViewList
		be.repoSubView = beRepoSubViewList
	}
	be.saveRepoForm()
	return be, nil
}

func (be BackendEditor) enterRepoFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.repoEditor
	f := ed.form[ed.formIdx]
	if !f.CanEditAsText() {
		return be, nil
	}
	be.internalMode = core.ModeInsert
	be.formInput.SetValue(f.TextInputValue())
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}
