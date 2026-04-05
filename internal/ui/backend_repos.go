package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── op_type options by DB technology ─────────────────────────────────────────

func opTypesForDBType(dbType string) []string {
	switch dbType {
	case "PostgreSQL", "MySQL", "SQLite":
		return []string{
			"read-one", "read-all", "read-page",
			"create", "update", "delete",
			"count", "exists", "aggregate", "raw-query",
		}
	case "MongoDB":
		return []string{
			"find-one", "find-many", "find-page",
			"insert-one", "insert-many",
			"update-one", "update-many",
			"delete-one", "delete-many",
			"count", "aggregate",
		}
	case "DynamoDB":
		return []string{
			"get-item", "query", "scan",
			"put-item", "update-item", "delete-item",
			"batch-get", "batch-write", "count",
		}
	default: // Redis, other, unknown
		return []string{"read", "read-all", "create", "update", "delete", "count", "aggregate"}
	}
}

var resultShapeOptions = []string{"Single item", "List", "Count", "Boolean", "Void"}
var paginationOptions = []string{"None", "Offset/Limit", "Cursor-based", "Page-based"}

// ── field constructors ────────────────────────────────────────────────────────

func defaultRepoFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{
			Key:     "entity_ref",
			Label:   "entity_ref    ",
			Kind:    KindSelect,
			Options: []string{"(no domains configured)"},
			Value:   "(no domains configured)",
		},
		{
			Key:     "fields",
			Label:   "fields        ",
			Kind:    KindMultiSelect,
			Options: []string{},
		},
		{
			Key:     "target_db",
			Label:   "target_db     ",
			Kind:    KindSelect,
			Options: []string{"(no databases configured)"},
			Value:   "(no databases configured)",
		},
	}
}

func defaultOpFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{
			Key:     "op_type",
			Label:   "op_type       ",
			Kind:    KindSelect,
			Options: []string{"read", "read-all", "create", "update", "delete", "count", "aggregate"},
			Value:   "read",
		},
		{Key: "filter_by", Label: "filter_by     ", Kind: KindMultiSelect},
		{
			Key:     "sort_by",
			Label:   "sort_by       ",
			Kind:    KindSelect,
			Options: []string{"(none)"},
			Value:   "(none)",
		},
		{
			Key:     "result_shape",
			Label:   "result_shape  ",
			Kind:    KindSelect,
			Options: resultShapeOptions,
			Value:   "Single item",
		},
		{
			Key:     "pagination",
			Label:   "pagination    ",
			Kind:    KindSelect,
			Options: paginationOptions,
			Value:   "None",
		},
		{Key: "query_hint", Label: "query_hint    ", Kind: KindText},
		{Key: "description", Label: "description   ", Kind: KindText},
	}
}

// ── manifest converters ───────────────────────────────────────────────────────

func repoDefFromFields(fields []Field) manifest.RepositoryDef {
	entityRef := fieldGet(fields, "entity_ref")
	if entityRef == "(no domains configured)" || entityRef == "(none)" {
		entityRef = ""
	}
	targetDB := fieldGet(fields, "target_db")
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
		Name:           fieldGet(fields, "name"),
		EntityRef:      entityRef,
		TargetDB:       targetDB,
		SelectedFields: selectedFields,
	}
}

