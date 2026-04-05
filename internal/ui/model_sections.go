package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// sectionEntry groups the editor-getter and update-handler for one section.
// To add a new pillar, register one entry here — no other switches need changing.
type sectionEntry struct {
	// editor returns the Editor interface value for the section (used for Mode,
	// HintLine, and View dispatch). May return nil for sections without a
	// delegated editor.
	editor func(m *Model) Editor
	// update injects cross-section context, calls the editor's Update, and
	// stores the result back on m. Returns the bubbletea Cmd (if any).
	update func(m *Model, msg tea.Msg) tea.Cmd
}

// sectionRegistry maps section IDs to their entry. Populated once at startup.
var sectionRegistry = buildSectionRegistry()

// sectionOrder is the canonical sequence of section IDs, used for resize
// propagation so that adding a new section only requires one registration.
var sectionOrder = []string{
	"describe", "backend", "data", "contracts",
	"frontend", "infrastructure", "crosscut", "realize",
}

// resizeAllEditors propagates a WindowSizeMsg to every registered editor so
// that text inputs resize immediately, regardless of which tab is active.
func resizeAllEditors(m *Model, wsz tea.WindowSizeMsg) {
	for _, id := range sectionOrder {
		if entry, ok := sectionRegistry[id]; ok {
			entry.update(m, wsz)
		}
	}
}

