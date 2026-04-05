package ui

import "fmt"

// viewRepoEditor renders the repository list or form for the current service.
func (be BackendEditor) viewRepoEditor(w int) []string {
	ed := be.repoEditor
	svcIdx := be.serviceEditor.itemIdx
	svcName := ""
	if svcIdx < len(be.Services) {
		svcName = be.Services[svcIdx].Name
	}
	if svcName == "" {
		svcName = fmt.Sprintf("service #%d", svcIdx+1)
	}

	if ed.itemView == beListViewList {
		var lines []string
		lines = append(lines,
			StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(svcName)+
				StyleSectionDesc.Render(" · Data Access — a: add  d: delete  Enter: edit"),
			"",
		)
		repos := be.currentServiceRepos()
		if len(repos) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no repositories yet — press 'a' to add)"))
		} else {
			for i, r := range repos {
				name := r.Name
				if name == "" {
					name = fmt.Sprintf("(repo #%d)", i+1)
				}
				// Extra: entity_ref + op count
				extra := r.EntityRef
				if nOps := len(r.Operations); nOps > 0 {
					opStr := "1 op"
					if nOps > 1 {
						opStr = fmt.Sprintf("%d ops", nOps)
					}
					if extra != "" {
						extra = extra + "  " + opStr
					} else {
						extra = opStr
					}
				}
				lines = append(lines, renderListItem(w, i == ed.itemIdx, "  ▶ ", name, extra))
			}
		}
		return lines
	}

	// Form view
	repoName := fieldGet(ed.form, "name")
	if repoName == "" {
		repoName = "(new repository)"
	}

	// Count ops for this repo.
	svcRepos := be.currentServiceRepos()
	opCountHint := ""
	if ed.itemIdx < len(svcRepos) {
		nOps := len(svcRepos[ed.itemIdx].Operations)
		if nOps == 0 {
			opCountHint = "  no operations — press O to add"
		} else if nOps == 1 {
			opCountHint = "  1 operation — press O to manage"
		} else {
			opCountHint = fmt.Sprintf("  %d operations — press O to manage", nOps)
		}
	}

	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(repoName), "")
	lines = append(lines, StyleSectionDesc.Render("  (O: operations"+opCountHint+")"), "")
	lines = append(lines, renderFormFields(w, ed.form, ed.formIdx, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
	return lines
}

// viewOpEditor renders the operation list or form for the current repository.
func (be BackendEditor) viewOpEditor(w int) []string {
	ed := be.opEditor

	repoName := ""
	if be.repoEditor.itemIdx < len(be.repoEditor.items) {
		repoName = fieldGet(be.repoEditor.items[be.repoEditor.itemIdx], "name")
	}
	if repoName == "" {
		repoName = "(repository)"
	}

	if ed.itemView == beListViewList {
		var lines []string
		lines = append(lines,
			StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(repoName)+
				StyleSectionDesc.Render(" · Operations — a: add  d: delete  Enter: edit"),
			"",
		)
		svcIdx := be.serviceEditor.itemIdx
		repoIdx := be.repoEditor.itemIdx
		if svcIdx < len(be.Services) && repoIdx < len(be.Services[svcIdx].Repositories) {
			ops := be.Services[svcIdx].Repositories[repoIdx].Operations
			if len(ops) == 0 {
				lines = append(lines, StyleSectionDesc.Render("  (no operations yet — press 'a' to add)"))
			} else {
				for i, op := range ops {
					name := op.Name
					if name == "" {
						name = fmt.Sprintf("(op #%d)", i+1)
					}
					extra := op.OpType
					if len(op.FilterBy) > 0 {
						filterStr := "by " + joinMax(op.FilterBy, ", ", 3)
						if extra != "" {
							extra = extra + "  " + filterStr
						} else {
							extra = filterStr
						}
					}
					lines = append(lines, renderListItem(w, i == ed.itemIdx, "  ▶ ", name, extra))
				}
			}
		} else {
			lines = append(lines, StyleSectionDesc.Render("  (no operations yet — press 'a' to add)"))
		}
		return lines
	}

	// Form view
	opName := fieldGet(ed.form, "name")
	if opName == "" {
		opName = "(new operation)"
	}
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(opName), "")
	lines = append(lines, renderFormFields(w, ed.form, ed.formIdx, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
	return lines
}

// joinMax joins at most n items from s with sep; if more exist, appends "…".
func joinMax(s []string, sep string, n int) string {
	if len(s) <= n {
		result := ""
		for i, v := range s {
			if i > 0 {
				result += sep
			}
			result += v
		}
		return result
	}
	result := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			result += sep
		}
		result += s[i]
	}
	return result + "…"
}
