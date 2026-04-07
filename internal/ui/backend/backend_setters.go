package backend

import (
	"strings"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── dependency-injection setters & appliers ──────────────────────────────────
//
// These methods inject cross-pillar data into the BackendEditor's dropdowns and
// multiselects, keeping backend_editor.go focused on struct layout, constructor,
// mode, and update dispatch.

// SetDomainNames stores domain names from the Data pillar for dropdown population.
func (be *BackendEditor) SetDomainNames(names []string) {
	be.DomainNames = names
}

// SetDomainAttributes stores domain attribute maps from the Data pillar for
// repo field selection dropdowns.
func (be *BackendEditor) SetDomainAttributes(attrs map[string][]string) {
	be.domainAttributes = attrs
}

// SetDomainsByDB stores the DB alias → domain names map for repo entity_ref filtering.
func (be *BackendEditor) SetDomainsByDB(m map[string][]string) {
	be.domainsByDB = m
}

// SetDBSourceTypes stores the DB alias → type map from the Data pillar for
// technology-aware op_type options in the repo editor.
func (be *BackendEditor) SetDBSourceTypes(types map[string]string) {
	be.dbSourceTypes = types
}

// SetCacheAliases stores the IsCache DB aliases from the Data pillar.
// Options are applied lazily when the rate_limit_backend dropdown is opened,
// not on every keypress, to avoid corrupting SelIdx during dropdown navigation.
func (be *BackendEditor) SetCacheAliases(aliases []string) {
	be.cacheAliases = aliases
	be.clearStaleCacheRef()
}

// SetDBSourceAliases stores all DB source aliases from the Data pillar and
// refreshes the health_deps multiselect options in the monolith CONFIG tab and
// in every per-service form (non-monolith arches).
func (be *BackendEditor) SetDBSourceAliases(aliases []string) {
	be.dbSourceAliases = aliases
	// Monolith: global health_deps lives in EnvFields (CONFIG tab).
	be.applyHealthDepsOptionsToFields(be.EnvFields)
	// Non-monolith: per-service health_deps lives in service forms.
	be.applyHealthDepsOptionsToFields(be.serviceEditor.form)
	for _, item := range be.serviceEditor.items {
		be.applyHealthDepsOptionsToFields(item)
	}
}

// applyHealthDepsOptionsToFields refreshes the health_deps multiselect options
// in a field slice, preserving any currently selected aliases by name.
// Also handles lazy restoration when options haven't been set yet but Value
// holds a comma-separated list of names (written by serviceFieldsFromDef).
func (be *BackendEditor) applyHealthDepsOptionsToFields(fields []core.Field) {
	for i := range fields {
		if fields[i].Key != "health_deps" {
			continue
		}
		// Collect previously selected names before replacing options.
		var selectedNames []string
		if len(fields[i].Options) > 0 {
			for _, idx := range fields[i].SelectedIdxs {
				if idx < len(fields[i].Options) {
					selectedNames = append(selectedNames, fields[i].Options[idx])
				}
			}
		} else if fields[i].Value != "" {
			// Options not yet populated — Value holds comma-sep names from manifest restore.
			for _, name := range strings.Split(fields[i].Value, ", ") {
				if name != "" {
					selectedNames = append(selectedNames, name)
				}
			}
		}
		fields[i].Options = be.dbSourceAliases
		fields[i].Value = ""
		fields[i].SelectedIdxs = nil
		for _, name := range selectedNames {
			for j, opt := range fields[i].Options {
				if opt == name {
					fields[i].SelectedIdxs = append(fields[i].SelectedIdxs, j)
					break
				}
			}
		}
		break
	}
}

// SetDTONames injects DTO names from the contracts tab and immediately refreshes
// all open comm/event/job forms that reference DTOs.
func (be *BackendEditor) SetDTONames(names []string) {
	be.availableDTOs = names
	be.applyDTONamesToForms()
}

// applyDTONamesToForms refreshes DTO-referencing dropdowns in all open comm,
// event, and job queue forms without resetting the current selection.
func (be *BackendEditor) applyDTONamesToForms() {
	dtoOpts, dtoPlaceholder := core.NoneOrPlaceholder(be.availableDTOs, "(no DTOs configured)")

	applyMultiDTO := func(fields []core.Field) {
		for i := range fields {
			if fields[i].Key != "payload_dto" && fields[i].Key != "response_dto" {
				continue
			}
			var selectedNames []string
			if len(fields[i].Options) > 0 {
				for _, idx := range fields[i].SelectedIdxs {
					if idx < len(fields[i].Options) {
						selectedNames = append(selectedNames, fields[i].Options[idx])
					}
				}
			} else if fields[i].Value != "" {
				selectedNames = strings.Split(fields[i].Value, ", ")
			}
			fields[i].Options = be.availableDTOs
			fields[i].SelectedIdxs = nil
			fields[i].Value = ""
			for _, name := range selectedNames {
				for j, opt := range fields[i].Options {
					if opt == name {
						fields[i].SelectedIdxs = append(fields[i].SelectedIdxs, j)
						break
					}
				}
			}
		}
	}
	applyMultiDTO(be.commEditor.form)
	for _, item := range be.commEditor.items {
		applyMultiDTO(item)
	}

	applySingleDTO := func(fields []core.Field) {
		for i := range fields {
			if fields[i].Key != "dto" {
				continue
			}
			cur := fields[i].Value
			fields[i].Kind = core.KindSelect
			fields[i].Options = dtoOpts
			fields[i].SelIdx = 0
			if len(dtoOpts) == 0 {
				fields[i].Value = dtoPlaceholder
				break
			}
			for j, o := range dtoOpts {
				if o == cur {
					fields[i].SelIdx = j
					fields[i].Value = o
					break
				}
			}
			if fields[i].Value != cur {
				fields[i].Value = dtoOpts[0]
			}
			break
		}
	}
	applyMultiDTO(be.eventEditor.form)
	for _, item := range be.eventEditor.items {
		applyMultiDTO(item)
		applySingleDTO(item)
	}
	applySingleDTO(be.eventEditor.form)

	// Jobs form payload_dto is core.KindSelect (single).
	for i := range be.jobsForm {
		if be.jobsForm[i].Key != "payload_dto" {
			continue
		}
		workerOpts, workerVal := core.NoneOrPlaceholder(be.availableDTOs, "(no DTOs configured)")
		cur := be.jobsForm[i].Value
		be.jobsForm[i].Options = workerOpts
		be.jobsForm[i].SelIdx = 0
		for j, o := range workerOpts {
			if o == cur {
				be.jobsForm[i].SelIdx = j
				be.jobsForm[i].Value = o
				break
			}
		}
		if be.jobsForm[i].Value != cur {
			be.jobsForm[i].Value = workerVal
		}
		break
	}
}

// applyServiceNamesToForms refreshes service-name dropdowns in all open comm,
// event, and job queue forms without resetting the current selection.
func (be *BackendEditor) applyServiceNamesToForms() {
	names := be.ServiceNames()
	svcOpts, svcPlaceholder := core.NoneOrPlaceholder(names, "(no services configured)")

	applyToFields := func(fields []core.Field) {
		for i := range fields {
			switch fields[i].Key {
			case "from", "to", "publisher_service", "consumer_service", "worker_service":
				cur := fields[i].Value
				fields[i].Kind = core.KindSelect
				fields[i].Options = svcOpts
				fields[i].SelIdx = 0
				if len(svcOpts) == 0 {
					fields[i].Value = svcPlaceholder
					continue
				}
				found := false
				for j, o := range svcOpts {
					if o == cur {
						fields[i].SelIdx = j
						fields[i].Value = o
						found = true
						break
					}
				}
				if !found {
					fields[i].Value = svcOpts[0]
				}
			}
		}
	}
	applyToFields(be.commEditor.form)
	for _, item := range be.commEditor.items {
		applyToFields(item)
	}
	applyToFields(be.eventEditor.form)
	for _, item := range be.eventEditor.items {
		applyToFields(item)
	}
	applyToFields(be.jobsForm)
}

// dtoProtocolToSerialization maps a DTO protocol name to the corresponding
// messaging serialization option, or "" if no mapping exists.
func dtoProtocolToSerialization(proto string) string {
	switch proto {
	case "Protobuf":
		return "Protobuf"
	case "Avro":
		return "Avro"
	case "MessagePack":
		return "MessagePack"
	case "REST/JSON":
		return "JSON"
	default:
		return ""
	}
}

// SetDTOProtocols injects the unique serialisation protocols used by DTOs in
// the Contracts pillar. When all DTOs share a single protocol that maps to a
// messaging serialization option (Protobuf, Avro, MessagePack), the messaging
// serialization field is updated to match. Mixed or unmappable protocols leave
// the current selection unchanged.
func (be *BackendEditor) SetDTOProtocols(protocols []string) {
	if core.StringSlicesEqual(be.dtoProtocols, protocols) {
		return
	}
	be.dtoProtocols = protocols

	// Determine a single dominant serialization suggestion.
	if len(protocols) != 1 {
		return // mixed or no DTOs — leave current selection
	}
	suggested := dtoProtocolToSerialization(protocols[0])
	if suggested == "" || suggested == "JSON" {
		return // no actionable mapping
	}

	for i := range be.MessagingFields {
		if be.MessagingFields[i].Key != "serialization" {
			continue
		}
		for j, opt := range be.MessagingFields[i].Options {
			if opt == suggested {
				be.MessagingFields[i].SelIdx = j
				be.MessagingFields[i].Value = suggested
				break
			}
		}
		break
	}
}

// SetEndpointNames injects endpoint names from the contracts tab for the API
// Gateway endpoints multiselect, preserving any existing selection by name.
func (be *BackendEditor) SetEndpointNames(names []string) {
	be.availableEndpoints = names
	for i := range be.APIGWFields {
		if be.APIGWFields[i].Key != "endpoints" {
			continue
		}
		// Collect currently selected names before replacing options.
		var selectedNames []string
		if len(be.APIGWFields[i].Options) > 0 {
			for _, idx := range be.APIGWFields[i].SelectedIdxs {
				if idx < len(be.APIGWFields[i].Options) {
					selectedNames = append(selectedNames, be.APIGWFields[i].Options[idx])
				}
			}
		} else if be.APIGWFields[i].Value != "" {
			selectedNames = strings.Split(be.APIGWFields[i].Value, ", ")
		}
		be.APIGWFields[i].Options = names
		be.APIGWFields[i].SelectedIdxs = nil
		for _, name := range selectedNames {
			for j, opt := range names {
				if opt == name {
					be.APIGWFields[i].SelectedIdxs = append(be.APIGWFields[i].SelectedIdxs, j)
					break
				}
			}
		}
		break
	}
}

// updateJobQueueTechOptions refreshes the technology options in the active jobs
// form based on the currently selected config_ref. Called after config_ref changes.
func (be *BackendEditor) updateJobQueueTechOptions() {
	lang := be.langForConfig(core.FieldGet(be.jobsForm, "config_ref"))
	var langs []string
	if lang != "" {
		langs = []string{lang}
	} else {
		langs = be.Languages()
	}
	opts, defaultVal := jobQueueTechOptions(langs)
	cur := core.FieldGet(be.jobsForm, "technology")
	for i := range be.jobsForm {
		if be.jobsForm[i].Key != "technology" {
			continue
		}
		be.jobsForm[i].Options = opts
		found := false
		for j, o := range opts {
			if o == cur {
				be.jobsForm[i].SelIdx = j
				be.jobsForm[i].Value = o
				found = true
				break
			}
		}
		if !found {
			be.jobsForm[i].SelIdx = 0
			be.jobsForm[i].Value = defaultVal
		}
		break
	}
}

// applyStackConfigNamesToServices updates the config_ref dropdown in all service
// and job queue forms to reflect the current set of stack config names. Called
// whenever stack configs are added, renamed, or deleted.
func (be *BackendEditor) applyStackConfigNamesToServices() {
	var names []string
	for _, item := range be.stackConfigEditor.items {
		if n := core.FieldGet(item, "name"); n != "" {
			names = append(names, n)
		}
	}
	// Service forms use core.NoneOrPlaceholder: "(none)" prefix when configs exist.
	opts, placeholder := core.NoneOrPlaceholder(names, "(no configs defined)")
	applyOpts := func(fields []core.Field) {
		for i := range fields {
			if fields[i].Key != "config_ref" {
				continue
			}
			fields[i].Options = opts
			found := false
			for j, o := range opts {
				if o == fields[i].Value {
					fields[i].SelIdx = j
					found = true
					break
				}
			}
			if !found {
				fields[i].Value = placeholder
				fields[i].SelIdx = 0
			}
			break
		}
	}
	applyOpts(be.serviceEditor.form)
	for _, item := range be.serviceEditor.items {
		applyOpts(item)
	}

	// Job queue forms use "(any)" as first option when configs exist.
	var jobsOpts []string
	var jobsPlaceholder string
	if len(names) > 0 {
		jobsOpts = append([]string{"(any)"}, names...)
		jobsPlaceholder = "(any)"
	} else {
		jobsOpts = []string{"(no configs defined)"}
		jobsPlaceholder = "(no configs defined)"
	}
	applyJobsOpts := func(fields []core.Field) {
		for i := range fields {
			if fields[i].Key != "config_ref" {
				continue
			}
			fields[i].Options = jobsOpts
			found := false
			for j, o := range jobsOpts {
				if o == fields[i].Value {
					fields[i].SelIdx = j
					found = true
					break
				}
			}
			if !found {
				fields[i].Value = jobsPlaceholder
				fields[i].SelIdx = 0
			}
			break
		}
	}
	applyJobsOpts(be.jobsForm)
}

// SetEnvironmentNames injects environment names from the infra tab so that
// the monolith env tab, service forms, and messaging broker config can show
// an environment selector dropdown.
func (be *BackendEditor) SetEnvironmentNames(names []string) {
	be.environmentNames = names
	// Refresh the monolith shared environment dropdown in the env tab.
	be.applyEnvNamesToServiceFields(be.EnvFields)
	// Refresh environment dropdowns in the active form and all stored items.
	be.applyEnvNamesToServiceFields(be.serviceEditor.form)
	for _, item := range be.serviceEditor.items {
		be.applyEnvNamesToServiceFields(item)
	}
	// Refresh environment dropdown in the messaging broker config.
	be.applyEnvNamesToServiceFields(be.MessagingFields)
	// Refresh environment dropdown in the API gateway config.
	be.applyEnvNamesToServiceFields(be.APIGWFields)
	// Clear stale environment refs in committed domain data.
	be.clearStaleEnvRefs()
}

// SetEnvironmentDefs injects full environment definitions so the API Gateway
// technology options can be filtered by the selected environment's orchestrator
// and cloud provider.
func (be *BackendEditor) SetEnvironmentDefs(defs []manifest.ServerEnvironmentDef) {
	be.environmentDefs = defs
	be.updateAPIGWTechOptions()
}

// updateAPIGWTechOptions re-filters the API gateway technology options based
// on the currently selected environment's orchestrator and cloud provider.
func (be *BackendEditor) updateAPIGWTechOptions() {
	envVal := core.FieldGet(be.APIGWFields, "environment")
	var orch, cloud string
	for _, d := range be.environmentDefs {
		if d.Name == envVal {
			orch = d.Orchestrator
			cloud = d.CloudProvider
			break
		}
	}
	opts := apiGWTechOptionsForEnv(orch, cloud)
	cur := core.FieldGet(be.APIGWFields, "technology")
	for i := range be.APIGWFields {
		if be.APIGWFields[i].Key != "technology" {
			continue
		}
		be.APIGWFields[i].Options = opts
		// Keep current value when still valid; otherwise reset to first option.
		valid := false
		for j, o := range opts {
			if o == cur {
				be.APIGWFields[i].SelIdx = j
				valid = true
				break
			}
		}
		if !valid && len(opts) > 0 {
			be.APIGWFields[i].SelIdx = 0
			be.APIGWFields[i].Value = opts[0]
		}
		break
	}
}

// SetMessagingCloudProvider injects the primary cloud provider from infra so
// that the messaging deployment dropdown shows cloud-specific managed options.
func (be *BackendEditor) SetMessagingCloudProvider(cp string) {
	if be.cloudProvider == cp {
		return
	}
	be.cloudProvider = cp
	be.refreshMessagingDeploymentOptions()
}

// SetOrchestrator injects the primary orchestrator from infra for narrowing
// service discovery options. A no-op when unchanged.
func (be *BackendEditor) SetOrchestrator(orch string) {
	if be.orchestrator == orch {
		return
	}
	be.orchestrator = orch
	be.updateServiceDiscoveryOptions()
}

// applyEnvNamesToServiceFields sets the environment field options in a field slice.
func (be *BackendEditor) applyEnvNamesToServiceFields(fields []core.Field) {
	opts, val := core.NoneOrPlaceholder(be.environmentNames, "(no environments configured)")
	for i := range fields {
		if fields[i].Key != "environment" {
			continue
		}
		fields[i].Options = opts
		// Keep current value when still valid.
		found := false
		for j, o := range opts {
			if o == fields[i].Value {
				fields[i].SelIdx = j
				found = true
				break
			}
		}
		if !found {
			fields[i].Value = val
			fields[i].SelIdx = 0
		}
		break
	}
}

// ── stale reference clearing ─────────────────────────────────────────────────

// clearStaleEnvRefs clears environment references in committed Services,
// Messaging, and API Gateway when the referenced environment no longer exists.
func (be *BackendEditor) clearStaleEnvRefs() {
	envSet := make(map[string]bool, len(be.environmentNames))
	for _, n := range be.environmentNames {
		envSet[n] = true
	}
	for i := range be.Services {
		if be.Services[i].Environment != "" && !envSet[be.Services[i].Environment] {
			be.Services[i].Environment = ""
		}
	}
}

// ClearStaleServiceRefs removes references to deleted services from CommLinks,
// Events, and JobQueues. Called after a service is deleted.
func (be *BackendEditor) ClearStaleServiceRefs() {
	svcSet := make(map[string]bool, len(be.Services))
	for _, s := range be.Services {
		if s.Name != "" {
			svcSet[s.Name] = true
		}
	}
	for i := range be.CommLinks {
		if be.CommLinks[i].From != "" && !svcSet[be.CommLinks[i].From] {
			be.CommLinks[i].From = ""
		}
		if be.CommLinks[i].To != "" && !svcSet[be.CommLinks[i].To] {
			be.CommLinks[i].To = ""
		}
	}
	for i := range be.Events {
		if be.Events[i].PublisherService != "" && !svcSet[be.Events[i].PublisherService] {
			be.Events[i].PublisherService = ""
		}
		if be.Events[i].ConsumerService != "" && !svcSet[be.Events[i].ConsumerService] {
			be.Events[i].ConsumerService = ""
		}
	}
	for i := range be.jobQueues {
		if be.jobQueues[i].WorkerService != "" && !svcSet[be.jobQueues[i].WorkerService] {
			be.jobQueues[i].WorkerService = ""
		}
	}
}

// clearStaleCacheRef clears the rate_limit_backend security field when the
// referenced cache alias no longer exists.
func (be *BackendEditor) clearStaleCacheRef() {
	cur := core.FieldGet(be.securityFields, "rate_limit_backend")
	if cur == "" {
		return
	}
	for _, alias := range be.cacheAliases {
		if alias == cur {
			return
		}
	}
	be.securityFields = core.SetFieldValue(be.securityFields, "rate_limit_backend", "")
}