func repoFieldsFromDef(r manifest.RepositoryDef) []Field {
	f := defaultRepoFields()
	f = setFieldValue(f, "name", r.Name)
	if r.EntityRef != "" {
		f = setFieldValue(f, "entity_ref", r.EntityRef)
	}
	if r.TargetDB != "" {
		f = setFieldValue(f, "target_db", r.TargetDB)
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

func opDefFromFields(fields []Field) manifest.DataAccessOp {
	var filterBy []string
	for _, f := range fields {
		if f.Key == "filter_by" {
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					filterBy = append(filterBy, f.Options[idx])
				}
			}
			break
		}
	}
	sortBy := fieldGet(fields, "sort_by")
	if sortBy == "(none)" {
		sortBy = ""
	}
	pagination := fieldGet(fields, "pagination")
	if pagination == "None" {
		pagination = ""
	}
	return manifest.DataAccessOp{
		Name:        fieldGet(fields, "name"),
		OpType:      fieldGet(fields, "op_type"),
		FilterBy:    filterBy,
		SortBy:      sortBy,
		ResultShape: fieldGet(fields, "result_shape"),
		Pagination:  pagination,
		QueryHint:   fieldGet(fields, "query_hint"),
		Description: fieldGet(fields, "description"),
	}
}

func opFieldsFromDef(op manifest.DataAccessOp) []Field {
	f := defaultOpFields()
	f = setFieldValue(f, "name", op.Name)
	if op.OpType != "" {
		f = setFieldValue(f, "op_type", op.OpType)
	}
	// filter_by: store as Value for lazy restoration via withOpRefs.
	if len(op.FilterBy) > 0 {
		for i := range f {
			if f[i].Key == "filter_by" {
				f[i].Value = strings.Join(op.FilterBy, ", ")
				break
			}
		}
	}
	if op.SortBy != "" {
		f = setFieldValue(f, "sort_by", op.SortBy)
	}
	if op.ResultShape != "" {
		f = setFieldValue(f, "result_shape", op.ResultShape)
	}
	if op.Pagination != "" {
		f = setFieldValue(f, "pagination", op.Pagination)
	}
	f = setFieldValue(f, "query_hint", op.QueryHint)
	f = setFieldValue(f, "description", op.Description)
	return f
}

// ── reference injection helpers ───────────────────────────────────────────────

