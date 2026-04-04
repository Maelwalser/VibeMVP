package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── Jobs list update ──────────────────────────────────────────────────────────

func (be BackendEditor) updateJobsList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.jobQueues)
	switch key.String() {
	case "j", "down":
		if n > 0 && be.jobsIdx < n-1 {
			be.jobsIdx++
		}
	case "k", "up":
		if be.jobsIdx > 0 {
			be.jobsIdx--
		}
	case "a":
		be.jobQueues = append(be.jobQueues, manifest.JobQueueDef{})
		be.jobsIdx = len(be.jobQueues) - 1
		be.jobsForm = defaultJobQueueFormFields(be.ServiceNames(), be.availableDTOs, be.Languages(), be.stackConfigNames())
		existing := make([]string, 0, len(be.jobQueues)-1)
		for i, jq := range be.jobQueues {
			if i != be.jobsIdx {
				existing = append(existing, jq.Name)
			}
		}
		be.jobsForm = setFieldValue(be.jobsForm, "name", uniqueName("queue", existing))
		be.jobsFormIdx = 0
		be.jobsSubView = beViewForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.jobQueues = append(be.jobQueues[:be.jobsIdx], be.jobQueues[be.jobsIdx+1:]...)
			if be.jobsIdx > 0 && be.jobsIdx >= len(be.jobQueues) {
				be.jobsIdx = len(be.jobQueues) - 1
			}
		}
	case "enter":
		if n > 0 {
			jq := be.jobQueues[be.jobsIdx]
			// Determine effective language for technology filtering.
			lang := be.langForConfig(jq.ConfigRef)
			var langs []string
			if lang != "" {
				langs = []string{lang}
			} else {
				langs = be.Languages()
			}
			be.jobsForm = defaultJobQueueFormFields(be.ServiceNames(), be.availableDTOs, langs, be.stackConfigNames())
			be.jobsForm = setFieldValue(be.jobsForm, "name", jq.Name)
			be.jobsForm = setFieldValue(be.jobsForm, "description", jq.Description)
			if jq.ConfigRef != "" {
				be.jobsForm = setFieldValue(be.jobsForm, "config_ref", jq.ConfigRef)
			}
			be.jobsForm = setFieldValue(be.jobsForm, "technology", jq.Technology)
			be.jobsForm = setFieldValue(be.jobsForm, "concurrency", jq.Concurrency)
			be.jobsForm = setFieldValue(be.jobsForm, "max_retries", jq.MaxRetries)
			be.jobsForm = setFieldValue(be.jobsForm, "retry_policy", jq.RetryPolicy)
			be.jobsForm = setFieldValue(be.jobsForm, "dlq", jq.DLQ)
			be.jobsForm = setFieldValue(be.jobsForm, "worker_service", jq.WorkerService)
			be.jobsForm = setFieldValue(be.jobsForm, "payload_dto", jq.PayloadDTO)
			be.jobsFormIdx = 0
			be.jobsSubView = beViewForm
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

// ── Jobs form update ──────────────────────────────────────────────────────────

func (be BackendEditor) updateJobsForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.jobsForm)
	switch key.String() {
	case "j", "down":
		if be.jobsFormIdx < n-1 {
			be.jobsFormIdx++
		}
		be.activeField = be.jobsFormIdx
	case "k", "up":
		if be.jobsFormIdx > 0 {
			be.jobsFormIdx--
		}
		be.activeField = be.jobsFormIdx
	case "enter", " ":
		if be.jobsFormIdx < n {
			f := &be.jobsForm[be.jobsFormIdx]
			if f.Kind == KindSelect {
				be.dd.Open = true
				be.dd.OptIdx = f.SelIdx
			} else {
				return be.enterJobsFormInsert()
			}
		}
	case "H", "shift+left":
		if be.jobsFormIdx < n {
			f := &be.jobsForm[be.jobsFormIdx]
			if f.Kind == KindSelect {
				f.CyclePrev()
				if f.Key == "config_ref" {
					be.updateJobQueueTechOptions()
				}
			}
		}
	case "i", "a":
		if be.jobsFormIdx < n && be.jobsForm[be.jobsFormIdx].CanEditAsText() {
			return be.enterJobsFormInsert()
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
		be.saveJobsForm()
		be.jobsSubView = beViewList
	}
	be.saveJobsForm()
	return be, nil
}

func (be BackendEditor) enterJobsFormInsert() (BackendEditor, tea.Cmd) {
	if be.jobsFormIdx >= len(be.jobsForm) {
		return be, nil
	}
	f := be.jobsForm[be.jobsFormIdx]
	if !f.CanEditAsText() {
		return be, nil
	}
	be.internalMode = ModeInsert
	be.formInput.SetValue(f.TextInputValue())
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveJobsForm() {
	if be.jobsIdx >= len(be.jobQueues) {
		return
	}
	jq := &be.jobQueues[be.jobsIdx]
	jq.Name = fieldGet(be.jobsForm, "name")
	jq.Description = fieldGet(be.jobsForm, "description")
	cfgRef := fieldGet(be.jobsForm, "config_ref")
	if cfgRef == "(any)" || cfgRef == "" {
		jq.ConfigRef = ""
	} else {
		jq.ConfigRef = cfgRef
	}
	jq.Technology = fieldGet(be.jobsForm, "technology")
	jq.Concurrency = fieldGet(be.jobsForm, "concurrency")
	jq.MaxRetries = fieldGet(be.jobsForm, "max_retries")
	jq.RetryPolicy = fieldGet(be.jobsForm, "retry_policy")
	jq.DLQ = fieldGet(be.jobsForm, "dlq")
	ws := fieldGet(be.jobsForm, "worker_service")
	if ws != "(none)" {
		jq.WorkerService = ws
	} else {
		jq.WorkerService = ""
	}
	pd := fieldGet(be.jobsForm, "payload_dto")
	if pd != "(none)" {
		jq.PayloadDTO = pd
	} else {
		jq.PayloadDTO = ""
	}
}

// ── Jobs view ─────────────────────────────────────────────────────────────────

func (be BackendEditor) viewJobs(w int) []string {
	if be.jobsSubView == beViewList {
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Job Queues — a: add  d: delete  Enter: edit"), "")
		if len(be.jobQueues) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no job queues yet — press 'a' to add)"))
		} else {
			for i, jq := range be.jobQueues {
				name := jq.Name
				if name == "" {
					name = fmt.Sprintf("(queue #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == be.jobsIdx, "  ▶ ", name, jq.Technology))
			}
		}
		return lines
	}
	// Form view
	name := fieldGet(be.jobsForm, "name")
	if name == "" {
		name = "(new job queue)"
	}
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
	lines = append(lines, renderFormFields(w, be.jobsForm, be.jobsFormIdx, be.internalMode == ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
	return lines
}