// buildSectionRegistry constructs the registry. The closures operate on the
// *Model parameter — no captured state — so the registry is safe to share.
func buildSectionRegistry() map[string]sectionEntry {
	return map[string]sectionEntry{
		"describe": {
			editor: func(m *Model) Editor { return m.descriptionEditor },
			update: func(m *Model, msg tea.Msg) tea.Cmd {
				var cmd tea.Cmd
				m.descriptionEditor, cmd = m.descriptionEditor.Update(msg)
				return cmd
			},
		},
		"backend": {
			editor: func(m *Model) Editor { return m.backendEditor },
			update: func(m *Model, msg tea.Msg) tea.Cmd {
				m.backendEditor.SetDomainNames(m.dataTabEditor.domainNames())
				m.backendEditor.SetDomainAttributes(m.dataTabEditor.DomainAttributeMap())
				m.backendEditor.SetDomainsByDB(m.dataTabEditor.DomainsByDB())
				m.backendEditor.SetDBSourceTypes(m.dataTabEditor.DBSourceTypeMap())
				m.backendEditor.SetDTONames(m.contractsEditor.DTONames())
				m.backendEditor.SetDTOProtocols(m.contractsEditor.DTOProtocols())
				m.backendEditor.SetEndpointNames(m.contractsEditor.EndpointNames())
				m.backendEditor.SetCacheAliases(m.dataTabEditor.CacheAliases())
				m.backendEditor.SetDBSourceAliases(m.dataTabEditor.AllDBSourceAliases())
				m.backendEditor.SetEnvironmentNames(m.infraEditor.EnvironmentNames())
				m.backendEditor.SetEnvironmentDefs(m.infraEditor.EnvironmentDefs())
				m.backendEditor.SetOrchestrator(m.infraEditor.PrimaryOrchestrator())
				m.backendEditor.SetMessagingCloudProvider(m.infraEditor.PrimaryCloudProvider())
				var cmd tea.Cmd
				m.backendEditor, cmd = m.backendEditor.Update(msg)
				return cmd
			},
		},
		"data": {
			editor: func(m *Model) Editor { return m.dataTabEditor },
			update: func(m *Model, msg tea.Msg) tea.Cmd {
				m.dataTabEditor.SetMigrationContext(m.backendEditor.Languages())
				m.dataTabEditor.SetServiceNames(m.backendEditor.ServiceNames())
				m.dataTabEditor.SetCloudProvider(m.infraEditor.PrimaryCloudProvider())
				m.dataTabEditor.SetEnvironmentNames(m.infraEditor.EnvironmentNames())
				m.dataTabEditor.SetDTONames(m.contractsEditor.DTONames())
				var cmd tea.Cmd
				m.dataTabEditor, cmd = m.dataTabEditor.Update(msg)
				// Refresh rate_limit_backend, health_deps, and repo references whenever data sources change.
				m.backendEditor.SetCacheAliases(m.dataTabEditor.CacheAliases())
				m.backendEditor.SetDBSourceAliases(m.dataTabEditor.AllDBSourceAliases())
				m.backendEditor.SetDomainAttributes(m.dataTabEditor.DomainAttributeMap())
				m.backendEditor.SetDomainsByDB(m.dataTabEditor.DomainsByDB())
				m.backendEditor.SetDBSourceTypes(m.dataTabEditor.DBSourceTypeMap())
				return cmd
			},
		},
		"contracts": {
			editor: func(m *Model) Editor { return m.contractsEditor },
			update: func(m *Model, msg tea.Msg) tea.Cmd {
				m.contractsEditor.SetDomains(m.dataTabEditor.domainNames())
				m.contractsEditor.SetDomainDefs(m.dataTabEditor.domains)
				m.contractsEditor.SetServices(m.backendEditor.ServiceNames())
				m.contractsEditor.SetServiceDefs(m.backendEditor.ServiceDefs())
				m.contractsEditor.SetAuthRoles(m.backendEditor.AuthRoleOptions())
				m.contractsEditor.SetWAFRateLimitStrategy(m.backendEditor.WAFRateLimitStrategy())
				var cmd tea.Cmd
				m.contractsEditor, cmd = m.contractsEditor.Update(msg)
				return cmd
			},
		},
		"frontend": {
			editor: func(m *Model) Editor { return m.frontendEditor },
			update: func(m *Model, msg tea.Msg) tea.Cmd {
				m.frontendEditor.SetAuthRoles(m.backendEditor.AuthRoleOptions())
				m.frontendEditor.SetBackendProtocols(m.backendEditor.CommProtocols(), m.backendEditor.ServiceFrameworks())
				m.frontendEditor.SetBackendAuthStrategy(m.backendEditor.AuthStrategy())
				m.frontendEditor.SetAvailableEndpoints(m.contractsEditor.EndpointNames())
				var cmd tea.Cmd
				m.frontendEditor, cmd = m.frontendEditor.Update(msg)
				return cmd
			},
		},
		"infrastructure": {
			editor: func(m *Model) Editor { return m.infraEditor },
			update: func(m *Model, msg tea.Msg) tea.Cmd {
				m.infraEditor.SetBackendLanguages(m.backendEditor.Languages())
				var cmd tea.Cmd
				m.infraEditor, cmd = m.infraEditor.Update(msg)
				// Propagate environment names to backend and data after infra updates.
				m.backendEditor.SetEnvironmentNames(m.infraEditor.EnvironmentNames())
				m.backendEditor.SetEnvironmentDefs(m.infraEditor.EnvironmentDefs())
				m.backendEditor.SetOrchestrator(m.infraEditor.PrimaryOrchestrator())
				m.backendEditor.SetMessagingCloudProvider(m.infraEditor.PrimaryCloudProvider())
				m.dataTabEditor.SetEnvironmentNames(m.infraEditor.EnvironmentNames())
				m.dataTabEditor.SetCloudProvider(m.infraEditor.PrimaryCloudProvider())
				return cmd
			},
		},
		"crosscut": {
			editor: func(m *Model) Editor { return m.crossCutEditor },
			update: func(m *Model, msg tea.Msg) tea.Cmd {
				m.crossCutEditor.SetTestingContext(
					m.backendEditor.Languages(),
					m.backendEditor.CommProtocols(),
					m.backendEditor.ArchPattern(),
					m.frontendEditor.Language(),
					m.frontendEditor.Framework(),
				)
				m.crossCutEditor.SetDocsContext(m.contractsEditor.ActiveDocProtocols())
				var cmd tea.Cmd
				m.crossCutEditor, cmd = m.crossCutEditor.Update(msg)
				return cmd
			},
		},
		"realize": {
			// editor is handled specially in activeEditor() to swap in the
			// RealizationScreen when it is active.
			editor: func(m *Model) Editor { return m.realizeEditor },
			update: func(m *Model, msg tea.Msg) tea.Cmd {
				var cmd tea.Cmd
				m.realizeEditor, cmd = m.realizeEditor.Update(msg)
				return cmd
			},
		},
	}
}