// withRepoRefs injects target_db, entity_ref (filtered by target_db), and
// fields options based on the BackendEditor's current domain and DB source references.
func (be *BackendEditor) withRepoRefs(fields []Field) []Field {
	f := copyFields(fields)

	// target_db: DB source aliases — resolve first so entity_ref can be filtered.
	dbOpts, _ := noneOrPlaceholder(be.dbSourceAliases, "(no databases configured)")
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
		f[i].Kind = KindSelect
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
	entityOpts, _ := noneOrPlaceholder(domainCandidates, "(no domains configured)")
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
		f[i].Kind = KindSelect
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
func (be *BackendEditor) applyDomainFieldsToRepo(fields []Field, entityRef string) {
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
	entityRef := fieldGet(ed.form, "entity_ref")
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
	targetDB := fieldGet(ed.form, "target_db")
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

	opts, _ := noneOrPlaceholder(candidates, "(no domains configured)")
	curEntity := fieldGet(ed.form, "entity_ref")

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

// withOpRefs injects filter_by and sort_by options (from the repo's selected
// fields) and op_type options (based on target_db technology) into an op field
// slice.
func (be *BackendEditor) withOpRefs(fields []Field) []Field {
	f := copyFields(fields)
	// Resolve the parent repo to get selected fields and target_db.
	svcIdx := be.serviceEditor.itemIdx
	repoIdx := be.repoEditor.itemIdx
	var repoFields []string
	var dbType string
	if svcIdx < len(be.Services) && repoIdx < len(be.Services[svcIdx].Repositories) {
		repo := be.Services[svcIdx].Repositories[repoIdx]
		if len(repo.SelectedFields) > 0 {
			repoFields = repo.SelectedFields
		} else if repo.EntityRef != "" {
			repoFields = be.domainAttributes[repo.EntityRef]
		}
		if repo.TargetDB != "" {
			dbType = be.dbSourceTypes[repo.TargetDB]
		}
	}

	// op_type: technology-aware options
	opTypes := opTypesForDBType(dbType)
	for i := range f {
		if f[i].Key != "op_type" {
			continue
		}
		cur := f[i].Value
		f[i].Options = opTypes
		found := false
		for j, o := range opTypes {
			if o == cur {
				f[i].SelIdx = j
				f[i].Value = o
				found = true
				break
			}
		}
		if !found {
			f[i].SelIdx = 0
			f[i].Value = opTypes[0]
		}
		break
	}

	// filter_by and sort_by: from repo's selected fields.
	// When no fields are available, show a descriptive placeholder.
	const entityPlaceholder = "(configure entity_ref in repo first)"
	noFields := len(repoFields) == 0
	sortOpts := append([]string{"(none)"}, repoFields...)
	if noFields {
		sortOpts = []string{entityPlaceholder}
	}
	for i := range f {
		switch f[i].Key {
		case "filter_by":
			if noFields {
				f[i].Options = []string{entityPlaceholder}
				f[i].SelectedIdxs = nil
				f[i].Value = ""
				break
			}
			// Restore previously selected filter fields by name.
			var selectedNames []string
			if len(f[i].SelectedIdxs) > 0 && len(f[i].Options) > 0 {
				for _, idx := range f[i].SelectedIdxs {
					if idx < len(f[i].Options) {
						selectedNames = append(selectedNames, f[i].Options[idx])
					}
				}
			} else if f[i].Value != "" {
				for _, name := range strings.Split(f[i].Value, ", ") {
					name = strings.TrimSpace(name)
					if name != "" {
						selectedNames = append(selectedNames, name)
					}
				}
			}
			f[i].Options = repoFields
			f[i].SelectedIdxs = nil
			f[i].Value = ""
			for _, name := range selectedNames {
				for j, opt := range repoFields {
					if opt == name {
						f[i].SelectedIdxs = append(f[i].SelectedIdxs, j)
						break
					}
				}
			}
		case "sort_by":
			cur := f[i].Value
			f[i].Options = sortOpts
			if noFields {
				f[i].SelIdx = 0
				f[i].Value = entityPlaceholder
				break
			}
			found := false
			for j, o := range sortOpts {
				if o == cur {
					f[i].SelIdx = j
					f[i].Value = o
					found = true
					break
				}
			}
			if !found {
				f[i].SelIdx = 0
				f[i].Value = "(none)"
			}
		}
	}
	return f
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
	be.repoEditor.items = make([][]Field, len(repos))
	for i, r := range repos {
		be.repoEditor.items[i] = repoFieldsFromDef(r)
	}
}

// loadOpEditorItems populates opEditor.items from the current repo's operations.
func (be *BackendEditor) loadOpEditorItems() {
	svcIdx := be.serviceEditor.itemIdx
	repoIdx := be.repoEditor.itemIdx
	if svcIdx >= len(be.Services) || repoIdx >= len(be.Services[svcIdx].Repositories) {
		be.opEditor.items = nil
		return
	}
	ops := be.Services[svcIdx].Repositories[repoIdx].Operations
	be.opEditor.items = make([][]Field, len(ops))
	for i, op := range ops {
		be.opEditor.items[i] = opFieldsFromDef(op)
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
	ed.items[ed.itemIdx] = copyFields(ed.form)
	repo := repoDefFromFields(ed.form)
	repos := be.Services[svcIdx].Repositories
	if ed.itemIdx < len(repos) {
		repo.Operations = repos[ed.itemIdx].Operations // preserve ops
		be.Services[svcIdx].Repositories[ed.itemIdx] = repo
	}
}

func (be *BackendEditor) saveOpForm() {
	ed := &be.opEditor
	svcIdx := be.serviceEditor.itemIdx
	repoIdx := be.repoEditor.itemIdx
	if ed.itemIdx >= len(ed.items) || svcIdx >= len(be.Services) {
		return
	}
	if repoIdx >= len(be.Services[svcIdx].Repositories) {
		return
	}
	ed.items[ed.itemIdx] = copyFields(ed.form)
	op := opDefFromFields(ed.form)
	ops := be.Services[svcIdx].Repositories[repoIdx].Operations
	if ed.itemIdx < len(ops) {
		be.Services[svcIdx].Repositories[repoIdx].Operations[ed.itemIdx] = op
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
			newRepo := manifest.RepositoryDef{Name: uniqueName("repository", existing)}
			be.Services[svcIdx].Repositories = append(be.Services[svcIdx].Repositories, newRepo)
			newFields := be.withRepoRefs(defaultRepoFields())
			newFields = setFieldValue(newFields, "name", newRepo.Name)
			ed.items = append(ed.items, newFields)
			ed.itemIdx = len(ed.items) - 1
			ed.form = copyFields(ed.items[ed.itemIdx])
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
			ed.form = be.withRepoRefs(copyFields(ed.items[ed.itemIdx]))
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
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			be.dd.Open = true
			if f.Kind == KindSelect {
				be.dd.OptIdx = f.SelIdx
			} else {
				be.dd.OptIdx = f.DDCursor
			}
		} else {
			return be.enterRepoFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
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
	be.internalMode = ModeInsert
	be.formInput.SetValue(f.TextInputValue())
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

// ── op list update ────────────────────────────────────────────────────────────

func (be BackendEditor) updateOpList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.opEditor
	svcIdx := be.serviceEditor.itemIdx
	repoIdx := be.repoEditor.itemIdx
	n := len(ed.items)

	validRepo := svcIdx < len(be.Services) && repoIdx < len(be.Services[svcIdx].Repositories)

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
		if validRepo {
			existing := make([]string, 0)
			for _, op := range be.Services[svcIdx].Repositories[repoIdx].Operations {
				existing = append(existing, op.Name)
			}
			newOp := manifest.DataAccessOp{Name: uniqueName("op", existing), OpType: "read"}
			be.Services[svcIdx].Repositories[repoIdx].Operations = append(
				be.Services[svcIdx].Repositories[repoIdx].Operations, newOp)
			newFields := be.withOpRefs(defaultOpFields())
			newFields = setFieldValue(newFields, "name", newOp.Name)
			ed.items = append(ed.items, newFields)
			ed.itemIdx = len(ed.items) - 1
			ed.form = copyFields(ed.items[ed.itemIdx])
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.repoSubView = beRepoSubViewOpForm
		}
	case "d":
		if n > 0 && validRepo {
			ops := be.Services[svcIdx].Repositories[repoIdx].Operations
			be.Services[svcIdx].Repositories[repoIdx].Operations = append(ops[:ed.itemIdx], ops[ed.itemIdx+1:]...)
			ed.items = append(ed.items[:ed.itemIdx], ed.items[ed.itemIdx+1:]...)
			if ed.itemIdx > 0 && ed.itemIdx >= len(ed.items) {
				ed.itemIdx = len(ed.items) - 1
			}
		}
	case "enter":
		if n > 0 {
			ed.form = be.withOpRefs(copyFields(ed.items[ed.itemIdx]))
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.repoSubView = beRepoSubViewOpForm
		}
	case "b", "esc":
		ed.itemView = beListViewList
		be.repoEditor.itemView = beListViewForm
		be.repoSubView = beRepoSubViewForm
	}
	return be, nil
}

// ── op form update ────────────────────────────────────────────────────────────

func (be BackendEditor) updateOpForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.opEditor
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
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			be.dd.Open = true
			if f.Kind == KindSelect {
				be.dd.OptIdx = f.SelIdx
			} else {
				be.dd.OptIdx = f.DDCursor
			}
		} else {
			return be.enterOpFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if ed.form[ed.formIdx].CanEditAsText() {
			return be.enterOpFormInsert()
		}
	case "b", "esc":
		be.saveOpForm()
		ed.itemView = beListViewList
		be.repoSubView = beRepoSubViewOpList
	}
	be.saveOpForm()
	return be, nil
}

func (be BackendEditor) enterOpFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.opEditor
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
